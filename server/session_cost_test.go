package server

import (
	"encoding/json"
	"testing"

	"shelley.exe.dev/db"
	"shelley.exe.dev/llm"
)

func usageMsg(t *testing.T, cost float64) APIMessage {
	t.Helper()
	usageJSON, err := json.Marshal(llm.Usage{CostUSD: cost})
	if err != nil {
		t.Fatalf("marshal usage: %v", err)
	}
	s := string(usageJSON)
	return APIMessage{Type: string(db.MessageTypeAgent), UsageData: &s}
}

func TestCalculateSessionCost(t *testing.T) {
	t.Parallel()

	t.Run("sums message costs", func(t *testing.T) {
		messages := []APIMessage{
			usageMsg(t, 0.01),
			{Type: string(db.MessageTypeUser)}, // no usage data
			usageMsg(t, 0.02),
		}
		got := calculateSessionCost(messages, 0)
		if want := 0.03; !floatEq(got, want) {
			t.Errorf("calculateSessionCost() = %v, want %v", got, want)
		}
	})

	t.Run("adds base cost", func(t *testing.T) {
		messages := []APIMessage{usageMsg(t, 0.05)}
		got := calculateSessionCost(messages, 0.10)
		if want := 0.15; !floatEq(got, want) {
			t.Errorf("calculateSessionCost() = %v, want %v", got, want)
		}
	})

	t.Run("base only with no messages", func(t *testing.T) {
		got := calculateSessionCost(nil, 0.42)
		if want := 0.42; !floatEq(got, want) {
			t.Errorf("calculateSessionCost() = %v, want %v", got, want)
		}
	})
}

func floatEq(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	return d < eps && d > -eps
}
