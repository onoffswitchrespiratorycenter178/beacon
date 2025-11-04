// Test file for F-2 Package Structure rules
// This file intentionally violates layer boundaries
package testpackage

// ============================================================================
// F-2: Internal Imports Public (beacon-internal-imports-public)
// ============================================================================

// SHOULD TRIGGER: beacon-internal-imports-public
// Internal packages MUST NOT import public packages
import "github.com/joshuafuller/beacon/querier"

func BadInternalImportsPublic() {
	// This creates circular dependency
	_ = querier.New
}

// Correct: Internal should only import other internal packages
// import "github.com/joshuafuller/beacon/internal/message"
