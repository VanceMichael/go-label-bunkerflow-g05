package billing

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type RateBand struct {
	FromKG     float64
	ToKG       float64
	CentsPerKG int64
}
type PricingPolicy struct {
	Currency           string
	BaseCents          int64
	Bands              []RateBand
	NightFrom          int
	NightUntil         int
	HazardSurcharge    int64
	CurrencyMultiplier map[string]float64
}
type Quote struct {
	Currency   string
	Subtotal   int64
	Surcharges int64
	Total      int64
	Band       string
	Night      bool
	Hazardous  bool
}

func DefaultPolicy() PricingPolicy {
	return PricingPolicy{Currency: "USD", BaseCents: 5000, Bands: []RateBand{{0, 10000, 120}, {10000, 50000, 100}, {50000, math.Inf(1), 80}}, NightFrom: 22, NightUntil: 6, HazardSurcharge: 15000, CurrencyMultiplier: map[string]float64{"USD": 1, "CNY": 7.1}}
}
func (p PricingPolicy) band(quantity float64) RateBand {
	for _, band := range p.Bands {
		if quantity >= band.FromKG && quantity < band.ToKG {
			return band
		}
	}
	return p.Bands[len(p.Bands)-1]
}
func (p PricingPolicy) IsNight(at time.Time) bool {
	hour := at.Hour()
	if p.NightFrom > p.NightUntil {
		return hour >= p.NightFrom || hour < p.NightUntil
	}
	return hour >= p.NightFrom && hour < p.NightUntil
}
func (p PricingPolicy) Quote(quantity float64, at time.Time, hazardous bool, currency string) (Quote, error) {
	if quantity <= 0 {
		return Quote{}, domain.ErrInvalid
	}
	if p.CurrencyMultiplier[currency] == 0 {
		return Quote{}, fmt.Errorf("%w: %s", domain.ErrInvalid, currency)
	}
	band := p.band(quantity)
	subtotal := p.BaseCents + int64(math.Round(quantity))*band.CentsPerKG
	surcharge := int64(0)
	night := p.IsNight(at)
	if night {
		surcharge += 10000
	}
	if hazardous {
		surcharge += p.HazardSurcharge
	}
	multiplier := p.CurrencyMultiplier[currency]
	return Quote{Currency: currency, Subtotal: int64(float64(subtotal) * multiplier), Surcharges: int64(float64(surcharge) * multiplier), Total: int64(float64(subtotal+surcharge) * multiplier), Band: fmt.Sprintf("%.0f-%.0f", band.FromKG, band.ToKG), Night: night, Hazardous: hazardous}, nil
}
func (p PricingPolicy) Validate() error {
	if p.BaseCents < 0 || len(p.Bands) == 0 {
		return domain.ErrInvalid
	}
	copyOf := append([]RateBand(nil), p.Bands...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].FromKG < copyOf[j].FromKG })
	for i, band := range copyOf {
		if band.FromKG < 0 || band.CentsPerKG <= 0 || i > 0 && band.FromKG < copyOf[i-1].ToKG {
			return domain.ErrInvalid
		}
	}
	return nil
}
