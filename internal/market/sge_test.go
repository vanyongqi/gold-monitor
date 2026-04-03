package market

import (
	"strings"
	"testing"
	"time"
)

func TestParseQuoteHTML(t *testing.T) {
	t.Parallel()

	html := strings.Join([]string{
		`<h1>上海黄金交易所2026年04月02日延时行情</h1>`,
		`<tr class="ininfo">`,
		`<td class="insid">Au99.99</td>`,
		`<td>1039.99</td>`,
		`<td>1095.0</td>`,
		`<td>1033.0</td>`,
		`<td>1060.0</td>`,
		`</tr>`,
	}, "")

	got, err := parseQuoteHTML([]byte(html), "Au99.99", time.Date(2026, 4, 2, 11, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("parseQuoteHTML() err = %v", err)
	}

	if got.Instrument != "Au99.99" {
		t.Fatalf("instrument = %q, want %q", got.Instrument, "Au99.99")
	}
	if got.Price != 1039.99 {
		t.Fatalf("price = %.2f, want 1039.99", got.Price)
	}
	if got.High != 1095.0 {
		t.Fatalf("high = %.2f, want 1095.0", got.High)
	}
	if got.Low != 1033.0 {
		t.Fatalf("low = %.2f, want 1033.0", got.Low)
	}
	if got.Open != 1060.0 {
		t.Fatalf("open = %.2f, want 1060.0", got.Open)
	}
	wantDate := time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local)
	if !got.QuoteDate.Equal(wantDate) {
		t.Fatalf("quote date = %v, want %v", got.QuoteDate, wantDate)
	}
}

func TestParseQuoteHTMLBlockedResponse(t *testing.T) {
	t.Parallel()

	_, err := parseQuoteHTML([]byte("您的请求可能存在威胁，已被拦截！"), "Au99.99", time.Now())
	if err == nil {
		t.Fatal("expected error for blocked response")
	}
}

func TestParseQuoteHTMLInstrumentNotFound(t *testing.T) {
	t.Parallel()

	html := `<h1>上海黄金交易所2026年04月02日延时行情</h1>`

	_, err := parseQuoteHTML([]byte(html), "Au99.99", time.Now())
	if err == nil {
		t.Fatal("expected error for missing instrument")
	}
}
