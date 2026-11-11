package services

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Every list has USD before KHR, and groups are A to Z ignoring case with agents in no group last.
func TestReportOrder(t *testing.T) {
	if !currencyBefore("USD", "KHR") || currencyBefore("KHR", "USD") || !currencyBefore("KHR", "EUR") || currencyBefore("USD", "USD") {
		t.Error("USD, then KHR, then the rest")
	}
	lines := []ReportLine{{PlatformCode: "b", Sums: Sums{Currency: "KHR"}}, {PlatformCode: "B", Sums: Sums{Currency: "USD"}}, {PlatformCode: "a", Sums: Sums{Currency: "KHR"}}, {PlatformCode: "a", Sums: Sums{Currency: "USD"}}}
	sortLines(lines)
	var got []string
	for _, l := range lines {
		got = append(got, l.PlatformCode+" "+l.Currency)
	}
	if strings.Join(got, ", ") != "a USD, a KHR, B USD, b KHR" {
		t.Errorf("lines: %v", got)
	}
	id := uuid.New()
	groups := []GroupTotalsRow{{Title: ""}, {GroupID: &id, Title: "beta"}, {GroupID: &id, Title: "Alpha"}, {GroupID: &id, Title: "Gamma"}}
	sortGroups(groups)
	got = got[:0]
	for _, g := range groups {
		got = append(got, g.Title)
	}
	if strings.Join(got, ",") != "Alpha,beta,Gamma," {
		t.Errorf("groups: %v", got)
	}
}
