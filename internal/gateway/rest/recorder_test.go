package rest

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/repository/mock"
	otelcollector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

// collector is an OTLP/HTTP collector that keeps received spans.
type collector struct {
	mu    sync.Mutex
	spans []*trace.Span
}

func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	req := &otelcollector.ExportTraceServiceRequest{}
	if err := proto.Unmarshal(body, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	for _, rs := range req.ResourceSpans {
		for _, ss := range rs.ScopeSpans {
			c.spans = append(c.spans, ss.Spans...)
		}
	}
	c.mu.Unlock()

	res, _ := proto.Marshal(&otelcollector.ExportTraceServiceResponse{})
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(res)
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

	body, err := proto.Marshal(&otelcollector.ExportTraceServiceRequest{
		ResourceSpans: []*trace.ResourceSpans{{
			ScopeSpans: []*trace.ScopeSpans{{
				Spans: []*trace.Span{{
					Name: "op",
					Attributes: []*common.KeyValue{
						{Key: "a", Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: "1"}}},
						{Key: "secret", Value: &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: "token"}}},
					},
				}},
			}},
		}},
	})
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
			if err := proto.Unmarshal(rec.Body.Bytes(), &otelcollector.ExportTraceServiceResponse{}); err != nil {
				t.Errorf("response is not protobuf: %v", err)
			}

			if len(c.spans) != 1 {
				t.Fatalf("collector spans = %d, want 1", len(c.spans))
			}
			if attrs := c.spans[0].Attributes; len(attrs) != 1 || attrs[0].Key != "a" {
				t.Errorf("collector span attributes = %v, want only `a`", attrs)
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
		{"json is not supported", "application/json", []byte(`{"resourceSpans":[]}`), http.StatusUnsupportedMediaType},
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
