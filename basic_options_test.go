package metrics

import (
	"testing"
)

func TestBasicProvider_InstrumentsWithOptions(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		create   func(p *BasicProvider)
		check    func(t *testing.T, p *BasicProvider)
	}{
		{
			name:     "counter",
			typeName: "cnt",
			create: func(p *BasicProvider) {
				p.Counter("cnt", WithDescription("my counter"), WithUnit("1"), WithAttributes(map[string]string{"k": "v"}))
			},
			check: func(t *testing.T, p *BasicProvider) {
				c := p.Counter("cnt")
				bc, ok := c.(*BasicCounter)
				if !ok {
					t.Fatalf("expected *BasicCounter, got %T", c)
				}
				bc.Add(5)
				if got := bc.Snapshot(); got != 5 {
					t.Fatalf("unexpected counter value: got %d want %d", got, 5)
				}
				cfg, ok := p.metaLoad("cnt")
				if !ok {
					t.Fatal("expected metadata for 'cnt' to be present")
				}
				if cfg.Description != "my counter" {
					t.Fatalf("unexpected description: got %q want %q", cfg.Description, "my counter")
				}
				if cfg.Unit != "1" {
					t.Fatalf("unexpected unit: got %q want %q", cfg.Unit, "1")
				}
				if cfg.Attributes == nil {
					t.Fatalf("expected attributes map to be non-nil")
				}
				if v, ok := cfg.Attributes["k"]; !ok || v != "v" {
					t.Fatalf("unexpected attribute 'k': got %q want %q", v, "v")
				}
			},
		},
		{
			name:     "updown",
			typeName: "udc",
			create: func(p *BasicProvider) {
				p.UpDownCounter("udc", WithDescription("updown"), WithUnit("items"), WithAttributes(map[string]string{"a": "b"}))
			},
			check: func(t *testing.T, p *BasicProvider) {
				u := p.UpDownCounter("udc")
				bu, ok := u.(*BasicUpDownCounter)
				if !ok {
					t.Fatalf("expected *BasicUpDownCounter, got %T", u)
				}
				bu.Add(10)
				bu.Add(-3)
				if got := bu.Snapshot(); got != 7 {
					t.Fatalf("unexpected updown value: got %d want %d", got, 7)
				}
				cfg, ok := p.metaLoad("udc")
				if !ok {
					t.Fatal("expected metadata for 'udc' to be present")
				}
				if cfg.Description != "updown" {
					t.Fatalf("unexpected description: got %q want %q", cfg.Description, "updown")
				}
				if cfg.Unit != "items" {
					t.Fatalf("unexpected unit: got %q want %q", cfg.Unit, "items")
				}
				if v, ok := cfg.Attributes["a"]; !ok || v != "b" {
					t.Fatalf("unexpected attribute 'a': got %q want %q", v, "b")
				}
			},
		},
		{
			name:     "histogram",
			typeName: "h",
			create: func(p *BasicProvider) {
				p.Histogram("h", WithDescription("histogram"), WithUnit("ms"), WithAttributes(map[string]string{"x": "y"}))
			},
			check: func(t *testing.T, p *BasicProvider) {
				h := p.Histogram("h")
				bh, ok := h.(*BasicHistogram)
				if !ok {
					t.Fatalf("expected *BasicHistogram, got %T", h)
				}
				bh.Record(1.5)
				bh.Record(2.5)
				s := bh.Snapshot()
				if s.Count != 2 {
					t.Fatalf("unexpected count: got %d want %d", s.Count, 2)
				}
				if s.Sum != 4.0 {
					t.Fatalf("unexpected sum: got %v want %v", s.Sum, 4.0)
				}
				if s.Min != 1.5 || s.Max != 2.5 {
					t.Fatalf("unexpected min/max: got %v/%v want %v/%v", s.Min, s.Max, 1.5, 2.5)
				}
				if s.Mean != 2.0 {
					t.Fatalf("unexpected mean: got %v want %v", s.Mean, 2.0)
				}
				cfg, ok := p.metaLoad("h")
				if !ok {
					t.Fatal("expected metadata for 'h' to be present")
				}
				if cfg.Description != "histogram" {
					t.Fatalf("unexpected description: got %q want %q", cfg.Description, "histogram")
				}
				if cfg.Unit != "ms" {
					t.Fatalf("unexpected unit: got %q want %q", cfg.Unit, "ms")
				}
				if v, ok := cfg.Attributes["x"]; !ok || v != "y" {
					t.Fatalf("unexpected attribute 'x': got %q want %q", v, "y")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewBasicProvider()
			tc.create(p)
			tc.check(t, p)
		})
	}
}

func TestWithAttributesCopiesMap(t *testing.T) {
	p := NewBasicProvider()
	attrs := map[string]string{"m": "n"}
	p.Counter("a", WithAttributes(attrs))
	// mutate original
	attrs["m"] = "mutated"
	cfg, _ := p.metaLoad("a")
	if got := cfg.Attributes["m"]; got != "n" {
		t.Fatalf("expected stored attribute to remain 'n', got %q", got)
	}
}

func TestBasicProvider_OptionsAreOnlyAppliedOnFirstCreation(t *testing.T) {
	p := NewBasicProvider()
	p.Counter("dup", WithDescription("first"))
	p.Counter("dup", WithDescription("second"))
	cfg, _ := p.metaLoad("dup")
	if cfg.Description != "first" {
		t.Fatalf("expected first description to be kept, got %q", cfg.Description)
	}
	p.UpDownCounter("dupud", WithDescription("ud-first"))
	p.UpDownCounter("dupud", WithDescription("ud-second"))
	cfg, _ = p.metaLoad("dupud")
	if cfg.Description != "ud-first" {
		t.Fatalf("expected first updown description to be kept, got %q", cfg.Description)
	}
	p.Histogram("duph", WithDescription("h-first"))
	p.Histogram("duph", WithDescription("h-second"))
	cfg, _ = p.metaLoad("duph")
	if cfg.Description != "h-first" {
		t.Fatalf("expected first histogram description to be kept, got %q", cfg.Description)
	}
}
