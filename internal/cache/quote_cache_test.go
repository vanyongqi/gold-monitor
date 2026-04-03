package cache

import (
	"testing"
	"time"

	"gold_monitor/internal/market"
)

func TestQuoteCacheSetAndGet(t *testing.T) {
	t.Parallel()

	c := NewQuoteCache()
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1018,
		FetchedAt:  time.Date(2026, 4, 2, 14, 0, 0, 0, time.Local),
	}

	c.Set(quote)

	got, ok := c.Get("Au99.99")
	if !ok {
		t.Fatal("expected quote in cache")
	}
	if got.Price != 1018 {
		t.Fatalf("price = %.2f, want 1018", got.Price)
	}
}
