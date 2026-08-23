package worker

import (
	"errors"
	"github.com/VanceMichael/go-base-bunkerflow-g05/internal/domain"
	"testing"
	"time"
)

func TestRetryPolicyCapsAndClassifies(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.Delay(1) != time.Second {
		t.Fatalf("delay=%v", p.Delay(1))
	}
	if p.Delay(20) > p.Maximum {
		t.Fatal("delay exceeded cap")
	}
	if !p.Terminal(5) || p.Terminal(4) {
		t.Fatal("terminal attempts wrong")
	}
	if p.Classify(domain.ErrUnavailable) != "retryable" {
		t.Fatal("unavailable not retryable")
	}
	if p.Classify(errors.New("bad")) != "failed" {
		t.Fatal("generic error classified retryable")
	}
}
