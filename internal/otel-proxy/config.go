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
				return nil, fmt.Errorf("rule #%d: key `%s`: unknown `%s`", i, v, parts[0])
			}

		}

		rules = append(rules, Rule{
			key:      keyFn,
			value:    valueFn,
			strategy: strategy,
		})
	}

	return rules, nil
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
