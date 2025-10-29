- Run with the debug build tag:

```bash
go test -tags=debug ./...
```

Notes

- `WithInitCleanupDisabled()` disables removal of per-key init mutex entries and
  is useful for debugging or for very long-lived instrument names. The default
  behavior removes init mutex entries after initialization to allow the mutexes
  to be garbage-collected.

Contributing

Contributions are welcome. Please open issues or pull requests with tests.
/*
Package metrics provides a small, concurrency-safe in-memory metrics Provider used
for tests, examples, and lightweight applications.

It implements:

  - BasicProvider: creates and reuses instruments (Counter, UpDownCounter, Histogram)
    keyed by (type,name). Instrument creation is deduplicated with per-key mutexes.

  - Inspector helpers: CounterWithMeta, UpDownCounterWithMeta, HistogramWithMeta and
    ListMetadata which return defensive copies of instrument metadata.

Invariants

The provider performs internal invariant checks and reports unexpected internal
states (for example: "instrument exists but meta missing"). In debug and race
builds (controlled via build tags) invariant violations cause a panic to fail
fast; in non-debug builds they are logged and the provider attempts to continue.

Build and test

  - Run unit tests:

    go test ./...

  - Run with the race detector (enables stricter invariant behavior):

    go test -race ./...

  - Enable debug build tag (debug invariants enabled):

    go test -tags=debug ./...

Examples

  p := metrics.NewBasicProvider()
  c := p.Counter("requests", metrics.WithDescription("HTTP requests"), metrics.WithUnit("count"))
  c.Add(1)

  if inst, cfg, ok := p.CounterWithMeta("requests"); ok {
      // cfg is a defensive copy of the stored InstrumentConfig
      _ = cfg
      _ = inst
  }

Notes

  - InstrumentConfig returned by Inspector methods are defensive copies; mutating them
    will not change the provider's stored metadata.

  - Per-key init mutex entries are removed by default after initialization to allow
    GC of many ephemeral instrument names. Disable this behavior with
    metrics.WithInitCleanupDisabled().

*/
package metrics

