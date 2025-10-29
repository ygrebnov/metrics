package examples

import (
	"fmt"
	"sync"

	"github.com/ygrebnov/metrics"
)

// ExampleBasicProvider_concurrent demonstrates concurrent usage of BasicProvider.
func ExampleBasicProvider_concurrent() {
	p := metrics.NewBasicProvider()
	name := "concurrent_counter"
	var wg sync.WaitGroup
	const n = 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			c := p.Counter(name)
			c.Add(1)
		}()
	}
	wg.Wait()
	inst, _, ok := p.CounterWithMeta(name)
	if !ok {
		fmt.Println("counter missing")
		return
	}
	bc := inst.(*metrics.BasicCounter)
	fmt.Printf("final value: %d (expected %d)\n", bc.Snapshot(), n)

	// Output: final value: 100 (expected 100)
}
