package rest

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tekig/clerk/internal/logger"
	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	"github.com/tekig/clerk/internal/x/xbytes"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

var poolBuf = sync.Pool{
	New: func() any {
		return make([]byte, 0, 4*1024*1024)
	},
}

// codec is OTLP/HTTP encoding of the export request and response.
type codec struct {
	unmarshal func(ptraceotlp.ExportRequest, []byte) error
	marshal   func(ptraceotlp.ExportResponse) ([]byte, error)
}

var (
	protoCodec = codec{
		unmarshal: ptraceotlp.ExportRequest.UnmarshalProto,
		marshal:   ptraceotlp.ExportResponse.MarshalProto,
	}
	jsonCodec = codec{
		unmarshal: ptraceotlp.ExportRequest.UnmarshalJSON,
		marshal:   ptraceotlp.ExportResponse.MarshalJSON,
	}
)

type Recorder struct {
	otelProxy  *otelproxy.Proxy
	httpServer *echo.Echo
	codecs     map[string]codec
	address    string
}

type RecorderConfig struct {
	OTELProxy      *otelproxy.Proxy
	HTTPAddress    string
	MaxConcurrency int
}

func NewRecorder(config RecorderConfig) (*Recorder, error) {
	r := &Recorder{
		otelProxy:  config.OTELProxy,
		httpServer: echo.New(),
		// OTLP/HTTP binary protobuf and JSON
		codecs: map[string]codec{
			"application/x-protobuf": protoCodec,
			"application/protobuf":   protoCodec,
			"application/json":       jsonCodec,
		},
		address: config.HTTPAddress,
	}

	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 3
	}

	r.httpServer.Server.SetKeepAlivesEnabled(false)
	r.httpServer.HideBanner = true
	r.httpServer.HidePort = true
	r.httpServer.Pre(middleware.RemoveTrailingSlash())
	r.httpServer.Use(
		logger.EchoLogger(),
		ConcurrencyLimiter(config.MaxConcurrency),
		middleware.Recover(),
		middleware.Decompress(),
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
		buf = xbytes.Resize(buf, int(c.Request().ContentLength))[:0]
	}

	buf, err := ReadAll(c.Request().Body, buf)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	inbound, outbound, outboundContentType, ok := r.codec(c.Request())
	if !ok {
		return echo.NewHTTPError(
			http.StatusUnsupportedMediaType,
			fmt.Sprintf("unsupported content type `%s`", c.Request().Header.Get(echo.HeaderContentType)),
		)
	}

	req := ptraceotlp.NewExportRequest()
	if err := inbound.unmarshal(req, buf); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unmarshal: %s", err))
	}

	resp, err := r.otelProxy.Grep(c.Request().Context(), req.Traces())
	if err != nil {
		return fmt.Errorf("grep: %w", err)
	}

	body, err := outbound.marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	return c.Blob(http.StatusOK, outboundContentType, body)
}

// codec returns false if the content type is not supported.
func (r *Recorder) codec(req *http.Request) (inbound codec, outbound codec, outboundContentType string, ok bool) {
	var inboundContentType string
	for _, content := range req.Header[echo.HeaderContentType] {
		contentType, _, err := mime.ParseMediaType(content)
		if err != nil {
			continue
		}

		if inbound, ok = r.codecs[contentType]; ok {
			inboundContentType = contentType
			break
		}
	}

	if !ok {
		return codec{}, codec{}, "", false
	}

	for _, accept := range req.Header[echo.HeaderAccept] {
		if outbound, ok := r.codecs[accept]; ok {
			return inbound, outbound, accept, true
		}
	}

	return inbound, inbound, inboundContentType, true
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
