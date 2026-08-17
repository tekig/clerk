package otelproxy

import (
	"fmt"
	"regexp"
	"strings"
)

func parseRule(r []ConfigRule) ([]Rule, error) {
	var rules []Rule
	for i, r := range r {
		strategy, err := parseStrategy(r.Strategy)
		if err != nil {
			return nil, fmt.Errorf("rule #%d strategy: %w", i, err)
		}

		var keyFn []ruleKeyFn
		for _, k := range r.Key {
			parts := strings.SplitN(k, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("rule #%d: key `%s`: invalid format", i, k)
			}

			switch parts[0] {
			case "regex":
				r, err := regexp.Compile(parts[1])
				if err != nil {
					return nil, fmt.Errorf("rule #%d: key `%s`: %w", i, k, err)
				}
				keyFn = append(keyFn, r.MatchString)
			case "prefix":
				keyFn = append(keyFn, rulePrefix(parts[1]))
			case "equals":
				keyFn = append(keyFn, ruleEquals(parts[1]))
			default:
				return nil, fmt.Errorf("rule #%d: key `%s`: unknown `%s`", i, k, parts[0])
			}
		}

		var valueFn []ruleValueFn
		for _, v := range r.Value {
			parts := strings.SplitN(v, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("rule #%d: value `%s`: invalid format", i, v)
			}

			switch parts[0] {
			case "len_ge":
				r, err := ruleLenGE(parts[1])
				if err != nil {
					return nil, fmt.Errorf("rule #%d: value `%s`: %w", i, v, err)
				}

				valueFn = append(valueFn, r)
			default:
				return nil, fmt.Errorf("rule #%d: value `%s`: unknown `%s`", i, v, parts[0])
			}
		}

		var spanFn []ruleSpanFn
		for _, s := range r.Span {
			rule, err := parseRuleSpan(s)
			if err != nil {
				return nil, fmt.Errorf("rule #%d: %w", i, err)
			}

			spanFn = append(spanFn, rule)
		}

		rules = append(rules, Rule{
			key:      keyFn,
			value:    valueFn,
			span:     spanFn,
			strategy: strategy,
		})
	}

	return rules, nil
}

func parseRuleSpan(s string) (ruleSpanFn, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("span `%s`: invalid format", s)
	}

	if parts[0] == "name" {
		if parts[1] == "equals" {
			return ruleSpan(ruleEquals(parts[2])), nil
		}

		if parts[1] == "prefix" {
			return ruleSpan(rulePrefix(parts[2])), nil
		}

		if parts[1] == "regex" {
			r, err := regexp.Compile(parts[1])
			if err != nil {
				return nil, fmt.Errorf("key `%s`: %w", s, err)
			}

			return ruleSpan(r.MatchString), nil
		}
	}

	return nil, fmt.Errorf("span `%s`: unknown `%s`", s, parts[0])
}

func parseStrategy(s string) (RuleStrategy, error) {
	switch v := RuleStrategy(s); v {
	case RuleStrategyKeep:
		return RuleStrategyKeep, nil
	case RuleStrategyUnlink:
		return RuleStrategyUnlink, nil
	case RuleStrategyRemove:
		return RuleStrategyRemove, nil
	default:
		return "", fmt.Errorf("unknown `%s`", v)
	}
}
