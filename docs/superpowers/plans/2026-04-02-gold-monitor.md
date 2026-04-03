# Gold Monitor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个 Go 命令行程序，抓取上海黄金交易所延时金价，结合用户持仓成本和卖出手续费给出收益测算与操作建议。

**Architecture:** 程序分为三层：`market` 负责抓取并解析上金所公开行情页，`advice` 负责计算回本价和建议，`cmd` 负责参数解析、单次输出和轮询监控。首版只依赖官方公开延时行情页，不接入需要鉴权的内部接口。

**Tech Stack:** Go 1.24、标准库 `net/http` / `regexp` / `flag` / `time` / `context`

---

### Task 1: 初始化模块和测试骨架

**Files:**
- Create: `go.mod`
- Create: `internal/market/sge_test.go`
- Create: `internal/advice/advice_test.go`

- [ ] **Step 1: 写行情解析失败测试**

```go
func TestParseQuoteHTML(t *testing.T) {
    html := `<h1>上海黄金交易所2026年04月02日延时行情</h1>
<tr class="ininfo">
  <td class="insid">Au99.99</td><td>1039.99</td><td>1095.0</td><td>1033.0</td><td>1060.0</td>
</tr>`

    _, err := parseQuoteHTML([]byte(html), "Au99.99", time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local))

    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
}
```

- [ ] **Step 2: 运行测试确认因未实现而失败**

Run: `go test ./...`
Expected: FAIL with undefined `parseQuoteHTML`

- [ ] **Step 3: 写建议逻辑失败测试**

```go
func TestGenerateAdviceProfitTaking(t *testing.T) {
    quote := market.Quote{Instrument: "Au99.99", Price: 1048, High: 1050, Low: 1030, Open: 1032}
    pos := Position{CostPerGram: 1000, Grams: 10, SellFeeRate: 0.004}

    advice := GenerateAdvice(pos, quote)

    if advice.Level != "take_profit" {
        t.Fatalf("expected take_profit, got %s", advice.Level)
    }
}
```

- [ ] **Step 4: 再次运行测试确认失败**

Run: `go test ./...`
Expected: FAIL with undefined `GenerateAdvice`

### Task 2: 实现市场行情抓取

**Files:**
- Create: `internal/market/sge.go`

- [ ] **Step 1: 实现 `Quote` 和 `Client`**

```go
type Quote struct {
    Instrument string
    Price      float64
    High       float64
    Low        float64
    Open       float64
    QuoteDate  time.Time
    Source     string
    FetchedAt  time.Time
}

type Client struct {
    BaseURL    string
    HTTPClient *http.Client
    now        func() time.Time
}
```

- [ ] **Step 2: 实现最小抓取逻辑**

```go
func (c *Client) FetchQuote(ctx context.Context, instrument string) (Quote, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL, nil)
    if err != nil {
        return Quote{}, err
    }
    req.Header.Set("User-Agent", "gold-monitor/1.0")
    resp, err := c.httpClient().Do(req)
    if err != nil {
        return Quote{}, err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return Quote{}, err
    }
    return parseQuoteHTML(body, instrument, c.nowTime())
}
```

- [ ] **Step 3: 运行测试直到行情解析通过**

Run: `go test ./internal/market -v`
Expected: PASS

### Task 3: 实现收益计算和建议引擎

**Files:**
- Create: `internal/advice/advice.go`

- [ ] **Step 1: 实现收益测算**

```go
func CalculateMetrics(pos Position, quote market.Quote) Metrics {
    cost := pos.CostPerGram * pos.Grams
    gross := quote.Price * pos.Grams
    net := gross * (1 - pos.SellFeeRate)
    profit := net - cost
    return Metrics{
        BreakEvenPrice: pos.CostPerGram / (1 - pos.SellFeeRate),
        NetSellAmount:  net,
        ProfitAmount:   profit,
        ProfitRate:     profit / cost,
    }
}
```

- [ ] **Step 2: 实现建议规则**

```go
switch {
case metrics.ProfitRate >= 0.015:
    return Advice{Level: "take_profit", Summary: "已有较明显浮盈，可考虑分批止盈"}
case metrics.ProfitRate >= 0.004:
    return Advice{Level: "hold_profit", Summary: "已覆盖手续费，先拿着观察"}
case intradayNearHigh:
    return Advice{Level: "avoid_chasing", Summary: "价格接近日内高位，不建议追高"}
case intradayNearLow:
    return Advice{Level: "watch_buy", Summary: "价格靠近日内低位，可小仓分批关注"}
default:
    return Advice{Level: "hold_wait", Summary: "暂未出现清晰优势，先观察"}
}
```

- [ ] **Step 3: 运行建议测试**

Run: `go test ./internal/advice -v`
Expected: PASS

### Task 4: 实现命令行入口

**Files:**
- Create: `cmd/gold-monitor/main.go`

- [ ] **Step 1: 增加参数解析**

```go
instrument := flag.String("instrument", "Au99.99", "上金所合约")
cost := flag.Float64("cost", 0, "买入成本（元/克）")
grams := flag.Float64("grams", 0, "持仓克数")
fee := flag.Float64("sell-fee", 0.004, "卖出手续费率")
watch := flag.Duration("watch", 0, "轮询间隔，例如 60s")
```

- [ ] **Step 2: 实现单次输出和轮询**

```go
if *watch <= 0 {
    runOnce(ctx, client, *instrument, pos)
    return
}
for {
    runOnce(ctx, client, *instrument, pos)
    time.Sleep(*watch)
}
```

- [ ] **Step 3: 运行程序手工验证**

Run: `go run ./cmd/gold-monitor -cost 1000 -grams 10`
Expected: 输出最新价、回本价、净收益、建议

### Task 5: 总体验证

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 补充使用说明**

```md
go run ./cmd/gold-monitor -cost 1000 -grams 10
go run ./cmd/gold-monitor -cost 1000 -grams 10 -watch 60s
```

- [ ] **Step 2: 运行完整验证**

Run: `go test ./... && go run ./cmd/gold-monitor -cost 1000 -grams 10`
Expected: 测试全绿，程序输出一份完整建议
