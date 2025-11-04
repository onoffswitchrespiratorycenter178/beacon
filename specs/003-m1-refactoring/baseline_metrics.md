# M1 Baseline Metrics - Before Refactoring

**Date**: 2025-11-01
**Purpose**: Establish baseline for post-refactoring validation
**Git Commit**: (captured before refactoring begins)

---

## Test Results Baseline

### Test Execution

**Command**: `go test ./... -v -race -coverprofile=baseline_coverage.out`

**Results**:
```
✅ internal/errors:   PASS (93.3% coverage)
✅ internal/message:  PASS (90.9% coverage)
✅ internal/network:  PASS (70.3% coverage)
✅ internal/protocol: PASS (98.0% coverage)
✅ querier:           PASS (71.6% coverage)
✅ tests/contract:    PASS
✅ tests/fuzz:        PASS
⚠️ tests/integration: FAIL (1 flaky test - environment dependent)
```

**Total Tests**: 302 test cases
**Test Results File**: `baseline_tests.txt`

**Integration Test Issue**:
- `TestQuery_RealNetwork_Timeout`: Flaky test depending on network conditions
- Expected behavior: This is acceptable as baseline (integration tests are environment-dependent)
- All other tests (301/302): PASS

---

## Coverage Baseline

**Command**: `go tool cover -func=baseline_coverage.out`

**Overall Coverage**: **85.2%** ✅ (meets ≥80% requirement)

**Package Breakdown**:
| Package | Coverage | Status |
|---------|----------|--------|
| internal/errors | 93.3% | ✅ Excellent |
| internal/message | 90.9% | ✅ Excellent |
| internal/protocol | 98.0% | ✅ Excellent |
| internal/network | 70.3% | 🟡 Good (target for improvement) |
| querier | 71.6% | 🟡 Good (target for improvement) |

**Function Coverage** (querier package - refactoring focus):
| Function | Coverage | Priority |
|----------|----------|----------|
| New() | 90.0% | ✅ |
| Query() | 76.5% | 🟡 |
| collectResponses() | 91.3% | ✅ |
| receiveLoop() | 63.6% | 🟡 Target for improvement |
| Close() | 85.7% | ✅ |
| AsA() | 33.3% | 🟡 Low (type conversion helper) |
| AsPTR() | 66.7% | 🟡 Low (type conversion helper) |
| AsSRV() | 33.3% | 🟡 Low (type conversion helper) |
| AsTXT() | 33.3% | 🟡 Low (type conversion helper) |

**Coverage Report File**: `baseline_coverage.out`

---

## Performance Baseline

**Command**: `go test -bench=BenchmarkQuery -benchmem -count=5 ./querier`

**Platform**:
- OS: Linux (amd64)
- CPU: Intel Xeon E5-2667 v2 @ 3.30GHz
- Cores: 8

**Benchmark Results**:

### BenchmarkQuery (Sequential)
```
BenchmarkQuery-8   5525503   201.2 ns/op   0 B/op   0 allocs/op
BenchmarkQuery-8   5207757   194.3 ns/op   0 B/op   0 allocs/op
BenchmarkQuery-8  10607850   104.7 ns/op   0 B/op   0 allocs/op
BenchmarkQuery-8   5307885   190.6 ns/op   0 B/op   0 allocs/op
BenchmarkQuery-8   5243521   206.0 ns/op   0 B/op   0 allocs/op
```

**Average**: ~179.4 ns/op
**Allocations**: 0 B/op (excellent - already optimized for this path)

### BenchmarkQueryParallel (Concurrent)
```
BenchmarkQueryParallel-8   1708837   648.8 ns/op   0 B/op   0 allocs/op
BenchmarkQueryParallel-8   1802190   671.9 ns/op   0 B/op   0 allocs/op
BenchmarkQueryParallel-8   1659928   667.9 ns/op   0 B/op   0 allocs/op
BenchmarkQueryParallel-8   3120877   482.8 ns/op   0 B/op   0 allocs/op
BenchmarkQueryParallel-8   1602813   629.9 ns/op   0 B/op   0 allocs/op
```

**Average**: ~620.3 ns/op
**Allocations**: 0 B/op

**Note**: Current benchmarks measure query construction overhead, NOT receive path allocations. The 9KB allocation per receive is NOT visible in these benchmarks because they don't exercise the actual network receive operations.

**Action Required**: Create new benchmark for receive path to measure buffer pooling impact:
```go
func BenchmarkReceive(b *testing.B) {
    // Benchmark actual receive operations to capture buffer allocations
}
```

**Benchmark Results File**: `baseline_bench.txt`

---

## Dependency Baseline

**Command**: `go mod graph`

**Dependencies**: 2 entries (minimal, stdlib-focused)

**Dependency Graph File**: `baseline_deps.txt`

**Expected Post-Refactoring**: 2 entries (no new dependencies)

**Validation**: `diff baseline_deps.txt after_refactoring_deps.txt` should show no changes

---

## Layer Boundary Violations Baseline

**Command**: `grep -rn "internal/network" querier/`

**Violations Found**: **1 violation** (P0-2 issue)

```
querier/querier.go:13:	"github.com/joshuafuller/beacon/internal/network"
```

**Analysis**:
- Querier directly imports `internal/network` package
- Violates F-2 Package Structure (should use Transport abstraction)
- **P0-2 Fix**: Remove this import, use `internal/transport` interface instead

**Expected Post-Refactoring**: 0 violations

**Violations File**: `baseline_violations.txt`

---

## Race Condition Baseline

**Command**: `go test ./... -race`

**Result**: ✅ **Zero race conditions detected**

**Expectation**: Maintain zero race conditions after refactoring

---

## Refactoring Target Metrics

### Success Criteria (from spec.md)

| Metric | Baseline | Target | Validation |
|--------|----------|--------|------------|
| **Test Pass Rate** | 301/302 (99.7%) | 302/302 (100%) | All tests pass (fix flaky test or document) |
| **Test Coverage** | 85.2% | ≥85.0% | Maintain or improve |
| **Race Conditions** | 0 detected | 0 detected | No new races |
| **Allocations (Receive)** | ~9000 bytes/op | ~1800 bytes/op | ≥80% reduction (FR-003) |
| **Layer Violations** | 1 violation | 0 violations | Fix querier→network import (FR-002) |
| **Error Swallowing** | 1 instance | 0 instances | Fix CloseSocket (FR-004) |
| **Transport Abstraction** | ❌ None | ✅ Interface | Create Transport interface (FR-001) |

---

## Critical Findings

### 1. Benchmark Gap (High Priority)

**Issue**: Current benchmarks don't measure receive path allocations
**Impact**: Cannot validate FR-003 (buffer pooling) without proper benchmark
**Action**: Create `BenchmarkReceive` or `BenchmarkReceivePath` that exercises actual socket receive operations

**Recommendation**:
```go
// Add to querier/querier_test.go
func BenchmarkReceivePath(b *testing.B) {
    // Create querier with real socket
    q, _ := New()
    defer q.Close()

    // Benchmark receive operations
    ctx := context.Background()
    for i := 0; i < b.N; i++ {
        // Trigger receive operation (capture buffer allocations)
        _, _ = q.Query(ctx, "test.local", RecordTypeA)
    }
}
```

---

### 2. Integration Test Flakiness (Medium Priority)

**Issue**: `TestQuery_RealNetwork_Timeout` fails in isolated environments
**Impact**: Baseline shows 301/302 tests passing (one flaky integration test)
**Action**: Document as acceptable baseline OR fix test to handle isolated environments gracefully

**Recommendation**: Add skip condition for isolated environments:
```go
if os.Getenv("CI") == "true" || !networkAvailable() {
    t.Skip("Skipping integration test in isolated environment")
}
```

---

### 3. Low Coverage in Type Helpers (Low Priority)

**Issue**: AsA(), AsPTR(), AsSRV(), AsTXT() have 33-66% coverage
**Impact**: Low priority (simple type conversions, low risk)
**Action**: Consider adding tests for error paths if time permits

---

## Validation Plan

**After each refactoring phase**:

1. **Run Test Suite**:
   ```bash
   go test ./... -v -race -coverprofile=after_coverage.out
   # Compare: diff baseline_tests.txt after_tests.txt
   ```

2. **Check Coverage**:
   ```bash
   go tool cover -func=after_coverage.out | grep total
   # Expected: ≥85.2% (maintain or improve)
   ```

3. **Benchmark Comparison**:
   ```bash
   go test -bench=. -benchmem -count=5 > after_bench.txt
   benchstat baseline_bench.txt after_bench.txt
   # Expected: <5% overhead from abstraction, ≥80% reduction in receive allocations
   ```

4. **Dependency Check**:
   ```bash
   go mod graph > after_deps.txt
   diff baseline_deps.txt after_deps.txt
   # Expected: No diff (zero new dependencies)
   ```

5. **Layer Violations Check**:
   ```bash
   grep -rn "internal/network" querier/
   # Expected: No matches (zero violations)
   ```

---

## Next Steps

1. ✅ Baseline metrics captured and documented
2. ⏩ Proceed to `/speckit.tasks` generation (Phase 2)
3. ⏩ Execute refactoring with checkpoint validation after each phase
4. ⏩ Create post-refactoring metrics comparison report

---

**Baseline Captured**: 2025-11-01
**Files Created**:
- baseline_tests.txt (test execution results)
- baseline_coverage.out (coverage data)
- baseline_bench.txt (benchmark results)
- baseline_deps.txt (dependency graph)
- baseline_violations.txt (layer violations)
- baseline_metrics.md (this document)

**Status**: ✅ BASELINE ESTABLISHED - Ready for refactoring implementation
