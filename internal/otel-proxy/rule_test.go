package otelproxy

import (
	"strings"
	"testing"

	common "go.opentelemetry.io/proto/otlp/common/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
)

func stringValue(v string) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: v}}
}

func TestRuleLenGE(t *testing.T) {
	fn, err := ruleLenGE("3")
	if err != nil {
		t.Fatalf("ruleLenGE: %v", err)
	}

	tests := []struct {
		name  string
		value *common.AnyValue
		want  bool
	}{
		{"short string", stringValue("ab"), false},
		{"string of exact length", stringValue("abc"), true},
		{"long string", stringValue("abcde"), true},
		{"short bytes", &common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: []byte("ab")}}, false},
		{"long bytes", &common.AnyValue{Value: &common.AnyValue_BytesValue{BytesValue: []byte("abcde")}}, true},
		{"long array", &common.AnyValue{Value: &common.AnyValue_ArrayValue{ArrayValue: &common.ArrayValue{
			Values: []*common.AnyValue{stringValue("1"), stringValue("2"), stringValue("3"), stringValue("4")},
		}}}, true},
		{"long kvlist", &common.AnyValue{Value: &common.AnyValue_KvlistValue{KvlistValue: &common.KeyValueList{
			Values: []*common.KeyValue{{Key: "1"}, {Key: "2"}, {Key: "3"}, {Key: "4"}},
		}}}, true},
		{"int", &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: 123456}}, false},
		{"bool", &common.AnyValue{Value: &common.AnyValue_BoolValue{BoolValue: true}}, false},
	}
	for _, tt := range tests {
		if got := fn(tt.value); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}

	if _, err := ruleLenGE("abc"); err == nil {
		t.Errorf("ruleLenGE(abc): expected error")
	}
}

func TestRuleKey(t *testing.T) {
	if !rulePrefix("http.")("http.method") || rulePrefix("http.")("db.statement") {
		t.Errorf("rulePrefix mismatch")
	}
	if !ruleEquals("http.method")("http.method") || ruleEquals("http.method")("http.method.x") {
		t.Errorf("ruleEquals mismatch")
	}
	if !ruleSpan(ruleEquals("Request"))(&trace.Span{Name: "Request"}) {
		t.Errorf("ruleSpan mismatch")
	}
}

func TestParseRule(t *testing.T) {
	rules, err := parseRule([]ConfigRule{{
		Key:      []string{"regex:^http\\.", "prefix:db.", "equals:secret"},
		Value:    []string{"len_ge:10"},
		Span:     []string{"name:equals:Request", "name:prefix:Req", "name:regex:^Req"},
		Strategy: "unlink",
	}, {
		Strategy: "remove",
	}})
	if err != nil {
		t.Fatalf("parseRule: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(rules))
	}
	if len(rules[0].key) != 3 || len(rules[0].value) != 1 || len(rules[0].span) != 3 {
		t.Errorf("rule #0 matchers: key=%d value=%d span=%d", len(rules[0].key), len(rules[0].value), len(rules[0].span))
	}
	if rules[0].strategy != RuleStrategyUnlink || rules[1].strategy != RuleStrategyRemove {
		t.Errorf("strategies = %s, %s", rules[0].strategy, rules[1].strategy)
	}
}

func TestParseRule_Errors(t *testing.T) {
	tests := []struct {
		name string
		rule ConfigRule
		err  string
	}{
		{"strategy", ConfigRule{Strategy: "drop"}, "strategy"},
		{"key format", ConfigRule{Key: []string{"http"}, Strategy: "keep"}, "invalid format"},
		{"key type", ConfigRule{Key: []string{"suffix:http"}, Strategy: "keep"}, "unknown"},
		{"key regex", ConfigRule{Key: []string{"regex:("}, Strategy: "keep"}, "regex"},
		{"value format", ConfigRule{Value: []string{"100"}, Strategy: "keep"}, "invalid format"},
		{"value type", ConfigRule{Value: []string{"len_le:100"}, Strategy: "keep"}, "unknown"},
		{"value size", ConfigRule{Value: []string{"len_ge:big"}, Strategy: "keep"}, "atoi"},
		{"span format", ConfigRule{Span: []string{"name:Request"}, Strategy: "keep"}, "invalid format"},
		{"span field", ConfigRule{Span: []string{"kind:equals:server"}, Strategy: "keep"}, "unknown"},
		{"span matcher", ConfigRule{Span: []string{"name:suffix:Request"}, Strategy: "keep"}, "unknown"},
		{"span regex", ConfigRule{Span: []string{"name:regex:("}, Strategy: "keep"}, "error parsing regexp"},
	}
	for _, tt := range tests {
		_, err := parseRule([]ConfigRule{tt.rule})
		if err == nil {
			t.Errorf("%s: expected error", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.err) {
			t.Errorf("%s: error %q does not contain %q", tt.name, err, tt.err)
		}
	}
}

func TestParseStrategy(t *testing.T) {
	for _, s := range []RuleStrategy{RuleStrategyKeep, RuleStrategyUnlink, RuleStrategyRemove} {
		got, err := parseStrategy(string(s))
		if err != nil || got != s {
			t.Errorf("parseStrategy(%s) = %s, %v", s, got, err)
		}
	}

	if _, err := parseStrategy(""); err == nil {
		t.Errorf("parseStrategy(\"\"): expected error")
	}
}

func TestProxy_Rule(t *testing.T) {
	rules, err := parseRule([]ConfigRule{
		{Key: []string{"equals:http.body"}, Span: []string{"name:equals:Upload"}, Strategy: "unlink"},
		{Key: []string{"prefix:http."}, Value: []string{"len_ge:5"}, Strategy: "remove"},
		{Key: []string{"prefix:http."}, Strategy: "unlink"},
	})
	if err != nil {
		t.Fatalf("parseRule: %v", err)
	}
	p := &Proxy{rules: rules, defaultStrategy: RuleStrategyKeep}

	tests := []struct {
		name string
		span string
		kv   *common.KeyValue
		want RuleStrategy
	}{
		{"span rule matches", "Upload", &common.KeyValue{Key: "http.body", Value: stringValue("long value")}, RuleStrategyUnlink},
		{"span rule skipped for other span", "Download", &common.KeyValue{Key: "http.body", Value: stringValue("long value")}, RuleStrategyRemove},
		{"value does not match, next rule", "Download", &common.KeyValue{Key: "http.body", Value: stringValue("ab")}, RuleStrategyUnlink},
		{"nil value never matches", "Download", &common.KeyValue{Key: "http.body"}, RuleStrategyKeep},
		{"no rule, default", "Download", &common.KeyValue{Key: "db.statement", Value: stringValue("select")}, RuleStrategyKeep},
	}
	for _, tt := range tests {
		span := &trace.Span{Name: tt.span}
		if got := p.rule(tt.kv, p.spanRules(span, nil)); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
		}
	}
}
