package metrics

import (
	"math"
	"sort"
	"sync"
)

const DefaultCapacity = 1000

// Metrics maintains a thread-safe ring buffer of the latest latency measurements
// (in milliseconds) and a counter for total queries.
type Metrics struct {
	mu           sync.RWMutex
	capacity     int
	buffer       []float64
	cursor       int
	queriesTotal int64
}

// New creates a new Metrics instance with the default capacity of 1000.
func New() *Metrics {
	return NewWithCapacity(DefaultCapacity)
}

// NewWithCapacity creates a Metrics instance with custom ring buffer capacity.
func NewWithCapacity(capacity int) *Metrics {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Metrics{
		capacity: capacity,
		buffer:   make([]float64, 0, capacity),
	}
}

// Record records a latency measurement in milliseconds.
func (m *Metrics) Record(durationMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queriesTotal++

	if len(m.buffer) < m.capacity {
		m.buffer = append(m.buffer, durationMs)
	} else {
		m.buffer[m.cursor] = durationMs
		m.cursor = (m.cursor + 1) % m.capacity
	}
}

// Stats returns p50, p95, p99 latency percentiles and total queries count.
// If no measurements have been recorded, percentiles are returned as 0.
func (m *Metrics) Stats() (p50, p95, p99 float64, total int64) {
	m.mu.RLock()
	total = m.queriesTotal
	n := len(m.buffer)
	if n == 0 {
		m.mu.RUnlock()
		return 0, 0, 0, total
	}

	samples := make([]float64, n)
	copy(samples, m.buffer)
	m.mu.RUnlock()

	sort.Float64s(samples)

	p50 = percentile(samples, 50)
	p95 = percentile(samples, 95)
	p99 = percentile(samples, 99)
	return p50, p95, p99, total
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100.0*float64(n))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
