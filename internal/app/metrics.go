package app

import (
	"sync"
	"sync/atomic"
)

type Metrics struct {
	requests  atomic.Uint64
	failures  atomic.Uint64
	active    atomic.Int64
	mu        sync.RWMutex
	endpoints map[string]uint64
}

func NewMetrics() *Metrics { return &Metrics{endpoints: make(map[string]uint64)} }
func (m *Metrics) Begin(path string) {
	m.requests.Add(1)
	m.active.Add(1)
	m.mu.Lock()
	m.endpoints[path]++
	m.mu.Unlock()
}
func (m *Metrics) End(success bool) {
	m.active.Add(-1)
	if !success {
		m.failures.Add(1)
	}
}
func (m *Metrics) Snapshot() (uint64, uint64, int64, map[string]uint64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copyOf := make(map[string]uint64, len(m.endpoints))
	for key, value := range m.endpoints {
		copyOf[key] = value
	}
	return m.requests.Load(), m.failures.Load(), m.active.Load(), copyOf
}
