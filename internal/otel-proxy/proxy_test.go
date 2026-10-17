package otelproxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/golang/snappy"
	"github.com/tekig/clerk/internal/block2"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/repository/mock"
	"github.com/tekig/clerk/internal/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

// collector is an OTLP/HTTP collector that keeps received requests.
type collector struct {
	mu       sync.Mutex
	requests []ptraceotlp.ExportRequest
	status   int
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	req := ptraceotlp.NewExportRequest()
	if err := req.UnmarshalProto(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	c.requests = append(c.requests, req)
	status := c.status
	c.mu.Unlock()

	if status != 0 {
		http.Error(w, "fail", status)
		return
	}

	res, _ := ptraceotlp.NewExportResponse().MarshalProto()
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(res)
}

type proxyEnv struct {
	proxy     *Proxy
	recorder  *recorder.Recorder
	storage   *mem.Storage
	collector *collector
}

func newProxyEnv(t *testing.T, rules []ConfigRule) *proxyEnv {
	t.Helper()

	c := &collector{}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	searcher := mock.NewMockSearcher(gomock.NewController(t))
	searcher.EXPECT().AppendBlock(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	storage := mem.NewStorage()
	r, err := recorder.NewRecorder(storage, searcher)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	p, err := New(Config{
		Target:          srv.URL,
		Recorder:        r,
		FormatURL:       "http://searcher/events/%s",
		DefaultStrategy: "keep",
		Rules:           rules,
	})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	return &proxyEnv{proxy: p, recorder: r, storage: storage, collector: c}
}

// events stops the recorder and reads all recorded events from the storage.
func (e *proxyEnv) events(t *testing.T) []*pb.Event {
	t.Helper()

	if err := e.recorder.Shutdown(); err != nil {
		t.Fatalf("shutdown recorder: %v", err)
	}

	ctx := context.Background()
	blocks, err := e.storage.Blocks(ctx)
	if err != nil {
		t.Fatalf("blocks: %v", err)
	}

	var events []*pb.Event
	for _, block := range blocks {
		r, err := e.storage.Read(ctx, block, entity.NameData)
		if err != nil {
			t.Fatalf("read data: %v", err)
		}

		// Chunks are sequential snappy streams, they are read as one stream
		snap := snappy.NewReader(r)
		for {
			event := &pb.Event{}
			if err := block2.Decode(event, snap); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Fatalf("decode event: %v", err)
			}
			events = append(events, event)
		}
		r.Close()
	}

	return events
}

func keys(attrs pcommon.Map) []string {
	result := []string{}
	for k := range attrs.All() {
		result = append(result, k)
	}
	return result
}

// newTraces returns traces with a single resource, the service is not set if empty,
// and its spans to fill.
func newTraces(service string, spans int) (ptrace.Traces, ptrace.SpanSlice) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	if service != "" {
		rs.Resource().Attributes().PutStr("service.name", service)
	}

	slice := rs.ScopeSpans().AppendEmpty().Spans()
	for range spans {
		slice.AppendEmpty()
	}

	return td, slice
}

// spans returns spans received by the collector in the request.
func (c *collector) spans(t *testing.T, request int) ptrace.SpanSlice {
	t.Helper()

	if len(c.requests) <= request {
		t.Fatalf("collector requests = %d, want > %d", len(c.requests), request)
	}

	return c.requests[request].Traces().ResourceSpans().At(0).ScopeSpans().At(0).Spans()
}

func TestProxy_Grep(t *testing.T) {
	env := newProxyEnv(t, []ConfigRule{
		{Key: []string{"prefix:secret."}, Strategy: "remove"},
		{Key: []string{"prefix:big."}, Strategy: "unlink"},
	})

	td, spans := newTraces("svc", 2)

	unchanged := spans.At(0)
	unchanged.SetName("Unchanged")
	unchanged.Attributes().PutStr("a", "1")
	unchanged.Attributes().PutStr("b", "2")

	changed := spans.At(1)
	changed.SetName("Changed")
	changed.SetTraceID(pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	changed.SetSpanID(pcommon.SpanID{1, 2, 3, 4, 5, 6, 7, 8})
	changed.Attributes().PutStr("secret.token", "token")
	changed.Attributes().PutStr("a", "1")
	changed.Attributes().PutStr("big.string", "body")
	changed.Attributes().PutInt("big.int", 42)
	changed.Attributes().PutBool("big.bool", true)
	changed.Attributes().PutDouble("big.double", 1.5)
	changed.Attributes().PutEmptySlice("big.array")
	changed.Attributes().PutStr("b", "2")
	traceID, spanID := changed.TraceID().String(), changed.SpanID().String()

	if _, err := env.proxy.Grep(context.Background(), td); err != nil {
		t.Fatalf("grep: %v", err)
	}

	// Spans sent to the collector
	if len(env.collector.requests) != 1 {
		t.Fatalf("collector requests = %d, want 1", len(env.collector.requests))
	}
	got := env.collector.spans(t, 0)
	if got := keys(got.At(0).Attributes()); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("unchanged span attributes = %v", got)
	}
	if got := keys(got.At(1).Attributes()); !slices.Equal(got, []string{"a", "b", "event_url"}) {
		t.Errorf("changed span attributes = %v", got)
	}
	eventURLValue, _ := got.At(1).Attributes().Get("event_url")
	eventURL := eventURLValue.Str()
	if !strings.HasPrefix(eventURL, "http://searcher/events/") {
		t.Fatalf("event_url = %q", eventURL)
	}

	// Event recorded for the changed span only
	events := env.events(t)
	if len(events) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(events))
	}
	event := events[0]

	if id := strings.TrimPrefix(eventURL, "http://searcher/events/"); id != uuid.UUID(event.Id).String() {
		t.Errorf("event_url id %s, recorded id %s", id, uuid.UUID(event.Id).String())
	}

	want := []*pb.Attribute{
		{Key: entity.MetaServiceName, Value: &pb.Attribute_AsString{AsString: "svc"}},
		{Key: entity.MetaSpanName, Value: &pb.Attribute_AsString{AsString: "Changed"}},
		{Key: entity.MetaTraceID, Value: &pb.Attribute_AsString{AsString: traceID}},
		{Key: entity.MetaSpanID, Value: &pb.Attribute_AsString{AsString: spanID}},
		{Key: "big.string", Value: &pb.Attribute_AsString{AsString: "body"}},
		{Key: "big.int", Value: &pb.Attribute_AsInt64{AsInt64: 42}},
		{Key: "big.bool", Value: &pb.Attribute_AsBool{AsBool: true}},
		{Key: "big.double", Value: &pb.Attribute_AsDouble{AsDouble: 1.5}},
		{Key: "big.array", Value: &pb.Attribute_AsString{AsString: "unsupport attribute `Slice`"}},
	}
	if len(event.Attributes) != len(want) {
		t.Fatalf("event attributes = %v, want %v", event.Attributes, want)
	}
	for i := range want {
		if !proto.Equal(event.Attributes[i], want[i]) {
			t.Errorf("event attribute #%d = %v, want %v", i, event.Attributes[i], want[i])
		}
	}
}

func TestProxy_GrepUnknownService(t *testing.T) {
	env := newProxyEnv(t, []ConfigRule{{Key: []string{"equals:body"}, Strategy: "unlink"}})

	td, spans := newTraces("", 1)
	spans.At(0).SetName("op")
	spans.At(0).Attributes().PutStr("body", "v")
	if _, err := env.proxy.Grep(context.Background(), td); err != nil {
		t.Fatalf("grep: %v", err)
	}

	events := env.events(t)
	if len(events) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(events))
	}
	if got := events[0].Attributes[0].GetAsString(); got != entity.MetaValueUnknown {
		t.Errorf("service name = %q, want %q", got, entity.MetaValueUnknown)
	}
}

func TestProxy_GrepCollectorError(t *testing.T) {
	env := newProxyEnv(t, nil)
	env.collector.status = http.StatusServiceUnavailable

	td, spans := newTraces("svc", 1)
	spans.At(0).SetName("op")
	spans.At(0).Attributes().PutStr("a", "1")
	if _, err := env.proxy.Grep(context.Background(), td); err == nil {
		t.Errorf("grep with failed collector: expected error")
	}
	if events := env.events(t); len(events) != 0 {
		t.Errorf("recorded events = %d, want 0", len(events))
	}
}
