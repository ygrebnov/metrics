package metrics

import (
	"sync"
	"sync/atomic"
)

// BasicCounter is a thread-safe monotonic counter.
type BasicCounter struct {
	val atomic.Int64
}

// Add increments the counter by n (n may be negative but it's not recommended for monotonic counters).
func (c *BasicCounter) Add(n int64) { c.val.Add(n) }

// Snapshot returns the current value.
func (c *BasicCounter) Snapshot() int64 { return c.val.Load() }

// BasicUpDownCounter is a thread-safe up/down counter.
type BasicUpDownCounter struct {
	val atomic.Int64
}

// Add adds n (positive or negative) to the current value.
func (u *BasicUpDownCounter) Add(n int64) { u.val.Add(n) }

// Snapshot returns the current value.
func (u *BasicUpDownCounter) Snapshot() int64 { return u.val.Load() }

// BasicHistogram is a thread-safe histogram that tracks count, sum, min, and max.
// It does not maintain buckets; it's intended as a lightweight, general-purpose aggregator.
type BasicHistogram struct {
	mu    sync.Mutex
	count int64
	sum   float64
	min   float64
	max   float64
}

// Record adds a measurement to the histogram.
func (h *BasicHistogram) Record(v float64) {
	h.mu.Lock()
	if h.count == 0 {
		// initialize min/max on first record
		h.min, h.max = v, v
	} else {
		if v < h.min {
			h.min = v
		}
		if v > h.max {
			h.max = v
		}
	}
	h.count++
	h.sum += v
	h.mu.Unlock()
}

// HistSnapshot is an immutable snapshot of a BasicHistogram.
type HistSnapshot struct {
	Count int64
	Sum   float64
	Min   float64
	Max   float64
	Mean  float64
}

// Snapshot returns a copy of the histogram state at the time of call.
func (h *BasicHistogram) Snapshot() HistSnapshot {
	h.mu.Lock()
	count := h.count
	sum := h.sum
	minV := h.min
	maxV := h.max
	h.mu.Unlock()
	mean := 0.0
	if count > 0 {
		mean = sum / float64(count)
	}
	return HistSnapshot{Count: count, Sum: sum, Min: minV, Max: maxV, Mean: mean}
}
