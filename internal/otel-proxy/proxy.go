package otelproxy

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository/http"
	"github.com/tekig/clerk/internal/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	otelcollector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	trace "go.opentelemetry.io/proto/otlp/trace/v1"
)

type Proxy struct {
	target          otelcollector.TraceServiceClient
	recorder        *recorder.Recorder
	formatURL       string
	defaultStrategy RuleStrategy
	rules           []Rule
}

type Config struct {
	Target          string
	Recorder        *recorder.Recorder
	FormatURL       string
	DefaultStrategy string
	Rules           []ConfigRule
}
type ConfigRule struct {
	Key      []string
	Value    []string
	Strategy string
}

type (
	ruleKeyFn   func(v string) bool
	ruleValueFn func(v *common.AnyValue) bool
)

type RuleStrategy string

const (
	RuleStrategyKeep   RuleStrategy = "keep"
	RuleStrategyUnlink RuleStrategy = "unlink"
	RuleStrategyRemove RuleStrategy = "remove"
)

var (
	metricInit       sync.Once
	meterSpanCounter metric.Int64Counter
)

type Rule struct {
	key      []ruleKeyFn
	value    []ruleValueFn
	strategy RuleStrategy
}

func New(c Config) (*Proxy, error) {
	metricInit.Do(func() {
		meter := otel.Meter("otel_proxy")
		counter, err := meter.Int64Counter("span_count")
		if err != nil {
			log.Fatalf("init meter span_count: %s", err.Error())
		}

		meterSpanCounter = counter
	})

	target, err := http.NewOTELCollector(c.Target)
	if err != nil {
		return nil, fmt.Errorf("target collector: %w", err)
	}

	defaultStrategy, err := parseStrategy(c.DefaultStrategy)
	if err != nil {
		return nil, fmt.Errorf("default strategy: %w", err)
	}

	rules, err := parseRule(c.Rules)
	if err != nil {
		return nil, fmt.Errorf("parse rule: %w", err)
	}

	return &Proxy{
		target:          target,
		recorder:        c.Recorder,
		formatURL:       c.FormatURL,
		defaultStrategy: defaultStrategy,
		rules:           rules,
	}, nil
}

func (p *Proxy) Grep(ctx context.Context, res []*trace.ResourceSpans) (*otelcollector.ExportTraceServiceResponse, error) {
	var events []*pb.Event
	for _, prevRes := range res {
		var serviceName = "unknown"
		for _, kv := range prevRes.GetResource().GetAttributes() {
			if kv.GetKey() == "service.name" {
				serviceName = kv.GetValue().GetStringValue()
			}
		}

		for _, prevScope := range prevRes.ScopeSpans {
			meterSpanCounter.Add(
				ctx,
				int64(len(prevScope.Spans)),
				metric.WithAttributes(
					attribute.String("service.name", serviceName),
				),
			)

			for _, prevSpan := range prevScope.Spans {
				id := uuid.New()
				var event = &pb.Event{
					Id:         id[:],
					Attributes: make([]*pb.Attribute, 0),
				}
				var nextAttributes = make([]*common.KeyValue, 0, len(prevSpan.Attributes))
				for _, prevAttribute := range prevSpan.Attributes {
					switch p.rule(prevAttribute) {
					case RuleStrategyUnlink:
						var eventAttribute *pb.Attribute
						switch v := prevAttribute.GetValue().GetValue().(type) {
						case *common.AnyValue_StringValue:
							eventAttribute = &pb.Attribute{
								Key: prevAttribute.Key,
								Value: &pb.Attribute_AsString{
									AsString: v.StringValue,
								},
							}
						case *common.AnyValue_BoolValue:
							eventAttribute = &pb.Attribute{
								Key: prevAttribute.Key,
								Value: &pb.Attribute_AsBool{
									AsBool: v.BoolValue,
								},
							}
						case *common.AnyValue_IntValue:
							eventAttribute = &pb.Attribute{
								Key: prevAttribute.Key,
								Value: &pb.Attribute_AsInt64{
									AsInt64: v.IntValue,
								},
							}
						case *common.AnyValue_DoubleValue:
							eventAttribute = &pb.Attribute{
								Key: prevAttribute.Key,
								Value: &pb.Attribute_AsDouble{
									AsDouble: v.DoubleValue,
								},
							}
						default:
							eventAttribute = &pb.Attribute{
								Key: prevAttribute.Key,
								Value: &pb.Attribute_AsString{
									AsString: fmt.Sprintf("unsupport attribute `%T`", v),
								},
							}
						}

						event.Attributes = append(event.Attributes, eventAttribute)
					case RuleStrategyRemove:
						// remove
					default: // RuleStrategyKeep
						nextAttributes = append(nextAttributes, prevAttribute)
					}
				}
				if len(event.Attributes) > 0 {
					nextAttributes = append(nextAttributes,
						&common.KeyValue{
							Key: "event_url",
							Value: &common.AnyValue{
								Value: &common.AnyValue_StringValue{
									StringValue: fmt.Sprintf(p.formatURL, id.String()),
								},
							},
						},
					)
					events = append(events, event)
				}
				prevSpan.Attributes = nextAttributes
			}
		}
	}

	if len(events) > 0 {
		if err := p.recorder.Write(ctx, events); err != nil {
			return nil, fmt.Errorf("recoreder write: %w", err)
		}
	}

	response, err := p.target.Export(ctx, &otelcollector.ExportTraceServiceRequest{
		ResourceSpans: res,
	})
	if err != nil {
		return nil, fmt.Errorf("target export: %w", err)
	}

	return response, nil
}

func (p *Proxy) rule(kv *common.KeyValue) RuleStrategy {
	ruleKeyFn := func(fn []ruleKeyFn, k string) bool {
		if len(fn) == 0 {
			return true
		}

		for _, fn := range fn {
			if fn(k) {
				return true
			}
		}

		return false
	}

	ruleValueFn := func(fn []ruleValueFn, v *common.AnyValue) bool {
		if v == nil {
			return false
		}

		if len(fn) == 0 {
			return true
		}

		for _, fn := range fn {
			if fn(v) {
				return true
			}
		}

		return false
	}

	for _, rule := range p.rules {
		if !ruleKeyFn(rule.key, kv.Key) {
			continue
		}

		if !ruleValueFn(rule.value, kv.Value) {
			continue
		}

		return rule.strategy
	}

	return p.defaultStrategy
}
