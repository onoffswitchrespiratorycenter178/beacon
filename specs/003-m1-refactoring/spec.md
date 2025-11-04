# Feature Specification: M1 Architectural Alignment and Refactoring

**Feature ID**: 003-m1-refactoring
**Milestone**: M1-Refactoring (Post-M1, Pre-M1.1)
**Status**: Specification Phase
**Priority**: P0 - CRITICAL (Blocks M1.1 Implementation)
**Created**: 2025-11-01

---

## Overview

### Purpose

Architectural alignment and refactoring of M1 Basic mDNS Querier to address critical technical debt identified in post-implementation analysis. This milestone creates a clean, spec-compliant foundation for M1.1 Transport Layer implementation while preserving all M1 functionality.

### Background

M1 (Basic mDNS Querier) successfully delivered a fully functional, well-tested implementation (107/107 tasks complete, 85.9% coverage). However, post-implementation analysis identified 4 critical architectural gaps that must be addressed before M1.1:

1. **No Transport Interface Abstraction** - Prevents IPv6 support, makes testing harder
2. **Layer Boundary Violations** - Querier bypasses protocol layer, violates F-2 architecture
3. **Buffer Allocation in Hot Path** - 9KB allocation per packet creates GC pressure
4. **Error Swallowing in Cleanup** - CloseSocket prevents resource leak detection

These issues were identified through comprehensive 6-agent refactoring analysis (docs/M1_REFACTORING_ANALYSIS.md) totaling 110 hours of findings. The P0 issues (14 hours effort) naturally align with M1.1 Transport Layer requirements (F-9, F-10, F-11), making this the optimal intervention point.

### Scope

**In Scope**:
- ✅ Address 4 P0 critical issues identified in refactoring analysis
- ✅ Align M1 implementation with F-2 Package Structure specification
- ✅ Create Transport interface abstraction (foundation for M1.1)
- ✅ Implement buffer pooling per F-7 Resource Management
- ✅ Fix error handling per F-3 Error Handling Strategy
- ✅ Preserve all M1 functionality (zero regression)

**Out of Scope**:
- ❌ P1/P2 refactoring issues (32 high-priority + 38 medium-priority items deferred)
- ❌ New features from F-9/F-10/F-11 (socket configuration, interface management, security)
- ❌ IPv6 implementation (Transport interface created but UDPv6Transport deferred to M1.1+)
- ❌ Continuous query support (deferred to future milestone)
- ❌ Query result caching (deferred to future milestone)

### Success Criteria

**SC-001**: All 107 M1 tests pass after refactoring (zero regression)
**SC-002**: Transport interface enables future IPv6 support (validated via mock implementation)
**SC-003**: Buffer pooling reduces allocations by ≥80% (benchmark validated)
**SC-004**: Layer boundaries comply with F-2 specification (no protocol→transport imports)
**SC-005**: Error propagation enables resource leak detection (CloseSocket test validates)
**SC-006**: Test coverage maintained ≥85% (no coverage loss from refactoring)
**SC-007**: All refactoring tasks complete in ≤16 hours (14h P0 fixes + 2h validation)

---

## User Stories

### US1: As a Library Maintainer, I need Transport abstraction to enable future IPv6 support

**Goal**: Create Transport interface that decouples network operations from concrete UDP implementations

**Acceptance Criteria**:
- Transport interface defined with Send/Receive/Close methods
- UDPv4Transport implements Transport interface
- MockTransport exists for testing
- Querier uses Transport interface (not concrete socket)
- All M1 tests pass with Transport abstraction

**Value**: Enables M1.1 to add platform-specific socket configuration and future IPv6 support without querier changes

---

### US2: As a Library Maintainer, I need clean layer boundaries to maintain architectural integrity

**Goal**: Ensure querier uses protocol layer, not direct network calls

**Acceptance Criteria**:
- Querier imports internal/transport (not internal/network)
- Protocol layer orchestrates transport operations
- No F-2 layer boundary violations detected
- Dependency graph validates stable→unstable flow

**Value**: Maintains spec-driven architecture, prevents technical debt accumulation

---

### US3: As a Library User, I need performant packet processing without GC pressure

**Goal**: Eliminate hot path allocations via buffer pooling

**Acceptance Criteria**:
- sync.Pool implemented for receive buffers
- Benchmark shows ≥80% reduction in allocations
- Zero buffer leaks detected (defer pattern validated)
- All M1 tests pass with buffer pooling

**Value**: High-throughput scenarios (100+ queries/sec) don't degrade due to GC pauses

---

### US4: As a Library User, I need reliable error reporting for resource cleanup failures

**Goal**: Ensure CloseSocket propagates errors to caller

**Acceptance Criteria**:
- CloseSocket returns NetworkError on close failure
- Querier.Close() handles CloseSocket errors
- Test validates error propagation
- F-3 RULE-1 compliance validated

**Value**: Applications can detect and log resource leaks in production

---

## Functional Requirements

### FR-001: Transport Interface Abstraction (P0-1)

**Requirement**: Create Transport interface that abstracts UDP socket operations

**Rationale**:
- M1 currently uses concrete `net.PacketConn` throughout querier
- Prevents IPv6 support (no way to swap UDP4↔UDP6)
- Makes testing difficult (requires real network)
- M1.1 F-9 requires platform-specific socket configuration (needs abstraction)

**Implementation**:
1. Create `internal/transport/` package
2. Define Transport interface:
   ```go
   type Transport interface {
       Send(ctx context.Context, packet []byte, dest net.Addr) error
       Receive(ctx context.Context) ([]byte, net.Addr, error)
       Close() error
   }
   ```
3. Implement UDPv4Transport (migrates current `internal/network/socket.go` logic)
4. Create MockTransport for testing
5. Update Querier to accept Transport interface (constructor parameter or field)

**Acceptance Criteria**:
- [ ] Transport interface defined in `internal/transport/transport.go`
- [ ] UDPv4Transport implements Transport interface
- [ ] UDPv4Transport includes all M1 socket logic (multicast join, buffer configuration)
- [ ] MockTransport exists for unit testing
- [ ] Querier uses Transport interface (not concrete net.PacketConn)
- [ ] All 107 M1 tests pass
- [ ] New tests validate Transport interface contract

**References**:
- docs/M1_REFACTORING_ANALYSIS.md (P0-1, lines 34-66)
- F-9 Transport Layer Socket Configuration (REQ-F9-1: ListenConfig pattern)
- F-2 Package Structure (Transport layer abstraction)

**Effort**: 8 hours

---

### FR-002: Layer Boundary Compliance (P0-2)

**Requirement**: Ensure querier accesses network via protocol layer, not directly

**Rationale**:
- M1 querier imports `internal/network` and calls socket functions directly
- Violates F-2 Package Structure (Public API → Service → Protocol → Transport)
- Creates tight coupling between service and transport layers
- Prevents protocol layer from orchestrating complex operations

**Current Violation** (querier/querier.go:184):
```go
// ❌ WRONG: Direct network import and call
import "github.com/joshuafuller/beacon/internal/network"

err = network.SendQuery(q.socket, queryMsg)
```

**Required Pattern**:
```go
// ✅ CORRECT: Through transport abstraction
import "github.com/joshuafuller/beacon/internal/transport"

err = q.transport.Send(ctx, queryMsg, dest)
```

**Implementation**:
1. Update querier to use Transport interface (from FR-001)
2. Remove direct `internal/network` imports from querier
3. Validate dependency graph (use `go mod graph` or tool)
4. Ensure F-2 layer flow: querier → protocol → transport

**Acceptance Criteria**:
- [ ] Querier does NOT import `internal/network`
- [ ] Querier uses Transport interface for all network operations
- [ ] Dependency analysis validates F-2 layer boundaries
- [ ] No protocol→transport imports detected (inverted dependency)
- [ ] All 107 M1 tests pass

**References**:
- docs/M1_REFACTORING_ANALYSIS.md (P0-2, lines 69-107)
- F-2 Package Structure (Dependency Direction)
- docs/M1_SPEC_ALIGNMENT_CRITICAL.md (Layer violation analysis)

**Effort**: 4 hours (naturally resolved by FR-001 Transport abstraction)

---

### FR-003: Buffer Pooling for Hot Path Optimization (P0-3)

**Requirement**: Implement sync.Pool for receive buffer reuse to eliminate hot path allocations

**Rationale**:
- M1 allocates 9KB buffer on every `ReceiveResponse()` call
- Hot path per F-6 Logging & Observability specification
- 100 queries/sec = 900KB/sec allocations = 54MB/min
- Forces frequent GC cycles, degrades high-throughput performance

**Current Issue** (internal/network/socket.go:132):
```go
func ReceiveResponse(conn net.PacketConn, timeout time.Duration) ([]byte, error) {
    buffer := make([]byte, 9000)  // ⚠️ HOT PATH ALLOCATION
    n, _, err := conn.ReadFrom(buffer)
    return buffer[:n], nil
}
```

**Required Pattern** (per F-7 Resource Management lines 286-311):
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 9000)
        return &buf
    },
}

func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    bufPtr := bufferPool.Get().(*[]byte)
    defer bufferPool.Put(bufPtr)

    buffer := *bufPtr
    n, srcAddr, err := t.conn.ReadFrom(buffer)
    if err != nil {
        return nil, nil, err
    }

    // Copy to return (caller owns memory)
    result := make([]byte, n)
    copy(result, buffer[:n])
    return result, srcAddr, nil
}
```

**Implementation**:
1. Create buffer pool in `internal/transport/buffer_pool.go`
2. Update Transport.Receive() to use pooled buffers
3. Ensure defer pattern (no buffer leaks)
4. Copy buffer contents before returning (caller owns memory)
5. Benchmark allocation reduction

**Acceptance Criteria**:
- [ ] sync.Pool implemented for 9KB receive buffers
- [ ] Transport.Receive() uses pooled buffers
- [ ] defer pattern ensures buffers returned to pool
- [ ] Benchmark shows ≥80% allocation reduction (pprof validation)
- [ ] Zero buffer leaks detected (no pooled buffers retained)
- [ ] All 107 M1 tests pass

**Benchmarking**:
```bash
# Before buffer pooling
go test -bench=BenchmarkReceive -memprofile=before.mem

# After buffer pooling
go test -bench=BenchmarkReceive -memprofile=after.mem

# Compare allocations
go tool pprof -base=before.mem after.mem
# Expected: ≥80% reduction in alloc_space
```

**References**:
- docs/M1_REFACTORING_ANALYSIS.md (P0-3, lines 110-159)
- F-7 Resource Management (Buffer Pooling Pattern, lines 286-311)
- F-6 Logging & Observability (Hot Path Definition)

**Effort**: 2 hours

---

### FR-004: Error Propagation in Cleanup Operations (P0-4)

**Requirement**: CloseSocket must propagate errors to caller, not swallow them

**Rationale**:
- M1 CloseSocket silently swallows close errors
- Violates F-3 RULE-1: "Return errors to caller"
- Prevents applications from detecting resource leaks
- Violates F-7 cleanup patterns (errors must be reportable)

**Current Issue** (internal/network/socket.go:166-179):
```go
func CloseSocket(conn net.PacketConn) error {
    if conn == nil {
        return nil
    }

    err := conn.Close()
    if err != nil {
        // In M1, we log but don't fail on close errors
        return nil  // ❌ ERROR SWALLOWED
    }

    return nil
}
```

**Required Fix**:
```go
func CloseSocket(conn net.PacketConn) error {
    if conn == nil {
        return nil  // Graceful nil handling OK
    }

    err := conn.Close()
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

**Implementation**:
1. Update CloseSocket to propagate errors (return NetworkError)
2. Verify Querier.Close() already handles errors (it does per querier.go:334-337)
3. Add test for close error propagation
4. Validate F-3 RULE-1 compliance

**Acceptance Criteria**:
- [ ] CloseSocket propagates close errors (no swallowing)
- [ ] Errors wrapped as NetworkError with context
- [ ] Querier.Close() handles CloseSocket errors correctly
- [ ] Test validates error propagation (mock socket with close failure)
- [ ] F-3 RULE-1 compliance validated
- [ ] All 107 M1 tests pass

**References**:
- docs/M1_REFACTORING_ANALYSIS.md (P0-4, lines 162-229)
- F-3 Error Handling (RULE-1: Return Errors to Caller)
- F-7 Resource Management (Cleanup Patterns, lines 218-285)

**Effort**: 0.5 hours

---

## Non-Functional Requirements

### NFR-001: Zero Regression (MANDATORY)

**Requirement**: All 107 M1 tests must pass after refactoring

**Validation**:
```bash
go test ./... -v -race -coverprofile=coverage.out
# Expected: 107/107 tests PASS, 0 race conditions
```

**Rationale**: M1 is production-ready. Refactoring must preserve all functionality.

---

### NFR-002: Coverage Maintenance (MANDATORY)

**Requirement**: Test coverage must remain ≥85% (M1 baseline: 85.9%)

**Validation**:
```bash
go tool cover -func=coverage.out | grep total
# Expected: total coverage ≥85.0%
```

**Rationale**: Refactoring should not reduce test quality. New code (Transport interface) requires tests.

---

### NFR-003: Performance Improvement (MANDATORY for FR-003)

**Requirement**: Buffer pooling must reduce allocations by ≥80%

**Validation**:
```bash
go test -bench=BenchmarkReceive -benchmem
# Before: ~9000 allocs per operation
# After:  ~1800 allocs per operation (80% reduction)
```

**Rationale**: Performance optimization must be measurable.

---

### NFR-004: Layer Boundary Validation (MANDATORY for FR-002)

**Requirement**: No F-2 layer violations after refactoring

**Validation**:
```bash
# Check imports in querier package
grep -r "internal/network" querier/
# Expected: No matches

# Check protocol package doesn't import transport
grep -r "internal/transport" internal/protocol/
# Expected: No matches (protocol uses interfaces, not concrete transport)
```

**Rationale**: Architectural integrity must be verifiable.

---

## Constraints

### Technical Constraints

**TC-001**: Must maintain Go 1.21+ compatibility (M1 baseline)
**TC-002**: Must use stdlib only (no new external dependencies)
**TC-003**: Must maintain RFC 6762/1035/2782 compliance (no protocol changes)
**TC-004**: Must complete in ≤16 hours (14h fixes + 2h validation)

### Implementation Constraints

**IC-001**: Refactoring must be surgical (no wholesale rewrites)
**IC-002**: Must preserve M1 public API (no breaking changes to querier package)
**IC-003**: Must maintain M1 git history (no squash/rebase)
**IC-004**: Each FR must be independently testable (can validate incrementally)

### Quality Constraints

**QC-001**: All M1 tests must pass at each checkpoint (no broken states)
**QC-002**: Code must pass `go vet` and `gofmt` (no new lint issues)
**QC-003**: Documentation must reflect architectural changes (godoc updates)
**QC-004**: Benchmark results must be documented (before/after comparison)

---

## Dependencies

### Specification Dependencies

**F-2**: Package Structure and Dependencies (Layer boundaries)
**F-3**: Error Handling Strategy (RULE-1: Return errors)
**F-6**: Logging & Observability (Hot path definitions)
**F-7**: Resource Management (Buffer pooling, cleanup patterns)
**F-9**: Transport Layer Socket Configuration (Context propagation, interface abstraction)

### Analysis Dependencies

**docs/M1_REFACTORING_ANALYSIS.md**: Comprehensive refactoring findings (74 issues)
**docs/M1_SPEC_ALIGNMENT_CRITICAL.md**: Specification-implementation gap analysis
**docs/CONTEXT_AND_LOGGING_REVIEW.md**: Context propagation mandate (REQ-F9-7)

### Milestone Dependencies

**M1 (002-mdns-querier)**: ✅ COMPLETE - 107/107 tasks, 85.9% coverage
**M1.1 (004-transport-layer)**: ⏳ BLOCKED - Requires M1-Refactoring completion

---

## Success Metrics

### Completion Metrics

| Metric | Baseline (M1) | Target (Post-Refactoring) | Validation |
|--------|---------------|---------------------------|------------|
| **Test Pass Rate** | 107/107 (100%) | 107/107 (100%) | `go test ./...` |
| **Test Coverage** | 85.9% | ≥85.0% | `go tool cover` |
| **Race Conditions** | 0 detected | 0 detected | `go test -race` |
| **Allocations (Receive)** | ~9000 bytes/op | ~1800 bytes/op | `go test -benchmem` |
| **Layer Violations** | 2 detected | 0 detected | `grep` analysis |
| **Error Swallowing** | 1 instance | 0 instances | Test validation |

### Architectural Metrics

| Metric | Baseline (M1) | Target (Post-Refactoring) |
|--------|---------------|---------------------------|
| **Transport Abstraction** | ❌ None | ✅ Interface defined |
| **IPv6 Ready** | ❌ No | ✅ Yes (via Transport) |
| **Testability** | 🟡 Requires network | ✅ Mockable (MockTransport) |
| **F-2 Compliance** | 🟡 75% | ✅ 100% |
| **F-3 Compliance** | 🟡 Partial (error swallowing) | ✅ 100% |
| **F-7 Compliance** | 🟡 Partial (no pooling) | ✅ 100% |

---

## Risk Assessment

### High Risk: Regression in M1 Functionality

**Probability**: Medium
**Impact**: High (breaks production-ready M1)
**Mitigation**:
- Run full M1 test suite after each FR implementation
- Use checkpoints (validate after FR-001, FR-002, FR-003, FR-004)
- Keep M1 baseline in git for comparison
- Automated CI validation on every commit

### Medium Risk: Performance Degradation from Abstraction

**Probability**: Low
**Impact**: Medium (Transport interface adds indirection)
**Mitigation**:
- Benchmark before/after (ensure <5% overhead)
- Interface method calls are inlinable by Go compiler
- Buffer pooling (FR-003) offsets any abstraction cost

### Low Risk: Test Coverage Reduction

**Probability**: Low
**Impact**: Medium (violates NFR-002)
**Mitigation**:
- Add tests for new Transport interface
- Add tests for buffer pooling
- Add test for CloseSocket error propagation
- Validate coverage after each FR

---

## Out of Scope (Deferred Items)

The M1 refactoring analysis identified 74 total issues. This milestone addresses **ONLY the 4 critical P0 issues** (14 hours). The following are **explicitly deferred**:

### Deferred to M1.2 or Later (P1 - High Priority, 32 issues, 45 hours)

**Architecture & Design** (7 issues):
- Query mutex too conservative (serializes concurrent queries)
- Strategy pattern for continuous queries
- Network package inverted dependency (imports protocol)
- Missing query result cache (TTL-based)
- No graceful degradation for partial failures
- No rate limiting (F-11 requirement)

**Code Smells & Duplication** (6 issues):
- Multicast address resolution duplicated
- Error wrapping pattern duplicated
- Magic numbers (buffer size, timeouts)
- Long functions (collectResponses, ParseName)

**Performance & Efficiency** (3 issues):
- String concatenation in hot path (ParseName)
- Response channel buffer size not justified
- Deduplication map never cleaned

**Error Handling & Resilience** (4 issues):
- Limited errors.As usage
- No sentinel errors (ErrTimeout, ErrNoRecordsFound)
- Context cancellation not always checked
- No validation of response message size

**Readability & Maintainability** (6 issues):
- Inconsistent error message formatting
- Missing package-level documentation
- Unexported types lack documentation
- Magic constants not documented
- Test table names not descriptive
- No Architecture Decision Records (ADRs)

**Test Quality & Coverage** (6 issues):
- Coverage gaps (querier 74.7%, network 70.3%)
- No test helper package
- Integration tests environment-dependent
- No benchmark tests
- Table-driven tests could be fuzzed

### Deferred to Polish Phase (P2 - Medium Priority, 38 issues, 51 hours)

See docs/M1_REFACTORING_ANALYSIS.md for complete P2 list.

---

## Implementation Phases

### Phase 0: Preparation (1 hour)

**Goal**: Set up refactoring environment and baselines

**Tasks**:
- Create M1 baseline branch (`git checkout -b m1-baseline`)
- Run full test suite and capture baseline metrics
- Run benchmarks and capture allocation baseline
- Analyze dependency graph and identify violations
- Create refactoring branch (`git checkout -b 003-m1-refactoring`)

**Deliverables**:
- Baseline test results (107/107 passing)
- Baseline coverage report (85.9%)
- Baseline benchmark results (allocations per receive)
- Dependency graph visualization

---

### Phase 1: Transport Interface Abstraction (8 hours)

**Goal**: Implement FR-001 and FR-002 (Transport interface + layer boundaries)

**Tasks**:
1. Create `internal/transport/` package structure
2. Define Transport interface
3. Implement UDPv4Transport (migrate socket.go logic)
4. Create MockTransport for testing
5. Update Querier to use Transport interface
6. Remove direct network imports from querier
7. Add Transport interface tests
8. Validate all M1 tests pass

**Checkpoint**: Run `go test ./... -v -race` - Expected: 107/107 PASS

---

### Phase 2: Buffer Pooling (2 hours)

**Goal**: Implement FR-003 (sync.Pool for receive buffers)

**Tasks**:
1. Create buffer pool in transport package
2. Update Transport.Receive() to use pooled buffers
3. Add benchmark for allocation comparison
4. Validate zero buffer leaks (defer pattern)
5. Validate all M1 tests pass

**Checkpoint**: Run benchmark - Expected: ≥80% allocation reduction

---

### Phase 3: Error Handling Cleanup (0.5 hours)

**Goal**: Implement FR-004 (CloseSocket error propagation)

**Tasks**:
1. Update CloseSocket to propagate errors
2. Add test for close error handling
3. Validate F-3 RULE-1 compliance
4. Validate all M1 tests pass

**Checkpoint**: Run error handling tests - Expected: All pass

---

### Phase 4: Validation and Documentation (2 hours)

**Goal**: Validate all FRs and update documentation

**Tasks**:
1. Run full test suite (go test ./... -v -race)
2. Validate coverage ≥85% (go tool cover)
3. Run benchmarks and document improvements
4. Validate layer boundaries (grep analysis)
5. Update godoc comments (Transport interface, buffer pooling)
6. Create ADR for Transport abstraction decision
7. Update CHANGELOG.md with refactoring summary

**Deliverables**:
- Test results (107/107 passing)
- Coverage report (≥85.0%)
- Benchmark comparison (before/after)
- Layer boundary validation report
- Updated documentation

---

## Acceptance Criteria

### Functional Acceptance

- [x] **FR-001**: Transport interface abstraction complete
  - [ ] Transport interface defined
  - [ ] UDPv4Transport implements interface
  - [ ] MockTransport exists
  - [ ] Querier uses Transport
  - [ ] Tests pass

- [x] **FR-002**: Layer boundaries compliant
  - [ ] No querier→network imports
  - [ ] Dependency graph validates F-2
  - [ ] Tests pass

- [x] **FR-003**: Buffer pooling implemented
  - [ ] sync.Pool in use
  - [ ] ≥80% allocation reduction
  - [ ] Zero buffer leaks
  - [ ] Tests pass

- [x] **FR-004**: Error propagation fixed
  - [ ] CloseSocket propagates errors
  - [ ] Tests validate propagation
  - [ ] F-3 RULE-1 compliant

### Non-Functional Acceptance

- [ ] **NFR-001**: All 107 M1 tests pass
- [ ] **NFR-002**: Coverage ≥85.0%
- [ ] **NFR-003**: ≥80% allocation reduction (benchmarked)
- [ ] **NFR-004**: Zero layer violations (validated)

### Quality Acceptance

- [ ] Code passes `go vet`
- [ ] Code passes `gofmt -l` (zero files)
- [ ] Documentation updated (godoc, ADRs)
- [ ] CHANGELOG.md updated

---

## References

### Analysis Documents

- **docs/M1_REFACTORING_ANALYSIS.md**: Comprehensive 74-issue analysis (6 specialized agents)
- **docs/M1_SPEC_ALIGNMENT_CRITICAL.md**: Specification-implementation gap analysis
- **docs/M1_ANALYSIS_COMPLETE_SUMMARY.md**: Executive summary with decision points
- **docs/CONTEXT_AND_LOGGING_REVIEW.md**: Context propagation requirements
- **docs/CONTEXT_AND_LOGGING_COMPLIANCE_MATRIX.md**: Compliance validation

### Specifications

- **F-2**: Package Structure and Dependencies
- **F-3**: Error Handling Strategy
- **F-6**: Logging & Observability
- **F-7**: Resource Management
- **F-9**: Transport Layer Socket Configuration (references for M1.1 alignment)

### Milestones

- **M1 (002-mdns-querier)**: ✅ COMPLETE - Baseline for refactoring
- **M1.1 (004-transport-layer)**: ⏳ PENDING - Requires this refactoring complete

### RFC Standards

- **RFC 6762**: Multicast DNS (no changes required)
- **RFC 1035**: DNS Specification (no changes required)
- **RFC 2782**: DNS SRV Records (no changes required)

---

## Approval

**Specification Status**: ✅ READY FOR PLANNING

**Next Steps**:
1. Run `/speckit.plan` to generate implementation plan
2. Review and approve plan.md
3. Run `/speckit.tasks` to generate surgical refactoring tasks
4. Execute refactoring with full test validation

**Constitutional Alignment**:
- ✅ Principle II: Spec-Driven Development (formal specification created)
- ✅ Principle III: Test-Driven Development (all changes validated by existing 107 tests)
- ✅ Principle VIII: Excellence (technical debt addressed systematically)

---

**Specification Created**: 2025-11-01
**Specification Version**: 1.0
**Status**: Draft - Awaiting Planning Phase
