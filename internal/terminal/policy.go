package terminal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type OperatingHours struct {
	From     time.Duration
	Until    time.Duration
	Location *time.Location
}

func ParseHours(from, until string, location *time.Location) (OperatingHours, error) {
	parse := func(value string) (time.Duration, error) {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return 0, domain.ErrInvalid
		}
		hour, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		minute, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return 0, domain.ErrInvalid
		}
		return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute, nil
	}
	a, err := parse(from)
	if err != nil {
		return OperatingHours{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	b, err := parse(until)
	if err != nil {
		return OperatingHours{}, fmt.Errorf("%w: %v", domain.ErrInvalid, err)
	}
	if location == nil {
		location = time.UTC
	}
	return OperatingHours{From: a, Until: b, Location: location}, nil
}
func (h OperatingHours) OpenAt(value time.Time) bool {
	local := value.In(h.Location)
	current := time.Duration(local.Hour())*time.Hour + time.Duration(local.Minute())*time.Minute
	if h.From <= h.Until {
		return current >= h.From && current <= h.Until
	}
	return current >= h.From || current <= h.Until
}
func (h OperatingHours) NextOpening(value time.Time) time.Time {
	if h.OpenAt(value) {
		return value
	}
	local := value.In(h.Location)
	candidate := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, h.Location).Add(h.From)
	if !candidate.After(local) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}
