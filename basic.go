package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

// BasicProvider is a simple in-memory implementation of Provider.
// It is concurrency-safe and suitable for tests, examples, and lightweight apps.
// Instruments are created on demand by name and reused for the same name.
// Instrument options are currently advisory and stored for potential introspection.
type BasicProvider struct {
	counters   sync.Map // map[string]*BasicCounter
	updowns    sync.Map // map[string]*BasicUpDownCounter
	histograms sync.Map // map[string]*BasicHistogram
	meta       sync.Map // map[string]InstrumentConfig — use sync.Map for concurrent access
	// per-key init mutexes: protect concurrent initialization for the same key
	inits sync.Map // map[string]*sync.Mutex
}

// NewBasicProvider constructs a new BasicProvider.
func NewBasicProvider() *BasicProvider {
	return &BasicProvider{
		// counters/updowns/histograms/meta/inits are zero-value ready for use
	}
}

const (
	InstrumentTypeCounter   = "counter"
	InstrumentTypeUpDown    = "updown"
	InstrumentTypeHistogram = "histogram"
)

// keyMu returns a per-key mutex for the given key, creating one if necessary.
// The returned mutex is owned by the provider and should be locked/unlocked by callers.
func (p *BasicProvider) keyMu(key string) *sync.Mutex {
	m, _ := p.inits.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// metaLoad returns the stored InstrumentConfig for given instrument kind and name, if any.
// key is composed as typ+":"+name.
func (p *BasicProvider) metaLoad(typ, name string) (InstrumentConfig, bool) {
	key := typ + ":" + name
	v, ok := p.meta.Load(key)
	if !ok {
		return InstrumentConfig{}, false
	}
	cfg, ok := v.(InstrumentConfig)
	return cfg, ok
}

// applyOptions builds InstrumentConfig from options.
func applyOptions(opts []InstrumentOption) InstrumentConfig {
	var cfg InstrumentConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	return cfg
}

// Counter returns a monotonic counter instrument for the given name (created once).
func (p *BasicProvider) Counter(name string, opts ...InstrumentOption) Counter {
	v := p.getOrCreate(
		InstrumentTypeCounter+":"+name,
		opts,
		func() (interface{}, bool) {
			if vv, ok := p.counters.Load(name); ok {
				return vv.(*BasicCounter), true
			}
			return nil, false
		},
		func() interface{} { c := &BasicCounter{}; p.counters.Store(name, c); return c },
	)
	return v.(*BasicCounter)
}

// UpDownCounter returns an up/down counter instrument for the given name (created once).
func (p *BasicProvider) UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter {
	v := p.getOrCreate(
		InstrumentTypeUpDown+":"+name,
		opts,
		func() (interface{}, bool) {
			if vv, ok := p.updowns.Load(name); ok {
				return vv.(*BasicUpDownCounter), true
			}
			return nil, false
		},
		func() interface{} { u := &BasicUpDownCounter{}; p.updowns.Store(name, u); return u },
	)
	return v.(*BasicUpDownCounter)
}

// Histogram returns a histogram instrument for the given name (created once).
func (p *BasicProvider) Histogram(name string, opts ...InstrumentOption) Histogram {
	v := p.getOrCreate(
		InstrumentTypeHistogram+":"+name,
		opts,
		func() (interface{}, bool) {
			if vv, ok := p.histograms.Load(name); ok {
				return vv.(*BasicHistogram), true
			}
			return nil, false
		},
		func() interface{} {
			h := &BasicHistogram{min: math.Inf(1), max: math.Inf(-1)}
			p.histograms.Store(name, h)
			return h
		},
	)
	return v.(*BasicHistogram)
}

// getOrCreate is a helper that implements a fast read path, computes options before
// acquiring locks, and uses a per-key mutex to deduplicate concurrent initializations.
//   - typ is a short prefix used as part of the key to avoid collisions between different instrument kinds.
//   - name is the instrument name.
//   - opts are the instrument options (passed to applyOptions).
//   - check is called under the appropriate lock to check for existing instance.
//   - create is called under write lock to construct and store a new instance; it
//     must assign the created instance into the appropriate provider map and return it.
func (p *BasicProvider) getOrCreate(
	key string,
	opts []InstrumentOption,
	check func() (interface{}, bool),
	create func() interface{},
) interface{} {
	// fast read path using sync.Map loads (safe without a global lock)
	if v, ok := check(); ok {
		return v
	}

	// compute config off-lock to avoid holding per-key mutex during option application
	cfg := applyOptions(opts)

	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	// re-check after acquiring per-key mutex
	if v, ok := check(); ok {
		return v
	}
	// store metadata computed earlier using the compound key typ:name
	p.meta.Store(key, cfg)
	inst := create()
	return inst
}

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
