package app

import (
	"context"
	"sync"
	"time"
)

type Supervisor struct {
	mu      sync.Mutex
	workers map[string]context.CancelFunc
	stopped bool
}

func NewSupervisor() *Supervisor { return &Supervisor{workers: make(map[string]context.CancelFunc)} }
func (s *Supervisor) Add(parent context.Context, name string, run func(context.Context)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return context.Canceled
	}
	if _, ok := s.workers[name]; ok {
		return context.DeadlineExceeded
	}
	ctx, cancel := context.WithCancel(parent)
	s.workers[name] = cancel
	go func() { run(ctx); s.mu.Lock(); delete(s.workers, name); s.mu.Unlock() }()
	return nil
}
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	s.stopped = true
	for _, cancel := range s.workers {
		cancel()
	}
	s.mu.Unlock()
	deadline := time.NewTimer(10 * time.Millisecond)
	defer deadline.Stop()
	select {
	case <-deadline.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Supervisor) Active() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.workers) }
