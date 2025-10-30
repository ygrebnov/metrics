[![GoDoc](https://pkg.go.dev/badge/github.com/ygrebnov/metrics)](https://pkg.go.dev/github.com/ygrebnov/metrics)
[![Build Status](https://github.com/ygrebnov/metrics/actions/workflows/build.yml/badge.svg)](https://github.com/ygrebnov/metrics/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/ygrebnov/metrics/graph/badge.svg?token=27GL7ZQJ4U)](https://codecov.io/gh/ygrebnov/metrics)
[![Go Report Card](https://goreportcard.com/badge/github.com/ygrebnov/metrics)](https://goreportcard.com/report/github.com/ygrebnov/metrics)

**Metrics** is a small, concurrency‑safe in‑memory metrics provider for Go. It is designed for tests, examples and lightweight applications where you need simple counters, up/down counters and histograms without pulling in heavy dependencies. Instruments are created lazily and deduplicated by type and name, and internal invariant checks help surface bugs early. Written by [ygrebnov](https://github.com/ygrebnov).

The library is structured around two main interfaces: [Provider](#provider) (for creating and managing instruments) and [Inspector](#inspector) (for inspecting and listing instrument state and metadata). It provides a simple, in-memory, concurrency-safe implementation of both interfaces: [BasicProvider](#basicprovider). While this README focuses primarily on [BasicProvider](#basicprovider), you may also implement custom providers or inspectors by implementing the respective interfaces.

## Installation

To add `metrics` to your project, run:

```bash
go get example.com/ygrebnov/metrics
```

## Provider

The **Provider** interface defines the contract for creating and managing metrics instruments such as counters, up/down counters, and histograms. It is responsible for instrument lifecycle management, ensuring that instruments are uniquely identified by their name and type, and providing concurrency-safe access to them.

Implementations are expected to include methods:

- `Counter(name string, opts ...InstrumentOption) Counter`
- `UpDownCounter(name string, opts ...InstrumentOption) UpDownCounter`
- `Histogram(name string, opts ...InstrumentOption) Histogram`

Implementations of `Provider` must guarantee safe concurrent access from multiple goroutines. Instruments should be created lazily on demand and deduplicated by their key to avoid redundant instances.

The [BasicProvider](#basicprovider) implements this interface by maintaining separate internal maps for each instrument type keyed by name. It uses per-key mutexes to synchronize initialization, ensuring that each instrument is created exactly once. After creation, instruments are cached and returned on subsequent requests without locking.

A minimal custom provider might look like this:

```go
type MyProvider struct {
    counters sync.Map // map[string]Counter
    // similarly for upDownCounters and histograms
}

func (p *MyProvider) Counter(name string, opts ...InstrumentOption) Counter {
    if inst, ok := p.counters.Load(name); ok {
        return inst.(Counter)
    }
    // create new counter with options, store in map
    c := NewCounter(name, opts...)
    p.counters.Store(name, c)
    return c
}

// Implement UpDownCounter and Histogram similarly
```

## Inspector

The **Inspector** interface provides read-only access to instruments and their associated metadata. It enables callers to retrieve instrument instances along with a defensive copy of their configuration, or to list all registered instruments.

Implementations are expected to include methods:

- `CounterWithMeta(name string) (Counter, InstrumentConfig, bool)`
- `UpDownCounterWithMeta(name string) (UpDownCounter, InstrumentConfig, bool)`
- `HistogramWithMeta(name string) (Histogram, InstrumentConfig, bool)`
- `ListMetadata() []InstrumentEntry`

Inspector implementations must ensure consistent snapshots of instrument state and metadata, typically by acquiring the same locks used during instrument initialization. This prevents race conditions and guarantees that the returned metadata accurately reflects the current state.

[BasicProvider](#basicprovider) offers a simple, concurrency-safe in-memory implementation of `Inspector`. It uses the same per-key mutexes as the `Provider` to synchronize access, returning copies of the stored configurations to protect internal state from mutation.

A custom inspector implementation might embed or mimic [BasicProvider's](#basicprovider) pattern as follows:

```go
type MyInspector struct {
    provider *MyProvider
}

func (i *MyInspector) CounterWithMeta(name string) (Counter, InstrumentConfig, bool) {
    // Acquire per-key lock to ensure consistent snapshot
    // Retrieve counter and config from provider's internal maps
    // Return defensive copy of config
}
```

This design allows for flexible inspection of metrics while preserving thread safety and data integrity.

## BasicProvider

### Description

BasicProvider stores instruments in three sync.Maps keyed by the instrument name and uses a separate sync.Map to hold per‑key mutexes for initialization. When you request an instrument:
1.	It performs a fast read using the appropriate map. If the instrument exists, it returns immediately.
2.	Otherwise it applies any options to build an InstrumentConfig off‑lock, acquires the per‑key mutex, re‑checks that the instrument doesn’t already exist, stores the metadata and creates the instrument.
3.	After initialization it optionally deletes the per‑key mutex from the inits map to allow the mutex to be garbage collected.

[Inspector](#inspector) methods such as CounterWithMeta take the same per‑key mutex to provide a consistent snapshot of the instrument and its metadata. They return a defensive copy of the InstrumentConfig so callers cannot mutate the provider’s internal state.

The provider includes internal invariant checks that detect impossible situations (for example, an instrument exists but no metadata is stored). In debug and race builds these checks panic to fail fast; in non‑debug builds the provider logs a warning and continues.

### Features

- **Concurrency‑safe in‑memory implementation** – uses `sync.Map` and per‑key mutexes to safely share instruments across goroutines.
- **Lazy instrument creation** – counters, up/down counters and histograms are created on demand and reused for the same key.
- **Inspector helpers** – retrieve instrument instances along with a defensive copy of their metadata, or list all registered instruments.
- **Configurable instrument metadata** – set descriptions, units and static attributes on instruments through functional options.
- **Invariant checking** – detects unexpected internal states such as missing metadata and optionally fails fast under debug or race builds.
- **Optional init mutex cleanup** – remove per‑key initialization mutexes after use to reduce memory for many short‑lived instrument names.

### Quick start

```go
package main

import (
    "fmt"

    "github.com/ygrebnov/metrics"
)

func main() {
    // Create a new provider with default options.
    p := metrics.NewBasicProvider()

    // Create or retrieve a counter instrument with optional metadata.
    c := p.Counter("requests_total",
        metrics.WithDescription("HTTP requests"),
        metrics.WithUnit("count"),
    )

    // Record a value.
    c.Add(1)

    // Inspect the counter and its metadata.
    if inst, cfg, ok := p.CounterWithMeta("requests_total"); ok {
        fmt.Printf("description: %s\n", cfg.Description)
        _ = inst // use inst as metrics.Counter
    }
}
```
By default, the provider removes per‑key initialization mutexes once an instrument has been created. To disable this cleanup (e.g. for debugging), call metrics.WithInitCleanupDisabled() when constructing the provider.

### Examples

Create and use an up/down counter:
```go
u := p.UpDownCounter("in_flight",
    metrics.WithDescription("in‑flight requests"),
    metrics.WithUnit("count"),
)
u.Add(+1) // increment
u.Add(-1) // decrement
```

Record histogram measurements:
```go
h := p.Histogram("request_duration_seconds",
    metrics.WithDescription("request latency"),
    metrics.WithUnit("seconds"),
)
h.Record(0.123)
h.Record(0.256)
snapshot := h.Snapshot()
fmt.Printf("count=%d min=%f max=%f mean=%f\n",
    snapshot.Count, snapshot.Min, snapshot.Max, snapshot.Mean)
```

List all registered instruments and their metadata:
```go
for _, entry := range p.ListMetadata() {
    fmt.Printf("%s %s: %s\n", entry.Type, entry.Name, entry.Config.Description)
}
```

Enable the race detector and debug invariants when running tests to catch concurrency bugs:
```go
go test -race ./...
go test -tags=debug ./...
```

## Contributing

Contributions are welcome!  
Please open an [issue](https://github.com/ygrebnov/metrics/issues) or submit a [pull request](https://github.com/ygrebnov/metrics/pulls).

## License

Distributed under the MIT License. See the [LICENSE](LICENSE) file for details.
