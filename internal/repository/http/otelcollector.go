package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

type OTELCollector struct {
	address string
}

func NewOTELCollector(address string) (*OTELCollector, error) {
	return &OTELCollector{
		address: address,
	}, nil
}

func (c *OTELCollector) Export(ctx context.Context, in ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	body, err := in.MarshalProto()
	if err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("proto marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.address+"/v1/traces", bytes.NewReader(body))
	if err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("do: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("read response: %w", err)
	}

	if res.Header.Get("Content-Type") != "application/x-protobuf" {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("invalid response `%s`", res.Header.Get("Content-Type"))
	}

	out := ptraceotlp.NewExportResponse()
	if err := out.UnmarshalProto(resBody); err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("proto unmarshal: %w", err)
	}

	return out, nil
}
