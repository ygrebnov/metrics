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
	cfg        *basicProviderConfig
	counters   sync.Map // map[string]*BasicCounter
	updowns    sync.Map // map[string]*BasicUpDownCounter
	histograms sync.Map // map[string]*BasicHistogram
	meta       sync.Map // map[string]InstrumentConfig — use sync.Map for concurrent access
	// per-key init mutexes: protect concurrent initialization for the same key
	inits sync.Map // map[string]*sync.Mutex
}

type basicProviderConfig struct {
	// when false, remove per-key mutex entries from `inits` after initialization to
	// allow GC of mutexes for many ephemeral instrument names. Default: false.
	doNotCleanupInits bool
}

// BasicProviderOption configures a BasicProvider constructed by NewBasicProvider.
type BasicProviderOption func(*basicProviderConfig)

// WithInitCleanupDisabled controls whether per-key init mutex entries are removed from
// the provider's internal `inits` map after initialization. When enabled the
// entries are deleted to allow GC of mutexes for ephemeral instrument names.
// Init cleanup is enabled by default; this option disables it.
func WithInitCleanupDisabled() BasicProviderOption {
	return func(cfg *basicProviderConfig) { cfg.doNotCleanupInits = true }
}

// NewBasicProvider constructs a new BasicProvider.
// Accepts optional functional options to customize behavior.
func NewBasicProvider(opts ...BasicProviderOption) *BasicProvider {
	cfg := &basicProviderConfig{}
	for _, o := range opts {
		if o != nil {
			o(cfg)
		}
	}
	return &BasicProvider{cfg: cfg}
}

// keyMu returns a per-key mutex for the given key, creating one if necessary.
// The returned mutex is owned by the provider and should be locked/unlocked by callers.
func (p *BasicProvider) keyMu(key string) *sync.Mutex {
	m, _ := p.inits.LoadOrStore(key, &sync.Mutex{})
	return m.(*sync.Mutex)
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

// get retrieves an existing instrument by type and name.
func (p *BasicProvider) get(t InstrumentType, name string) (interface{}, bool) {
	switch t {
	case InstrumentTypeCounter:
		if v, ok := p.counters.Load(name); ok {
			return v.(*BasicCounter), true
		}
	case InstrumentTypeUpDown:
		if v, ok := p.updowns.Load(name); ok {
			return v.(*BasicUpDownCounter), true
		}
	case InstrumentTypeHistogram:
		if v, ok := p.histograms.Load(name); ok {
			return v.(*BasicHistogram), true
		}
	}
	return nil, false
}

// create constructs and stores a new instance into the appropriate sync.Map.
func (p *BasicProvider) create(t InstrumentType, name string) interface{} {
	switch t {
	case InstrumentTypeCounter:
		c := &BasicCounter{}
		p.counters.Store(name, c)
		return c
	case InstrumentTypeUpDown:
		u := &BasicUpDownCounter{}
		p.updowns.Store(name, u)
		return u
	case InstrumentTypeHistogram:
		h := &BasicHistogram{min: math.Inf(1), max: math.Inf(-1)}
		p.histograms.Store(name, h)
		return h
	default:
		return nil
	}
}

// Counter returns a monotonic counter instrument for the given name (created once).
func (p *BasicProvider) Counter(name string, opts ...InstrumentOption) Counter {
	return p.getOrCreate(InstrumentTypeCounter, name, opts).(*BasicCounter)
}

// UpDownCounter returns an up/down counter instrument for the given name (created once).
func (p *BasicProvider) UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter {
	return p.getOrCreate(InstrumentTypeUpDown, name, opts).(*BasicUpDownCounter)
}

// Histogram returns a histogram instrument for the given name (created once).
func (p *BasicProvider) Histogram(name string, opts ...InstrumentOption) Histogram {
	return p.getOrCreate(InstrumentTypeHistogram, name, opts).(*BasicHistogram)
}

// getOrCreate is a helper that implements a fast read path, computes options before
// acquiring locks, and uses a per-key mutex to deduplicate concurrent initializations.
//   - key is a compound "typ:name" key used for both the per-key mutex and meta storage.
//   - opts are the instrument options (passed to applyOptions).
func (p *BasicProvider) getOrCreate(t InstrumentType, name string, opts []InstrumentOption) interface{} {
	// fast read path using sync.Map loads (safe without a global lock)
	if v, ok := p.get(t, name); ok {
		return v
	}

	// compute config off-lock to avoid holding per-key mutex during option application
	cfg := applyOptions(opts)

	key := string(t) + ":" + name
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	// re-check after acquiring per-key mutex
	if v, ok := p.get(t, name); ok {
		return v
	}
	// store metadata computed earlier using the compound key typ:name
	p.meta.Store(key, cfg)
	inst := p.create(t, name)
	// optional cleanup: remove the per-key mutex from the inits map to allow GC of mutexes
	// It's safe to delete while holding the mutex; existing goroutines that already
	// hold the pointer will continue to use it, and new callers will get a new mutex.
	if !p.cfg.doNotCleanupInits {
		p.inits.Delete(key)
	}
	return inst
}

// copyConfig makes a defensive copy of InstrumentConfig (copies Attributes map).
func copyConfig(in InstrumentConfig) InstrumentConfig {
	out := InstrumentConfig{Description: in.Description, Unit: in.Unit}
	if len(in.Attributes) > 0 {
		out.Attributes = make(map[string]string, len(in.Attributes))
		for k, v := range in.Attributes {
			out.Attributes[k] = v
		}
	}
	return out
}

// CounterWithMeta implements Inspector.CounterWithMeta for BasicProvider.
// It acquires the per-key init mutex, re-checks, then reads both the instance
// and metadata before unlocking in order to provide a consistent snapshot.
func (p *BasicProvider) CounterWithMeta(name string) (Counter, InstrumentConfig, bool) {
	key := InstrumentTypeCounter.String() + ":" + name
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.counters.Load(name)
	if !ok {
		// not created
		return nil, InstrumentConfig{}, false
	}
	inst := v.(*BasicCounter)

	var cfg InstrumentConfig
	if m, ok := p.meta.Load(key); ok {
		if c, ok2 := m.(InstrumentConfig); ok2 {
			cfg = copyConfig(c)
		}
	}
	return inst, cfg, true
}

// UpDownCounterWithMeta implements Inspector.UpDownCounterWithMeta for BasicProvider.
func (p *BasicProvider) UpDownCounterWithMeta(name string) (UpDownCounter, InstrumentConfig, bool) {
	key := InstrumentTypeUpDown.String() + ":" + name
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.updowns.Load(name)
	if !ok {
		return nil, InstrumentConfig{}, false
	}
	inst := v.(*BasicUpDownCounter)

	var cfg InstrumentConfig
	if m, ok := p.meta.Load(key); ok {
		if c, ok2 := m.(InstrumentConfig); ok2 {
			cfg = copyConfig(c)
		}
	}
	return inst, cfg, true
}

// HistogramWithMeta implements Inspector.HistogramWithMeta for BasicProvider.
func (p *BasicProvider) HistogramWithMeta(name string) (Histogram, InstrumentConfig, bool) {
	key := InstrumentTypeHistogram.String() + ":" + name
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.histograms.Load(name)
	if !ok {
		return nil, InstrumentConfig{}, false
	}
	inst := v.(*BasicHistogram)

	var cfg InstrumentConfig
	if m, ok := p.meta.Load(key); ok {
		if c, ok2 := m.(InstrumentConfig); ok2 {
			cfg = copyConfig(c)
		}
	}
	return inst, cfg, true
}

// ListMetadata returns a best-effort snapshot of metadata entries. It does not
// acquire per-key init mutexes for each entry; callers should treat the result
// as a point-in-time snapshot that may race with concurrent creations.
func (p *BasicProvider) ListMetadata() []InstrumentEntry {
	out := make([]InstrumentEntry, 0)
	p.meta.Range(func(k, v interface{}) bool {
		ks, ok := k.(string)
		if !ok {
			return true
		}
		// expect "type:name"; find ':' without importing strings
		idx := -1
		for i, r := range ks {
			if r == ':' {
				idx = i
				break
			}
		}
		if idx < 0 {
			return true
		}
		typ := InstrumentType(ks[:idx])
		name := ks[idx+1:]
		cfg, _ := v.(InstrumentConfig)
		out = append(out, InstrumentEntry{Type: typ, Name: name, Config: copyConfig(cfg)})
		return true
	})
	return out
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
