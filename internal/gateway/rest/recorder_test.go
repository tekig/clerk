package rest

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/repository/mock"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/mock/gomock"
)

// collector is an OTLP/HTTP collector that keeps received spans.
type collector struct {
	mu    sync.Mutex
	spans []ptrace.Span
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	req := ptraceotlp.NewExportRequest()
	if err := req.UnmarshalProto(body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	for _, rs := range req.Traces().ResourceSpans().All() {
		for _, ss := range rs.ScopeSpans().All() {
			for _, span := range ss.Spans().All() {
				c.spans = append(c.spans, span)
			}
		}
	}
	c.mu.Unlock()

	res, _ := ptraceotlp.NewExportResponse().MarshalProto()
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(res)
}

// onlyA reports whether attrs keep only `a`, other attributes are removed by the proxy.
func onlyA(attrs pcommon.Map) bool {
	_, ok := attrs.Get("a")

	return ok && attrs.Len() == 1
}

func newGateway(t *testing.T) (*Recorder, *collector) {
	t.Helper()

	c := &collector{}
	srv := httptest.NewServer(c)
	t.Cleanup(srv.Close)

	searcher := mock.NewMockSearcher(gomock.NewController(t))
	searcher.EXPECT().AppendBlock(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	r, err := recorder.NewRecorder(mem.NewStorage(), searcher)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	t.Cleanup(func() {
		r.Shutdown()
	})

	proxy, err := otelproxy.New(otelproxy.Config{
		Target:          srv.URL,
		Recorder:        r,
		FormatURL:       "http://searcher/events/%s",
		DefaultStrategy: "keep",
		Rules: []otelproxy.ConfigRule{
			{Key: []string{"equals:secret"}, Strategy: "remove"},
		},
	})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	g, err := NewRecorder(RecorderConfig{OTELProxy: proxy, HTTPAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}

	return g, c
}

func exportBody(t *testing.T) []byte {
	t.Helper()

	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("op")
	span.Attributes().PutStr("a", "1")
	span.Attributes().PutStr("secret", "token")

	body, err := ptraceotlp.NewExportRequestFromTraces(td).MarshalProto()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return body
}

func TestRecorder_Export(t *testing.T) {
	gzipBody := func(body []byte) []byte {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write(body)
		w.Close()
		return buf.Bytes()
	}

	tests := []struct {
		name     string
		gzip     bool
		mimeType string
	}{
		{"protobuf", false, "application/x-protobuf"},
		{"protobuf alias", false, "application/protobuf"},
		{"gzip", true, "application/x-protobuf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, c := newGateway(t)

			body := exportBody(t)
			if tt.gzip {
				body = gzipBody(body)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
			req.Header.Set("Content-Type", tt.mimeType)
			if tt.gzip {
				req.Header.Set("Content-Encoding", "gzip")
			}
			rec := httptest.NewRecorder()
			g.httpServer.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != tt.mimeType {
				t.Errorf("response content type = %q, want %q", ct, tt.mimeType)
			}
			if err := ptraceotlp.NewExportResponse().UnmarshalProto(rec.Body.Bytes()); err != nil {
				t.Errorf("response is not protobuf: %v", err)
			}

			if len(c.spans) != 1 {
				t.Fatalf("collector spans = %d, want 1", len(c.spans))
			}
			if attrs := c.spans[0].Attributes(); !onlyA(attrs) {
				t.Errorf("collector span attributes = %v, want only `a`", attrs.AsRaw())
			}
		})
	}
}

func TestRecorder_ExportErrors(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		body     []byte
		status   int
	}{
		{"invalid body", "application/x-protobuf", []byte("not a protobuf"), http.StatusBadRequest},
		{"invalid json", "application/json", []byte(`{"resourceSpans":`), http.StatusBadRequest},
		{"json id is not hex", "application/json", []byte(`{"resourceSpans":[{"scopeSpans":[{"spans":[{"traceId":"zz"}]}]}]}`), http.StatusBadRequest},
		{"text is not supported", "text/plain", []byte(`{"resourceSpans":[]}`), http.StatusUnsupportedMediaType},
		{"no content type", "", nil, http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, c := newGateway(t)

			body := tt.body
			if body == nil {
				body = exportBody(t)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
			if tt.mimeType != "" {
				req.Header.Set("Content-Type", tt.mimeType)
			}
			rec := httptest.NewRecorder()
			g.httpServer.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d, body %s", rec.Code, tt.status, rec.Body.String())
			}
			if len(c.spans) != 0 {
				t.Errorf("collector spans = %d, want 0", len(c.spans))
			}
		})
	}
}

func TestRecorder_ExportJSON(t *testing.T) {
	g, c := newGateway(t)

	// OTLP/JSON: ids are hex, enums are numbers
	body := `{"resourceSpans":[{"scopeSpans":[{"spans":[{
		"traceId":"5b8efff798038103d269b633813fc60c",
		"spanId":"eee19b7ec3c1b174",
		"parentSpanId":"eee19b7ec3c1b173",
		"name":"op",
		"kind":2,
		"startTimeUnixNano":"1544712660000000000",
		"links":[{"traceId":"5b8efff798038103d269b633813fc60d","spanId":"eee19b7ec3c1b175"}],
		"attributes":[
			{"key":"a","value":{"stringValue":"1"}},
			{"key":"secret","value":{"stringValue":"token"}}
		]
	}]}]}]}`

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	g.httpServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("response content type = %q, want application/json", ct)
	}
	if err := ptraceotlp.NewExportResponse().UnmarshalJSON(rec.Body.Bytes()); err != nil {
		t.Errorf("response is not json: %v", err)
	}

	if len(c.spans) != 1 {
		t.Fatalf("collector spans = %d, want 1", len(c.spans))
	}

	span := c.spans[0]
	link := span.Links().At(0)
	for want, got := range map[string]string{
		"5b8efff798038103d269b633813fc60c": span.TraceID().String(),
		"eee19b7ec3c1b174":                 span.SpanID().String(),
		"eee19b7ec3c1b173":                 span.ParentSpanID().String(),
		"5b8efff798038103d269b633813fc60d": link.TraceID().String(),
		"eee19b7ec3c1b175":                 link.SpanID().String(),
	} {
		if got != want {
			t.Errorf("id = %s, want %s", got, want)
		}
	}
	if span.Kind() != ptrace.SpanKindServer {
		t.Errorf("kind = %v, want server", span.Kind())
	}
	if span.StartTimestamp() != 1544712660000000000 {
		t.Errorf("start time = %d", span.StartTimestamp())
	}
	if attrs := span.Attributes(); !onlyA(attrs) {
		t.Errorf("collector span attributes = %v, want only `a`", attrs.AsRaw())
	}
}

func TestRecorder_TrailingSlash(t *testing.T) {
	g, c := newGateway(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces/", bytes.NewReader(exportBody(t)))
	req.Header.Set("Content-Type", "application/x-protobuf")
	rec := httptest.NewRecorder()
	g.httpServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if len(c.spans) != 1 {
		t.Errorf("collector spans = %d, want 1", len(c.spans))
	}
}

func TestReadAll(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 1000)

	for _, capacity := range []int{0, 7, len(data), 2 * len(data)} {
		got, err := ReadAll(bytes.NewReader(data), make([]byte, 0, capacity))
		if err != nil {
			t.Fatalf("read all: %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("cap %d: read %d bytes, want %d", capacity, len(got), len(data))
		}
	}
}
