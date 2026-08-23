package vessel

import (
	"sort"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Registry struct{ Vessels []domain.Vessel }

func (r Registry) FindIMO(imo string) (domain.Vessel, bool) {
	imo = domain.NormalizeIMO(imo)
	for _, item := range r.Vessels {
		if item.IMO == imo {
			return item, true
		}
	}
	return domain.Vessel{}, false
}
func (r Registry) Search(term string) []domain.Vessel {
	term = strings.ToUpper(strings.TrimSpace(term))
	result := make([]domain.Vessel, 0)
	for _, item := range r.Vessels {
		if strings.Contains(strings.ToUpper(item.IMO), term) || strings.Contains(strings.ToUpper(item.Name), term) {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func (r Registry) Expiring(now time.Time, within time.Duration) []domain.Vessel {
	limit := now.Add(within)
	result := make([]domain.Vessel, 0)
	for _, item := range r.Vessels {
		if item.Certificate.ExpiresAt.After(now) && item.Certificate.ExpiresAt.Before(limit) {
			result = append(result, item)
		}
	}
	return result
}
func (r *Registry) Add(item domain.Vessel) error {
	for _, existing := range r.Vessels {
		if existing.TenantID == item.TenantID && existing.IMO == item.IMO {
			return domain.ErrConflict
		}
	}
	r.Vessels = append(r.Vessels, item)
	return nil
}
