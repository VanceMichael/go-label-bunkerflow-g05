package weighing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Reading struct {
	ID         string
	OrderID    string
	Sequence   int
	GrossKG    float64
	TareKG     float64
	MeasuredAt time.Time
	DeviceID   string
}

func (r Reading) NetKG() float64 { return r.GrossKG - r.TareKG }

func (r Reading) Validate() error {
	if r.ID == "" || r.OrderID == "" || r.Sequence < 1 || r.DeviceID == "" {
		return domain.ErrInvalid
	}
	if r.GrossKG <= 0 || r.TareKG < 0 || r.GrossKG <= r.TareKG {
		return fmt.Errorf("%w: weighing values", domain.ErrInvalid)
	}
	if r.MeasuredAt.IsZero() {
		return fmt.Errorf("%w: measurement time", domain.ErrInvalid)
	}
	return nil
}

type Series struct{ Readings []Reading }

func (s Series) Ordered() []Reading {
	result := append([]Reading(nil), s.Readings...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Sequence == result[j].Sequence {
			return result[i].MeasuredAt.Before(result[j].MeasuredAt)
		}
		return result[i].Sequence < result[j].Sequence
	})
	return result
}

func (s Series) Validate() error {
	ordered := s.Ordered()
	seen := make(map[int]bool, len(ordered))
	for index, reading := range ordered {
		if err := reading.Validate(); err != nil {
			return err
		}
		if seen[reading.Sequence] {
			return fmt.Errorf("%w: duplicate sequence", domain.ErrConflict)
		}
		seen[reading.Sequence] = true
		if reading.Sequence != index+1 {
			return fmt.Errorf("%w: sequence gap", domain.ErrConflict)
		}
		if index > 0 && reading.MeasuredAt.Before(ordered[index-1].MeasuredAt) {
			return fmt.Errorf("%w: time moved backwards", domain.ErrConflict)
		}
	}
	return nil
}

func (s Series) DeliveredKG() (float64, error) {
	if err := s.Validate(); err != nil {
		return 0, err
	}
	if len(s.Readings) < 2 {
		return 0, fmt.Errorf("%w: at least two readings required", domain.ErrInvalid)
	}
	ordered := s.Ordered()
	delivered := ordered[len(ordered)-1].NetKG() - ordered[0].NetKG()
	if delivered < 0 {
		return 0, fmt.Errorf("%w: negative delivered quantity", domain.ErrConflict)
	}
	return math.Round(delivered*1000) / 1000, nil
}

func (s Series) WithinTolerance(target, percent float64) (bool, error) {
	if target <= 0 || percent < 0 {
		return false, domain.ErrInvalid
	}
	delivered, err := s.DeliveredKG()
	if err != nil {
		return false, err
	}
	delta := math.Abs(delivered - target)
	return delta <= target*percent/100, nil
}

type Client interface {
	Read(context.Context, string) (Reading, error)
}
type Service struct{ Client Client }

func (s Service) Capture(ctx context.Context, orderID string) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, fmt.Errorf("%w: %v", domain.ErrCancelled, err)
	}
	reading, err := s.Client.Read(ctx, orderID)
	if err != nil {
		return Reading{}, fmt.Errorf("read weighing device: %w", err)
	}
	if err := reading.Validate(); err != nil {
		return Reading{}, err
	}
	return reading, nil
}
