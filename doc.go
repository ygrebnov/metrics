/*
Package metrics provides a small, concurrency-safe in-memory metrics library for Go.

# Overview

The library is organized around two main interfaces:

1. Provider: creation and lifecycle management of instruments (Counter, UpDownCounter, Histogram).
Providers must be safe for concurrent use by multiple goroutines, create instruments lazily,
and deduplicate by a stable key (typically (type,name)).

	type Provider interface {
	  Counter(name string, opts ...Option) Counter
	  UpDownCounter(name string, opts ...Option) UpDownCounter
	  Histogram(name string, opts ...Option) Histogram
	}

2. Inspector: read-only access to instruments and their metadata.
Inspector methods (e.g., CounterWithMeta, UpDownCounterWithMeta, HistogramWithMeta, ListMetadata)
return instruments together with a defensive copy of their InstrumentConfig and aim to provide a
consistent per-key snapshot.

	type Inspector interface {
	  CounterWithMeta(name string) (Counter, InstrumentConfig, bool)
	  UpDownCounterWithMeta(name string) (UpDownCounter, InstrumentConfig, bool)
	  HistogramWithMeta(name string) (Histogram, InstrumentConfig, bool)
	  ListMetadata() []MetadataEntry
	}

# Reference implementation

BasicProvider implements both Provider and Inspector using in-memory data structures.
It stores instruments in per-type sync.Maps keyed by name and uses a separate sync.Map of
per-key mutexes to serialize first-time initialization. Metadata is stored alongside and
Inspector methods acquire the same per-key mutex to return a consistent (instrument, meta)
snapshot. After initialization, the per-key mutex entry may be removed to reduce memory
(this cleanup can be disabled via options).

How it works (high level)

 1. Fast path: look up the instrument in the appropriate sync.Map and return it if present.
 2. Slow path: build InstrumentConfig off-lock from options; acquire the per-key mutex; re-check;
    store metadata; create and store the instrument; optionally delete the init mutex entry.
 3. Inspector methods take the same per-key mutex, read instrument and metadata, and return a
    defensive copy of the config so callers cannot mutate internal state.
 4. The provider performs internal invariant checks and reports unexpected internal states
    (for example: "instrument exists but meta missing"). In debug and race builds (controlled via
    build tags) invariant violations cause a panic to fail fast; in non-debug builds they are
    logged and the provider attempts to continue.

Examples

	p := metrics.NewBasicProvider()
	c := p.Counter("requests_total", metrics.WithDescription("HTTP requests"), metrics.WithUnit("count"))
	c.Add(1)

	if inst, cfg, ok := p.CounterWithMeta("requests_total"); ok {
	    // cfg is a defensive copy of the stored InstrumentConfig
	    _ = cfg
	    _ = inst
	}

	// List all registered instruments and their metadata snapshot
	for _, entry := range p.ListMetadata() {
	    _ = entry // entry.Type, entry.Name, entry.Config
	}

# Build and test

- Run unit tests:

	go test ./...

- Run with the race detector (enables stricter invariant behavior):

	go test -race ./...

- Enable debug build tag (debug invariants enabled):

	go test -tags=debug ./...

# Notes

- InstrumentConfig returned by Inspector methods are defensive copies; mutating them will not
change the provider's stored metadata.

- Per-key init mutex entries are removed by default after initialization to allow GC of many
ephemeral instrument names. Disable this behavior with metrics.WithInitCleanupDisabled().
*/
package metrics
