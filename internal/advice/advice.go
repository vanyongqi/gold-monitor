package advice

import (
	"fmt"

	"gold_monitor/internal/market"
)

type Position struct {
	CostPerGram float64 `json:"cost_per_gram"`
	Grams       float64 `json:"grams"`
	SellFeeRate float64 `json:"sell_fee_rate"`
}

type Metrics struct {
	BreakEvenPrice     float64 `json:"break_even_price"`
	GrossSellAmount    float64 `json:"gross_sell_amount"`
	NetSellAmount      float64 `json:"net_sell_amount"`
	ProfitAmount       float64 `json:"profit_amount"`
	ProfitRate         float64 `json:"profit_rate"`
	IntradayChangePct  float64 `json:"intraday_change_pct"`
	DistanceToHighPct  float64 `json:"distance_to_high_pct"`
	DistanceFromLowPct float64 `json:"distance_from_low_pct"`
}

type Advice struct {
	Level   string   `json:"level"`
	Summary string   `json:"summary"`
	Reasons []string `json:"reasons"`
}

func CalculateMetrics(pos Position, quote market.Quote) Metrics {
	breakEvenPrice := 0.0
	if 1-pos.SellFeeRate > 0 {
		breakEvenPrice = pos.CostPerGram / (1 - pos.SellFeeRate)
	}

	grossSellAmount := quote.Price * pos.Grams
	netSellAmount := grossSellAmount * (1 - pos.SellFeeRate)
	costAmount := pos.CostPerGram * pos.Grams
	profitAmount := netSellAmount - costAmount

	profitRate := 0.0
	if costAmount > 0 {
		profitRate = profitAmount / costAmount
	}

	intradayChangePct := 0.0
	if quote.Open > 0 {
		intradayChangePct = (quote.Price - quote.Open) / quote.Open
	}

	distanceToHighPct := 0.0
	if quote.High > 0 {
		distanceToHighPct = (quote.High - quote.Price) / quote.High
	}

	distanceFromLowPct := 0.0
	if quote.Low > 0 {
		distanceFromLowPct = (quote.Price - quote.Low) / quote.Low
	}

	return Metrics{
		BreakEvenPrice:     breakEvenPrice,
		GrossSellAmount:    grossSellAmount,
		NetSellAmount:      netSellAmount,
		ProfitAmount:       profitAmount,
		ProfitRate:         profitRate,
		IntradayChangePct:  intradayChangePct,
		DistanceToHighPct:  distanceToHighPct,
		DistanceFromLowPct: distanceFromLowPct,
	}
}

func GenerateAdvice(pos Position, quote market.Quote) Advice {
	metrics := CalculateMetrics(pos, quote)
	reasons := []string{
		fmt.Sprintf("当前价 %.2f 元/克，日内区间 %.2f - %.2f", quote.Price, quote.Low, quote.High),
	}

	if metrics.BreakEvenPrice > 0 {
		reasons = append(reasons, fmt.Sprintf("按卖出手续费 %.2f%% 计算，回本价约 %.2f 元/克", pos.SellFeeRate*100, metrics.BreakEvenPrice))
	}

	switch {
	case metrics.ProfitRate >= 0.015:
		reasons = append(reasons, "当前净收益已经明显覆盖手续费和常规波动缓冲")
		return Advice{
			Level:   "take_profit",
			Summary: "已有较明显浮盈，可考虑分批止盈。",
			Reasons: reasons,
		}
	case metrics.ProfitRate >= 0.004:
		reasons = append(reasons, "当前价格已经覆盖卖出手续费，但还没到很舒服的止盈区")
		return Advice{
			Level:   "hold_profit",
			Summary: "已经覆盖手续费，可继续持有观察，不必急着卖。",
			Reasons: reasons,
		}
	case metrics.DistanceToHighPct <= 0.002 && metrics.IntradayChangePct >= 0.015:
		reasons = append(reasons, "价格接近日内高位，同时日内涨幅偏大，追高胜率一般")
		return Advice{
			Level:   "avoid_chasing",
			Summary: "价格接近日内高位，不建议追高。",
			Reasons: reasons,
		}
	case metrics.DistanceFromLowPct <= 0.003 && metrics.IntradayChangePct <= -0.01:
		reasons = append(reasons, "价格靠近日内低位，且日内有明显回撤，适合小仓分批观察")
		return Advice{
			Level:   "watch_buy",
			Summary: "价格靠近日内低位，可小仓分批关注。",
			Reasons: reasons,
		}
	default:
		reasons = append(reasons, "当前既不算强势突破，也不算明显低吸位置")
		return Advice{
			Level:   "hold_wait",
			Summary: "暂时没有很清晰的优势位置，先观察。",
			Reasons: reasons,
		}
	}
}
