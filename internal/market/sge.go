package market

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultSGEURL = "https://www.sge.com.cn/h5_sjzx/yshq"

var (
	quoteDateRe = regexp.MustCompile(`上海黄金交易所(\d{4})年(\d{2})月(\d{2})日延时行情`)
	quoteRowRe  = regexp.MustCompile(`(?s)<tr class="ininfo">\s*<td[^>]*class="insid"[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*</tr>`)
)

type Quote struct {
	Instrument string    `json:"instrument"`
	Price      float64   `json:"price"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Open       float64   `json:"open"`
	QuoteDate  time.Time `json:"quote_date"`
	Source     string    `json:"source"`
	FetchedAt  time.Time `json:"fetched_at"`
}

type Client struct {
	BaseURL        string
	HistoryBaseURL string
	HTTPClient     *http.Client
	now            func() time.Time
}

func NewClient() *Client {
	return &Client{
		BaseURL: defaultSGEURL,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		now: time.Now,
	}
}

func (c *Client) FetchQuote(ctx context.Context, instrument string) (Quote, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL(), nil)
	if err != nil {
		return Quote{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "gold-monitor/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("fetch sge quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Quote{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("read response body: %w", err)
	}

	return parseQuoteHTML(body, instrument, c.nowTime())
}

func (c *Client) baseURL() string {
	if strings.TrimSpace(c.BaseURL) != "" {
		return c.BaseURL
	}
	return defaultSGEURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) nowTime() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func parseQuoteHTML(body []byte, instrument string, fetchedAt time.Time) (Quote, error) {
	html := string(body)
	if strings.Contains(html, "请求可能存在威胁") || strings.Contains(html, "请求已被阻断") {
		return Quote{}, errors.New("sge blocked the request")
	}

	quoteDate, err := parseQuoteDate(html, fetchedAt.Location())
	if err != nil {
		return Quote{}, err
	}

	matches := quoteRowRe.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) != 6 {
			continue
		}

		name := strings.TrimSpace(match[1])
		if name != instrument {
			continue
		}

		price, err := parsePrice(match[2])
		if err != nil {
			return Quote{}, fmt.Errorf("parse latest price: %w", err)
		}
		high, err := parsePrice(match[3])
		if err != nil {
			return Quote{}, fmt.Errorf("parse high price: %w", err)
		}
		low, err := parsePrice(match[4])
		if err != nil {
			return Quote{}, fmt.Errorf("parse low price: %w", err)
		}
		open, err := parsePrice(match[5])
		if err != nil {
			return Quote{}, fmt.Errorf("parse open price: %w", err)
		}

		quote := Quote{
			Instrument: instrument,
			Price:      price,
			High:       high,
			Low:        low,
			Open:       open,
			QuoteDate:  quoteDate,
			Source:     "上海黄金交易所延时行情",
			FetchedAt:  fetchedAt,
		}
		if isZeroQuote(quote) {
			return Quote{}, errors.New("invalid zero quote from sge")
		}

		return quote, nil
	}

	return Quote{}, fmt.Errorf("instrument %q not found in sge quote page", instrument)
}

func isZeroQuote(quote Quote) bool {
	return quote.Price == 0 && quote.High == 0 && quote.Low == 0 && quote.Open == 0
}

func parseQuoteDate(html string, loc *time.Location) (time.Time, error) {
	match := quoteDateRe.FindStringSubmatch(html)
	if len(match) != 4 {
		return time.Time{}, errors.New("quote date not found")
	}

	year, err := strconv.Atoi(match[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse quote year: %w", err)
	}
	month, err := strconv.Atoi(match[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse quote month: %w", err)
	}
	day, err := strconv.Atoi(match[3])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse quote day: %w", err)
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc), nil
}

func parsePrice(raw string) (float64, error) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if cleaned == "" {
		return 0, errors.New("empty price")
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float %q: %w", cleaned, err)
	}
	return value, nil
}
