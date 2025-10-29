package metrics

import "testing"

func TestInitsCleanupEnabled(t *testing.T) {
	p := NewBasicProvider() // default: cleanup enabled
	p.Counter("cleanup_enabled")
	key := InstrumentTypeCounter.String() + ":" + "cleanup_enabled"
	if _, ok := p.inits.Load(key); ok {
		t.Fatalf("expected inits entry to be deleted when cleanup enabled")
	}
}

func TestInitsCleanupDisabled(t *testing.T) {
	p := NewBasicProvider(WithInitCleanupDisabled())
	p.Counter("cleanup_disabled")
	key := InstrumentTypeCounter.String() + ":" + "cleanup_disabled"
	v, ok := p.inits.Load(key)
	if !ok || v == nil {
		t.Fatalf("expected inits entry to be present when cleanup disabled")
	}
}
