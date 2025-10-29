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
	cfg    *basicProviderConfig
	logger logger

	counters   sync.Map // map[string]*BasicCounter
	updowns    sync.Map // map[string]*BasicUpDownCounter
	histograms sync.Map // map[string]*BasicHistogram
	meta       sync.Map // map[InstrumentKey]InstrumentConfig
	// per-key init mutexes: protect concurrent initialization for the same key
	inits sync.Map // map[InstrumentKey]*sync.Mutex
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
	l := cfg.logger
	if l == nil {
		l = newNoopLogger()
	}
	return &BasicProvider{cfg: cfg, logger: l}
}

// keyMu returns a per-key mutex for the given key, creating one if necessary.
// The returned mutex is owned by the provider and should be locked/unlocked by callers.
func (p *BasicProvider) keyMu(key InstrumentKey) *sync.Mutex {
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

// get retrieves an existing instrument by key.
func (p *BasicProvider) get(key InstrumentKey) (interface{}, bool) {
	switch key.Type {
	case InstrumentTypeCounter:
		if v, ok := p.counters.Load(key.Name); ok {
			return v.(*BasicCounter), true
		}
	case InstrumentTypeUpDown:
		if v, ok := p.updowns.Load(key.Name); ok {
			return v.(*BasicUpDownCounter), true
		}
	case InstrumentTypeHistogram:
		if v, ok := p.histograms.Load(key.Name); ok {
			return v.(*BasicHistogram), true
		}
	}
	return nil, false
}

// create constructs and stores a new instance into the appropriate sync.Map.
func (p *BasicProvider) create(key InstrumentKey) interface{} {
	switch key.Type {
	case InstrumentTypeCounter:
		c := &BasicCounter{}
		p.counters.Store(key.Name, c)
		return c
	case InstrumentTypeUpDown:
		u := &BasicUpDownCounter{}
		p.updowns.Store(key.Name, u)
		return u
	case InstrumentTypeHistogram:
		h := &BasicHistogram{min: math.Inf(1), max: math.Inf(-1)}
		p.histograms.Store(key.Name, h)
		return h
	default:
		return nil
	}
}

// Counter returns a monotonic counter instrument for the given name (created once).
func (p *BasicProvider) Counter(name string, opts ...InstrumentOption) Counter {
	key := NewInstrumentKey(InstrumentTypeCounter, name)
	return p.getOrCreate(key, opts).(*BasicCounter)
}

// UpDownCounter returns an up/down counter instrument for the given name (created once).
func (p *BasicProvider) UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter {
	key := NewInstrumentKey(InstrumentTypeUpDown, name)
	return p.getOrCreate(key, opts).(*BasicUpDownCounter)
}

// Histogram returns a histogram instrument for the given name (created once).
func (p *BasicProvider) Histogram(name string, opts ...InstrumentOption) Histogram {
	key := NewInstrumentKey(InstrumentTypeHistogram, name)
	return p.getOrCreate(key, opts).(*BasicHistogram)
}

// getOrCreate is a helper that implements a fast read path, computes options before
// acquiring locks, and uses a per-key mutex to deduplicate concurrent initializations.
//   - key is a compound "typ:name" key used for both the per-key mutex and meta storage.
//   - opts are the instrument options (passed to applyOptions).
func (p *BasicProvider) getOrCreate(key InstrumentKey, opts []InstrumentOption) interface{} {
	// fast read path using sync.Map loads (safe without a global lock)
	if v, ok := p.get(key); ok {
		return v
	}

	// compute config off-lock to avoid holding per-key mutex during option application
	cfg := applyOptions(opts)

	// acquire per-key mutex to deduplicate concurrent initializations
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	// re-check after acquiring per-key mutex
	if v, ok := p.get(key); ok {
		return v
	}
	// store metadata computed earlier using the compound key typ:name
	p.meta.Store(key, cfg)
	inst := p.create(key)
	// optional cleanup: remove the per-key mutex from the inits map to allow GC of mutexes
	// It's safe to delete while holding the mutex; existing goroutines that already
	// hold the pointer will continue to use it, and new callers will get a new mutex.
	if !p.cfg.doNotCleanupInits {
		p.inits.Delete(key)
	}
	return inst
}

// reportInvariantViolation reports unexpected internal states such as
// "instrument exists but meta missing". In release builds it logs up to 10 times per key;
// in debug builds (or under race detector) it panics to catch bugs early.
func (p *BasicProvider) reportInvariantViolation(kind string, key InstrumentKey) {
	// Avoid spamming logs for the same key
	const maxReports = 10
	var count int32
	if v, ok := p.meta.Load(InstrumentKey{Type: InstrumentTypeCounter, Name: "__invariant_counter__"}); ok {
		if c, ok2 := v.(*atomic.Int32); ok2 {
			count = c.Add(1)
		}
	} else {
		c := &atomic.Int32{}
		p.meta.Store(InstrumentKey{Type: InstrumentTypeCounter, Name: "__invariant_counter__"}, c)
		count = c.Add(1)
	}
	if count > maxReports {
		return
	}

	msg := "[metrics] invariant violation: " + kind + " for " + key.String()

	// In debug builds, fail fast.
	if isDebugBuild() {
		panic(msg)
	}

	// In release builds, just log a warning.
	p.logger.Warnf(msg)
}

// isDebugBuild reports whether we're in a "debug" or "race" build.
// This uses Go's built-in race detector flag or a debug build tag.
func isDebugBuild() bool {
	return raceBuild || debugBuild
}
