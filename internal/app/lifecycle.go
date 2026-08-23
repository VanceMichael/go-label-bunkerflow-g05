package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Lifecycle struct {
	mu       sync.Mutex
	started  bool
	stopping bool
	stopped  bool
	stopAt   time.Time
}

func (l *Lifecycle) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started && !l.stopped {
		return errors.New("lifecycle already started")
	}
	l.started = true
	l.stopping = false
	l.stopped = false
	return nil
}
func (l *Lifecycle) BeginStop() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.started || l.stopped {
		return false
	}
	if l.stopping {
		return false
	}
	l.stopping = true
	l.stopAt = time.Now()
	return true
}
func (l *Lifecycle) FinishStop() { l.mu.Lock(); l.stopped = true; l.mu.Unlock() }
func (l *Lifecycle) Ready() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.started && !l.stopping && !l.stopped
}
func (l *Lifecycle) Stop(ctx context.Context, wait func(context.Context) error) error {
	if !l.BeginStop() {
		return nil
	}
	err := wait(ctx)
	l.FinishStop()
	return err
}
