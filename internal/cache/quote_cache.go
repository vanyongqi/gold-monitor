package cache

import (
	"sync"

	"gold_monitor/internal/market"
)

type QuoteCache struct {
	mu     sync.RWMutex
	quotes map[string]market.Quote
}

func NewQuoteCache() *QuoteCache {
	return &QuoteCache{
		quotes: make(map[string]market.Quote),
	}
}

func (c *QuoteCache) Set(quote market.Quote) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.quotes[quote.Instrument] = quote
}

func (c *QuoteCache) Get(instrument string) (market.Quote, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	quote, ok := c.quotes[instrument]
	return quote, ok
}
