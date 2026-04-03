package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gold_monitor/internal/advice"
	"gold_monitor/internal/market"
	"gold_monitor/internal/store"
)

func TestBuildFallsBackToLatestSnapshotWhenFetchQuoteFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	db, err := store.OpenSQLite(fmt.Sprintf("file:dashboard-fallback-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	repo := store.NewSQLiteStore(db)
	fallback := market.Quote{
		Instrument: "Au99.99",
		Price:      812.34,
		Open:       808,
		High:       820,
		Low:        801,
		QuoteDate:  time.Date(2026, 4, 3, 0, 0, 0, 0, time.Local),
		FetchedAt:  time.Now().Add(-5 * time.Minute),
		Source:     "snapshot",
	}
	if err := repo.SavePriceSnapshot(context.Background(), fallback); err != nil {
		t.Fatalf("SavePriceSnapshot() err = %v", err)
	}

	client := market.NewClient()
	client.BaseURL = server.URL
	svc := NewService(client)
	svc.Store = repo

	got, err := svc.Build(context.Background(), "Au99.99", advice.Position{})
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if got.Quote.Price != fallback.Price {
		t.Fatalf("quote price = %.2f, want %.2f", got.Quote.Price, fallback.Price)
	}
	if len(got.LiveTrend) == 0 {
		t.Fatal("expected live trend from snapshot fallback")
	}
}

func TestBuildReturnsErrorWhenFetchQuoteFailsAndNoFallback(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failed", http.StatusBadGateway)
	}))
	defer server.Close()

	client := market.NewClient()
	client.BaseURL = server.URL
	svc := NewService(client)

	_, err := svc.Build(context.Background(), "Au99.99", advice.Position{})
	if err == nil {
		t.Fatal("expected error without fallback")
	}
}

func TestBuildFallsBackToOlderSnapshotWhenMarketIsClosed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<h1>上海黄金交易所2026年04月04日延时行情</h1>
<tr class="ininfo">
  <td class="insid">Au99.99</td>
  <td>0.0</td>
  <td>0.0</td>
  <td>0.0</td>
  <td>0.0</td>
</tr>`))
	}))
	defer server.Close()

	db, err := store.OpenSQLite(fmt.Sprintf("file:dashboard-closed-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	repo := store.NewSQLiteStore(db)
	fallback := market.Quote{
		Instrument: "Au99.99",
		Price:      812.34,
		Open:       808,
		High:       820,
		Low:        801,
		QuoteDate:  time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local),
		FetchedAt:  time.Now().Add(-48 * time.Hour),
		Source:     "snapshot",
	}
	if err := repo.SavePriceSnapshot(context.Background(), fallback); err != nil {
		t.Fatalf("SavePriceSnapshot() err = %v", err)
	}

	client := market.NewClient()
	client.BaseURL = server.URL
	svc := NewService(client)
	svc.Store = repo

	got, err := svc.Build(context.Background(), "Au99.99", advice.Position{})
	if err != nil {
		t.Fatalf("Build() err = %v", err)
	}
	if got.Quote.Price != fallback.Price {
		t.Fatalf("quote price = %.2f, want %.2f", got.Quote.Price, fallback.Price)
	}
}
