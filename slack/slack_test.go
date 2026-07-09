package slack

import (
	"testing"

	"shelley.exe.dev/claudetool"
)

func TestRenderReplyContext(t *testing.T) {
	b := &Bot{botUID: "UBOT"}
	msgs := []claudetool.SlackMessage{
		{User: "UAAA", Text: "the parent message", Timestamp: "1.0"},
		{User: "UBBB", Text: "a follow-up", Timestamp: "2.0"},
		{User: "UCCC", Text: "<@UBOT> what does this mean?", Timestamp: "3.0"}, // the mention itself
	}
	got := b.renderReplyContext(msgs, "3.0")
	want := "Replying to the following message(s):\n<@UAAA>: the parent message\n<@UBBB>: a follow-up"
	if got != want {
		t.Errorf("renderReplyContext:\n got: %q\nwant: %q", got, want)
	}

	// Only the mention present -> no parent context.
	only := []claudetool.SlackMessage{{User: "UCCC", Text: "<@UBOT> hi", Timestamp: "3.0"}}
	if got := b.renderReplyContext(only, "3.0"); got != "" {
		t.Errorf("expected empty reply context, got %q", got)
	}
}

func TestChunkText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   int // expected number of chunks
	}{
		{"short", "hello", 100, 1},
		{"exact", "hello", 5, 1},
		{"needs_split", "hello world", 6, 2},
		{"split_at_newline", "hello\nworld\nfoo", 12, 2},
		{"empty", "", 100, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkText(tt.text, tt.maxLen)
			if len(chunks) != tt.want {
				t.Errorf("chunkText(%q, %d) = %d chunks, want %d; chunks: %v", tt.text, tt.maxLen, len(chunks), tt.want, chunks)
			}
			// Verify all text is preserved
			var combined string
			for _, c := range chunks {
				combined += c
			}
			if combined != tt.text {
				t.Errorf("chunks don't reconstruct original text")
			}
			// Verify no chunk exceeds maxLen
			for i, c := range chunks {
				if len(c) > tt.maxLen {
					t.Errorf("chunk %d exceeds maxLen: %d > %d", i, len(c), tt.maxLen)
				}
			}
		})
	}
}
