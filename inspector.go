package metrics

// Inspector provides an optional capability of metadata inspection/snapshot.
// Implementations should return defensive copies of configs.
// WithMeta methods return the instrument (if it exists/created), a snapshot of its config,
// and a flag of whether it was found.
// Snapshot semantics: best-effort at call time.
// Methods must be safe for concurrent use.
type Inspector interface {
	CounterWithMeta(name string) (Counter, InstrumentConfig, bool)
	UpDownCounterWithMeta(name string) (UpDownCounter, InstrumentConfig, bool)
	HistogramWithMeta(name string) (Histogram, InstrumentConfig, bool)

	// ListMetadata returns enumeration for admin/debug UIs.
	ListMetadata() []InstrumentEntry
}

type InstrumentEntry struct {
	Type   InstrumentType
	Name   string
	Config InstrumentConfig // defensive copy
}
