package otelproxy

import (
	"fmt"
	"strconv"

	common "go.opentelemetry.io/proto/otlp/common/v1"
)

func ruleLenGE(rule string) (ruleValueFn, error) {
	size, err := strconv.Atoi(rule)
	if err != nil {
		return nil, fmt.Errorf("atoi: %w", err)
	}

	return func(v *common.AnyValue) bool {
		switch v := v.Value.(type) {
		case *common.AnyValue_ArrayValue:
			return len(v.ArrayValue.Values) > size
		case *common.AnyValue_BytesValue:
			return len(v.BytesValue) > size
		case *common.AnyValue_KvlistValue:
			return len(v.KvlistValue.Values) > size
		case *common.AnyValue_StringValue:
			return len(v.StringValue) > size
		default:
			return false
		}
	}, nil
}
