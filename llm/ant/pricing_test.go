package ant

import (
	"testing"

	"shelley.exe.dev/llm"
)

func TestCostUSD(t *testing.T) {
	// 1M output tokens on an Opus model = $75.
	got := costUSD(Claude48Opus, llm.Usage{OutputTokens: 1_000_000})
	if got != 75 {
		t.Errorf("Opus 1M output: costUSD = %f, want 75", got)
	}

	// 1M input tokens on a Sonnet model = $3.
	got = costUSD(Claude46Sonnet, llm.Usage{InputTokens: 1_000_000})
	if got != 3 {
		t.Errorf("Sonnet 1M input: costUSD = %f, want 3", got)
	}

	// Every Opus model shares the standard $15/$75 tier.
	for _, m := range []string{Claude45Opus, Claude46Opus, Claude47Opus, Claude48Opus} {
		c := costUSD(m, llm.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000})
		if c != 90 {
			t.Errorf("%s: costUSD = %f, want 90", m, c)
		}
	}

	// Unknown model costs nothing.
	if got := costUSD("nonexistent-model", llm.Usage{InputTokens: 1_000_000}); got != 0 {
		t.Errorf("unknown model: costUSD = %f, want 0", got)
	}
}
