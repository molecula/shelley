package oai

import "shelley.exe.dev/llm"

// modelPricing maps OpenAI ModelName values to per-million-token prices in USD.
// Cache-read prices reflect OpenAI's discounted cached-input rate. OpenAI does
// not bill separately for cache writes, so CacheWrite is left at the base input
// rate. These values are the source of truth for computing spend from token
// counts returned by the API.
var modelPricing = map[string]llm.ModelPrice{
	// GPT-5 family: $1.25 / $10 per MTok in/out, cached input $0.125.
	"gpt-5.5":       {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.4":       {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.3-codex": {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.2-codex": {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.1":       {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.1-codex": {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	"gpt-5.1-mini":  {Input: 0.25, Output: 2, CacheWrite: 0.25, CacheRead: 0.025},
	"gpt-5.1-nano":  {Input: 0.05, Output: 0.4, CacheWrite: 0.05, CacheRead: 0.005},
	// o-series reasoning models.
	"o3-2025-04-16":      {Input: 2, Output: 8, CacheWrite: 2, CacheRead: 0.5},
	"o4-mini-2025-04-16": {Input: 1.1, Output: 4.4, CacheWrite: 1.1, CacheRead: 0.275},
	// GPT-4.1 family.
	"gpt-4.1-2025-04-14":      {Input: 2, Output: 8, CacheWrite: 2, CacheRead: 0.5},
	"gpt-4.1-mini-2025-04-14": {Input: 0.4, Output: 1.6, CacheWrite: 0.4, CacheRead: 0.1},
	"gpt-4.1-nano-2025-04-14": {Input: 0.1, Output: 0.4, CacheWrite: 0.1, CacheRead: 0.025},
	// GPT-4o family.
	"gpt-4o-2024-08-06":      {Input: 2.5, Output: 10, CacheWrite: 2.5, CacheRead: 1.25},
	"gpt-4o-mini-2024-07-18": {Input: 0.15, Output: 0.6, CacheWrite: 0.15, CacheRead: 0.075},
}

// costUSD computes the dollar cost for usage on the given model. Unknown models
// (e.g. third-party Fireworks/Together models without a price entry) return 0.
func costUSD(modelName string, usage llm.Usage) float64 {
	price, ok := modelPricing[modelName]
	if !ok {
		return 0
	}
	return price.CostUSD(usage)
}
