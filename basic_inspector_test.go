package metrics

import (
	"sync"
	"testing"
)

const (
	mut     = "mut"
	ext     = "ext"
	mutated = "mutated"
	u1      = "u1"
	d1      = "d1"
)

func TestCounterWithMeta_TableDriven(t *testing.T) {
	t.Run("not_created", func(t *testing.T) {
		p := NewBasicProvider()
		if inst, cfg, ok := p.CounterWithMeta("missing"); ok || inst != nil || cfg.Description != "" {
			t.Fatalf("expected not found; got ok=%v inst=%v cfg=%v", ok, inst, cfg)
		}
	})

	t.Run("created_and_snapshot", func(t *testing.T) {
		p := NewBasicProvider()
		c := p.Counter("cnt1")
		c.Add(3)
		inst, cfg, ok := p.CounterWithMeta("cnt1")
		if !ok || inst == nil {
			t.Fatal("expected found counter")
		}
		bc, ok2 := inst.(*BasicCounter)
		if !ok2 {
			t.Fatalf("expected *BasicCounter, got %T", inst)
		}
		if got := bc.Snapshot(); got != 3 {
			t.Fatalf("expected snapshot 3; got %d", got)
		}
		// config should be empty/default
		if cfg.Description != "" || cfg.Unit != "" || len(cfg.Attributes) != 0 {
			t.Fatalf("unexpected config: %v", cfg)
		}
	})

	t.Run("created_with_options_and_defensive_copy", func(t *testing.T) {
		p := NewBasicProvider()
		attrs := map[string]string{"k": "v"}
		p.Counter("cnt2", WithDescription("desc"), WithUnit("u"), WithAttributes(attrs))
		_, cfg1, ok1 := p.CounterWithMeta("cnt2")
		if !ok1 {
			t.Fatal("expected found")
		}
		if cfg1.Description != "desc" || cfg1.Unit != "u" {
			t.Fatalf("unexpected cfg fields: %v", cfg1)
		}
		if cfg1.Attributes["k"] != "v" {
			t.Fatalf("unexpected attrs: %v", cfg1.Attributes)
		}

		// Mutate returned config and original attrs map; provider should keep a defensive copy
		cfg1.Attributes["k"] = mutated
		attrs["k"] = "external"
		_, cfg2, ok2 := p.CounterWithMeta("cnt2")
		if !ok2 {
			t.Fatal("expected found on second read")
		}
		if cfg2.Attributes["k"] != "v" {
			t.Fatalf("provider config mutated; want v got %v", cfg2.Attributes["k"])
		}
	})
}

func TestUpDownCounterWithMeta_TableDriven(t *testing.T) {
	t.Run("not_created", func(t *testing.T) {
		p := NewBasicProvider()
		if inst, cfg, ok := p.UpDownCounterWithMeta("missing"); ok || inst != nil || cfg.Description != "" {
			t.Fatalf("expected not found; got ok=%v inst=%v cfg=%v", ok, inst, cfg)
		}
	})

	t.Run("created_and_snapshot", func(t *testing.T) {
		p := NewBasicProvider()
		u := p.UpDownCounter("ud1")
		u.Add(7)
		inst, cfg, ok := p.UpDownCounterWithMeta("ud1")
		if !ok || inst == nil {
			t.Fatal("expected found updown")
		}
		bu, ok2 := inst.(*BasicUpDownCounter)
		if !ok2 {
			t.Fatalf("expected *BasicUpDownCounter, got %T", inst)
		}
		if got := bu.Snapshot(); got != 7 {
			t.Fatalf("expected snapshot 7; got %d", got)
		}
		if cfg.Description != "" || cfg.Unit != "" || len(cfg.Attributes) != 0 {
			t.Fatalf("unexpected config: %v", cfg)
		}
	})

	t.Run("created_with_options_and_defensive_copy", func(t *testing.T) {
		p := NewBasicProvider()
		attrs := map[string]string{"x": "y"}
		p.UpDownCounter("ud2", WithDescription("dud"), WithUnit("units"), WithAttributes(attrs))
		_, cfg1, ok1 := p.UpDownCounterWithMeta("ud2")
		if !ok1 {
			t.Fatal("expected found")
		}
		if cfg1.Description != "dud" || cfg1.Unit != "units" {
			t.Fatalf("unexpected cfg fields: %v", cfg1)
		}
		if cfg1.Attributes["x"] != "y" {
			t.Fatalf("unexpected attrs: %v", cfg1.Attributes)
		}
		cfg1.Attributes["x"] = mut
		attrs["x"] = ext
		_, cfg2, ok2 := p.UpDownCounterWithMeta("ud2")
		if cfg2.Attributes["x"] != "y" {
			t.Fatalf("provider config mutated; want y got %v", cfg2.Attributes["x"])
		}
		if !ok2 {
			t.Fatal("expected found on second read")
		}
	})
}

func TestHistogramWithMeta_TableDriven(t *testing.T) {
	t.Run("not_created", func(t *testing.T) {
		p := NewBasicProvider()
		if inst, cfg, ok := p.HistogramWithMeta("missing"); ok || inst != nil || cfg.Description != "" {
			t.Fatalf("expected not found; got ok=%v inst=%v cfg=%v", ok, inst, cfg)
		}
	})

	t.Run("created_and_snapshot", func(t *testing.T) {
		p := NewBasicProvider()
		h := p.Histogram("h1")
		h.Record(1.5)
		h.Record(2.5)
		inst, cfg, ok := p.HistogramWithMeta("h1")
		if !ok || inst == nil {
			t.Fatal("expected found histogram")
		}
		hh, ok2 := inst.(*BasicHistogram)
		if !ok2 {
			t.Fatalf("expected *BasicHistogram, got %T", inst)
		}
		snap := hh.Snapshot()
		if snap.Count != 2 || snap.Sum != 4.0 {
			t.Fatalf("unexpected snapshot: %+v", snap)
		}
		if cfg.Description != "" || cfg.Unit != "" || len(cfg.Attributes) != 0 {
			t.Fatalf("unexpected config: %v", cfg)
		}
	})

	t.Run("created_with_options_and_defensive_copy", func(t *testing.T) {
		p := NewBasicProvider()
		attrs := map[string]string{"a": "b"}
		p.Histogram("h2", WithDescription("hd"), WithUnit("s"), WithAttributes(attrs))
		_, cfg1, ok1 := p.HistogramWithMeta("h2")
		if !ok1 {
			t.Fatal("expected found")
		}
		if cfg1.Description != "hd" || cfg1.Unit != "s" {
			t.Fatalf("unexpected cfg fields: %v", cfg1)
		}
		if cfg1.Attributes["a"] != "b" {
			t.Fatalf("unexpected attrs: %v", cfg1.Attributes)
		}
		cfg1.Attributes["a"] = mut
		attrs["a"] = ext
		_, cfg2, ok2 := p.HistogramWithMeta("h2")
		if cfg2.Attributes["a"] != "b" {
			t.Fatalf("provider config mutated; want b got %v", cfg2.Attributes["a"])
		}
		if !ok2 {
			t.Fatal("expected found on second read")
		}
	})
}

func TestListMetadata_TableDriven(t *testing.T) {
	p := NewBasicProvider()
	p.Counter("c", WithUnit(u1))
	p.UpDownCounter("u", WithDescription(d1))
	p.Histogram("h", WithAttributes(map[string]string{"k": "v"}))

	entries := p.ListMetadata()
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 metadata entries; got %d", len(entries))
	}

	find := func(typ InstrumentType, name string) (InstrumentConfig, bool) {
		for _, e := range entries {
			if e.Type == typ && e.Name == name {
				return e.Config, true
			}
		}
		return InstrumentConfig{}, false
	}

	if cfg, ok := find(InstrumentTypeCounter, "c"); !ok || cfg.Unit != u1 {
		t.Fatalf("counter entry missing or wrong: %v %v", ok, cfg)
	}
	if cfg, ok := find(InstrumentTypeUpDown, "u"); !ok || cfg.Description != d1 {
		t.Fatalf("updown entry missing or wrong: %v %v", ok, cfg)
	}
	if cfg, ok := find(InstrumentTypeHistogram, "h"); !ok || cfg.Attributes["k"] != "v" {
		t.Fatalf("histogram entry missing or wrong: %v %v", ok, cfg)
	}

	// Defensive copy: mutate entries and ensure provider's meta remains unchanged
	for i := range entries {
		if entries[i].Config.Attributes != nil {
			entries[i].Config.Attributes["k"] = mutated
		}
	}
	// Re-query provider to ensure config still original
	_, cfgC, okC := p.CounterWithMeta("c")
	if !okC {
		t.Fatal("expected counter found after mutation")
	}
	if cfgC.Unit != u1 {
		t.Fatalf("provider metadata mutated via ListMetadata copy: got %v", cfgC)
	}
}

func TestConcurrentCreationAndInitCleanup(t *testing.T) {
	// Default provider: init cleanup enabled -> per-key mutex should be deleted after create
	p := NewBasicProvider()
	name := "race_counter"
	var wg sync.WaitGroup
	const goroutines = 50
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c := p.Counter(name)
			c.Add(1)
		}()
	}
	wg.Wait()

	// Instrument must exist and meta must be present
	if v, ok := p.counters.Load(name); !ok || v == nil {
		t.Fatalf("expected instrument created in counters map; got ok=%v v=%v", ok, v)
	}
	key := NewInstrumentKey(InstrumentTypeCounter, name)
	if _, ok := p.meta.Load(key); !ok {
		t.Fatalf("expected meta stored for instrument; missing")
	}

	// per-key mutex should have been removed from inits map (cleanup enabled by default)
	if _, ok := p.inits.Load(key); ok {
		t.Fatalf("expected per-key mutex to be cleaned up, but found one in inits map")
	}

	// Now with cleanup disabled: mutex should remain
	p2 := NewBasicProvider(WithInitCleanupDisabled())
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			c := p2.Counter(name + "2")
			c.Add(1)
		}()
	}
	wg.Wait()
	key2 := NewInstrumentKey(InstrumentTypeCounter, name+"2")
	// ensure instrument/meta created without calling keyMu (avoid creating a mutex)
	if v, ok := p2.counters.Load(name + "2"); !ok || v == nil {
		t.Fatalf("expected instrument created in counters map for p2; got ok=%v v=%v", ok, v)
	}
	if _, ok := p2.meta.Load(key2); !ok {
		t.Fatalf("expected meta stored for instrument p2; missing")
	}
	if v, ok := p2.inits.Load(key2); !ok {
		t.Fatalf("expected per-key mutex entry to remain when cleanup disabled")
	} else {
		if _, ok2 := v.(*sync.Mutex); !ok2 {
			t.Fatalf("unexpected type in inits map: %T", v)
		}
	}
}

func TestInvalidMapEntriesAndInvariantBehavior(t *testing.T) {
	p := NewBasicProvider()

	// Case 1: instrument exists but meta missing -> should return inst, ok=false
	p.counters.Store("onlyinst", &BasicCounter{})
	inst, cfg, ok := p.CounterWithMeta("onlyinst")
	if inst == nil {
		t.Fatalf("expected instance returned even when meta missing")
	}
	if ok {
		t.Fatalf("expected ok==false when meta missing; got true; cfg=%v", cfg)
	}

	// Case 2: wrong type stored in counters map -> should report invariant and return not found
	p2 := NewBasicProvider()
	p2.counters.Store("bad", "not-a-counter")
	p2.meta.Store(NewInstrumentKey(InstrumentTypeCounter, "bad"), InstrumentConfig{})
	inst2, _, ok2 := p2.CounterWithMeta("bad")
	if inst2 != nil || ok2 {
		t.Fatalf("expected nil,false when wrong type in counters map; got %v,%v", inst2, ok2)
	}

	// Case 3: wrong type in meta map -> getInstrumentMeta should report invariant and return false
	p3 := NewBasicProvider()
	p3.counters.Store("badmeta", &BasicCounter{})
	p3.meta.Store(NewInstrumentKey(InstrumentTypeCounter, "badmeta"), "not-a-config")
	inst3, cfg3, ok3 := p3.CounterWithMeta("badmeta")
	if inst3 == nil {
		t.Fatalf("expected instance returned even when meta has wrong type")
	}
	if ok3 {
		t.Fatalf("expected ok==false when meta type invalid; got true; cfg=%v", cfg3)
	}

	// Case 4: ListMetadata should skip invalid entries
	p4 := NewBasicProvider()
	p4.meta.Store("not-a-key", InstrumentConfig{})
	p4.meta.Store(NewInstrumentKey(InstrumentTypeCounter, "good"), InstrumentConfig{Unit: "u"})
	entries := p4.ListMetadata()
	// only the valid entry should be present
	found := 0
	for _, e := range entries {
		if e.Name == "good" && e.Type == InstrumentTypeCounter && e.Config.Unit == "u" {
			found++
		}
		if e.Name == "not-a-key" {
			t.Fatalf("invalid meta entry should have been skipped: %v", e)
		}
	}
	if found != 1 {
		t.Fatalf("expected 1 valid metadata entry; got %d", found)
	}
}
