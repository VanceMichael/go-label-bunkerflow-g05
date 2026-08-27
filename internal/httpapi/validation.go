package httpapi

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

func required(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is required", domain.ErrInvalid, name)
	}
	return nil
}
func positive(value float64, name string) error {
	if value <= 0 {
		return fmt.Errorf("%w: %s must be positive", domain.ErrInvalid, name)
	}
	return nil
}
func parseLimit(raw string, defaultValue, max int) int {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return defaultValue
	}
	if value > max {
		return max
	}
	return value
}
func parseTimeQuery(values url.Values, name string) (time.Time, error) {
	raw := values.Get(name)
	if raw == "" {
		return time.Time{}, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid %s", domain.ErrInvalid, name)
	}
	return value, nil
}
func parseBoolQuery(values url.Values, name string) (bool, error) {
	raw := values.Get(name)
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%w: invalid %s", domain.ErrInvalid, name)
	}
	return value, nil
}
func ensureDateRange(start, end time.Time) error {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return fmt.Errorf("%w: date range", domain.ErrInvalid)
	}
	return nil
}
