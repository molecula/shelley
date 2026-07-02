package claudetool

import (
	"context"
	"encoding/json"

	"shelley.exe.dev/llm"
)

// NotifyTool sends the user a desktop notification on demand, decoupled from
// turn-end. The Notify callback is provided per-conversation by the server and
// emits a "notify" notification event over the stream (and external channels).
type NotifyTool struct {
	// Notify delivers the notification. It is optional; when nil the tool is a
	// no-op (used by test/non-server toolsets). title is an optional title
	// override; when empty the conversation title is used by the receiver.
	Notify func(ctx context.Context, message, title string)
}

func (t *NotifyTool) Tool() *llm.Tool {
	return &llm.Tool{
		Name:        notifyName,
		Description: notifyDescription,
		InputSchema: llm.MustSchema(notifyInputSchema),
		Run:         t.Run,
	}
}

const (
	notifyName        = "notify"
	notifyDescription = `Send the user a desktop notification immediately. Use when the user asked to be told when something happens — e.g. a deployment going live, a long poll completing, a watched condition being met — especially from long-running or background/subagent tasks. Call it the moment the condition is true; do not wait for the turn to end. message is the notification body; optional title.`

	notifyInputSchema = `
{
  "type": "object",
  "required": ["message"],
  "properties": {
    "message": {
      "type": "string",
      "description": "The notification body text to show the user."
    },
    "title": {
      "type": "string",
      "description": "Optional title override. If omitted, the conversation title is used."
    }
  }
}
`
)

func (t *NotifyTool) Run(ctx context.Context, m json.RawMessage) llm.ToolOut {
	var input struct {
		Message string `json:"message"`
		Title   string `json:"title"`
	}
	if err := json.Unmarshal(m, &input); err != nil {
		return llm.ErrorToolOut(err)
	}
	if input.Message == "" {
		return llm.ErrorfToolOut("message is required")
	}
	if t.Notify != nil {
		t.Notify(ctx, input.Message, input.Title)
	}
	return llm.ToolOut{LLMContent: llm.TextContent("notified user")}
}
