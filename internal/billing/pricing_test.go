package billing

import (
	"testing"
	"time"
)

func TestPricingAppliesBandsAndNightSurcharge(t *testing.T) {
	p := DefaultPolicy()
	day := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	night := time.Date(2026, 8, 23, 23, 0, 0, 0, time.UTC)
	a, err := p.Quote(1000, day, false, "USD")
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Quote(1000, night, false, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if b.Total <= a.Total || !b.Night {
		t.Fatalf("day=%+v night=%+v", a, b)
	}
}
func TestPricingHazardAndCurrency(t *testing.T) {
	p := DefaultPolicy()
	q, err := p.Quote(12000, time.Now(), true, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	if q.Total <= q.Subtotal || q.Currency != "CNY" {
		t.Fatalf("quote=%+v", q)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}
