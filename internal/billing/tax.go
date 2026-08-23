package billing

import (
	"math"
	"strings"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type TaxRule struct {
	Country string
	Rate    float64
	Exempt  bool
}

func (r TaxRule) Apply(amount int64) int64 {
	if r.Exempt || r.Rate <= 0 {
		return amount
	}
	return amount + int64(math.Round(float64(amount)*r.Rate))
}
func (r TaxRule) Validate() error {
	if len(strings.TrimSpace(r.Country)) != 2 || r.Rate < 0 || r.Rate > 1 {
		return domain.ErrInvalid
	}
	return nil
}
func TaxRules() []TaxRule {
	return []TaxRule{{Country: "CN", Rate: .06}, {Country: "SG", Rate: .08}, {Country: "PA", Rate: 0, Exempt: true}}
}
func FindTax(country string) (TaxRule, bool) {
	for _, rule := range TaxRules() {
		if rule.Country == strings.ToUpper(country) {
			return rule, true
		}
	}
	return TaxRule{}, false
}
