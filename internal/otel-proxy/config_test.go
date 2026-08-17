package otelproxy

import "testing"

func Test_parseRuleSpan(t *testing.T) {
	tests := []string{
		"name:equals:Request",
		"name:regex:Req.*",
		"name:prefix:Req",
	}
	for _, tt := range tests {
		_, gotErr := parseRuleSpan(tt)
		if gotErr != nil {
			t.Errorf("parseRuleSpan() failed: %v", gotErr)
			return
		}
	}
}
