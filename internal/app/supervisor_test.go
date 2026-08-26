package app

import (
	"context"
	"testing"
	"time"
)

func TestSupervisorCancelsWorkers(t *testing.T) {
	supervisor := NewSupervisor()
	started := make(chan struct{})
	finished := make(chan struct{})
	if err := supervisor.Add(context.Background(), "outbox", func(ctx context.Context) { close(started); <-ctx.Done(); close(finished) }); err != nil {
		t.Fatal(err)
	}
	<-started
	if supervisor.Active() != 1 {
		t.Fatalf("active=%d", supervisor.Active())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("worker not cancelled")
	}
}
