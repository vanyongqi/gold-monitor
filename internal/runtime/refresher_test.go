package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gold_monitor/internal/cache"
	"gold_monitor/internal/market"
	"gold_monitor/internal/store"
)

func TestRefresherRefreshWritesCacheAndStore(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<h1>上海黄金交易所2026年04月02日延时行情</h1>
<tr class="ininfo">
  <td class="insid">Au99.99</td>
  <td>1017.07</td>
  <td>1020.00</td>
  <td>1010.00</td>
  <td>1012.00</td>
</tr>`))
	}))
	defer server.Close()

	db, err := store.OpenSQLite("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	repo := store.NewSQLiteStore(db)
	cacheLayer := cache.NewQuoteCache()
	client := market.NewClient()
	client.BaseURL = server.URL

	refresher := &Refresher{
		Market:     client,
		Cache:      cacheLayer,
		Store:      repo,
		Instrument: "Au99.99",
		Interval:   time.Minute,
	}

	if err := refresher.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() err = %v", err)
	}

	gotQuote, ok := cacheLayer.Get("Au99.99")
	if !ok {
		t.Fatal("expected quote in cache")
	}
	if gotQuote.Price != 1017.07 {
		t.Fatalf("cached price = %.2f, want 1017.07", gotQuote.Price)
	}

	now := time.Now()
	snapshots, err := repo.ListPriceSnapshots(context.Background(), "Au99.99", now.Add(-time.Hour), now.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("ListPriceSnapshots() err = %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("len = %d, want 1", len(snapshots))
	}
	if snapshots[0].Price != 1017.07 {
		t.Fatalf("stored price = %.2f, want 1017.07", snapshots[0].Price)
	}
}
