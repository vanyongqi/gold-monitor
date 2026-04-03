package market

import (
	"strings"
	"testing"
	"time"
)

func TestParseDailyQuotesHTML(t *testing.T) {
	t.Parallel()

	html := strings.Join([]string{
		`<table>`,
		`<tr>`,
		`<td style="width:80px">2026-03-31</td>`,
		`<td style="width:80px">Au99.99</td>`,
		`<td>1015.00</td>`,
		`<td>1029.99</td>`,
		`<td>1008.00</td>`,
		`<td>1018.90</td>`,
		`<td>10.15</td>`,
		`<td>1.01%</td>`,
		`<td>1020.33</td>`,
		`</tr>`,
		`</table>`,
	}, "")

	got, err := parseDailyQuotesHTML([]byte(html), "Au99.99", time.Local)
	if err != nil {
		t.Fatalf("parseDailyQuotesHTML() err = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Close != 1018.90 {
		t.Fatalf("close = %.2f, want 1018.90", got[0].Close)
	}
	if got[0].Date.Format("2006-01-02") != "2026-03-31" {
		t.Fatalf("date = %s", got[0].Date.Format("2006-01-02"))
	}
}
