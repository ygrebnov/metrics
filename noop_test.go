package metrics

import "testing"

func TestNoopProvider_Minimal(t *testing.T) {
	n := NewNoopProvider()

	// Counter
	c := n.Counter("x")
	if _, ok := c.(noopCounter); !ok {
		t.Fatalf("expected noopCounter type, got %T", c)
	}
	// should be no-op and not panic
	c.Add(123)

	// UpDownCounter
	u := n.UpDownCounter("y")
	if _, ok := u.(noopUpDownCounter); !ok {
		t.Fatalf("expected noopUpDownCounter type, got %T", u)
	}
	u.Add(-5)

	// Histogram
	h := n.Histogram("z")
	if _, ok := h.(noopHistogram); !ok {
		t.Fatalf("expected noopHistogram type, got %T", h)
	}
	h.Record(3.14)
}
