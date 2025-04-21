package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	v1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type OTELCollector struct {
	address string
}

func NewOTELCollector(address string) (*OTELCollector, error) {
	return &OTELCollector{
		address: address,
	}, nil
}

func (c *OTELCollector) Export(ctx context.Context, in *v1.ExportTraceServiceRequest, opts ...grpc.CallOption) (*v1.ExportTraceServiceResponse, error) {
	body, err := proto.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("proto marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.address+"/v1/traces", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if res.Header.Get("Content-Type") != "application/x-protobuf" {
		return nil, fmt.Errorf("invalid response `%s`", res.Header.Get("Content-Type"))
	}

	var out = &v1.ExportTraceServiceResponse{}
	if err := proto.Unmarshal(resBody, out); err != nil {
		return nil, fmt.Errorf("proto unmarshal: %w", err)
	}

	return out, nil
}
