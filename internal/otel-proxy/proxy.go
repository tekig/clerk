package otelproxy

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository/http"
	"github.com/tekig/clerk/internal/uuid"
	"github.com/tekig/clerk/internal/x/xsync"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Target interface {
	Export(ctx context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error)
}

type Proxy struct {
	target          Target
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
	Span     []string
	Strategy string
}

type (
	ruleKeyFn   func(v string) bool
	ruleValueFn func(v pcommon.Value) bool
	ruleSpanFn  func(v ptrace.Span) bool
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
	span     []ruleSpanFn
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

func (p *Proxy) Grep(ctx context.Context, td ptrace.Traces) (ptraceotlp.ExportResponse, error) {
	var events []*pb.Event
	// Rules applicable to the current span, reused between spans
	var spanRules []*Rule
	for _, prevRes := range td.ResourceSpans().All() {
		serviceName := entity.MetaValueUnknown
		if v, ok := prevRes.Resource().Attributes().Get("service.name"); ok {
			serviceName = v.Str()
		}

		for _, prevScope := range prevRes.ScopeSpans().All() {
			for _, prevSpan := range prevScope.Spans().All() {
				eventAttributes := make([]*pb.Attribute, 0)
				spanRules = p.spanRules(prevSpan, spanRules)
				prevSpan.Attributes().RemoveIf(func(key string, value pcommon.Value) bool {
					switch p.rule(key, value, spanRules) {
					case RuleStrategyUnlink:
						eventAttributes = append(eventAttributes, eventAttribute(key, value))

						return true
					case RuleStrategyRemove:
						return true
					default: // RuleStrategyKeep
						return false
					}
				})
				meterSpanCounter.Add(
					ctx,
					1,
					metric.WithAttributes(
						attribute.String("service.name", serviceName),
						attribute.String("span.name", prevSpan.Name()),
					),
				)
				if len(eventAttributes) > 0 {
					id := uuid.New()

					prevSpan.Attributes().PutStr("event_url", fmt.Sprintf(p.formatURL, id.String()))

					metaEvent := []*pb.Attribute{
						{
							Key: entity.MetaServiceName,
							Value: &pb.Attribute_AsString{
								AsString: serviceName,
							},
						}, {
							Key: entity.MetaSpanName,
							Value: &pb.Attribute_AsString{
								AsString: prevSpan.Name(),
							},
						}, {
							Key: entity.MetaTraceID,
							Value: &pb.Attribute_AsString{
								AsString: prevSpan.TraceID().String(),
							},
						}, {
							Key: entity.MetaSpanID,
							Value: &pb.Attribute_AsString{
								AsString: prevSpan.SpanID().String(),
							},
						},
					}

					event := &pb.Event{
						Id:         id[:],
						Attributes: append(metaEvent, eventAttributes...),
					}

					events = append(events, event)
				}
			}
		}
	}

	// Export can take a long time, and it's doing nothing the entire time
	// Let's run it in parallel to free up memory a little earlier
	var wg xsync.ErrGroup
	var response ptraceotlp.ExportResponse
	wg.Go(func() error {
		res, err := p.target.Export(ctx, ptraceotlp.NewExportRequestFromTraces(td))
		if err != nil {
			return fmt.Errorf("target export: %w", err)
		}

		response = res

		return nil
	})

	if len(events) > 0 {
		wg.Go(func() error {
			if err := p.recorder.Write(ctx, events); err != nil {
				return fmt.Errorf("recoreder write: %w", err)
			}

			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		return ptraceotlp.ExportResponse{}, fmt.Errorf("wait: %w", err)
	}

	return response, nil
}

func eventAttribute(key string, value pcommon.Value) *pb.Attribute {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		return &pb.Attribute{
			Key: key,
			Value: &pb.Attribute_AsString{
				AsString: value.Str(),
			},
		}
	case pcommon.ValueTypeBool:
		return &pb.Attribute{
			Key: key,
			Value: &pb.Attribute_AsBool{
				AsBool: value.Bool(),
			},
		}
	case pcommon.ValueTypeInt:
		return &pb.Attribute{
			Key: key,
			Value: &pb.Attribute_AsInt64{
				AsInt64: value.Int(),
			},
		}
	case pcommon.ValueTypeDouble:
		return &pb.Attribute{
			Key: key,
			Value: &pb.Attribute_AsDouble{
				AsDouble: value.Double(),
			},
		}
	default:
		return &pb.Attribute{
			Key: key,
			Value: &pb.Attribute_AsString{
				AsString: fmt.Sprintf("unsupport attribute `%s`", value.Type()),
			},
		}
	}
}

// spanRules returns the rules whose span conditions match the span, preserving their order.
// Span conditions do not depend on attributes, so they are checked once per span.
// buf is reused to avoid allocations.
func (p *Proxy) spanRules(span ptrace.Span, buf []*Rule) []*Rule {
	buf = buf[:0]
	for i := range p.rules {
		if matchAny(p.rules[i].span, span) {
			buf = append(buf, &p.rules[i])
		}
	}

	return buf
}

// rule returns the strategy of the first rule matching the attribute.
// rules must be prepared by spanRules.
func (p *Proxy) rule(key string, value pcommon.Value, rules []*Rule) RuleStrategy {
	for _, rule := range rules {
		if !matchAny(rule.key, key) {
			continue
		}

		if value.Type() == pcommon.ValueTypeEmpty || !matchAny(rule.value, value) {
			continue
		}

		return rule.strategy
	}

	return p.defaultStrategy
}

// matchAny combines the checks as OR, an empty list matches any value.
func matchAny[F ~func(T) bool, T any](fn []F, v T) bool {
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
