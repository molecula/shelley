package gem

import "shelley.exe.dev/llm"

// modelPricing maps Gemini model names to per-million-token prices in USD.
// Cache-read prices reflect Google's discounted cached-input rate. Gemini does
// not bill a separate cache-write rate, so CacheWrite uses the base input rate.
// These values are the source of truth for computing spend from token counts.
//
// Prices use the standard (<=200k token) context tier.
var modelPricing = map[string]llm.ModelPrice{
	// Gemini 3 Pro: $2 / $12 per MTok in/out.
	"gemini-3-pro-preview": {Input: 2, Output: 12, CacheWrite: 2, CacheRead: 0.2},
	// Gemini 3 Flash: $0.30 / $2.50 per MTok in/out.
	"gemini-3-flash-preview": {Input: 0.3, Output: 2.5, CacheWrite: 0.3, CacheRead: 0.03},
	// Gemini 2.5 Pro: $1.25 / $10 per MTok in/out.
	"gemini-2.5-pro": {Input: 1.25, Output: 10, CacheWrite: 1.25, CacheRead: 0.125},
	// Gemini 2.5 Flash: $0.30 / $2.50 per MTok in/out.
	"gemini-2.5-flash": {Input: 0.3, Output: 2.5, CacheWrite: 0.3, CacheRead: 0.03},
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
