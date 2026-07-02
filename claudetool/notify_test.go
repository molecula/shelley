package claudetool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNotifyRun(t *testing.T) {
	t.Run("empty message errors", func(t *testing.T) {
		called := false
		tool := &NotifyTool{Notify: func(ctx context.Context, message, title string) { called = true }}
		out := tool.Run(context.Background(), json.RawMessage(`{"message":""}`))
		if out.Error == nil {
			t.Fatalf("expected error for empty message, got none")
		}
		if called {
			t.Fatalf("callback should not be invoked for empty message")
		}
	})

	t.Run("valid message invokes callback", func(t *testing.T) {
		var gotMsg, gotTitle string
		calls := 0
		tool := &NotifyTool{Notify: func(ctx context.Context, message, title string) {
			calls++
			gotMsg = message
			gotTitle = title
		}}
		out := tool.Run(context.Background(), json.RawMessage(`{"message":"deploy is live","title":"Deploy"}`))
		if out.Error != nil {
			t.Fatalf("unexpected error: %v", out.Error)
		}
		if calls != 1 {
			t.Fatalf("expected callback called once, got %d", calls)
		}
		if gotMsg != "deploy is live" {
			t.Fatalf("unexpected message: %q", gotMsg)
		}
		if gotTitle != "Deploy" {
			t.Fatalf("unexpected title: %q", gotTitle)
		}
	})

	t.Run("nil callback is safe", func(t *testing.T) {
		tool := &NotifyTool{}
		out := tool.Run(context.Background(), json.RawMessage(`{"message":"hi"}`))
		if out.Error != nil {
			t.Fatalf("unexpected error: %v", out.Error)
		}
	})

	t.Run("invalid json errors", func(t *testing.T) {
		tool := &NotifyTool{}
		out := tool.Run(context.Background(), json.RawMessage(`{`))
		if out.Error == nil {
			t.Fatalf("expected error for invalid json")
		}
	})
}
