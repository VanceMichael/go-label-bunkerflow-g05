package bunkering

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type Manifest struct {
	OrderID     string
	VesselIMO   string
	Product     string
	TargetKG    float64
	Steps       []domain.TransferStep
	GeneratedAt time.Time
}

func (m Manifest) Validate() error {
	if m.OrderID == "" || !domain.ValidateIMO(m.VesselIMO) || m.Product != "green-methanol" || m.TargetKG <= 0 {
		return domain.ErrInvalid
	}
	if len(m.Steps) != 4 {
		return fmt.Errorf("%w: transfer steps", domain.ErrConflict)
	}
	copyOf := append([]domain.TransferStep(nil), m.Steps...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].Position < copyOf[j].Position })
	for index, step := range copyOf {
		if step.Position != index+1 || step.Name == "" {
			return domain.ErrConflict
		}
	}
	return nil
}
func (m Manifest) JSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
func (m Manifest) Copy() Manifest {
	copyOf := m
	copyOf.Steps = append([]domain.TransferStep(nil), m.Steps...)
	return copyOf
}
