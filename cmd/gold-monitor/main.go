package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gold_monitor/internal/advice"
	"gold_monitor/internal/cache"
	"gold_monitor/internal/dashboard"
	"gold_monitor/internal/market"
	gruntime "gold_monitor/internal/runtime"
	"gold_monitor/internal/store"
	"gold_monitor/internal/web"
)

func main() {
	var (
		instrument = flag.String("instrument", "Au99.99", "上金所合约名称")
		cost       = flag.Float64("cost", 0, "买入成本（元/克）")
		grams      = flag.Float64("grams", 0, "持仓克数")
		sellFee    = flag.Float64("sell-fee", 0.004, "卖出手续费率，例如 0.004 表示千分之四")
		watch      = flag.Duration("watch", 0, "轮询间隔，例如 60s；不传则只查询一次")
		httpMode   = flag.Bool("http", false, "启动本地前端监控页")
		collector  = flag.Bool("collector", false, "仅启动后台采集器，不启动前端")
		listenAddr = flag.String("listen", ":8080", "HTTP 服务监听地址")
		dbPath     = flag.String("db", filepath.Join("data", "gold_monitor.db"), "SQLite 数据库路径")
		interval   = flag.Duration("interval", time.Minute, "后台采集间隔")
	)
	flag.Parse()

	pos := advice.Position{
		CostPerGram: *cost,
		Grams:       *grams,
		SellFeeRate: *sellFee,
	}
	client := market.NewClient()
	ctx := context.Background()

	if *collector {
		if err := runCollector(client, *instrument, *dbPath, *interval); err != nil {
			fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *httpMode {
		if err := runHTTP(client, *instrument, pos, *listenAddr, *dbPath, *interval); err != nil {
			fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	var err error
	if *watch > 0 {
		err = runWatch(ctx, client, *instrument, pos, *watch)
	} else {
		err = runOnce(ctx, client, *instrument, pos)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "运行失败: %v\n", err)
		os.Exit(1)
	}
}

func runHTTP(client *market.Client, instrument string, pos advice.Position, listenAddr, dbPath string, interval time.Duration) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	cacheLayer := cache.NewQuoteCache()
	repo := store.NewSQLiteStore(db)
	service := dashboard.NewService(client)
	service.Cache = cacheLayer
	service.Store = repo

	refreshCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	refresher := &gruntime.Refresher{
		Market:     client,
		Cache:      cacheLayer,
		Store:      repo,
		Instrument: instrument,
		Interval:   interval,
		Logger:     log.New(os.Stderr, "[refresher] ", log.LstdFlags),
	}
	refresher.Start(refreshCtx)

	server := &web.Server{
		Service:           service,
		DefaultInstrument: instrument,
		DefaultPosition:   pos,
	}
	handler, err := server.Handler()
	if err != nil {
		return err
	}

	fmt.Printf("前端监控页已启动: http://localhost%s\n", listenAddr)
	return http.ListenAndServe(listenAddr, handler)
}

func runCollector(client *market.Client, instrument, dbPath string, interval time.Duration) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	refresher := &gruntime.Refresher{
		Market:     client,
		Store:      store.NewSQLiteStore(db),
		Instrument: instrument,
		Interval:   interval,
		Logger:     log.New(os.Stderr, "[collector] ", log.LstdFlags),
	}
	ctx := context.Background()
	if err := refresher.Refresh(ctx); err != nil {
		return err
	}

	refresher.Start(ctx)
	fmt.Printf("后台采集器已启动: instrument=%s interval=%s db=%s\n", instrument, interval, dbPath)
	select {}
}

func runWatch(ctx context.Context, client *market.Client, instrument string, pos advice.Position, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("watch 间隔必须大于 0")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := runOnce(ctx, client, instrument, pos); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] 获取失败: %v\n", time.Now().Format(time.DateTime), err)
		}

		<-ticker.C
		fmt.Println()
	}
}

func runOnce(ctx context.Context, client *market.Client, instrument string, pos advice.Position) error {
	quote, err := client.FetchQuote(ctx, instrument)
	if err != nil {
		return err
	}

	metrics := advice.CalculateMetrics(pos, quote)
	suggestion := advice.GenerateAdvice(pos, quote)

	fmt.Printf("[%s] %s / %s\n", time.Now().Format(time.DateTime), quote.Source, quote.Instrument)
	fmt.Printf("行情日期: %s\n", quote.QuoteDate.Format("2006-01-02"))
	fmt.Printf("最新价: %.2f 元/克\n", quote.Price)
	fmt.Printf("日内: 开 %.2f / 高 %.2f / 低 %.2f\n", quote.Open, quote.High, quote.Low)

	if pos.CostPerGram > 0 && pos.Grams > 0 {
		fmt.Printf("持仓: %.4f 克，成本 %.2f 元/克，卖出手续费 %.2f%%\n", pos.Grams, pos.CostPerGram, pos.SellFeeRate*100)
		fmt.Printf("回本价: %.2f 元/克\n", metrics.BreakEvenPrice)
		fmt.Printf("按当前价卖出到账: %.2f 元\n", metrics.NetSellAmount)
		fmt.Printf("净收益: %.2f 元 (%.2f%%)\n", metrics.ProfitAmount, metrics.ProfitRate*100)
	} else {
		fmt.Printf("未提供完整持仓参数，当前只输出盘口建议。\n")
	}

	fmt.Printf("建议: %s\n", suggestion.Summary)
	for _, reason := range suggestion.Reasons {
		fmt.Printf("- %s\n", reason)
	}

	return nil
}
