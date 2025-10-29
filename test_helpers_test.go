package metrics

// test helper: read metadata stored under the compound key "typ:name".
// Placed in a _test.go file so it is test-only and not part of the public API.
func metaLoad(p *BasicProvider, t InstrumentType, name string) (InstrumentConfig, bool) {
	key := NewInstrumentKey(t, name)
	v, ok := p.meta.Load(key)
	if !ok {
		return InstrumentConfig{}, false
	}
	cfg, ok := v.(InstrumentConfig)
	return cfg, ok
}
