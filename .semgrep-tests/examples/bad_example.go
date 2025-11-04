// Test file for F-2 Package Structure - Examples
// Examples should only import public APIs
package main

// ============================================================================
// F-2: Examples Import Internal (beacon-test-imports-internal)
// ============================================================================

// SHOULD TRIGGER: beacon-test-imports-internal
// Examples MUST only use public API
import "github.com/joshuafuller/beacon/internal/message"

func main() {
	// If examples need internal packages, public API is insufficient
	_ = message.Parse
}

// Correct: Examples should only import public packages
// import "github.com/joshuafuller/beacon/querier"
