package rest

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tekig/clerk/internal/bytes"
	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	tracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
)

var poolBuf = sync.Pool{
	New: func() any {
		return make([]byte, 0, 4*1024*1024)
	},
}

type Recorder struct {
	otelProxy  *otelproxy.Proxy
	httpServer *echo.Echo
	marshalers map[string]runtime.Marshaler
	address    string
}

type RecorderConfig struct {
	OTELProxy   *otelproxy.Proxy
	HTTPAddress string
}

func NewRecorder(config RecorderConfig) (*Recorder, error) {
	r := &Recorder{
		otelProxy:  config.OTELProxy,
		httpServer: echo.New(),
		marshalers: map[string]runtime.Marshaler{
			runtime.MIMEWildcard:     &runtime.HTTPBodyMarshaler{},
			"application/x-protobuf": &runtime.ProtoMarshaller{},
			"application/protobuf":   &runtime.ProtoMarshaller{},
		},
		address: config.HTTPAddress,
	}

	r.httpServer.HideBanner = true
	r.httpServer.Use(
		middleware.Recover(),
		middleware.Logger(),
	)

	r.httpServer.POST("/v1/traces", r.Export)

	return r, nil
}

func (r *Recorder) Run() error {
	return r.httpServer.Start(r.address)
}

func (g *Recorder) Shutdown() error {
	return g.httpServer.Shutdown(context.TODO())
}

func (r *Recorder) Export(c echo.Context) error {
	buf := poolBuf.Get().([]byte)
	defer func() {
		// put in buf final slice
		poolBuf.Put(buf[:0])
	}()

	if c.Request().ContentLength > 0 {
		buf = bytes.Resize(buf, int(c.Request().ContentLength))[:0]
	}

	buf, err := ReadAll(c.Request().Body, buf)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	inboundMarshaler, outboundMarshaler := r.marshaler(c.Request())

	var protoReq tracev1.ExportTraceServiceRequest
	if err := inboundMarshaler.Unmarshal(buf, &protoReq); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	resp, err := r.otelProxy.Grep(c.Request().Context(), protoReq.ResourceSpans)
	if err != nil {
		return fmt.Errorf("grep: %w", err)
	}

	if err := outboundMarshaler.NewEncoder(c.Response().Writer).Encode(resp); err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return nil
}

func (r *Recorder) marshaler(req *http.Request) (inbound runtime.Marshaler, outbound runtime.Marshaler) {
	for _, accept := range req.Header[echo.HeaderAccept] {
		if m, ok := r.marshalers[accept]; ok {
			outbound = m
			break
		}
	}

	for _, content := range req.Header[echo.HeaderContentType] {
		contentType, _, err := mime.ParseMediaType(content)
		if err != nil {
			continue
		}

		if m, ok := r.marshalers[contentType]; ok {
			inbound = m
			break
		}
	}

	if inbound == nil {
		inbound = r.marshalers[runtime.MIMEWildcard]
	}
	if outbound == nil {
		outbound = inbound
	}

	return inbound, outbound
}

func ReadAll(r io.Reader, b []byte) ([]byte, error) {
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}

		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
	}
}
