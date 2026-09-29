package observability

import (
	"context"
	"testing"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingCreatesRootAndContinuesRemoteParent(t *testing.T) {
	tracing := NewTracing()
	t.Cleanup(func() { _ = tracing.Shutdown(context.Background()) })

	rootCtx, rootSpan := tracing.StartTool(context.Background(), SourceMCP)
	rootTraceID, rootSpanID := TraceIdentifiers(rootCtx)
	if len(rootTraceID) != 32 || len(rootSpanID) != 16 {
		t.Fatalf("root trace identifiers = %q / %q", rootTraceID, rootSpanID)
	}
	rootSpan.End()

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	state, err := trace.ParseTraceState("rojo=00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, TraceState: state, Remote: true,
	})
	parentCtx := trace.ContextWithRemoteSpanContext(context.Background(), parent)
	childCtx, childSpan := tracing.StartTool(parentCtx, SourceNexus)
	child := trace.SpanContextFromContext(childCtx)
	defer childSpan.End()
	if child.TraceID() != parent.TraceID() || child.SpanID() == parent.SpanID() {
		t.Fatalf("child span context = %s/%s, parent = %s/%s", child.TraceID(), child.SpanID(), parent.TraceID(), parent.SpanID())
	}
	if !child.IsSampled() || child.TraceState().String() != parent.TraceState().String() {
		t.Fatalf("child sampling/tracestate = sampled:%v state:%q", child.IsSampled(), child.TraceState())
	}
}

func TestAddStageEventsUsesOriginalStageTiming(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	tracer := provider.Tracer("test")
	startedAt := time.Now().Add(-time.Second)
	ctx, span := tracer.Start(context.Background(), "tool", trace.WithTimestamp(startedAt))
	_ = ctx
	AddStageEvents(span, startedAt, []StageRecord{{
		Name: StageCommandStart, StartedOffsetMS: 10, DurationMS: 5, Success: true,
	}})
	span.End()
	ended := recorder.Ended()
	if len(ended) != 1 || len(ended[0].Events()) != 1 {
		t.Fatalf("ended spans/events = %d/%v", len(ended), ended)
	}
	event := ended[0].Events()[0]
	if event.Name != string(StageCommandStart) {
		t.Fatalf("event name = %q", event.Name)
	}
	want := startedAt.Add(15 * time.Millisecond)
	if event.Time.Sub(want) > time.Microsecond || want.Sub(event.Time) > time.Microsecond {
		t.Fatalf("event time = %v, want %v", event.Time, want)
	}
}
