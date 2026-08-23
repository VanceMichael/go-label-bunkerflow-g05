package domain

import "fmt"

type OperationState string

const (
	StatePlanned      OperationState = "planned"
	StateApproved     OperationState = "approved"
	StateAlongside    OperationState = "alongside"
	StateTransferring OperationState = "transferring"
	StateSampled      OperationState = "sampled"
	StateCompleted    OperationState = "completed"
	StateCancelled    OperationState = "cancelled"
)

func (s OperationState) Valid() bool {
	switch s {
	case StatePlanned, StateApproved, StateAlongside, StateTransferring, StateSampled, StateCompleted, StateCancelled:
		return true
	default:
		return false
	}
}

func (s OperationState) CanMove(to OperationState) bool {
	switch s {
	case StatePlanned:
		return to == StateApproved || to == StateCancelled
	case StateApproved:
		return to == StateAlongside || to == StateCancelled
	case StateAlongside:
		return to == StateTransferring || to == StateCancelled
	case StateTransferring:
		return to == StateSampled || to == StateCancelled
	case StateSampled:
		return to == StateCompleted || to == StateCancelled
	default:
		return false
	}
}

func Transition(s, to OperationState) error {
	if !s.Valid() || !to.Valid() || !s.CanMove(to) {
		return fmt.Errorf("%w: operation transition %s -> %s", ErrConflict, s, to)
	}
	return nil
}

type QualityState string

const (
	QualityReceived QualityState = "received"
	QualityApproved QualityState = "approved"
	QualityRejected QualityState = "rejected"
)

func (s QualityState) CanMove(to QualityState) bool {
	return (s == QualityReceived && (to == QualityApproved || to == QualityRejected)) || s == to
}
