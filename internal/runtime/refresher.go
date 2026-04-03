package runtime

import (
	"context"
	"log"
	"time"

	"gold_monitor/internal/cache"
	"gold_monitor/internal/market"
	"gold_monitor/internal/store"
)

type Refresher struct {
	Market     *market.Client
	Cache      *cache.QuoteCache
	Store      *store.SQLiteStore
	Instrument string
	Interval   time.Duration
	Logger     *log.Logger
}

func (r *Refresher) Refresh(ctx context.Context) error {
	quote, err := r.Market.FetchQuote(ctx, r.Instrument)
	if err != nil {
		return err
	}
	if r.Cache != nil {
		r.Cache.Set(quote)
	}
	if r.Store != nil {
		if err := r.Store.SavePriceSnapshot(ctx, quote); err != nil {
			return err
		}
	}
	return nil
}

func (r *Refresher) Start(ctx context.Context) {
	interval := r.Interval
	if interval <= 0 {
		interval = time.Minute
	}

	go func() {
		if err := r.Refresh(ctx); err != nil && r.Logger != nil {
			r.Logger.Printf("initial refresh failed: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Refresh(ctx); err != nil && r.Logger != nil {
					r.Logger.Printf("refresh failed: %v", err)
				}
			}
		}
	}()
}
