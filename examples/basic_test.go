package examples

import (
	"fmt"
	"time"

	"github.com/ygrebnov/metrics"
)

// ExampleBasicProvider_nominal demonstrates the nominal usage of BasicProvider.
func ExampleBasicProvider_nominal() {
	p := metrics.NewBasicProvider()

	// create a counter with some metadata
	c := p.Counter(
		"requests",
		metrics.WithDescription("HTTP requests"),
		metrics.WithUnit("count"),
	)

	// record some values
	c.Add(1)
	c.Add(2)

	// small delay to simulate work
	time.Sleep(10 * time.Millisecond)

	// fetch instrument and metadata
	inst, cfg, ok := p.CounterWithMeta("requests")
	if !ok {
		fmt.Println("counter not found")
		return
	}
	bc := inst.(*metrics.BasicCounter)
	fmt.Printf("Counter snapshot=%d, description=%q, unit=%q\n", bc.Snapshot(), cfg.Description, cfg.Unit)

	// Output: Counter snapshot=3, description="HTTP requests", unit="count"
}
