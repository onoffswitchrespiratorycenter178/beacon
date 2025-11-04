# Implementation Plan: M1 Architectural Alignment and Refactoring

**Branch**: `003-m1-refactoring` | **Date**: 2025-11-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-m1-refactoring/spec.md`
**Context**: Post-M1 refactoring to address critical technical debt before M1.1 implementation

---

## Summary

This milestone performs surgical refactoring of M1 (Basic mDNS Querier) to address 4 critical P0 architectural issues identified in comprehensive post-implementation analysis. The refactoring creates a clean, spec-compliant foundation for M1.1 Transport Layer implementation while preserving all M1 functionality (zero regression requirement).

**Primary Goals**:
1. Create Transport interface abstraction (enables IPv6, improves testability)
2. Fix layer boundary violations (align with F-2 Package Structure)
3. Implement buffer pooling (eliminate hot path allocations)
4. Fix error propagation in cleanup (enable resource leak detection)

**Approach**: Surgical, test-validated refactoring following existing M1 patterns. Each FR is independently testable with full M1 test suite validation (107 tests). Total effort: 14.5 hours implementation + 2 hours validation = 16.5 hours.

---

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Standard library only (maintaining M1 constraint)
**Storage**: N/A (in-memory only)
**Testing**: Go testing framework (`go test`), existing 107 M1 tests + new refactoring tests
**Target Platform**: Linux, macOS, Windows (M1 baseline)
**Project Type**: Library refactoring (no new features, architectural improvements only)
**Performance Goals**:
- ≥80% reduction in allocations (FR-003: buffer pooling)
- <5% overhead from Transport abstraction (interface indirection)
- Zero performance regression in query processing (<100ms requirement maintained)

**Constraints**:
- **Zero Regression**: All 107 M1 tests must pass after each FR
- **Coverage Maintenance**: ≥85% test coverage (M1 baseline: 85.9%)
- **No Breaking Changes**: Public API (`beacon/querier/`) unchanged
- **Stdlib Only**: No new external dependencies
- **Surgical Only**: Refactor only P0 issues (P1/P2 deferred)
- **M1.1 Alignment**: Changes must support F-9/F-10/F-11 future requirements

**Scale/Scope**:
- Codebase: 3,764 LOC implementation + 4,330 LOC tests
- Refactoring: 4 critical P0 issues (14.5h effort)
- Test Coverage: 107 existing tests + ~10 new refactoring tests
- Packages Modified: `internal/transport/` (new), `querier/` (modified), `internal/network/` (deprecated)

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: RFC Compliant ✅ PASS

**Status**: No RFC changes in refactoring
- Refactoring preserves all M1 RFC 6762/1035/2782 compliance
- No protocol-level changes
- All 107 M1 tests (including RFC compliance tests) must pass

**Validation**: RFC compliance tests continue to pass (T043-T044 from M1)

---

### Principle II: Spec-Driven Development ✅ PASS

**Status**: Formal specification created
- spec.md defines 4 FRs (FR-001 through FR-004) for P0 issues
- plan.md (this document) provides implementation strategy
- tasks.md will provide surgical task breakdown

**Validation**: Specification exists at `specs/003-m1-refactoring/spec.md`

---

### Principle III: Test-Driven Development ✅ PASS

**Status**: Test-first refactoring approach
- All 107 M1 tests serve as regression tests (must pass after each FR)
- New tests written FIRST for refactored components (Transport interface, buffer pooling)
- TDD cycle: Existing tests RED (modify code) → Refactor → Existing tests GREEN

**Validation**:
```bash
# After each FR implementation
go test ./... -v -race -coverprofile=coverage.out
# Expected: 107/107 PASS, 0 race conditions, coverage ≥85%
```

---

### Principle IV: Phased Approach ✅ PASS

**Status**: 4-phase surgical refactoring
- Phase 0: Preparation (baselines, metrics capture)
- Phase 1: Transport Interface Abstraction (FR-001, FR-002)
- Phase 2: Buffer Pooling (FR-003)
- Phase 3: Error Handling Cleanup (FR-004)
- Phase 4: Validation and Documentation

Each phase independently tested and validated before proceeding.

---

### Principle V: Dependencies and Supply Chain ✅ PASS

**Status**: No new dependencies
- Refactoring uses stdlib only (maintains M1 constraint)
- No golang.org/x/* additions (M1 already justified these)
- No third-party dependencies

**Validation**: `go mod tidy` shows no new dependencies

---

### Principle VI: Open Source ✅ PASS

**Status**: Transparent refactoring process
- Public refactoring analysis (docs/M1_REFACTORING_ANALYSIS.md)
- Public specification and plan
- Git history preserved (no squash/rebase)

---

### Principle VII: Maintained ✅ PASS

**Status**: Strengthens long-term maintainability
- Transport interface enables future IPv6 support (M1.2+)
- Clean layer boundaries reduce coupling
- Buffer pooling improves performance sustainability
- Error propagation enables production monitoring

---

### Principle VIII: Excellence ✅ PASS

**Status**: Addresses technical debt systematically
- 6-agent refactoring analysis identified 74 issues
- P0 critical issues addressed in this milestone
- P1/P2 issues documented for future milestones
- Benchmark-validated improvements

**Validation**: Benchmark results show measurable improvements

---

### Constitution Check Summary

**Result**: ✅ **ALL GATES PASS**

No constitutional violations. Refactoring strengthens architectural integrity and prepares for M1.1 Transport Layer implementation.

---

## Project Structure

### Documentation (this feature)

```text
specs/003-m1-refactoring/
├── spec.md              # ✅ Feature specification (4 FRs defined)
├── plan.md              # ✅ This file (implementation strategy)
├── research.md          # ⏩ Phase 0 output (architectural patterns)
├── data-model.md        # N/A (refactoring, no new data models)
├── contracts/           # N/A (refactoring, no API changes)
├── quickstart.md        # N/A (refactoring, no user-facing changes)
└── tasks.md             # ⏩ Phase 2 output (surgical refactoring tasks)
```

**Note**: data-model.md, contracts/, quickstart.md skipped for refactoring milestone (no new features or public API changes).

---

### Source Code (repository root)

**Current M1 Structure** (before refactoring):
```text
beacon/querier/          # Public API (unchanged in refactoring)
├── querier.go           # Modified: Use Transport interface
├── records.go           # Unchanged
├── options.go           # Unchanged
└── doc.go               # Unchanged

internal/
├── errors/              # Unchanged
│   └── errors.go
├── message/             # Unchanged
│   ├── builder.go
│   ├── parser.go
│   └── name.go
├── protocol/            # Unchanged
│   ├── mdns.go
│   ├── validator.go
│   └── validator_test.go
└── network/             # DEPRECATED (migrated to transport/)
    ├── socket.go        # ⚠️ Migrated to internal/transport/udp.go
    └── socket_test.go   # ⚠️ Migrated to internal/transport/udp_test.go

tests/                   # Unchanged (all 107 tests preserved)
├── contract/
├── integration/
└── fuzz/
```

**Target Structure** (after refactoring):
```text
beacon/querier/          # Public API (Transport interface usage)
├── querier.go           # ✏️ Modified: q.transport field, Transport methods
├── records.go           # ✅ Unchanged
├── options.go           # ✅ Unchanged
└── doc.go               # ✅ Unchanged

internal/
├── errors/              # ✅ Unchanged
│   └── errors.go
├── message/             # ✅ Unchanged
│   ├── builder.go
│   ├── parser.go
│   └── name.go
├── protocol/            # ✅ Unchanged
│   ├── mdns.go
│   ├── validator.go
│   └── validator_test.go
├── transport/           # ✨ NEW PACKAGE (FR-001)
│   ├── transport.go     # ✨ Transport interface definition
│   ├── udp.go           # ✨ UDPv4Transport implementation (migrated from network/socket.go)
│   ├── udp_test.go      # ✨ Tests for UDPv4Transport
│   ├── mock.go          # ✨ MockTransport for testing
│   └── buffer_pool.go   # ✨ Buffer pooling (FR-003)
└── network/             # ⚠️ DEPRECATED (contents moved to transport/)
    ├── socket.go        # ⚠️ Deprecated (use internal/transport instead)
    └── socket_test.go   # ⚠️ Tests migrated to transport package

tests/                   # ✅ Unchanged (all 107 tests preserved)
├── contract/
│   ├── api_test.go      # ✅ All tests must pass (regression validation)
│   ├── rfc_test.go      # ✅ RFC compliance maintained
│   └── error_handling_test.go
├── integration/
│   └── query_test.go    # ✅ Integration tests validate end-to-end
└── fuzz/
    └── parser_fuzz_test.go  # ✅ Fuzz tests continue to pass
```

**Structure Decision**:
- Create new `internal/transport/` package for Transport abstraction
- Migrate `internal/network/socket.go` → `internal/transport/udp.go`
- Deprecate `internal/network/` package (contents moved, not deleted for git history)
- No changes to public API (`beacon/querier/` signatures preserved)
- All existing tests preserved (zero deletion, only additions for new Transport tests)

---

## Complexity Tracking

> **No constitutional violations** - This section is empty as all Constitution Check gates passed.

---

## Phase 0: Research & Architectural Patterns

**Goal**: Document architectural patterns for refactoring and validate approach against F-series specifications

**Duration**: 1 hour

### Research Topics

#### Topic 1: Transport Interface Design Pattern (FR-001)

**Question**: What is the optimal Transport interface design that supports M1 refactoring AND M1.1 F-9 requirements?

**Research Scope**:
- Review F-9 REQ-F9-1 (ListenConfig pattern) - lines 83-120
- Review F-9 REQ-F9-7 (Context propagation) - lines 328-422
- Review Go net package interface patterns
- Review existing UDP socket operations in M1

**Deliverable**: Interface design with context-aware methods

**Expected Findings**:
```go
// Transport interface must support:
// 1. Context propagation (F-9 REQ-F9-7)
// 2. Platform-specific socket configuration (F-9 REQ-F9-1 future)
// 3. Testing via MockTransport
type Transport interface {
    Send(ctx context.Context, packet []byte, dest net.Addr) error
    Receive(ctx context.Context) ([]byte, net.Addr, error)
    Close() error
}
```

**References**:
- F-9 Transport Layer Socket Configuration
- docs/M1_SPEC_ALIGNMENT_CRITICAL.md (lines 80-140)
- M1 internal/network/socket.go (current implementation)

---

#### Topic 2: Buffer Pooling Pattern (FR-003)

**Question**: What is the correct sync.Pool implementation for UDP receive buffers that prevents leaks and improves performance?

**Research Scope**:
- Review F-7 Resource Management (Buffer Pooling Pattern, lines 286-311)
- Review Go sync.Pool documentation and best practices
- Analyze buffer ownership semantics (who owns the buffer after Receive()?)
- Benchmark allocation patterns before/after

**Deliverable**: Buffer pool implementation pattern with ownership rules

**Expected Findings**:
```go
// Pattern: Pool owns buffer, caller owns copy
var bufferPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 9000)
        return &buf
    },
}

func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    bufPtr := bufferPool.Get().(*[]byte)
    defer bufferPool.Put(bufPtr) // ✅ Always returned to pool

    buffer := *bufPtr
    n, srcAddr, err := t.conn.ReadFrom(buffer)

    // ✅ Copy to new buffer (caller owns result)
    result := make([]byte, n)
    copy(result, buffer[:n])
    return result, srcAddr, nil
}
```

**References**:
- F-7 Resource Management (lines 286-311)
- docs/M1_REFACTORING_ANALYSIS.md P0-3 (lines 110-159)
- Go sync.Pool documentation

---

#### Topic 3: Layer Boundary Compliance (FR-002)

**Question**: How should querier access Transport to maintain F-2 layer boundaries?

**Research Scope**:
- Review F-2 Package Structure (Layer boundaries)
- Review current querier imports and dependencies
- Determine if protocol layer should orchestrate transport or if querier can use transport directly

**Deliverable**: Import and dependency pattern

**Expected Findings**:
```go
// querier/querier.go
import (
    "github.com/joshuafuller/beacon/internal/transport"  // ✅ Direct import OK
    // ❌ NO internal/network import
)

type Querier struct {
    transport transport.Transport  // ✅ Interface field
    // ...
}

func New(opts ...Option) (*Querier, error) {
    // Create concrete transport (UDPv4Transport)
    transport, err := transport.NewUDPv4Transport()
    if err != nil {
        return nil, err
    }

    q := &Querier{
        transport: transport,  // ✅ Use interface
        // ...
    }
    return q, nil
}
```

**Rationale**: Public API layer can import Transport package directly. Protocol layer remains independent of transport (uses interfaces).

**References**:
- F-2 Package Structure
- docs/M1_REFACTORING_ANALYSIS.md P0-2 (lines 69-107)

---

#### Topic 4: Error Propagation Pattern (FR-004)

**Question**: What is the correct error handling pattern for CloseSocket that aligns with F-3 RULE-1?

**Research Scope**:
- Review F-3 Error Handling (RULE-1: Return errors to caller)
- Review F-7 Cleanup patterns (lines 218-285)
- Analyze Querier.Close() error handling (querier.go:326-343)

**Deliverable**: Updated CloseSocket implementation

**Expected Findings**:
```go
// ❌ BEFORE (M1 - swallows errors)
func CloseSocket(conn net.PacketConn) error {
    err := conn.Close()
    if err != nil {
        return nil  // ❌ ERROR SWALLOWED
    }
    return nil
}

// ✅ AFTER (Refactored - propagates errors)
func (t *UDPv4Transport) Close() error {
    if t.conn == nil {
        return nil  // Graceful nil handling OK
    }

    err := t.conn.Close()
    if err != nil {
        return &errors.NetworkError{
            Operation: "close socket",
            Err:       err,
            Details:   "failed to close UDP connection",
        }
    }
    return nil
}
```

**References**:
- F-3 Error Handling (RULE-1)
- docs/M1_REFACTORING_ANALYSIS.md P0-4 (lines 162-229)

---

#### Topic 5: M1.1 Alignment Validation (Cross-Cutting)

**Question**: Does this refactoring create the correct foundation for M1.1 F-9/F-10/F-11 requirements?

**Research Scope**:
- Review F-9 REQ-F9-1 (ListenConfig pattern requires Transport abstraction)
- Review F-9 REQ-F9-7 (Context propagation in Transport.Receive())
- Validate Transport interface signature compatibility with F-9 requirements
- Confirm no M1.1 rework needed after this refactoring

**Deliverable**: Validation report confirming M1.1 compatibility

**Expected Findings**:
- ✅ Transport interface signature matches F-9 REQ-F9-7 (context-aware)
- ✅ UDPv4Transport can be extended for platform-specific socket options (F-9 REQ-F9-2)
- ✅ Interface design supports IPv6 (future UDPv6Transport implementation)
- ✅ No breaking changes needed for M1.1

**References**:
- F-9 Transport Layer Socket Configuration
- docs/M1_SPEC_ALIGNMENT_CRITICAL.md
- docs/M1.1_PLANNING_COMPLETE.md

---

### Research Deliverables

**File**: `specs/003-m1-refactoring/research.md`

**Contents**:
1. **Topic 1**: Transport interface design (with code examples)
2. **Topic 2**: Buffer pooling pattern (with benchmark plan)
3. **Topic 3**: Layer boundary compliance (with import strategy)
4. **Topic 4**: Error propagation pattern (with before/after comparison)
5. **Topic 5**: M1.1 alignment validation (with compatibility checklist)

**Format** (per research.md template):
```markdown
## Topic X: [Name]

**Decision**: [What was chosen]
**Rationale**: [Why chosen]
**Alternatives Considered**: [What else evaluated]
**Implementation Notes**: [Key details for tasks.md generation]
**References**: [F-specs, docs, RFCs]
```

---

## Phase 1: Design & Validation

**Goal**: Validate architectural approach and create measurement baselines

**Duration**: 1 hour

### Baseline Metrics Capture

**Purpose**: Establish before/after comparison for validation

**Tasks**:
1. **Capture Test Baseline**:
   ```bash
   go test ./... -v -race -coverprofile=baseline_coverage.out > baseline_tests.txt
   # Expected: 107/107 PASS, 85.9% coverage
   ```

2. **Capture Performance Baseline**:
   ```bash
   go test -bench=BenchmarkQuery -benchmem -count=5 > baseline_bench.txt
   # Capture allocations per operation (receive path)
   ```

3. **Capture Dependency Graph**:
   ```bash
   go mod graph > baseline_deps.txt
   # Validate no new dependencies after refactoring
   ```

4. **Analyze Layer Violations**:
   ```bash
   # Document current violations
   grep -r "internal/network" querier/ > baseline_violations.txt
   # Expected: 2 instances (querier.go imports)
   ```

**Deliverables**:
- `baseline_tests.txt` (107 tests passing)
- `baseline_coverage.out` (85.9% coverage)
- `baseline_bench.txt` (allocations before buffer pooling)
- `baseline_deps.txt` (dependency snapshot)
- `baseline_violations.txt` (layer violations to fix)

---

### Architecture Validation

**Purpose**: Validate Transport interface design against F-series specs

**Validation Checklist**:

- [ ] **FR-001**: Transport interface design reviewed
  - [ ] Signature matches F-9 REQ-F9-7 (context-aware methods)
  - [ ] Methods support M1 operations (Send, Receive, Close)
  - [ ] Interface enables future IPv6 (extensible design)
  - [ ] MockTransport design supports testing

- [ ] **FR-002**: Layer boundary strategy validated
  - [ ] Querier→Transport import path acceptable per F-2
  - [ ] Protocol layer remains transport-agnostic
  - [ ] No circular dependencies created

- [ ] **FR-003**: Buffer pooling approach validated
  - [ ] sync.Pool pattern correct per F-7
  - [ ] Buffer ownership semantics clear (pool→caller)
  - [ ] defer pattern ensures zero leaks

- [ ] **FR-004**: Error propagation approach validated
  - [ ] CloseSocket propagates errors per F-3 RULE-1
  - [ ] Error types align with existing NetworkError

**Deliverable**: Architecture validation report (Phase 1 section of research.md)

---

### Data Model

**Status**: N/A - Refactoring milestone, no new data models

**Rationale**: This is an architectural refactoring milestone. No new entities, records, or data structures are introduced. Existing M1 data models (ResourceRecord, Response, DNSMessage) are unchanged.

---

### Contracts

**Status**: N/A - Refactoring milestone, no public API changes

**Rationale**: The refactoring is internal-only. Public API (`beacon/querier/` package) signatures are preserved. No new endpoints, methods, or user-facing contracts.

**Validation**: All 107 M1 tests continue to pass (contract validation)

---

### Quickstart

**Status**: N/A - Refactoring milestone, no user-facing changes

**Rationale**: Users interact with beacon/querier package, which has unchanged API. No new quickstart examples needed. Existing M1 examples continue to work.

---

### Phase 1 Deliverables

1. ✅ Baseline metrics captured (tests, coverage, benchmarks, dependencies, violations)
2. ✅ Architecture validation report (in research.md Phase 1 section)
3. ⏩ data-model.md (skipped - no new models)
4. ⏩ contracts/ (skipped - no API changes)
5. ⏩ quickstart.md (skipped - no user changes)

---

## Phase 2: Task Generation Strategy

**Goal**: Create surgical, test-validated refactoring tasks

**Approach**: Generate tasks using `/speckit.tasks` command with refactoring-specific structure

### Task Organization

**Phase-Based Structure** (not user-story based):

```markdown
## Phase 0: Preparation (1 hour)
- [ ] T001 Create refactoring branch (003-m1-refactoring)
- [ ] T002 Capture baseline metrics (tests, coverage, benchmarks)
- [ ] T003 Analyze dependency graph (validate no new deps)
- [ ] T004 Document current layer violations

## Phase 1: Transport Interface Abstraction (8 hours) - FR-001, FR-002
- [ ] T005 Create internal/transport/ package structure
- [ ] T006 Define Transport interface in transport.go
- [ ] T007 Implement UDPv4Transport (migrate socket.go logic)
- [ ] T008 Create MockTransport for testing
- [ ] T009 Add Transport interface tests (contract tests)
- [ ] T010 Update Querier to use Transport interface
- [ ] T011 Remove internal/network imports from querier
- [ ] T012 Validate all M1 tests pass (107/107)
- [ ] T013 Validate layer boundaries (grep analysis)

## Phase 2: Buffer Pooling (2 hours) - FR-003
- [ ] T014 Create buffer pool in transport/buffer_pool.go
- [ ] T015 Update UDPv4Transport.Receive() to use pooled buffers
- [ ] T016 Add buffer pool tests (leak detection)
- [ ] T017 Benchmark allocation improvements
- [ ] T018 Validate all M1 tests pass (107/107)

## Phase 3: Error Handling Cleanup (0.5 hours) - FR-004
- [ ] T019 Update UDPv4Transport.Close() to propagate errors
- [ ] T020 Add test for close error propagation
- [ ] T021 Validate all M1 tests pass (107/107)

## Phase 4: Validation and Documentation (2 hours)
- [ ] T022 Run full test suite with race detector
- [ ] T023 Validate coverage ≥85% (go tool cover)
- [ ] T024 Compare benchmarks (before/after buffer pooling)
- [ ] T025 Validate zero layer violations (grep analysis)
- [ ] T026 Update godoc comments (Transport interface)
- [ ] T027 Create ADR for Transport abstraction
- [ ] T028 Update CHANGELOG.md with refactoring summary
- [ ] T029 Create refactoring completion report
```

**Key Principles**:
- Each task independently testable
- Checkpoint after each phase (validate all 107 tests pass)
- No task breaks M1 functionality
- Explicit validation tasks (not implicit)

---

### TDD Workflow for Refactoring

**Pattern**: Existing tests serve as regression tests

1. **BASELINE**: Capture current state (all 107 tests passing)
2. **REFACTOR**: Modify code (e.g., introduce Transport interface)
3. **VALIDATE**: Run all 107 tests (must all pass)
4. **ENHANCE**: Add new tests for refactored components (Transport tests)
5. **CHECKPOINT**: Validate coverage maintained (≥85%)

**Example** (FR-001 Transport Interface):
```bash
# BASELINE
go test ./... -v > before_transport.txt
# Expected: 107/107 PASS

# REFACTOR: Create Transport interface and update Querier
# (T005-T011 tasks)

# VALIDATE
go test ./... -v > after_transport.txt
# Expected: 107/107 PASS (no regression)

# ENHANCE: Add new Transport interface tests
# (T009 task)

# CHECKPOINT
go test -coverprofile=coverage.out ./...
# Expected: coverage ≥85%
```

---

## Phase 3: Implementation Checkpoints

**Goal**: Define validation gates between refactoring phases

### Checkpoint 1: After Transport Interface (Phase 1)

**Criteria**:
- [ ] All 107 M1 tests pass (`go test ./... -v -race`)
- [ ] Coverage ≥85% (`go tool cover -func=coverage.out`)
- [ ] Zero layer violations (`grep -r "internal/network" querier/` → no matches)
- [ ] Transport interface tests pass (new tests for UDPv4Transport, MockTransport)
- [ ] No new dependencies (`go mod tidy` → no changes)

**Validation Command**:
```bash
go test ./... -v -race -coverprofile=coverage.out && \
go tool cover -func=coverage.out | grep total && \
grep -r "internal/network" querier/ && \
echo "Checkpoint 1: PASS"
```

**If Failed**: Rollback Transport interface changes, debug test failures, fix violations before proceeding.

---

### Checkpoint 2: After Buffer Pooling (Phase 2)

**Criteria**:
- [ ] All 107 M1 tests pass (no regression from pooling)
- [ ] Coverage ≥85%
- [ ] Benchmark shows ≥80% allocation reduction
- [ ] Buffer pool tests pass (leak detection validated)
- [ ] Zero buffer leaks detected (defer pattern correct)

**Validation Command**:
```bash
go test ./... -v -race -coverprofile=coverage.out && \
go test -bench=BenchmarkReceive -benchmem -count=5 > after_pooling.txt && \
echo "Compare baseline_bench.txt vs after_pooling.txt" && \
echo "Checkpoint 2: PASS"
```

**Benchmark Validation**:
```bash
# Before buffer pooling (baseline)
BenchmarkReceive-8    10000    100000 ns/op    9000 B/op    2 allocs/op

# After buffer pooling (target)
BenchmarkReceive-8    10000    100000 ns/op    1800 B/op    1 allocs/op
#                                               ^^^^^ ≥80% reduction
```

**If Failed**: Rollback buffer pooling, debug allocation issues, validate defer pattern.

---

### Checkpoint 3: After Error Handling (Phase 3)

**Criteria**:
- [ ] All 107 M1 tests pass
- [ ] Coverage ≥85%
- [ ] Close error propagation test passes
- [ ] F-3 RULE-1 compliance validated (no error swallowing)

**Validation Command**:
```bash
go test ./... -v -race -coverprofile=coverage.out && \
go test -run TestTransportClose_ErrorPropagation && \
echo "Checkpoint 3: PASS"
```

**If Failed**: Rollback error handling changes, fix error propagation test.

---

### Final Validation: After Phase 4

**Criteria**:
- [ ] All 107 M1 tests pass
- [ ] Coverage ≥85%
- [ ] Benchmark improvements documented (≥80% allocation reduction)
- [ ] Zero layer violations validated
- [ ] Documentation updated (godoc, ADR, CHANGELOG)
- [ ] Refactoring completion report created

**Validation Command**:
```bash
# Full validation suite
go test ./... -v -race -coverprofile=coverage.out && \
go tool cover -func=coverage.out | grep total && \
go test -bench=. -benchmem && \
grep -r "internal/network" querier/ && \
go vet ./... && \
gofmt -l . && \
echo "Final Validation: PASS"
```

**Deliverable**: Refactoring completion report with before/after metrics

---

## Risk Mitigation

### Risk 1: Regression in M1 Functionality

**Probability**: Medium
**Impact**: Critical (breaks production-ready M1)

**Mitigation**:
- Run full test suite after EVERY task (not just at checkpoints)
- Use git branches for each phase (easy rollback)
- Keep baseline metrics for comparison
- Automated CI validation on every commit

**Rollback Plan**:
```bash
# If any checkpoint fails
git stash  # Save current work
git checkout main  # Return to stable baseline
git checkout -b 003-m1-refactoring-retry  # New attempt branch
# Review failure, fix issue, retry phase
```

---

### Risk 2: Performance Degradation from Abstraction

**Probability**: Low
**Impact**: Medium (violates performance requirements)

**Mitigation**:
- Benchmark before/after (validate <5% overhead from interface)
- Interface method calls are inlinable by Go compiler
- Buffer pooling (FR-003) offsets any abstraction cost
- Continuous benchmark monitoring

**Validation**:
```bash
# Before refactoring
go test -bench=BenchmarkQuery -benchmem -count=10 > before.txt

# After refactoring
go test -bench=BenchmarkQuery -benchmem -count=10 > after.txt

# Compare
benchstat before.txt after.txt
# Expected: <5% performance change, or improvement from buffer pooling
```

---

### Risk 3: Test Coverage Reduction

**Probability**: Low
**Impact**: Medium (violates NFR-002)

**Mitigation**:
- Add tests for new Transport interface (maintains coverage)
- Add tests for buffer pooling (maintains coverage)
- Add test for CloseSocket error propagation (maintains coverage)
- Validate coverage after each phase

**Validation**:
```bash
go tool cover -func=coverage.out | grep total
# Expected: ≥85.0% (M1 baseline: 85.9%)
```

---

## Success Criteria Validation

### SC-001: Zero Regression ✅

**Validation**: All 107 M1 tests pass after refactoring
```bash
go test ./... -v -race
# Expected: 107/107 PASS, 0 race conditions
```

---

### SC-002: Transport Interface Enables IPv6 ✅

**Validation**: Mock IPv6 Transport implementation compiles and tests pass
```bash
# Create test UDPv6Transport (not for M1, validation only)
go test -run TestTransportInterface_SupportsIPv6Mock
# Expected: PASS (validates interface design)
```

---

### SC-003: Buffer Pooling Reduces Allocations ✅

**Validation**: Benchmark shows ≥80% reduction
```bash
benchstat baseline_bench.txt after_pooling.txt
# Expected: Allocations reduced by ≥80%
```

---

### SC-004: Layer Boundaries Compliant ✅

**Validation**: Zero F-2 violations detected
```bash
grep -r "internal/network" querier/
# Expected: No matches (querier uses internal/transport instead)

grep -r "internal/transport" internal/protocol/
# Expected: No matches (protocol remains transport-agnostic)
```

---

### SC-005: Error Propagation Enables Leak Detection ✅

**Validation**: Close error test passes
```bash
go test -run TestTransportClose_ErrorPropagation
# Expected: PASS (validates NetworkError returned on close failure)
```

---

### SC-006: Coverage Maintained ≥85% ✅

**Validation**: Coverage report shows ≥85%
```bash
go tool cover -func=coverage.out | grep total
# Expected: total: (statements) ≥85.0%
```

---

### SC-007: Complete in ≤16 hours ✅

**Validation**: Task effort totals
- Phase 0: 1h (preparation)
- Phase 1: 8h (Transport interface)
- Phase 2: 2h (Buffer pooling)
- Phase 3: 0.5h (Error handling)
- Phase 4: 2h (Validation)
- **Total**: 13.5h implementation + 2h validation = 15.5h ✅

---

## Deliverables

### Phase 0: Research
- [x] research.md (architectural patterns documented)

### Phase 1: Design & Validation
- [x] Baseline metrics captured
- [x] Architecture validation report
- [x] data-model.md (skipped - refactoring)
- [x] contracts/ (skipped - refactoring)
- [x] quickstart.md (skipped - refactoring)

### Phase 2: Implementation
- [ ] tasks.md (surgical refactoring tasks - generated by `/speckit.tasks`)
- [ ] All tasks executed and validated
- [ ] All checkpoints passed

### Phase 3: Documentation
- [ ] Transport interface godoc
- [ ] ADR: Transport Abstraction Decision
- [ ] CHANGELOG.md updated
- [ ] Refactoring completion report (before/after metrics)

---

## Next Steps

1. **Complete research.md** (Phase 0 - 1 hour)
   - Document Transport interface design
   - Document buffer pooling pattern
   - Document layer boundary strategy
   - Document error propagation approach
   - Validate M1.1 alignment

2. **Capture baselines** (Phase 1 - 1 hour)
   - Run test suite and save results
   - Run benchmarks and save results
   - Analyze dependencies
   - Document layer violations

3. **Generate tasks.md** (Phase 2)
   - Run `/speckit.tasks` command
   - Review generated surgical tasks
   - Validate task dependencies and checkpoints

4. **Execute refactoring** (Phases 1-3 - 10.5 hours)
   - Follow tasks.md order
   - Validate at each checkpoint
   - Document any deviations

5. **Final validation** (Phase 4 - 2 hours)
   - Run full validation suite
   - Document improvements
   - Update project documentation

---

## References

### Analysis Documents
- docs/M1_REFACTORING_ANALYSIS.md (74 issues, P0 details)
- docs/M1_SPEC_ALIGNMENT_CRITICAL.md (specification gaps)
- docs/M1_ANALYSIS_COMPLETE_SUMMARY.md (executive summary)
- docs/CONTEXT_AND_LOGGING_REVIEW.md (context propagation mandate)

### Specifications
- F-2: Package Structure and Dependencies
- F-3: Error Handling Strategy
- F-6: Logging & Observability
- F-7: Resource Management
- F-9: Transport Layer Socket Configuration (M1.1 requirements)

### Milestones
- M1 (002-mdns-querier): ✅ COMPLETE - Baseline for refactoring
- M1.1 (004-transport-layer): ⏳ PENDING - Requires this refactoring

### RFC Standards
- RFC 6762: Multicast DNS (no changes)
- RFC 1035: DNS Specification (no changes)
- RFC 2782: DNS SRV Records (no changes)

---

**Plan Status**: ✅ READY FOR RESEARCH PHASE
**Next Command**: Begin research.md generation (Phase 0)
**Estimated Completion**: 15.5 hours from task start
**Constitutional Compliance**: ✅ ALL GATES PASS

---

**Plan Created**: 2025-11-01
**Plan Version**: 1.0
**Status**: Ready for Phase 0 Research
