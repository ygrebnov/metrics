# Metrics

Metrics is a small, concurrency‑safe in‑memory metrics provider for Go. It is designed for tests, examples and lightweight applications where you need simple counters, up/down counters and histograms without pulling in heavy dependencies. Instruments are created lazily and deduplicated by type and name, and internal invariant checks help surface bugs early.

## Features

- **Concurrency‑safe in‑memory implementation** – uses `sync.Map` and per‑key mutexes to safely share instruments across goroutines.
- **Lazy instrument creation** – counters, up/down counters and histograms are created on demand and reused for the same key.
- **Inspector helpers** – retrieve instrument instances along with a defensive copy of their metadata, or list all registered instruments.
- **Configurable instrument metadata** – set descriptions, units and static attributes on instruments through functional options.
- **Invariant checking** – detects unexpected internal states such as missing metadata and optionally fails fast under debug or race builds.
- **Optional init mutex cleanup** – remove per‑key initialization mutexes after use to reduce memory for many short‑lived instrument names.

## Installation

To add `metrics` to your project, run:

```bash
go get example.com/ygrebnov/metrics
```

## Quick start
```go
package main

import (
    "fmt"

    "example.com/your-module-path/metrics"
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

## How it works

BasicProvider stores instruments in three sync.Maps keyed by the instrument name and uses a separate sync.Map to hold per‑key mutexes for initialization. When you request an instrument:
1.	It performs a fast read using the appropriate map. If the instrument exists, it returns immediately.
2.	Otherwise it applies any options to build an InstrumentConfig off‑lock, acquires the per‑key mutex, re‑checks that the instrument doesn’t already exist, stores the metadata and creates the instrument.
3.	After initialization it optionally deletes the per‑key mutex from the inits map to allow the mutex to be garbage collected.

Inspector methods such as CounterWithMeta take the same per‑key mutex to provide a consistent snapshot of the instrument and its metadata. They return a defensive copy of the InstrumentConfig so callers cannot mutate the provider’s internal state.

The provider includes internal invariant checks that detect impossible situations (for example, an instrument exists but no metadata is stored). In debug and race builds these checks panic to fail fast; in non‑debug builds the provider logs a warning and continues.

## Examples

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

Contributions are welcome! If you find a bug or would like to add a feature:
1.	Open an issue to discuss your idea or report a problem.
2.	Fork the repository and create a new branch for your changes.
3.	Add tests for new functionality or to reproduce bugs.
4.	Submit a pull request describing your changes.

We value clear, well‑documented code and comprehensive tests. See existing code and tests for guidance.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE)￼file for details.
