# Beacon Development Guidelines

**Last Updated**: 2025-11-04 (006-mdns-responder implementation 94.6% complete)

This file provides context for Claude AI when working on the Beacon project.

---

## Project Overview

**Beacon** is a lightweight, high-performance mDNS (Multicast DNS) library for Go, implementing RFC 6762 for service discovery on local networks.

**Current Status**: M2 Responder Implementation 94.6% Complete (122/129 tasks), Ready for Final Polish

---

## Technology Stack

### Language & Runtime
- **Go 1.21+** (required)
- Standard library only - zero external dependencies
  - `net` - Network I/O
  - `context` - Cancellation and timeouts
  - `time` - Timing operations
  - `sync` - Concurrency primitives (sync.Pool for buffer pooling)
  - `encoding/binary` - DNS wire format encoding

### Architecture
- **Clean Architecture** with strict layer boundaries (F-2)
- **TDD Methodology** (RED → GREEN → REFACTOR)
- **Context-aware** operations throughout (F-9)

---

## Project Structure

```
beacon/
├── querier/                    # Public API - mDNS query interface
│   ├── querier.go             # Main Querier type
│   ├── querier_test.go        # Unit & integration tests
│   └── types.go               # Public types (ResourceRecord, etc.)
│
├── responder/                  # Public API - mDNS responder interface (006-mdns-responder)
│   ├── responder.go           # Main Responder type
│   ├── service.go             # Service definition and validation
│   ├── options.go             # Configuration options
│   └── conflict_detector.go   # RFC 6762 §8.2 tie-breaking logic
│
├── internal/                   # Internal implementation (not importable)
│   ├── transport/             # Network transport abstraction (M1-Refactoring)
│   │   ├── transport.go       # Transport interface
│   │   ├── udp.go             # UDPv4Transport (IPv4 multicast)
│   │   ├── buffer_pool.go     # sync.Pool for 9KB buffers
│   │   └── mock_transport.go  # Test double
│   │
│   ├── message/               # DNS message parsing/building
│   │   ├── builder.go         # Query/response message construction
│   │   ├── parser.go          # Response parsing
│   │   ├── name.go            # DNS name encoding (RFC 6763 §4.3)
│   │   └── message.go         # DNS message types
│   │
│   ├── protocol/              # mDNS protocol constants
│   │   └── constants.go       # Port 5353, multicast addresses, record types
│   │
│   ├── errors/                # Custom error types
│   │   └── errors.go          # NetworkError, ValidationError, etc.
│   │
│   ├── responder/             # Responder implementation (006-mdns-responder)
│   │   ├── registry.go        # Thread-safe service registry
│   │   ├── response_builder.go # RFC 6762 §6 response construction
│   │   └── known_answer.go    # RFC 6762 §7.1 suppression logic
│   │
│   ├── state/                 # State machine for probing/announcing (006-mdns-responder)
│   │   ├── machine.go         # State machine orchestration
│   │   ├── prober.go          # RFC 6762 §8.1 probing
│   │   └── announcer.go       # RFC 6762 §8.3 announcing
│   │
│   ├── records/               # DNS record construction (006-mdns-responder)
│   │   ├── record_set.go      # PTR, SRV, TXT, A record builders
│   │   └── ttl.go             # RFC 6762 §10 TTL values
│   │
│   ├── security/              # Input validation and rate limiting (006-mdns-responder)
│   │   ├── validation.go      # RFC-compliant input validation
│   │   └── rate_limiter.go    # RFC 6762 §6.2 per-interface rate limiting
│   │
│   └── network/               # Socket operations (legacy, being phased out)
│       └── socket.go          # Legacy socket code
│
├── tests/                      # Test suites
│   ├── contract/              # Contract tests (RFC compliance)
│   ├── integration/           # Real network tests
│   └── fuzz/                  # Fuzz testing
│
├── .specify/                   # ⭐ Spec Kit Framework Configuration
│   ├── specs/                 # Foundation specifications (F-1 through F-11)
│   │   ├── F-2-architecture-layers.md     # Layer boundaries
│   │   ├── F-3-error-handling.md          # Error propagation
│   │   └── F-9-transport-layer-config.md  # Socket configuration
│   ├── memory/                # Constitutional memory
│   │   └── constitution.md    # Project principles
│   └── templates/             # Spec/plan/task templates
│
├── specs/                      # ⭐ Feature Specifications (Milestone work)
│   ├── 001-spec-kit-migration/
│   ├── 002-mdns-querier/      # M1 implementation spec
│   ├── 003-m1-refactoring/    # M1-Refactoring spec (97 tasks complete)
│   │   ├── spec.md            # Requirements and success criteria
│   │   ├── plan.md            # Implementation strategy
│   │   └── tasks.md           # Executable tasks with checkpoints
│   └── 006-mdns-responder/    # M2 responder implementation (129 tasks, 122 complete)
│       ├── spec.md            # Responder requirements and RFC compliance
│       ├── plan.md            # State machine and record construction strategy
│       ├── tasks.md           # Executable tasks (94.6% complete)
│       ├── SECURITY_AUDIT.md  # Security validation (Grade: STRONG)
│       ├── CODE_REVIEW.md     # Code quality review (Grade: A)
│       └── PERFORMANCE_ANALYSIS.md # Performance profiling (Grade: A+)
│
├── RFC%20Docs/                   # ⭐ Protocol Specifications (SOURCE OF TRUTH)
│   ├── rfc6762.txt            # mDNS specification
│   └── rfc1035.txt            # DNS message format
│
├── docs/decisions/             # Architecture Decision Records (ADRs)
│   ├── 001-transport-interface-abstraction.md
│   ├── 002-buffer-pooling-pattern.md
│   └── 003-integration-test-timing-tolerance.md
│
└── archive/                    # Historical artifacts
    └── m1-refactoring/        # M1-Refactoring metrics and reports
        ├── README.md          # Archive documentation
        ├── reports/           # Completion reports
        └── metrics/           # Test/benchmark data

```

---

## Key Architectural Patterns

### 1. Transport Interface Abstraction
**Why**: Decouples querier from network implementation, enables IPv6 (M2), testability
**ADR**: docs/decisions/001-transport-interface-abstraction.md

```go
type Transport interface {
    Send(ctx context.Context, packet []byte, dest net.Addr) error
    Receive(ctx context.Context) ([]byte, net.Addr, error)
    Close() error
}
```

**Implementations**:
- `UDPv4Transport` - Production IPv4 multicast (224.0.0.251:5353)
- `MockTransport` - Test double for unit testing

---

### 2. Buffer Pooling
**Why**: Eliminates 900KB/sec allocations (9KB per receive call)
**ADR**: docs/decisions/002-buffer-pooling-pattern.md
**Result**: 99% allocation reduction (9000 B/op → 48 B/op)

```go
// UDPv4Transport.Receive() uses buffer pool
bufPtr := GetBuffer()
defer PutBuffer(bufPtr)  // Returns buffer to pool
```

---

### 3. Layer Boundaries (F-2)
**Rule**: Strict import restrictions

```
querier → transport → network
       ↘ protocol
       ↘ message
       ↘ errors
```

**Validation**: `grep -rn "internal/network" querier/` must return 0 matches

---

### 4. Error Propagation (FR-004)
**Rule**: Never swallow errors
**Pattern**: All errors wrapped in typed errors (`NetworkError`, `ValidationError`, `WireFormatError`)

```go
// ✅ CORRECT
func (t *UDPv4Transport) Close() error {
    return t.conn.Close()  // Propagate error
}

// ❌ WRONG
func (t *UDPv4Transport) Close() error {
    t.conn.Close()
    return nil  // Error swallowed!
}
```

---

## Common Commands

### Testing
```bash
# Run all tests
make test                    # Basic test run
make test-race              # With race detector
make test-coverage          # With coverage (≥80% gate)

# Detailed coverage report
make test-coverage-report   # Pretty formatted by package

# Track coverage over time
./scripts/coverage-trend.sh              # Record current
./scripts/coverage-trend.sh --show       # Show history
./scripts/coverage-trend.sh --graph      # Show trend graph

# Specific test suites
go test ./querier -run TestQuery              # Single test
go test ./tests/integration -v                # Integration tests
go test -bench=. -benchmem ./...              # Benchmarks
go test -fuzz=FuzzParseMessage -fuzztime=10s ./tests/fuzz  # Fuzz tests

# CI pipelines
make ci-fast                # Fast CI (unit + race + coverage)
make ci-full                # Full CI (all tests)
```

**Coverage Philosophy**:
- **Target**: ≥80% (Constitution requirement, REQ-F8-2)
- **Aspiration**: 85%+
- **Current**: 81.3%
- **Enforcement**: CI only (NOT in pre-commit hook)

**Why no pre-commit coverage gate?**
Coverage measures aggregate codebase health, not commit safety. Coverage checks:
- Are slow (3-5s vs. 1-2s for gofmt/vet/semgrep)
- Break TDD RED phase (adding tests before implementation)
- Are non-deterministic (depend on global state)
- Can be gamed with empty tests
- Should answer "Is codebase healthy?" (CI) not "Is this commit safe?" (pre-commit)

**Where coverage IS enforced:**
- ✅ `make ci-fast` / `make ci-full` - Blocks merge if <80%
- ✅ Developer visibility via `make test-coverage-report`
- ✅ Trend tracking via `./scripts/coverage-trend.sh`

**Best practices:**
- Write tests FIRST (TDD: RED → GREEN → REFACTOR)
- Use coverage to find gaps, not to hit a number
- Test behavior, not implementation details
- Don't panic over small drops (<2%) - check if expected (RED phase, refactoring)
- Focus on critical paths: error handling, edge cases, concurrency

**Package targets (current status):**
- `internal/protocol`: 100% ✅ (constants)
- `internal/errors`: 93.3% ✅
- `internal/message`: 90.3% ✅
- `internal/records`: 94.1% ✅
- `internal/security`: 92.1% ✅ (security-critical)
- `internal/state`: 83.6% ✅
- `internal/responder`: 76.5% ⚠️ (WIP)
- `internal/network`: 73.9% ⚠️
- `internal/transport`: 71.1% ⚠️
- `querier`: 60.6% ⚠️ (needs more integration tests)

### Building
```bash
# Build library (check for errors)
go build ./...

# Vet code
go vet ./...

# Format code
gofmt -w .

# Run all quality checks (via Makefile)
make test
```

### Semgrep (Static Analysis)

**Automated Enforcement**: Semgrep runs automatically via:
1. **Pre-commit hook** - Blocks commits with findings (`.githooks/pre-commit`)
2. **Makefile CI** - `make semgrep-check` in `verify`, `ci-fast`, `ci-full`
3. **Manual runs** - `make semgrep` (informational) or `make semgrep-check` (strict)

**When to use Semgrep:**
- ✅ **AUTOMATIC** - Pre-commit hook runs on every `git commit`
- ✅ **CI/CD** - All CI targets include `semgrep-check`
- ✅ **MANUAL** - Run `make semgrep` to see all findings (informational)
- ✅ **BEFORE PR** - Run `make semgrep-check` to verify clean state

**DO NOT** use Semgrep:
- ❌ During initial exploration/prototyping
- ❌ Bypass with `git commit --no-verify` without user permission

```bash
# ✅ RECOMMENDED: Use Makefile targets
make semgrep              # Informational scan (won't fail)
make semgrep-check        # Strict scan (fails on findings) - used in CI

# ✅ Pre-commit hook (automatic)
git commit -m "message"   # Hook runs semgrep automatically

# Manual semgrep commands (if needed)
semgrep --config=.semgrep.yml .                                # All findings
semgrep --config=.semgrep.yml --severity ERROR .               # Critical only
semgrep --config=.semgrep.yml --validate                       # Check config
semgrep --config=.semgrep.yml .semgrep-tests/ --no-git-ignore  # Test rules

# Bypass hook (NOT recommended - ask user first)
git commit --no-verify
```

**Rules enforce:**
- 🛡️ **Constitution principles** (RFC compliance, error handling, dependencies)
- 🏗️ **F-Spec requirements** (concurrency, architecture, security)
- 📋 **RFC 6762 compliance** (mDNS protocol constants, TTLs)

**Common findings you should fix:**
1. **Timer/Ticker leaks** (`beacon-timer-leak`, `beacon-ticker-leak`)
   - Add `defer timer.Stop()` after creating timers
2. **Mutex without defer** (`beacon-mutex-defer-unlock`)
   - Add `defer mu.Unlock()` after `mu.Lock()`
3. **File leaks** (`beacon-file-missing-defer-close`)
   - Add `defer file.Close()` after opening files
4. **WaitGroup leaks** (`beacon-waitgroup-missing-done`)
   - Add `defer wg.Done()` as first line in goroutine
5. **Security violations** (`beacon-unsafe-in-parser`, `beacon-panic-on-network-input`)
   - Never use `unsafe` in packet parsing code
   - Never `panic()` on network input - return `WireFormatError` instead

**Integration:**
- **Pre-commit hook**: `.githooks/pre-commit` (auto-installed via `git config core.hooksPath .githooks`)
- **Makefile**: `make semgrep-check` in `verify`, `ci-fast`, `ci-full` targets
- **Testing**: See `.semgrep-tests/README.md` for TDD approach
- **Rules**: See `SEMGREP_RULES_SUMMARY.md` for complete documentation

**Suppressing findings:**
```go
// Valid reason: Lock upgrade pattern requires manual unlock
mu.RLock() // nosemgrep: beacon-mutex-defer-unlock
```
Always add a comment explaining WHY suppression is necessary.

**Current Status:**
```bash
# As of 2025-11-02:
# - 25 rules active
# - 13 ERROR severity (critical bugs/security/library design)
# - 10 WARNING severity (best practices)
# - 2 INFO severity (style)
```

---

## Code Style

### Go Standards
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` (no exceptions)
- Use `go vet` (must pass with zero warnings)

### Testing Standards
- **TDD**: Write tests FIRST (RED phase)
- **Coverage**: Maintain ≥85% (current: 84.8%)
- **Stability**: Zero flaky tests (achieved in M1-Refactoring)
- **Naming**: Test functions describe behavior (`TestQuery_Timeout_ReturnsEmptyResponse`)

### Documentation Standards
- **Godoc**: All exported types/functions must have documentation
- **ADRs**: Major architectural decisions documented in `docs/decisions/`
- **Comments**: Explain WHY, not WHAT
- **RFC References**: Link to RFC sections when implementing protocol behavior

---

## Recent Changes

### 006-mdns-responder (94.6% Complete - 2025-11-04)
**Branch**: 006-mdns-responder
**Status**: 🚧 94.6% Complete (122/129 tasks), Final documentation polish in progress
**Summary**: Full mDNS responder implementation with RFC 6762/6763 compliance

**Achievements**:
- ✅ **US1**: Service registration with probing and announcing (RFC 6762 §8)
- ✅ **US2**: Conflict resolution with lexicographic tie-breaking (RFC 6762 §8.2)
- ✅ **US3**: Query response with PTR/SRV/TXT/A records (RFC 6762 §6)
- ✅ **US4**: Cache coherency via known-answer suppression (RFC 6762 §7.1)
- ✅ **US5**: Multi-service support and service enumeration (RFC 6763 §9)
- ✅ **Security Audit**: STRONG security posture, zero panics on malformed input
- ✅ **Code Review**: Grade A, clean architecture (F-2 compliance)
- ✅ **Performance**: Grade A+ (4.8μs response, 20,833x under 100ms requirement)

**New Public APIs**:
- `responder.New(options...)` - Create responder with functional options
- `responder.Register(service)` - Register service with probing/announcing
- `responder.Unregister(instanceName)` - Unregister service with goodbye
- `responder.UpdateService(service)` - Update TXT records
- `responder.GetService(instanceName)` - Query service by name
- `responder.Service` - Service definition (InstanceName, ServiceType, Port, TXT, etc.)
- `responder.ConflictDetector` - RFC 6762 §8.2 tie-breaking logic

**Key Files Created**:
- `responder/` - Public API (responder.go, service.go, options.go, conflict_detector.go)
- `internal/responder/` - Implementation (registry.go, response_builder.go, known_answer.go)
- `internal/state/` - State machine (machine.go, prober.go, announcer.go)
- `internal/records/` - Record construction (record_set.go, ttl.go)
- `internal/security/` - Validation and rate limiting (validation.go, rate_limiter.go)
- `internal/message/name.go` - RFC 6763 §4.3 DNS name encoding
- `tests/contract/` - 36 RFC compliance tests (36/36 PASS)
- `tests/fuzz/` - 4 fuzzers (109,471 executions, 0 crashes)

**Documentation**:
- `specs/006-mdns-responder/SECURITY_AUDIT.md` - Security validation report
- `specs/006-mdns-responder/CODE_REVIEW.md` - Code quality review
- `specs/006-mdns-responder/PERFORMANCE_ANALYSIS.md` - Performance profiling

**Remaining**: T123-T126 (documentation updates), T116-T117 (deferred - require macOS/Avahi)

---

### M1.1 Architectural Hardening (Complete - 2025-11-02)
**Branch**: 004-m1-1-architectural-hardening
**Status**: ✅ Complete (ready to merge)
**Summary**: Production-ready socket configuration, security, and interface management

**Achievements**:
- ✅ SO_REUSEPORT socket options (Avahi/Bonjour coexistence on port 5353)
- ✅ Platform-specific socket configuration (Linux, macOS, Windows)

**New Public APIs**:

**Key Files Changed**:

**Documentation**:

---

### M1-Refactoring (Complete - 2025-11-01)
**Branch**: 003-m1-refactoring
**Status**: ✅ Merged to main
**Summary**: Comprehensive architectural refactoring

**Achievements**:

**Key Files Changed**:

**Documentation**:

---

### M1 Implementation (Complete - 2025-10-XX)
**Branch**: 002-mdns-querier
**Summary**: Initial mDNS query implementation

**Features**:

---

## Next Steps (M1.2 / M2)

### Planned Features (M1.2 - Service Discovery)
1. **Service Browsing**
   - Browse for services (`_services._dns-sd._udp.local`)
   - Service instance enumeration
   - Service type discovery

2. **Enhanced Resource Record Parsing**
   - Better handling of compressed names
   - Support for additional record types (AAAA for IPv6)

### Planned Features (M2 - IPv6 & Advanced)
1. **IPv6 Support**
   - Dual-stack operation (IPv4 + IPv6)
   - IPv6 multicast (FF02::FB)
   - Per-interface transport binding

2. **Full Source Filtering**
   - Per-interface SourceFilter integration
   - Same-subnet validation (requires M2 per-interface transports)

3. **Observability**
   - Structured logging (F-6 Logging & Observability spec)
   - Metrics and telemetry

**Spec**: See `specs/` directory for feature specifications

---

## Specification-Driven Development (Spec Kit Framework)

**This is a Spec Kit project** - all development follows the [Spec Kit](https://github.com/github/spec-kit) framework's specification-driven methodology.

### Core Philosophy

Development follows a strict **Spec → Plan → Tasks → TDD → Validate** cycle:

1. **Specification** - Define WHAT and WHY
   - Requirements (functional/non-functional)
   - Success criteria
   - RFC compliance constraints

2. **Planning** - Define HOW
   - Architecture decisions (documented as ADRs)
   - Implementation strategy
   - Risk analysis

3. **Tasks** - Define executable steps
   - Granular, testable tasks (T001, T002, ...)
   - TDD cycles (RED → GREEN → REFACTOR)
   - Checkpoint validation

4. **TDD Execution** - STRICT test-first implementation
   - Write tests BEFORE code
   - Validate against F-Specs
   - Check RFC compliance

5. **Validation** - Prove completion
   - All tasks complete
   - All success criteria met
   - Completion report generated

### Key Directories

#### `.specify/` - Framework Configuration
- **`.specify/specs/`** - Foundation specifications
  - **BEACON_FOUNDATIONS.md** - ⭐ DNS/mDNS/DNS-SD conceptual foundation (919 lines)
    - DNS fundamentals, mDNS essentials, DNS-SD concepts
    - Terminology glossary, reference tables
    - **Read this first** to understand the domain
  - **F-2**: Layer Boundaries - Defines clean architecture constraints
  - **F-3**: Error Handling - Error propagation patterns (FR-004)
  - **F-9**: Transport Layer Configuration - Socket options, Avahi/Bonjour coexistence
  - **F-10**: Network Interface Management - Interface selection, VPN exclusion
  - **F-11**: Security Architecture - Rate limiting, source IP filtering
- **`.specify/memory/constitution.md`** - Constitutional principles (see below)
- **`.specify/templates/`** - Templates for specs, plans, tasks

#### `specs/` - Feature Specifications
- **`specs/[milestone]/spec.md`** - Feature specification
- **`specs/[milestone]/plan.md`** - Implementation plan
- **`specs/[milestone]/tasks.md`** - Executable tasks
- Example: `specs/003-m1-refactoring/` (97 tasks, all complete)

#### `RFC%20Docs/` - Protocol Specifications (CRITICAL)
- **RFC 6762**: mDNS specification - **ALL behavior must comply**
- **RFC 1035**: DNS message format
- RFCs are the source of truth for protocol behavior
- When in doubt, reference the RFC

#### `docs/decisions/` - Architecture Decision Records
- **ADR-001**: Transport Interface Abstraction
- **ADR-002**: Buffer Pooling Pattern
- **ADR-003**: Integration Test Timing Tolerance
- ADRs document WHY we made key architectural choices

### Spec Kit Slash Commands

**IMPORTANT**: Use these commands when working on features:

- **`/speckit.specify`** - Create/update feature specification
  - Use when starting a new feature or milestone
  - Generates structured spec.md with requirements

- **`/speckit.plan`** - Generate implementation plan
  - Use after specification is complete
  - Creates plan.md with architecture decisions

- **`/speckit.tasks`** - Generate executable tasks
  - Use after plan is complete
  - Creates tasks.md with granular, testable tasks

- **`/speckit.implement`** - Execute implementation
  - Use to execute tasks in TDD cycles
  - Updates tasks.md as work progresses

- **`/speckit.analyze`** - Validate spec consistency
  - Use before starting implementation
  - Checks for conflicts between spec/plan/tasks

### Development Workflow Example (M1-Refactoring)

```bash
# 1. Create specification
/speckit.specify
# → Generated specs/003-m1-refactoring/spec.md

# 2. Plan implementation
/speckit.plan
# → Generated specs/003-m1-refactoring/plan.md with ADRs

# 3. Generate tasks
/speckit.tasks
# → Generated specs/003-m1-refactoring/tasks.md (97 tasks)

# 4. Execute tasks (TDD)
# T011-T018: RED - Write transport interface tests FIRST
# T019-T037: GREEN - Implement UDPv4Transport to make tests pass
# T038-T043: REFACTOR - Clean up, validate, checkpoint

# 5. Validate completion
# → COMPLETION_VALIDATION.md (9 criteria)
# → REFACTORING_COMPLETE.md (full report)
# → All tasks marked [x], all criteria met
```

---

## Constitutional Principles

From `.specify/memory/constitution.md`:

1. **Protocol Compliance First** - RFC 6762 (mDNS) compliance is non-negotiable
2. **Zero External Dependencies** - Standard library only
3. **Context-Aware Operations** - All blocking operations accept `context.Context`
4. **Clean Architecture** - Strict layer boundaries (F-2)
5. **Test-Driven Development** - Tests written first (TDD)
6. **Performance Matters** - Optimize hot paths (buffer pooling example)

---

## Performance Characteristics

### Current Metrics (Post M1-Refactoring)
- **Query Latency**: 163 ns/op (9% improvement)
- **Allocations**: 48 B/op in receive path (99% reduction)
- **Concurrent Queries**: 100+ supported (NFR-002)
- **Test Coverage**: 84.8%
- **Flaky Tests**: 0

### NFRs (Non-Functional Requirements)
- **NFR-001**: Query processing overhead <100ms ✅
- **NFR-002**: Support ≥100 concurrent queries ✅
- **NFR-003**: ≥80% allocation reduction (achieved 99%) ✅

---

## Troubleshooting

### Common Issues

**Test Failures**:
```bash
# Clear test cache
go clean -testcache

# Run with verbose output
go test ./... -v
```

**Integration Test Flakiness**:
- Integration tests use real mDNS traffic
- Timing-sensitive (100ms tolerance added in M1-Refactoring)
- See ADR-003 for timing tolerance rationale

**Layer Boundary Violations**:
```bash
# Check for violations
grep -rn "internal/network" querier/

# Should return: no matches
```

---

## Git Workflow

### Branch Naming
- Feature: `XXX-feature-name` (e.g., `003-m1-refactoring`)
- Where XXX matches spec directory number

### Commit Messages
- Use conventional commits style
- Reference spec/task numbers
- Include WHY, not just WHAT

### Testing Before Commit
```bash
go test ./... -race
go vet ./...
gofmt -l . | grep . && echo "Files need formatting"
```

---

## Resources

### RFCs
- [RFC 6762](https://www.rfc-editor.org/rfc/rfc6762.html) - Multicast DNS (primary)
- [RFC 1035](https://www.rfc-editor.org/rfc/rfc1035.html) - DNS specification (wire format)

### Internal Docs
- `docs/decisions/` - Architecture Decision Records
- `specs/` - Feature specifications
- `archive/m1-refactoring/reports/` - Historical completion reports

### External
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)

---

<!-- MANUAL ADDITIONS START -->
<!-- Add project-specific guidelines below -->

<!-- MANUAL ADDITIONS END -->

---

**Generated**: 2025-11-01
**Project**: Beacon mDNS Library
**Status**: M1-Refactoring Complete, M1.1 Planning
- Any time we remove a test, I would like to think critically over the decision and to consider if other tests need to adde in its place.

## Active Technologies
- Go 1.21+ + Standard library + `golang.org/x/sys` (platform-specific socket options from M1.1), `golang.org/x/net` (multicast group management from M1.1) (006-mdns-responder)
- In-memory (registered services, resource record sets with TTLs) (006-mdns-responder)
