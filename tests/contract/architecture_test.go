// Package contract provides architectural contract tests for the Beacon library.
//
// These tests validate architectural principles defined in the project constitution:
// - Package structure and layer boundaries (F-2)
// - Dependency constraints
// - Interface segregation
package contract

// nosemgrep: beacon-external-dependencies
import (
	"os"
	"os/exec" // Standard library, required for architecture tests (grep subprocess per gosec nolint)
	"path/filepath"
	"strings"
	"testing"
)

// ==============================================================================
// M1-Refactoring Architecture Tests (TDD - RED Phase)
// ==============================================================================
// T029: Layer boundary test - querier does NOT import internal/network

// TestLayerBoundaries_QuerierDoesNotImportInternalNetwork validates FR-002 layer compliance.
//
// FR-002 (Layer Boundary Compliance): Querier MUST use Transport abstraction, not internal/network directly.
//
// This test will FAIL until T036 removes the internal/network import from querier/querier.go.
//
// Expected violation (baseline):
//
//	querier/querier.go:13: "github.com/joshuafuller/beacon/internal/network"
//
// Expected after T036: No matches (zero layer violations)
func TestLayerBoundaries_QuerierDoesNotImportInternalNetwork(t *testing.T) {
	// Find project root (go up from tests/contract/ to project root)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(cwd, "../..")
	querierFile := filepath.Join(projectRoot, "querier/querier.go")

	// Use grep to find any imports of internal/network in querier/querier.go
	// G204: test code with controlled input (grep pattern is hardcoded, querierFile is constructed from project root)
	cmd := exec.Command("grep", "-n", `"github.com/joshuafuller/beacon/internal/network"`, querierFile) //nolint:gosec // G204: test code with controlled input
	output, err := cmd.CombinedOutput()

	// grep exit codes:
	//   0 = found matches (violation exists - test should FAIL)
	//   1 = no matches (no violation - test should PASS)
	//   2 = error (file not found, permission denied, etc.)

	if err == nil {
		// grep found matches (exit code 0) - layer violation exists!
		violations := strings.TrimSpace(string(output))
		t.Errorf("Layer boundary violation: querier imports internal/network directly\n"+
			"Expected: Querier should use internal/transport abstraction (FR-002)\n"+
			"Violations found:\n%s\n\n"+
			"Fix: T036 will remove internal/network import, T037 will add internal/transport import",
			violations)
		return
	}

	// If exit status 1, no matches found - test passes
	if strings.Contains(err.Error(), "exit status 1") {
		// No violations - test passes
		return
	}

	// Any other error (exit status 2, etc.) is a real error
	t.Fatalf("grep command failed: %v\nOutput: %s", err, string(output))
}
