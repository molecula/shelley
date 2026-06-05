package ant

import "shelley.exe.dev/llm"

// modelPricing maps Anthropic model names to their per-million-token prices in
// USD. Cache-write prices use the 5-minute TTL rate (1.25x base input) and
// cache-read prices use the cache-hit rate (0.1x base input), per Anthropic's
// published pricing. These values are the source of truth for computing spend
// from token counts returned by the API.
var modelPricing = map[string]llm.ModelPrice{
	// Opus tier: $15 / $75 per MTok in/out.
	Claude48Opus: {Input: 15, Output: 75, CacheWrite: 18.75, CacheRead: 1.5},
	Claude47Opus: {Input: 15, Output: 75, CacheWrite: 18.75, CacheRead: 1.5},
	Claude46Opus: {Input: 15, Output: 75, CacheWrite: 18.75, CacheRead: 1.5},
	Claude45Opus: {Input: 15, Output: 75, CacheWrite: 18.75, CacheRead: 1.5},
	// Sonnet tier: $3 / $15 per MTok in/out.
	Claude46Sonnet: {Input: 3, Output: 15, CacheWrite: 3.75, CacheRead: 0.3},
	Claude4Sonnet:  {Input: 3, Output: 15, CacheWrite: 3.75, CacheRead: 0.3},
	// Haiku tier: $1 / $5 per MTok in/out.
	Claude45Haiku: {Input: 1, Output: 5, CacheWrite: 1.25, CacheRead: 0.1},
}

// costUSD computes the dollar cost for usage on the given model. Unknown models
// return 0.
func costUSD(model string, usage llm.Usage) float64 {
	price, ok := modelPricing[model]
	if !ok {
		return 0
	}
	return price.CostUSD(usage)
}
