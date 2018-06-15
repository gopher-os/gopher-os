package sync

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestSpinlock(t *testing.T) {
	// Substitute the yieldFn with runtime.Gosched to avoid deadlocks while testing
	defer func(origYieldFn func()) { yieldFn = origYieldFn }(yieldFn)
	yieldFn = runtime.Gosched

	var (
		sl         Spinlock
		wg         sync.WaitGroup
		numWorkers = 10
	)

	sl.Acquire()

	if sl.TryToAcquire() != false {
		t.Error("expected TryToAcquire to return false when lock is held")
	}

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(worker int) {
			sl.Acquire()
			sl.Release()
			wg.Done()
		}(i)
	}

	<-time.After(100 * time.Millisecond)
	sl.Release()
	wg.Wait()
}
