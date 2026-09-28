package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/uvwt/agentdock/internal/observability"
)

func TestRuntimeAnalyticsRecordsSafeToolMetadata(t *testing.T) {
	runtime := newRuntimeValidationTestRuntime(t)
	ctx := observability.WithSource(context.Background(), observability.SourceMCP)
	secret := "/Users/example/private/token-file"

	if _, err := runtime.Call(ctx, "agentdock_context", map[string]any{"future_field": secret}); err == nil {
		t.Fatal("expected validation error")
	}
	if _, err := runtime.Call(ctx, "secret-tool-"+secret, nil); err == nil {
		t.Fatal("expected unknown tool error")
	}

	snapshot := runtime.observer.Snapshot()
	if snapshot.TotalCalls != 2 || snapshot.TotalErrors != 2 || snapshot.WindowCalls != 2 {
		t.Fatalf("analytics totals = calls %d errors %d window %d", snapshot.TotalCalls, snapshot.TotalErrors, snapshot.WindowCalls)
	}
	if snapshot.RecentCalls[0].Tool != "unknown" {
		t.Fatalf("unknown tool name was retained: %#v", snapshot.RecentCalls[0])
	}
	known := snapshot.RecentCalls[1]
	if known.Tool != "agentdock_context" || known.Source != observability.SourceMCP {
		t.Fatalf("known call metadata = %#v", known)
	}
	if known.ErrorCode != "INVALID_ARGUMENT" || known.ErrorCategory != "validation" {
		t.Fatalf("known call error metadata = %#v", known)
	}

	encoded, err := json.Marshal(runtime.RuntimeAnalytics())
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, forbidden := range []string{secret, "future_field", "secret-tool-"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("runtime analytics leaked %q: %s", forbidden, body)
		}
	}
}
