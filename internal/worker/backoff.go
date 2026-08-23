package worker

import (
	"math"
	"time"

	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
)

type RetryPolicy struct {
	MaxAttempts int
	Initial     time.Duration
	Maximum     time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 5, Initial: time.Second, Maximum: 5 * time.Minute}
}
func (p RetryPolicy) Delay(attempt int) time.Duration {
	if attempt < 1 {
		return p.Initial
	}
	factor := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(p.Initial) * factor)
	if delay > p.Maximum {
		return p.Maximum
	}
	return delay
}
func (p RetryPolicy) Terminal(attempt int) bool                 { return attempt >= p.MaxAttempts }
func (p RetryPolicy) Next(now time.Time, attempt int) time.Time { return now.Add(p.Delay(attempt)) }
func (p RetryPolicy) Classify(err error) string {
	if err == nil {
		return "success"
	}
	if err == domain.ErrCancelled {
		return "cancelled"
	}
	if err == domain.ErrUnavailable {
		return "retryable"
	}
	return "failed"
}
