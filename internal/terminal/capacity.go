package terminal

import (
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Capacity struct {
	Slots       int
	Active      int
	LastUpdated time.Time
}

func (c Capacity) Available() int {
	if c.Slots <= c.Active {
		return 0
	}
	return c.Slots - c.Active
}
func (c Capacity) CanAccept(count int) bool { return count > 0 && c.Available() >= count }
func (c Capacity) Reserve(count int) (Capacity, error) {
	if !c.CanAccept(count) {
		return c, domain.ErrConflict
	}
	c.Active += count
	c.LastUpdated = time.Now()
	return c, nil
}
func (c Capacity) Release(count int) (Capacity, error) {
	if count <= 0 || count > c.Active {
		return c, domain.ErrConflict
	}
	c.Active -= count
	c.LastUpdated = time.Now()
	return c, nil
}

type CapacityPoint struct {
	At    time.Time
	Value int
}

func Trend(points []CapacityPoint) []CapacityPoint {
	copyOf := append([]CapacityPoint(nil), points...)
	sort.SliceStable(copyOf, func(i, j int) bool { return copyOf[i].At.Before(copyOf[j].At) })
	return copyOf
}
