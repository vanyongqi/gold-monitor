package market

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultHistoryURL = "https://www.sge.com.cn/sjzx/quotation_daily_new"

var (
	dailyRowRe  = regexp.MustCompile(`(?s)<tr>\s*(?:<!--.*?-->\s*)?<td[^>]*>\s*(\d{4}-\d{2}-\d{2})\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>\s*<td[^>]*>\s*([^<]+?)\s*</td>`)
	totalPageRe = regexp.MustCompile(`var totalPage=(\d+)`)
)

type DailyQuote struct {
	Date       time.Time `json:"date"`
	Instrument string    `json:"instrument"`
	Open       float64   `json:"open"`
	High       float64   `json:"high"`
	Low        float64   `json:"low"`
	Close      float64   `json:"close"`
}

func (c *Client) FetchHistory(ctx context.Context, instrument string, start, end time.Time) ([]DailyQuote, error) {
	if end.Before(start) {
		return nil, errors.New("end date must be on or after start date")
	}

	all := make(map[string]DailyQuote)
	windowEnd := end
	for !windowEnd.Before(start) {
		windowStart := windowEnd.AddDate(0, 0, -30)
		if windowStart.Before(start) {
			windowStart = start
		}

		daily, err := c.fetchHistoryWindow(ctx, instrument, windowStart, windowEnd)
		if err != nil {
			return nil, err
		}
		for _, item := range daily {
			all[item.Date.Format("2006-01-02")] = item
		}

		if windowStart.Equal(start) {
			break
		}
		windowEnd = windowStart.AddDate(0, 0, -1)
	}

	result := make([]DailyQuote, 0, len(all))
	for _, item := range all {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result, nil
}

func (c *Client) fetchHistoryWindow(ctx context.Context, instrument string, start, end time.Time) ([]DailyQuote, error) {
	body, err := c.fetchHistoryPage(ctx, c.historyURL(start, end, instrument, 0))
	if err != nil {
		return nil, err
	}
	result, err := parseDailyQuotesHTML(body, instrument, c.nowTime().Location())
	if err != nil {
		return nil, err
	}

	totalPages := parseTotalPages(body)
	for page := 2; page <= totalPages; page++ {
		pageBody, err := c.fetchHistoryPage(ctx, c.historyURL(start, end, instrument, page))
		if err != nil {
			return nil, err
		}
		pageResult, err := parseDailyQuotesHTML(pageBody, instrument, c.nowTime().Location())
		if err != nil {
			return nil, err
		}
		result = append(result, pageResult...)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result, nil
}

func (c *Client) fetchHistoryPage(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create history request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 gold-monitor/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch sge history: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected history status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read history response body: %w", err)
	}
	return body, nil
}

func (c *Client) historyURL(start, end time.Time, instrument string, page int) string {
	values := url.Values{}
	values.Set("start_date", start.Format("2006-01-02"))
	values.Set("end_date", end.Format("2006-01-02"))
	values.Set("inst_ids", instrument)
	if page > 0 {
		values.Set("p", strconv.Itoa(page))
	}
	return c.historyBaseURL() + "?" + values.Encode()
}

func (c *Client) historyBaseURL() string {
	if strings.TrimSpace(c.HistoryBaseURL) != "" {
		return c.HistoryBaseURL
	}
	return defaultHistoryURL
}

func parseDailyQuotesHTML(body []byte, instrument string, loc *time.Location) ([]DailyQuote, error) {
	html := string(body)
	if strings.Contains(html, "请求可能存在威胁") || strings.Contains(html, "请求已被阻断") {
		return nil, errors.New("sge blocked the history request")
	}

	matches := dailyRowRe.FindAllStringSubmatch(html, -1)
	result := make([]DailyQuote, 0, len(matches))
	for _, match := range matches {
		if len(match) != 7 {
			continue
		}
		name := strings.TrimSpace(match[2])
		if name != instrument {
			continue
		}

		date, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(match[1]), loc)
		if err != nil {
			return nil, fmt.Errorf("parse history date: %w", err)
		}
		open, err := parsePrice(match[3])
		if err != nil {
			return nil, fmt.Errorf("parse history open: %w", err)
		}
		high, err := parsePrice(match[4])
		if err != nil {
			return nil, fmt.Errorf("parse history high: %w", err)
		}
		low, err := parsePrice(match[5])
		if err != nil {
			return nil, fmt.Errorf("parse history low: %w", err)
		}
		closePrice, err := parsePrice(match[6])
		if err != nil {
			return nil, fmt.Errorf("parse history close: %w", err)
		}

		result = append(result, DailyQuote{
			Date:       date,
			Instrument: instrument,
			Open:       open,
			High:       high,
			Low:        low,
			Close:      closePrice,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("instrument %q not found in history page", instrument)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})
	return result, nil
}

func parseTotalPages(body []byte) int {
	match := totalPageRe.FindSubmatch(body)
	if len(match) != 2 {
		return 1
	}
	total, err := strconv.Atoi(string(match[1]))
	if err != nil || total < 1 {
		return 1
	}
	return total
}
