package observability

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/uvwt/agentdock"

// Tracing owns AgentDock's process-local OpenTelemetry provider.
// The third observability phase deliberately has no exporter; the SDK is used
// for standard TraceID/SpanID generation, parent-child semantics, and W3C propagation.
type Tracing struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func NewTracing() *Tracing {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
	)
	return &Tracing{
		provider: provider,
		tracer:   provider.Tracer(instrumentationName),
	}
}

func (t *Tracing) StartTool(ctx context.Context, source Source) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	if t == nil || t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	kind := trace.SpanKindInternal
	if source == SourceMCP || source == SourceNexus {
		kind = trace.SpanKindServer
	}
	return t.tracer.Start(ctx, "agentdock.tool", trace.WithSpanKind(kind))
}

func (t *Tracing) Shutdown(ctx context.Context) error {
	if t == nil || t.provider == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return t.provider.Shutdown(ctx)
}

func TraceIdentifiers(ctx context.Context) (string, string) {
	if ctx == nil {
		return "", ""
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return "", ""
	}
	return spanContext.TraceID().String(), spanContext.SpanID().String()
}
