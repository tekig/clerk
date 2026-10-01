package otelproxy

import (
	"context"
	"encoding/hex"
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
	otelcollector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

// collector is an OTLP/HTTP collector that keeps received requests.
type collector struct {
	mu       sync.Mutex
	requests []*otelcollector.ExportTraceServiceRequest
	status   int
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	req := &otelcollector.ExportTraceServiceRequest{}
	if err := proto.Unmarshal(body, req); err != nil {
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

	res, _ := proto.Marshal(&otelcollector.ExportTraceServiceResponse{})
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

func keyValue(key string, value *common.AnyValue) *common.KeyValue {
	return &common.KeyValue{Key: key, Value: value}
}

func keys(attrs []*common.KeyValue) []string {
	result := []string{}
	for _, kv := range attrs {
		result = append(result, kv.Key)
	}
	return result
}

func newResourceSpans(service string, spans ...*trace.Span) []*trace.ResourceSpans {
	return []*trace.ResourceSpans{{
		Resource: &resource.Resource{Attributes: []*common.KeyValue{
			keyValue("service.name", stringValue(service)),
		}},
		ScopeSpans: []*trace.ScopeSpans{{Spans: spans}},
	}}
}

func TestProxy_Grep(t *testing.T) {
	env := newProxyEnv(t, []ConfigRule{
		{Key: []string{"prefix:secret."}, Strategy: "remove"},
		{Key: []string{"prefix:big."}, Strategy: "unlink"},
	})

	unchanged := &trace.Span{
		Name: "Unchanged",
		Attributes: []*common.KeyValue{
			keyValue("a", stringValue("1")),
			keyValue("b", stringValue("2")),
		},
	}
	changed := &trace.Span{
		Name:    "Changed",
		TraceId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Attributes: []*common.KeyValue{
			keyValue("secret.token", stringValue("token")),
			keyValue("a", stringValue("1")),
			keyValue("big.string", stringValue("body")),
			keyValue("big.int", &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: 42}}),
			keyValue("big.bool", &common.AnyValue{Value: &common.AnyValue_BoolValue{BoolValue: true}}),
			keyValue("big.double", &common.AnyValue{Value: &common.AnyValue_DoubleValue{DoubleValue: 1.5}}),
			keyValue("big.array", &common.AnyValue{Value: &common.AnyValue_ArrayValue{ArrayValue: &common.ArrayValue{}}}),
			keyValue("b", stringValue("2")),
		},
	}

	if _, err := env.proxy.Grep(context.Background(), newResourceSpans("svc", unchanged, changed)); err != nil {
		t.Fatalf("grep: %v", err)
	}

	// Spans sent to the collector
	if len(env.collector.requests) != 1 {
		t.Fatalf("collector requests = %d, want 1", len(env.collector.requests))
	}
	spans := env.collector.requests[0].ResourceSpans[0].ScopeSpans[0].Spans
	if got := keys(spans[0].Attributes); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("unchanged span attributes = %v", got)
	}
	if got := keys(spans[1].Attributes); !slices.Equal(got, []string{"a", "b", "event_url"}) {
		t.Errorf("changed span attributes = %v", got)
	}
	eventURL := spans[1].Attributes[2].GetValue().GetStringValue()
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
		{Key: entity.MetaTraceID, Value: &pb.Attribute_AsString{AsString: hex.EncodeToString(changed.TraceId)}},
		{Key: entity.MetaSpanID, Value: &pb.Attribute_AsString{AsString: hex.EncodeToString(changed.SpanId)}},
		{Key: "big.string", Value: &pb.Attribute_AsString{AsString: "body"}},
		{Key: "big.int", Value: &pb.Attribute_AsInt64{AsInt64: 42}},
		{Key: "big.bool", Value: &pb.Attribute_AsBool{AsBool: true}},
		{Key: "big.double", Value: &pb.Attribute_AsDouble{AsDouble: 1.5}},
		{Key: "big.array", Value: &pb.Attribute_AsString{AsString: "unsupport attribute `*v1.AnyValue_ArrayValue`"}},
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

	span := &trace.Span{Name: "op", Attributes: []*common.KeyValue{keyValue("body", stringValue("v"))}}
	res := []*trace.ResourceSpans{{ScopeSpans: []*trace.ScopeSpans{{Spans: []*trace.Span{span}}}}}
	if _, err := env.proxy.Grep(context.Background(), res); err != nil {
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

	span := &trace.Span{Name: "op", Attributes: []*common.KeyValue{keyValue("a", stringValue("1"))}}
	if _, err := env.proxy.Grep(context.Background(), newResourceSpans("svc", span)); err == nil {
		t.Errorf("grep with failed collector: expected error")
	}
	if events := env.events(t); len(events) != 0 {
		t.Errorf("recorded events = %d, want 0", len(events))
	}
}
