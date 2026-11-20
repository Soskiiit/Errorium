package metrics_test

import (
	"sync"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/internal/metrics"
)

func TestMetrics_EmptyStats(t *testing.T) {
	m := metrics.New()
	p50, p95, p99, total := m.Stats()

	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if p50 != 0 || p95 != 0 || p99 != 0 {
		t.Errorf("expected 0 for all percentiles, got p50=%f, p95=%f, p99=%f", p50, p95, p99)
	}
}

func TestMetrics_KnownValues(t *testing.T) {
	m := metrics.New()

	// Record 100 values: 1.0, 2.0, ..., 100.0
	for i := 1; i <= 100; i++ {
		m.Record(float64(i))
	}

	p50, p95, p99, total := m.Stats()

	if total != 100 {
		t.Errorf("expected total 100, got %d", total)
	}
	if p50 != 50 {
		t.Errorf("expected p50=50, got %f", p50)
	}
	if p95 != 95 {
		t.Errorf("expected p95=95, got %f", p95)
	}
	if p99 != 99 {
		t.Errorf("expected p99=99, got %f", p99)
	}
}

func TestMetrics_RingBufferWrap(t *testing.T) {
	// Ring buffer of capacity 5
	m := metrics.NewWithCapacity(5)

	// Record 1, 2, 3, 4, 5
	for i := 1; i <= 5; i++ {
		m.Record(float64(i))
	}

	// Overwrite with 10, 20
	m.Record(10)
	m.Record(20)

	// Total should be 7
	p50, p95, p99, total := m.Stats()
	if total != 7 {
		t.Errorf("expected total 7, got %d", total)
	}

	// Buffer should now contain: 10, 20, 3, 4, 5 -> sorted: 3, 4, 5, 10, 20
	// 5 elements:
	// p50: ceil(0.50 * 5) - 1 = 3 - 1 = 2 -> 5
	// p95: ceil(0.95 * 5) - 1 = 5 - 1 = 4 -> 20
	// p99: ceil(0.99 * 5) - 1 = 5 - 1 = 4 -> 20
	if p50 != 5 {
		t.Errorf("expected p50=5, got %f", p50)
	}
	if p95 != 20 {
		t.Errorf("expected p95=20, got %f", p95)
	}
	if p99 != 20 {
		t.Errorf("expected p99=20, got %f", p99)
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	m := metrics.NewWithCapacity(100)

	var wg sync.WaitGroup
	// 10 concurrent writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				m.Record(float64(workerID*50 + j))
			}
		}(i)
	}

	// 5 concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _, _, _ = m.Stats()
			}
		}()
	}

	wg.Wait()

	_, _, _, total := m.Stats()
	if total != 500 {
		t.Errorf("expected total 500, got %d", total)
	}
}
