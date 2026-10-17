package otelproxy

import (
	"testing"

	"go.opentelemetry.io/collector/pdata/ptrace"
)

func Test_parseRuleSpan(t *testing.T) {
	tests := []struct {
		rule  string
		name  string
		match bool
	}{
		{"name:equals:Request", "Request", true},
		{"name:equals:Request", "RequestX", false},
		{"name:regex:Req.*", "Request", true},
		{"name:regex:Req.*", "Response", false},
		{"name:regex:^Req$", "Request", false},
		{"name:prefix:Req", "Request", true},
		{"name:prefix:Req", "Response", false},
	}
	for _, tt := range tests {
		fn, err := parseRuleSpan(tt.rule)
		if err != nil {
			t.Errorf("parseRuleSpan(%q) failed: %v", tt.rule, err)
			continue
		}

		span := ptrace.NewSpan()
		span.SetName(tt.name)
		if got := fn(span); got != tt.match {
			t.Errorf("parseRuleSpan(%q)(%q) = %v, want %v", tt.rule, tt.name, got, tt.match)
		}
	}
}
