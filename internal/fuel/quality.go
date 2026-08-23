package fuel

import (
	"fmt"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Analysis struct {
	MethanolPercent float64
	WaterPPM        float64
	Density         float64
	SulfurPPM       float64
	TestedAt        time.Time
	Lab             string
}

func ValidateAnalysis(a Analysis) error {
	if a.MethanolPercent < 95 || a.MethanolPercent > 100 {
		return fmt.Errorf("%w: methanol percentage", domain.ErrInvalid)
	}
	if a.WaterPPM < 0 || a.WaterPPM > 500 {
		return fmt.Errorf("%w: water content", domain.ErrInvalid)
	}
	if a.Density < 0.7 || a.Density > 0.85 {
		return fmt.Errorf("%w: density", domain.ErrInvalid)
	}
	if a.SulfurPPM < 0 || a.SulfurPPM > 100 {
		return fmt.Errorf("%w: sulphur", domain.ErrInvalid)
	}
	if strings.TrimSpace(a.Lab) == "" || a.TestedAt.IsZero() {
		return fmt.Errorf("%w: laboratory evidence", domain.ErrInvalid)
	}
	return nil
}
func QualityDecision(a Analysis) domain.QualityState {
	if ValidateAnalysis(a) != nil {
		return domain.QualityRejected
	}
	return domain.QualityApproved
}
func IsFresh(a Analysis, now time.Time) bool {
	return !a.TestedAt.After(now) && now.Sub(a.TestedAt) <= 30*24*time.Hour
}
func Explain(a Analysis) string {
	if ValidateAnalysis(a) != nil {
		return "rejected"
	}
	return fmt.Sprintf("methanol %.2f%%, water %.1f ppm, density %.3f", a.MethanolPercent, a.WaterPPM, a.Density)
}
