package advice

import (
	"math"
	"testing"
	"time"

	"gold_monitor/internal/market"
)

func TestCalculateMetrics(t *testing.T) {
	t.Parallel()

	pos := Position{
		CostPerGram: 1000,
		Grams:       10,
		SellFeeRate: 0.004,
	}
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1048,
		Open:       1032,
		High:       1050,
		Low:        1030,
		QuoteDate:  time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local),
	}

	got := CalculateMetrics(pos, quote)

	if !almostEqual(got.BreakEvenPrice, 1004.0160642570282, 1e-9) {
		t.Fatalf("break even = %.12f", got.BreakEvenPrice)
	}
	if !almostEqual(got.NetSellAmount, 10438.08, 1e-9) {
		t.Fatalf("net sell amount = %.2f, want 10438.08", got.NetSellAmount)
	}
	if !almostEqual(got.ProfitAmount, 438.08, 1e-9) {
		t.Fatalf("profit amount = %.2f, want 438.08", got.ProfitAmount)
	}
	if !almostEqual(got.ProfitRate, 0.043808, 1e-9) {
		t.Fatalf("profit rate = %.6f, want 0.043808", got.ProfitRate)
	}
}

func TestGenerateAdviceTakeProfit(t *testing.T) {
	t.Parallel()

	pos := Position{
		CostPerGram: 1000,
		Grams:       10,
		SellFeeRate: 0.004,
	}
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1048,
		Open:       1032,
		High:       1050,
		Low:        1030,
	}

	got := GenerateAdvice(pos, quote)
	if got.Level != "take_profit" {
		t.Fatalf("level = %q, want %q", got.Level, "take_profit")
	}
}

func TestGenerateAdviceAvoidChasing(t *testing.T) {
	t.Parallel()

	pos := Position{
		CostPerGram: 1100,
		Grams:       10,
		SellFeeRate: 0.004,
	}
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1049,
		Open:       1028,
		High:       1050,
		Low:        1020,
	}

	got := GenerateAdvice(pos, quote)
	if got.Level != "avoid_chasing" {
		t.Fatalf("level = %q, want %q", got.Level, "avoid_chasing")
	}
}

func TestGenerateAdviceWatchBuy(t *testing.T) {
	t.Parallel()

	pos := Position{
		CostPerGram: 1100,
		Grams:       10,
		SellFeeRate: 0.004,
	}
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1001,
		Open:       1018,
		High:       1025,
		Low:        1000,
	}

	got := GenerateAdvice(pos, quote)
	if got.Level != "watch_buy" {
		t.Fatalf("level = %q, want %q", got.Level, "watch_buy")
	}
}

func TestGenerateAdviceNoPositionAvoidChasing(t *testing.T) {
	t.Parallel()

	pos := Position{}
	quote := market.Quote{
		Instrument: "Au99.99",
		Price:      1049,
		Open:       1028,
		High:       1050,
		Low:        1020,
	}

	got := GenerateAdvice(pos, quote)
	if got.Level != "avoid_chasing" {
		t.Fatalf("level = %q, want %q", got.Level, "avoid_chasing")
	}
}

func almostEqual(got, want, epsilon float64) bool {
	return math.Abs(got-want) <= epsilon
}
