package dashboard

import (
	"context"
	"fmt"
	"time"

	"gold_monitor/internal/advice"
	"gold_monitor/internal/cache"
	"gold_monitor/internal/market"
	"gold_monitor/internal/store"
)

type Service struct {
	Market *market.Client
	Cache  *cache.QuoteCache
	Store  *store.SQLiteStore
}

type Point struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

type KeyLevels struct {
	RecentSupport    float64 `json:"recent_support"`
	RecentResistance float64 `json:"recent_resistance"`
	BreakEven        float64 `json:"break_even"`
	TargetOne        float64 `json:"target_one"`
}

type Response struct {
	Instrument     string         `json:"instrument"`
	Quote          market.Quote   `json:"quote"`
	Metrics        advice.Metrics `json:"metrics"`
	Advice         advice.Advice  `json:"advice"`
	History        []Point        `json:"history"`
	ProfitTrend    []Point        `json:"profit_trend"`
	LiveTrend      []Point        `json:"live_trend"`
	KeyLevels      KeyLevels      `json:"key_levels"`
	HasPosition    bool           `json:"has_position"`
	RefreshSeconds int            `json:"refresh_seconds"`
}

func NewService(client *market.Client) *Service {
	return &Service{Market: client}
}

func (s *Service) Build(ctx context.Context, instrument string, pos advice.Position) (Response, error) {
	if s.Market == nil {
		return Response{}, fmt.Errorf("market client is required")
	}

	quote, ok := s.cachedQuote(instrument)
	if !ok {
		var err error
		quote, err = s.Market.FetchQuote(ctx, instrument)
		if err != nil {
			if fallback, fallbackOK := s.latestSnapshot(ctx, instrument); fallbackOK {
				quote = fallback
			} else {
				return Response{}, err
			}
		}
		if s.Cache != nil && quote.Price > 0 {
			s.Cache.Set(quote)
		}
	}

	history, err := s.getHistory(ctx, instrument, quote.QuoteDate)
	if err != nil {
		return Response{}, err
	}

	metrics := advice.CalculateMetrics(pos, quote)
	suggestion := advice.GenerateAdvice(pos, quote)

	return Response{
		Instrument:     instrument,
		Quote:          quote,
		Metrics:        metrics,
		Advice:         suggestion,
		History:        toClosePoints(history, quote),
		ProfitTrend:    toProfitPoints(history, quote, pos),
		LiveTrend:      toLivePoints(s.snapshotQuotes(ctx, instrument)),
		KeyLevels:      buildKeyLevels(history, quote, metrics),
		HasPosition:    pos.CostPerGram > 0 && pos.Grams > 0,
		RefreshSeconds: 60,
	}, nil
}

func (s *Service) cachedQuote(instrument string) (market.Quote, bool) {
	if s.Cache == nil {
		return market.Quote{}, false
	}
	return s.Cache.Get(instrument)
}

func (s *Service) getHistory(ctx context.Context, instrument string, end time.Time) ([]market.DailyQuote, error) {
	start := end.AddDate(-1, 0, 0)
	return s.Market.FetchHistory(ctx, instrument, start, end)
}

func (s *Service) snapshotQuotes(ctx context.Context, instrument string) []market.Quote {
	if s.Store != nil {
		end := time.Now()
		start := end.Add(-24 * time.Hour)
		items, err := s.Store.ListPriceSnapshots(ctx, instrument, start, end, 240)
		if err == nil && len(items) > 0 {
			return items
		}
	}

	if quote, ok := s.cachedQuote(instrument); ok {
		return []market.Quote{quote}
	}
	return nil
}

func (s *Service) latestSnapshot(ctx context.Context, instrument string) (market.Quote, bool) {
	if s.Store == nil {
		return market.Quote{}, false
	}

	item, ok, err := s.Store.GetLatestPriceSnapshot(ctx, instrument)
	if err != nil || !ok {
		return market.Quote{}, false
	}
	return item, true
}

func toClosePoints(history []market.DailyQuote, quote market.Quote) []Point {
	points := make([]Point, 0, len(history)+1)
	for _, item := range history {
		points = append(points, Point{
			Time:  item.Date.Format("2006-01-02"),
			Value: item.Close,
		})
	}

	if len(points) == 0 || points[len(points)-1].Time != quote.QuoteDate.Format("2006-01-02") {
		points = append(points, Point{
			Time:  quote.QuoteDate.Format("2006-01-02"),
			Value: quote.Price,
		})
	} else {
		points[len(points)-1].Value = quote.Price
	}
	return points
}

func toProfitPoints(history []market.DailyQuote, quote market.Quote, pos advice.Position) []Point {
	if pos.CostPerGram <= 0 || pos.Grams <= 0 {
		return nil
	}

	points := make([]Point, 0, len(history)+1)
	for _, item := range history {
		netSellAmount := item.Close * pos.Grams * (1 - pos.SellFeeRate)
		costAmount := pos.CostPerGram * pos.Grams
		profit := netSellAmount - costAmount
		points = append(points, Point{
			Time:  item.Date.Format("2006-01-02"),
			Value: profit,
		})
	}

	netSellAmount := quote.Price * pos.Grams * (1 - pos.SellFeeRate)
	costAmount := pos.CostPerGram * pos.Grams
	profit := netSellAmount - costAmount
	if len(points) == 0 || points[len(points)-1].Time != quote.QuoteDate.Format("2006-01-02") {
		points = append(points, Point{
			Time:  quote.QuoteDate.Format("2006-01-02"),
			Value: profit,
		})
	} else {
		points[len(points)-1].Value = profit
	}
	return points
}

func toLivePoints(items []market.Quote) []Point {
	points := make([]Point, 0, len(items))
	for _, item := range items {
		points = append(points, Point{
			Time:  item.FetchedAt.Format(time.RFC3339),
			Value: item.Price,
		})
	}
	return points
}

func buildKeyLevels(history []market.DailyQuote, quote market.Quote, metrics advice.Metrics) KeyLevels {
	recent := history
	if len(recent) > 20 {
		recent = recent[len(recent)-20:]
	}

	support := quote.Low
	resistance := quote.High
	if support == 0 {
		support = quote.Price
	}
	if resistance == 0 {
		resistance = quote.Price
	}
	for _, item := range recent {
		if item.Low > 0 && item.Low < support {
			support = item.Low
		}
		if item.High > resistance {
			resistance = item.High
		}
	}

	targetOne := quote.Price * 1.01
	if metrics.BreakEvenPrice > 0 {
		targetOne = max(metrics.BreakEvenPrice*1.01, quote.Price)
	}

	return KeyLevels{
		RecentSupport:    support,
		RecentResistance: resistance,
		BreakEven:        metrics.BreakEvenPrice,
		TargetOne:        targetOne,
	}
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
