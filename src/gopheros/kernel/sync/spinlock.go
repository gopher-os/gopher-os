// Package sync provides synchronization primitive implementations for spinlocks
// and semaphore.
package sync

import "sync/atomic"

var (
	// TODO: replace with real yield function when context-switching is implemented.
	yieldFn func()
)

// Spinlock implements a lock where each task trying to acquire it busy-waits
// till the lock becomes available.
type Spinlock struct {
	state uint32
}

// Acquire blocks until the lock can be acquired by the currently active task.
// Any attempt to re-acquire a lock already held by the current task will cause
// a deadlock.
func (l *Spinlock) Acquire() {
	archAcquireSpinlock(&l.state, 1)
}

// TryToAcquire attempts to acquire the lock and returns true if the lock could
// be acquired or false otherwise.
func (l *Spinlock) TryToAcquire() bool {
	return atomic.SwapUint32(&l.state, 1) == 0
}

// Release relinquishes a held lock allowing other tasks to acquire it. Calling
// Release while the lock is free has no effect.
func (l *Spinlock) Release() {
	atomic.StoreUint32(&l.state, 0)
}

// archAcquireSpinlock is an arch-specific implementation for acquiring the lock.
func archAcquireSpinlock(state *uint32, attemptsBeforeYielding uint32)
