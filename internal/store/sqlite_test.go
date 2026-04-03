package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gold_monitor/internal/market"
)

func TestSQLiteStoreSaveAndListSnapshots(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(fmt.Sprintf("file:sqlite-save-list-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLiteStore(db)

	snapshots := []market.Quote{
		{
			Instrument: "Au99.99",
			Price:      1001,
			High:       1005,
			Low:        998,
			Open:       999,
			QuoteDate:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local),
			FetchedAt:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local),
			Source:     "test",
		},
		{
			Instrument: "Au99.99",
			Price:      1008,
			High:       1010,
			Low:        1000,
			Open:       1002,
			QuoteDate:  time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local),
			FetchedAt:  time.Date(2026, 4, 2, 10, 0, 0, 0, time.Local),
			Source:     "test",
		},
	}

	for _, snapshot := range snapshots {
		if err := repo.SavePriceSnapshot(ctx, snapshot); err != nil {
			t.Fatalf("SavePriceSnapshot() err = %v", err)
		}
	}

	got, err := repo.ListPriceSnapshots(ctx, "Au99.99", time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local), time.Date(2026, 4, 2, 23, 59, 59, 0, time.Local), 10)
	if err != nil {
		t.Fatalf("ListPriceSnapshots() err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Price != 1001 {
		t.Fatalf("first price = %.2f, want 1001", got[0].Price)
	}
	if got[1].Price != 1008 {
		t.Fatalf("second price = %.2f, want 1008", got[1].Price)
	}
}

func TestSQLiteStoreListPriceSnapshotsReturnsLatestWindow(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(fmt.Sprintf("file:sqlite-window-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLiteStore(db)
	base := time.Date(2026, 4, 3, 10, 0, 0, 0, time.Local)
	for i := 0; i < 5; i++ {
		err := repo.SavePriceSnapshot(ctx, market.Quote{
			Instrument: "Au99.99",
			Price:      float64(1000 + i),
			Open:       1000,
			High:       1005,
			Low:        995,
			QuoteDate:  base,
			FetchedAt:  base.Add(time.Duration(i) * time.Minute),
			Source:     "test",
		})
		if err != nil {
			t.Fatalf("SavePriceSnapshot() err = %v", err)
		}
	}

	got, err := repo.ListPriceSnapshots(ctx, "Au99.99", base.Add(-time.Minute), base.Add(10*time.Minute), 2)
	if err != nil {
		t.Fatalf("ListPriceSnapshots() err = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Price != 1003 || got[1].Price != 1004 {
		t.Fatalf("unexpected prices: %.2f %.2f", got[0].Price, got[1].Price)
	}
}

func TestSQLiteStoreGetLatestPriceSnapshot(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(fmt.Sprintf("file:sqlite-latest-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLiteStore(db)
	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local)
	for i := 0; i < 3; i++ {
		err := repo.SavePriceSnapshot(ctx, market.Quote{
			Instrument: "Au99.99",
			Price:      float64(1000 + i),
			Open:       1000,
			High:       1005,
			Low:        995,
			QuoteDate:  base,
			FetchedAt:  base.Add(time.Duration(i) * 24 * time.Hour),
			Source:     "test",
		})
		if err != nil {
			t.Fatalf("SavePriceSnapshot() err = %v", err)
		}
	}

	got, ok, err := repo.GetLatestPriceSnapshot(ctx, "Au99.99")
	if err != nil {
		t.Fatalf("GetLatestPriceSnapshot() err = %v", err)
	}
	if !ok {
		t.Fatal("expected latest snapshot")
	}
	if got.Price != 1002 {
		t.Fatalf("latest price = %.2f, want 1002", got.Price)
	}
}

func TestSQLiteStoreGetLatestPriceSnapshotSkipsZeroQuotes(t *testing.T) {
	t.Parallel()

	db, err := OpenSQLite(fmt.Sprintf("file:sqlite-latest-valid-%d?mode=memory&cache=shared", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("OpenSQLite() err = %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLiteStore(db)
	base := time.Date(2026, 4, 3, 10, 0, 0, 0, time.Local)

	valid := market.Quote{
		Instrument: "Au99.99",
		Price:      1034.42,
		Open:       1030,
		High:       1042,
		Low:        1020,
		QuoteDate:  base,
		FetchedAt:  base,
		Source:     "test",
	}
	zero := market.Quote{
		Instrument: "Au99.99",
		Price:      0,
		Open:       0,
		High:       0,
		Low:        0,
		QuoteDate:  base.Add(24 * time.Hour),
		FetchedAt:  base.Add(24 * time.Hour),
		Source:     "test",
	}
	for _, snapshot := range []market.Quote{valid, zero} {
		if err := repo.SavePriceSnapshot(ctx, snapshot); err != nil {
			t.Fatalf("SavePriceSnapshot() err = %v", err)
		}
	}

	got, ok, err := repo.GetLatestPriceSnapshot(ctx, "Au99.99")
	if err != nil {
		t.Fatalf("GetLatestPriceSnapshot() err = %v", err)
	}
	if !ok {
		t.Fatal("expected latest valid snapshot")
	}
	if got.Price != valid.Price {
		t.Fatalf("latest valid price = %.2f, want %.2f", got.Price, valid.Price)
	}
}
