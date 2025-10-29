package metrics

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

func (p *BasicProvider) getInstrumentMeta(key InstrumentKey) (InstrumentConfig, bool) {
	m, ok := p.meta.Load(key)
	if !ok {
		// invariant violation: instrument without meta
		p.reportInvariantViolation(key.Type.String()+"_meta_missing", key)
		return InstrumentConfig{}, false
	}

	c, ok2 := m.(InstrumentConfig)
	if !ok2 {
		// invariant violation: wrong meta type
		p.reportInvariantViolation(key.Type.String()+"_meta_type", key)
		return InstrumentConfig{}, false
	}

	return copyConfig(c), true
}

// CounterWithMeta implements Inspector.CounterWithMeta for BasicProvider.
// It acquires the per-key init mutex, re-checks, then reads both the instance
// and metadata before unlocking in order to provide a consistent snapshot.
// The third return value is true if and only if both the instrument and the meta were found and both valid.
// Invariant violations (e.g., instrument exists but meta missing) are reported via logger.
func (p *BasicProvider) CounterWithMeta(name string) (Counter, InstrumentConfig, bool) {
	key := NewInstrumentKey(InstrumentTypeCounter, name)
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.counters.Load(name)
	if !ok {
		// not created
		return nil, InstrumentConfig{}, false
	}

	inst, ok2 := v.(*BasicCounter)
	if !ok2 {
		// invariant violation: wrong type in map
		p.reportInvariantViolation("counter_type", key)
		return nil, InstrumentConfig{}, false
	}

	c, okOverall := p.getInstrumentMeta(key)

	return inst, c, okOverall
}

// UpDownCounterWithMeta implements Inspector.UpDownCounterWithMeta for BasicProvider.
// It acquires the per-key init mutex, re-checks, then reads both the instance
// and metadata before unlocking in order to provide a consistent snapshot.
// The third return value is true if and only if both the instrument and the meta were found and both valid.
// Invariant violations (e.g., instrument exists but meta missing) are reported via logger.
func (p *BasicProvider) UpDownCounterWithMeta(name string) (UpDownCounter, InstrumentConfig, bool) {
	key := NewInstrumentKey(InstrumentTypeUpDown, name)
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.updowns.Load(name)
	if !ok {
		// not created
		return nil, InstrumentConfig{}, false
	}

	inst, ok2 := v.(*BasicUpDownCounter)
	if !ok2 {
		// invariant violation: wrong type in map
		p.reportInvariantViolation("updown_type", key)
		return nil, InstrumentConfig{}, false
	}

	c, okOverall := p.getInstrumentMeta(key)

	return inst, c, okOverall
}

// HistogramWithMeta implements Inspector.HistogramWithMeta for BasicProvider.
// It acquires the per-key init mutex, re-checks, then reads both the instance
// and metadata before unlocking in order to provide a consistent snapshot.
// The third return value is true if and only if both the instrument and the meta were found and both valid.
// Invariant violations (e.g., instrument exists but meta missing) are reported via logger.
func (p *BasicProvider) HistogramWithMeta(name string) (Histogram, InstrumentConfig, bool) {
	key := NewInstrumentKey(InstrumentTypeHistogram, name)
	km := p.keyMu(key)
	km.Lock()
	defer km.Unlock()

	v, ok := p.histograms.Load(name)
	if !ok {
		// not created
		return nil, InstrumentConfig{}, false
	}

	inst, ok2 := v.(*BasicHistogram)
	if !ok2 {
		// invariant violation: wrong type in map
		p.reportInvariantViolation("histogram_type", key)
		return nil, InstrumentConfig{}, false
	}

	c, okOverall := p.getInstrumentMeta(key)

	return inst, c, okOverall
}

// ListMetadata returns a best-effort snapshot of metadata entries. It does not
// acquire per-key init mutexes for each entry; callers should treat the result
// as a point-in-time snapshot that may race with concurrent creations.
func (p *BasicProvider) ListMetadata() []InstrumentEntry {
	out := make([]InstrumentEntry, 0)
	p.meta.Range(func(k, v interface{}) bool {
		key, ok := k.(InstrumentKey)
		cfg, ok2 := v.(InstrumentConfig)
		if !ok || !ok2 {
			return true // skip invalid entries
		}

		out = append(out, InstrumentEntry{Type: key.Type, Name: key.Name, Config: copyConfig(cfg)})
		return true
	})
	return out
}
