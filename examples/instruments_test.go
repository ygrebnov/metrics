package examples

import (
	"fmt"
	"sort"

	"github.com/ygrebnov/metrics"
)

// ExampleBasicProvider_instruments demonstrates how to create instruments with metadata
// using the BasicProvider and how to list the stored metadata.
func ExampleBasicProvider_instruments() {
	p := metrics.NewBasicProvider()
	// create instruments with metadata
	p.Counter(
		"c1",
		metrics.WithDescription("counter 1"),
		metrics.WithAttributes(map[string]string{"env": "dev"}),
	)
	p.UpDownCounter(
		"u1",
		metrics.WithDescription("updown 1"),
	)
	p.Histogram(
		"h1",
		metrics.WithUnit("ms"),
	)

	// list metadata
	entries := p.ListMetadata()

	// sort entries for consistent output
	// (not needed in real code)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})

	// print entries
	for _, e := range entries {
		fmt.Printf("%s:%s -> %#v\n", e.Type, e.Name, e.Config)
	}

	// Output: counter:c1 -> metrics.InstrumentConfig{Description:"counter 1", Unit:"", Attributes:map[string]string{"env":"dev"}}
	// histogram:h1 -> metrics.InstrumentConfig{Description:"", Unit:"ms", Attributes:map[string]string(nil)}
	// updown:u1 -> metrics.InstrumentConfig{Description:"updown 1", Unit:"", Attributes:map[string]string(nil)}
}
