package otelproxy

import (
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func ruleLenGE(rule string) (ruleValueFn, error) {
	size, err := strconv.Atoi(rule)
	if err != nil {
		return nil, fmt.Errorf("atoi: %w", err)
	}

	return func(v pcommon.Value) bool {
		switch v.Type() {
		case pcommon.ValueTypeSlice:
			return v.Slice().Len() >= size
		case pcommon.ValueTypeBytes:
			return v.Bytes().Len() >= size
		case pcommon.ValueTypeMap:
			return v.Map().Len() >= size
		case pcommon.ValueTypeStr:
			return len(v.Str()) >= size
		default:
			return false
		}
	}, nil
}

func rulePrefix(rule string) ruleKeyFn {
	return func(v string) bool {
		return strings.HasPrefix(v, rule)
	}
}

func ruleEquals(rule string) ruleKeyFn {
	return func(v string) bool {
		return v == rule
	}
}

func ruleSpan(fn ruleKeyFn) ruleSpanFn {
	return func(v ptrace.Span) bool {
		return fn(v.Name())
	}
}
