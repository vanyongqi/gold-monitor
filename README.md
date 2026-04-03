# gold_monitor

一个面向个人买金用户的 Go 命令行程序。

当前版本会：

- 抓取上海黄金交易所公开`延时行情`
- 输出指定合约的最新价、开盘价、最高价、最低价
- 结合你的`买入成本`、`持仓克数`、`卖出手续费`
- 计算`回本价`、`当前卖出到账金额`、`净收益`
- 给出保守型建议：`可止盈`、`继续观察`、`不建议追高`、`可小仓关注`

## 运行

单次查询：

```bash
go run ./cmd/gold-monitor -cost 1000 -grams 10
```

持续监控：

```bash
go run ./cmd/gold-monitor -cost 1000 -grams 10 -watch 60s
```

启动前端监控页：

```bash
go run ./cmd/gold-monitor -http -listen :8080
```

带持仓参数启动前端监控页：

```bash
go run ./cmd/gold-monitor -http -listen :8080 -cost 1000 -grams 10
```

只启动后台采集器：

```bash
go run ./cmd/gold-monitor -collector -db data/gold_monitor.db -interval 60s
```

一键启动本地采集器 + 网页：

```bash
./scripts/start-local-stack.sh
```

停止本地采集器 + 网页：

```bash
./scripts/stop-local-stack.sh
```

## Docker

构建镜像：

```bash
docker build -t gold-monitor:latest .
```

运行容器：

```bash
docker run -d \
  --name gold-monitor \
  -p 127.0.0.1:18090:8080 \
  -v "$(pwd)/data:/app/storage" \
  gold-monitor:latest
```

## GitHub Actions 与 `/gold` 子路径部署

- 已补充 GitHub Actions：`.github/workflows/publish-image.yml`
- 已补充 Dockerfile：`Dockerfile`
- 已补充服务器拉镜像脚本：`deploy/gold/deploy-ghcr.sh`
- 已补充 Nginx 子路径示例：`deploy/README.md`

线上建议通过 Nginx 把：

`https://718614413.xyz/gold/`

转发到：

`127.0.0.1:18090`

只看盘口建议：

```bash
go run ./cmd/gold-monitor
```

## 参数

```text
-instrument string
    上金所合约名称，默认 Au99.99
-cost float
    买入成本（元/克）
-grams float
    持仓克数
-sell-fee float
    卖出手续费率，默认 0.004
-watch duration
    轮询间隔，例如 60s
-http
    启动本地前端监控页
-collector
    仅启动后台采集器，不启动前端
-listen string
    HTTP 服务监听地址，默认 :8080
-db string
    SQLite 数据库路径，默认 data/gold_monitor.db
-interval duration
    后台采集间隔，默认 60s
```

## 说明

- 数据源为上海黄金交易所公开`延时行情`页面，不是逐笔实时成交。
- 后台采集器会统一抓价并写入 SQLite，适合后面给多人共享同一份监控数据。
- 页面会自动刷新当前价格，`实时监控曲线`优先读取后端统一采集到的快照。
- 历史趋势来自上金所公开`每日行情`页面，可切 `7天 / 30天 / 90天 / 1年`。
- 第一版建议引擎偏保守，目的是帮你过滤`追高`和识别`回本/止盈区`，不是自动交易系统。
