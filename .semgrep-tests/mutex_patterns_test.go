// Comprehensive test cases for beacon-mutex-defer-unlock Semgrep rule
// TDD: RED → GREEN → REFACTOR
package test

import "sync"

// ============================================================================
// BAD PATTERNS - SHOULD TRIGGER beacon-mutex-defer-unlock
// ============================================================================

// BadPattern1: Lock without any unlock
func BadPattern1() {
	var mu sync.Mutex
	mu.Lock() // ruleid: beacon-mutex-defer-unlock
	// Missing unlock entirely - major bug
}

// BadPattern2: Lock with manual unlock (no defer)
func BadPattern2() {
	var mu sync.Mutex
	mu.Lock() // ruleid: beacon-mutex-defer-unlock
	doWork()
	mu.Unlock() // Manual unlock - won't execute if doWork() panics
}

// BadPattern3: Lock with deferred unlock of WRONG mutex
func BadPattern3() {
	var mu1 sync.Mutex
	var mu2 sync.Mutex
	mu1.Lock()         // ruleid: beacon-mutex-defer-unlock
	defer mu2.Unlock() // Wrong mutex!
}

// BadPattern4: Lock with conditional unlock
func BadPattern4(condition bool) {
	var mu sync.Mutex
	mu.Lock() // ruleid: beacon-mutex-defer-unlock
	if condition {
		mu.Unlock() // Conditional unlock - may not execute
	}
}

// BadPattern5: RLock without defer RUnlock
func BadPattern5() {
	var mu sync.RWMutex
	mu.RLock() // ruleid: beacon-mutex-defer-unlock
	doWork()
	mu.RUnlock() // Manual unlock - unsafe
}

// BadPattern6: Lock with defer but separated by code
func BadPattern6() {
	var mu sync.Mutex
	mu.Lock()         // ruleid: beacon-mutex-defer-unlock
	doWork()          // Code between Lock and defer
	defer mu.Unlock() // Defer not immediately after Lock
}

// ============================================================================
// GOOD PATTERNS - SHOULD NOT TRIGGER
// ============================================================================

// GoodPattern1: Lock with immediate defer unlock
func GoodPattern1() {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock() // ok: defer immediately after Lock
	doWork()
}

// GoodPattern2: RLock with immediate defer RUnlock
func GoodPattern2() {
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock() // ok: defer immediately after RLock
	doWork()
}

// GoodPattern3: Multiple mutexes with proper defers
func GoodPattern3() {
	var mu1 sync.Mutex
	var mu2 sync.Mutex

	mu1.Lock()
	defer mu1.Unlock() // ok: proper defer for mu1

	mu2.Lock()
	defer mu2.Unlock() // ok: proper defer for mu2

	doWork()
}

// GoodPattern4: Nested locks with proper defers
func GoodPattern4() {
	var mu sync.RWMutex

	mu.RLock()
	defer mu.RUnlock() // ok: read lock with defer

	if needsWrite() {
		mu.RUnlock() // Upgrade to write lock
		mu.Lock()
		defer mu.Unlock() // ok: write lock with defer
	}
}

// GoodPattern5: Lock in conditional with defer
func GoodPattern5(condition bool) {
	var mu sync.Mutex

	if condition {
		mu.Lock()
		defer mu.Unlock() // ok: defer in same scope as Lock
		doWork()
	}
}

// GoodPattern6: Pointer receiver mutex with defer
func (s *Service) GoodPattern6() {
	s.mu.Lock()
	defer s.mu.Unlock() // ok: struct mutex with defer
	s.data = "updated"
}

// ============================================================================
// HELPER TYPES AND FUNCTIONS
// ============================================================================

type Service struct {
	mu   sync.Mutex
	data string
}

func doWork() {
	// Simulate work that could panic
}

func needsWrite() bool {
	return true
}
