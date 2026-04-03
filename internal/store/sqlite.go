package store

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	_ "modernc.org/sqlite"

	"gold_monitor/internal/market"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLite(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS price_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instrument TEXT NOT NULL,
			price REAL NOT NULL,
			open REAL NOT NULL,
			high REAL NOT NULL,
			low REAL NOT NULL,
			quote_date TEXT NOT NULL,
			source TEXT NOT NULL,
			fetched_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_price_snapshots_lookup ON price_snapshots (instrument, fetched_at);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("init sqlite: %w", err)
		}
	}

	return db, nil
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) SavePriceSnapshot(ctx context.Context, quote market.Quote) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO price_snapshots (instrument, price, open, high, low, quote_date, source, fetched_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		quote.Instrument,
		quote.Price,
		quote.Open,
		quote.High,
		quote.Low,
		quote.QuoteDate.Format("2006-01-02"),
		quote.Source,
		quote.FetchedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert price snapshot: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListPriceSnapshots(ctx context.Context, instrument string, start, end time.Time, limit int) ([]market.Quote, error) {
	query := `SELECT instrument, price, open, high, low, quote_date, source, fetched_at
		FROM price_snapshots
		WHERE instrument = ? AND fetched_at >= ? AND fetched_at <= ?`
	args := []any{
		instrument,
		start.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano),
	}
	if limit > 0 {
		query += ` ORDER BY fetched_at DESC LIMIT ?`
		args = append(args, limit)
	} else {
		query += ` ORDER BY fetched_at ASC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query price snapshots: %w", err)
	}
	defer rows.Close()

	var result []market.Quote
	for rows.Next() {
		var (
			quoteDateRaw string
			fetchedAtRaw string
			item         market.Quote
		)
		if err := rows.Scan(
			&item.Instrument,
			&item.Price,
			&item.Open,
			&item.High,
			&item.Low,
			&quoteDateRaw,
			&item.Source,
			&fetchedAtRaw,
		); err != nil {
			return nil, fmt.Errorf("scan price snapshot: %w", err)
		}
		item.QuoteDate, err = time.ParseInLocation("2006-01-02", quoteDateRaw, time.Local)
		if err != nil {
			return nil, fmt.Errorf("parse quote date: %w", err)
		}
		item.FetchedAt, err = time.Parse(time.RFC3339Nano, fetchedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse fetched at: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate price snapshots: %w", err)
	}
	if limit > 0 {
		slices.Reverse(result)
	}
	return result, nil
}
