# Implementation Plan: M1.1 Architectural Hardening

**Branch**: `004-m1-1-architectural-hardening` | **Date**: 2025-11-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-m1-1-architectural-hardening/spec.md`

## Summary

M1.1 implements production-grade socket configuration, interface management, and security features required for enterprise mDNS deployment. This milestone addresses critical architectural pitfalls identified in research: (1) Avahi/Bonjour coexistence via SO_REUSEPORT socket options, (2) VPN privacy protection via intelligent interface filtering, (3) multicast storm resilience via rate limiting, and (4) RFC 6762 §2 compliance via link-local source validation.

**Primary Requirement**: Enable Beacon to coexist with system mDNS daemons (Avahi, systemd-resolved, Bonjour) on shared port 5353, protect user privacy by excluding VPN/Docker interfaces from mDNS traffic, and provide resilience against multicast storms.

**Technical Approach** (from F-9, F-10, F-11 specifications):
- Replace `net.ListenMulticastUDP()` with `net.ListenConfig` pattern + platform-specific socket options
- Use `golang.org/x/sys` for SO_REUSEADDR/SO_REUSEPORT (Linux kernel 3.9+, macOS)
- Use `golang.org/x/net/ipv4` for multicast group membership (RFC 6762 §5)
- Implement `DefaultInterfaces()` with VPN/Docker exclusion patterns
- Add per-source-IP rate limiting (100 qps threshold, 60s cooldown)
- Validate source IPs before parsing (link-local or same subnet)

## Technical Context

**Language/Version**: Go 1.21+ (already established in M1)
**Primary Dependencies**:
- Standard library: `net`, `context`, `time`, `sync` (already used)
- **NEW**: `golang.org/x/sys/unix` (Linux/macOS socket options)
- **NEW**: `golang.org/x/sys/windows` (Windows socket options)
- **NEW**: `golang.org/x/net/ipv4` (multicast group management)

**Storage**: N/A (in-memory rate limiter map only)
**Testing**: Go testing (`go test`), race detector (`-race`), platform-specific build tags
**Target Platform**: Linux (kernel 3.9+), macOS, Windows (cross-platform)
**Project Type**: Single project (library, not application)
**Performance Goals**:
- Rate limiter: Process 1000 queries/second multicast storm without degradation (SC-005: CPU <20%, memory <10MB)
- Socket initialization: <100ms per interface
- Zero abstraction overhead from platform-specific code

**Constraints**:
- RFC 6762 compliance: Multicast group 224.0.0.251, TTL=255, link-local scope
- Platform compatibility: Must work on Linux (3.9+), macOS, Windows with appropriate socket option support
- Zero regression: All M1 tests must continue passing (SC-008)
- Coverage: Maintain ≥80% test coverage (SC-009)

**Scale/Scope**:
- 35 functional requirements (FR-001 through FR-035)
- 3 platform-specific socket option implementations (Linux, macOS, Windows)
- 11 success criteria across coexistence, privacy, resilience, quality

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: RFC Compliant ✅ PASS

- ✅ **RFC 6762 §5**: Multicast group membership (FR-005: `golang.org/x/net/ipv4` joins 224.0.0.251)
- ✅ **RFC 6762 §11**: TTL=255 for multicast (FR-006: SetMulticastTTL(255))
- ✅ **RFC 6762 §2**: Link-local scope (FR-023: Source IP validation)
- ✅ **RFC 6762 §17**: Packet size limits (FR-034: Reject >9000 bytes)

**Validation**: All RFC requirements referenced in FR section, validated against F-9, F-10, F-11 specifications.

### Principle II: Spec-Driven Development ✅ PASS

- ✅ Feature specification complete: [spec.md](./spec.md)
- ✅ Foundation specs exist: F-9 (Socket), F-10 (Interface), F-11 (Security)
- ✅ Architecture decisions documented: ADR-001 (Transport), ADR-002 (Buffer Pool) from M1-Refactoring provide foundation
- ✅ User scenarios defined: 4 prioritized stories (P1/P2/P3) with acceptance criteria

### Principle III: Test-Driven Development ✅ PASS

- ✅ Acceptance tests defined in spec (17 Given/When/Then scenarios)
- ✅ TDD approach: Tests written first per F-8 (Testing Strategy)
- ✅ Coverage target: ≥80% (SC-009)
- ✅ Race detection: All tests must pass with `-race` flag
- ✅ Platform-specific testing: Build tags for Linux/macOS/Windows socket tests

### Principle IV: Phased Approach ✅ PASS

- ✅ Milestone-based: M1.1 is part of M1-M6 roadmap
- ✅ Incremental delivery: Builds on M1-Refactoring transport interface
- ✅ Validation gates: Success criteria defined (SC-001 through SC-011)
- ✅ Scope bounded: Out of Scope section excludes M1.2, M2 features

### Principle V: Dependencies and Supply Chain ⚠️ REQUIRES JUSTIFICATION

**New External Dependencies** (semi-standard libraries):

1. **`golang.org/x/sys/unix`** (Linux/macOS)
   - **Why Needed**: SO_REUSEADDR and SO_REUSEPORT socket options unavailable in `net` package
   - **Platform-Specific**: Yes (syscalls differ by OS)
   - **Standard Library Alternative**: None (Go Issue #40752: `net.ListenConfig.Control` provides raw FD, but socket option constants not in stdlib)
   - **Maintained by Go Team**: Yes (golang.org/x/ repos are semi-standard, maintained by Go core team)
   - **Justification**: Coexistence with Avahi/Bonjour (P1 user story) impossible without SO_REUSEPORT. This is THE critical dependency enabling production deployment.

2. **`golang.org/x/sys/windows`** (Windows)
   - **Why Needed**: Platform-specific SO_REUSEADDR (Windows behavior differs from POSIX)
   - **Platform-Specific**: Yes
   - **Standard Library Alternative**: `syscall` package (deprecated, `x/sys/windows` is recommended replacement)
   - **Justification**: Windows socket options require Windows-specific constants

3. **`golang.org/x/net/ipv4`** (cross-platform)
   - **Why Needed**: Multicast group membership (JoinGroup, SetMulticastTTL, SetMulticastLoopback)
   - **RFC Requirement**: RFC 6762 §5 requires joining multicast group 224.0.0.251
   - **Standard Library Alternative**: None (`net.ListenMulticastUDP` has unfixable bugs per Go Issues #73484, #34728)
   - **Justification**: RFC 6762 compliance impossible with standard library alone

**Complexity Tracking**: See table below

### Principle VI: Open Source ✅ PASS

- ✅ MIT License already established
- ✅ No proprietary dependencies

### Principle VII: Maintained ✅ PASS

- ✅ Roadmap exists (M1 through M6)
- ✅ Documentation strategy defined in F-6 (Logging & Observability)

### Principle VIII: Excellence ✅ PASS

- ✅ Addresses known architectural pitfalls (documented in `docs/ARCHITECTURAL_PITFALLS_AND_MITIGATIONS.md`)
- ✅ Industry best practices: SO_REUSEPORT for port sharing (used by nginx, Apache, systemd)
- ✅ Production-hardened approach: Rate limiting (learned from Hubitat incident), VPN exclusion (GDPR/HIPAA compliance)

**GATE RESULT**: ✅ PASS with justifications for Principle V

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| golang.org/x/sys (not stdlib) | SO_REUSEPORT socket option for Avahi/Bonjour coexistence | Standard library `net` package does not expose SO_REUSEPORT constant or platform-specific socket options. `net.ListenConfig.Control` provides raw FD but requires `x/sys` for socket option constants. |
| golang.org/x/net (not stdlib) | Multicast group membership API (JoinGroup, SetMulticastTTL) | `net.ListenMulticastUDP()` has unfixable bugs (Go #73484: receives ALL port 5353 traffic, not just multicast; #34728: incorrect binding). RFC 6762 §5 compliance requires proper multicast group join. |
| Platform-specific files (3 impls) | Socket options differ: Linux (SO_REUSEPORT kernel 3.9+), macOS (SO_REUSEPORT), Windows (SO_REUSEADDR only, different semantics) | Unified cross-platform abstraction impossible - socket options are fundamentally OS-specific. Build tags ensure compile-time safety. |

## Project Structure

### Documentation (this feature)

```text
specs/004-m1-1-architectural-hardening/
├── spec.md                  # Feature specification (already complete)
├── plan.md                  # This file (implementation plan)
├── research.md              # Phase 0 output (technology decisions)
├── data-model.md            # Phase 1 output (entities: SocketConfig, RateLimitEntry, etc.)
├── quickstart.md            # Phase 1 output (developer quick reference)
├── contracts/               # Phase 1 output (API contracts for new options)
│   └── querier-options.md   # WithInterfaces(), WithInterfaceFilter(), WithRateLimit() API
├── checklists/
│   └── requirements.md      # Quality checklist (already complete, all PASS)
└── tasks.md                 # Phase 2 output (/speckit.tasks command - NOT created yet)
```

### Source Code (repository root)

```text
internal/
├── transport/               # M1-Refactoring foundation (already exists)
│   ├── transport.go         # Transport interface (already exists)
│   ├── udp.go               # UDPv4Transport (M1-Refactoring, WILL EXTEND)
│   ├── buffer_pool.go       # Buffer pool (already exists)
│   ├── mock.go              # MockTransport (already exists)
│   │
│   ├── socket_linux.go      # NEW: Linux socket options (SO_REUSEADDR + SO_REUSEPORT)
│   ├── socket_darwin.go     # NEW: macOS socket options (SO_REUSEADDR + SO_REUSEPORT)
│   ├── socket_windows.go    # NEW: Windows socket options (SO_REUSEADDR only)
│   └── socket_test.go       # NEW: Platform-specific socket option tests
│
├── network/                 # Socket layer (currently has CreateSocket(), WILL REFACTOR)
│   ├── socket.go            # REFACTOR: Move to ListenConfig pattern
│   ├── interfaces.go        # NEW: DefaultInterfaces(), interface filtering logic
│   └── interfaces_test.go   # NEW: Interface filtering tests
│
├── security/                # NEW: Security features (F-11)
│   ├── rate_limiter.go      # NEW: Per-source-IP rate limiting
│   ├── source_filter.go     # NEW: Link-local source IP validation
│   └── security_test.go     # NEW: Rate limiter + source filter tests
│
querier/                     # Public API (already exists, WILL EXTEND)
├── querier.go               # EXTEND: Add WithInterfaces(), WithInterfaceFilter() options
├── options.go               # EXTEND: Add new functional options (WithRateLimit, etc.)
└── querier_test.go          # EXTEND: Test new options

tests/
├── integration/             # Real network tests (already exists)
│   ├── query_test.go        # EXTEND: Add Avahi coexistence test (SC-001/SC-002)
│   ├── interfaces_test.go   # NEW: VPN/Docker exclusion tests (SC-003/SC-004)
│   └── storm_test.go        # NEW: Multicast storm simulation (SC-005/SC-006)
│
└── contract/                # RFC compliance tests (already exists)
    └── security_test.go     # NEW: Link-local scope enforcement (SC-007)
```

**Structure Decision**: Single project (library). M1.1 extends existing `internal/transport/` and `internal/network/` packages from M1-Refactoring with platform-specific socket configuration. New `internal/security/` package isolates rate limiting and source filtering logic per F-11. Public API (`querier/`) extended with new functional options for interface selection and rate limiting configuration.

## Architecture Decisions

### AD-001: ListenConfig Pattern for Socket Options

**Decision**: Replace `net.ListenMulticastUDP()` with `net.ListenConfig` + `Control` function

**Context**: Go standard library's `net.ListenMulticastUDP()` has two unfixable bugs:
- Go Issue #73484: Receives ALL UDP on port 5353, not just multicast (CPU waste, DoS vector)
- Go Issue #34728: Incorrect binding to 0.0.0.0 instead of multicast address

**Alternatives Considered**:
1. ~~Use `net.ListenMulticastUDP()` with workarounds~~ - Bugs are unfixable in stdlib
2. ~~Fork stdlib and fix bugs~~ - Unsustainable maintenance burden
3. **CHOSEN**: `net.ListenConfig` with `Control` function to set socket options BEFORE bind()

**Rationale**:
- Socket options (SO_REUSEADDR, SO_REUSEPORT) MUST be set after `socket()` but BEFORE `bind()`
- `net.ListenConfig.Control` provides access to raw file descriptor at correct lifecycle point
- Industry standard pattern (used by gRPC, Kubernetes, Docker)
- Enables platform-specific socket option configuration via build tags

**Implementation**:
```go
lc := net.ListenConfig{
    Control: func(network, address string, c syscall.RawConn) error {
        return c.Control(func(fd uintptr) {
            // Platform-specific socket options (build tags)
            unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
            unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
        })
    },
}
conn, err := lc.ListenPacket(ctx, "udp4", "0.0.0.0:5353")
```

**References**: F-9 REQ-F9-1, ADR-001 (from M1-Refactoring provides transport interface foundation)

---

### AD-002: Platform-Specific Socket Option Files with Build Tags

**Decision**: Use separate files with build tags for Linux, macOS, Windows socket configuration

**Context**: Socket options differ fundamentally across operating systems:
- Linux kernel 3.9+: SO_REUSEADDR + SO_REUSEPORT
- macOS: SO_REUSEADDR + SO_REUSEPORT (BSD semantics)
- Windows: SO_REUSEADDR only (different semantics than POSIX, no SO_REUSEPORT)

**Alternatives Considered**:
1. ~~Runtime OS detection with if/else~~ - Fragile, verbose, error-prone
2. ~~Single file with `//go:build` tags inline~~ - Unreadable, hard to test
3. **CHOSEN**: Separate files per platform with build tags

**Rationale**:
- Compile-time safety (wrong platform code cannot compile)
- Clear separation of concerns (each platform in its own file)
- Standard Go convention (stdlib uses this pattern extensively)
- Easier testing (platform-specific test files mirror implementation)

**Implementation**:
```
internal/transport/
├── socket_linux.go      // +build linux
├── socket_darwin.go     // +build darwin
├── socket_windows.go    // +build windows
└── socket_test.go       // Tests use runtime.GOOS checks
```

**References**: F-9 REQ-F9-2, F-9 REQ-F9-9, F-2 (Package Structure)

---

### AD-003: DefaultInterfaces() with Pattern-Based VPN/Docker Exclusion

**Decision**: Implement `DefaultInterfaces()` helper that excludes VPN and Docker interfaces by naming pattern

**Context**: Binding to all interfaces has security/privacy implications:
- VPN interfaces (utun*, tun*, ppp*) leak mDNS queries to VPN provider (GDPR/HIPAA violation)
- Docker interfaces (docker0, veth*, br-*) waste resources on isolated networks
- Users expect "local network discovery" to mean physical interfaces only

**Alternatives Considered**:
1. ~~Bind to all interfaces (current M1 behavior)~~ - Privacy violation, user confusion
2. ~~Require explicit interface specification~~ - Poor UX, breaks "just works" expectation
3. **CHOSEN**: Smart defaults with VPN/Docker exclusion, override available

**Rationale**:
- 95%+ of VPN clients use standard naming (utun*, tun*, ppp*, wg*, tailscale*)
- Docker always uses docker0/veth*/br-* patterns
- Users can override via `WithInterfaces()` if needed
- "Secure by default" principle

**Implementation**:
```go
func DefaultInterfaces() ([]net.Interface, error) {
    allIfaces, _ := net.Interfaces()
    var filtered []net.Interface
    for _, iface := range allIfaces {
        if iface.Flags&net.FlagUp == 0 { continue }           // Skip down
        if iface.Flags&net.FlagMulticast == 0 { continue }    // Skip non-multicast
        if iface.Flags&net.FlagLoopback != 0 { continue }     // Skip loopback
        if isVPN(iface.Name) { continue }                     // Skip VPN
        if isDocker(iface.Name) { continue }                  // Skip Docker
        filtered = append(filtered, iface)
    }
    return filtered, nil
}
```

**References**: F-10 REQ-F10-2, F-10 REQ-F10-6, SC-003, SC-004

---

### AD-004: Per-Source-IP Rate Limiter with Bounded Map

**Decision**: Implement sliding window rate limiter with LRU eviction (max 10,000 sources tracked)

**Context**: Real-world incident (Hubitat bug, 2020) sent 1000+ queries/sec, crashed ESP32 devices. Need protection against:
- Buggy devices flooding multicast group
- Malicious DoS attacks
- Resource exhaustion

**Alternatives Considered**:
1. ~~No rate limiting~~ - Vulnerable to known attack pattern
2. ~~Global rate limit (all sources)~~ - Punishes legitimate traffic
3. ~~Unbounded per-source tracking~~ - Memory exhaustion attack vector
4. **CHOSEN**: Bounded per-source map with LRU eviction

**Rationale**:
- Per-source tracking isolates misbehaving devices
- Bounded map (10,000 entries) prevents memory exhaustion
- LRU eviction handles high-churn scenarios
- 100 qps threshold allows legitimate high-volume use cases

**Implementation**:
```go
type RateLimiter struct {
    threshold   int                        // 100 queries/sec default
    cooldown    time.Duration              // 60 seconds default
    sources     map[string]*RateLimitEntry // Max 10,000 entries
    mu          sync.RWMutex
}

func (rl *RateLimiter) Allow(sourceIP string) bool {
    // Sliding window: count queries in last 1 second
    // If > threshold, apply cooldown
    // If map size > 10,000, evict oldest
}
```

**References**: F-11 REQ-F11-2, F-7 (Resource Management), SC-005, SC-006

---

### AD-005: Source IP Validation Before Parsing

**Decision**: Validate source IP is link-local OR same subnet as receiving interface, silently drop invalid packets BEFORE parsing

**Context**: RFC 6762 §2 specifies link-local scope (no routing). Non-link-local sources indicate:
- Misconfiguration (router forwarding multicast)
- Spoofing attempt
- CPU waste parsing invalid packets

**Alternatives Considered**:
1. ~~Parse all packets~~ - Wastes CPU, violates RFC scope
2. ~~Log and continue~~ - Still wastes parsing time
3. **CHOSEN**: Drop before parsing, log at debug level

**Rationale**:
- Early rejection = CPU efficiency
- RFC compliance (link-local scope enforcement)
- Silent drop for invalid traffic (standard practice)
- Debug logging for troubleshooting

**Implementation**:
```go
func isLinkLocalSource(srcIP net.IP, iface net.Interface) bool {
    // Check: 169.254.0.0/16 (IPv4 link-local)
    // OR: Same subnet as interface address
    // OR: Private IP on same subnet
}

// In receive loop:
if !isLinkLocalSource(srcIP, iface) {
    logger.Debug("Dropped non-link-local packet", "source", srcIP)
    continue // Skip parsing
}
```

**References**: F-11 REQ-F11-1, RFC 6762 §2, SC-007

## Risk Analysis

### High Risk: Platform-Specific Socket Options

**Risk**: Socket option behavior differs across platforms and kernel versions
- Linux kernel <3.9: SO_REUSEPORT not supported or buggy
- Windows: SO_REUSEPORT does not exist (SO_REUSEADDR has different semantics)
- macOS: BSD socket behavior differs from Linux

**Mitigation**:
1. ✅ Kernel version detection on Linux (log warning if <3.9)
2. ✅ Platform-specific test files with build tags
3. ✅ Integration tests on all target platforms (CI/CD)
4. ✅ Graceful degradation (continue with SO_REUSEADDR only if SO_REUSEPORT fails)

**Residual Risk**: LOW (mitigated)

---

### Medium Risk: Interface Pattern Detection for VPN/Docker

**Risk**: Custom VPN configurations may use non-standard interface names
- Enterprise VPNs with custom naming
- Exotic Docker configurations
- Future VPN clients with unknown patterns

**Mitigation**:
1. ✅ User override via `WithInterfaces()` (explicit interface selection)
2. ✅ Comprehensive pattern coverage (utun*, tun*, ppp*, wg*, tailscale*, wireguard*)
3. ✅ Logging of filtered interfaces (user visibility)
4. ✅ Documentation of override options

**Residual Risk**: LOW (user can override, 95%+ coverage of common patterns)

---

### Medium Risk: Rate Limiter Performance Under Load

**Risk**: Per-source tracking with 10,000 entry map may impact performance
- Mutex contention on high-volume networks
- Map lookup overhead on every packet
- LRU eviction cost

**Mitigation**:
1. ✅ Read-write mutex (allows concurrent reads)
2. ✅ Configurable disable (`WithRateLimit(false)`)
3. ✅ Benchmark tests for 1000 qps scenario (SC-005)
4. ✅ LRU eviction batched (periodic cleanup, not per-packet)

**Residual Risk**: LOW (configurable, benchmarked)

---

### Low Risk: golang.org/x Dependencies

**Risk**: Semi-standard libraries may have breaking changes
- golang.org/x/sys may change syscall signatures
- golang.org/x/net may deprecate APIs

**Mitigation**:
1. ✅ Maintained by Go team (high trust, stable)
2. ✅ Version pinning in go.mod
3. ✅ Constitution Principle V justifies usage
4. ✅ Widely adopted (gRPC, Kubernetes use same deps)

**Residual Risk**: VERY LOW (industry-standard dependencies)

## Implementation Phases

### Phase 0: Research (1 hour)

**Objective**: Resolve any remaining unknowns about platform-specific socket options

**Tasks**:
1. Verify SO_REUSEPORT availability on target platforms:
   - Linux kernel 3.9+ documentation
   - macOS socket option behavior
   - Windows SO_REUSEADDR semantics
2. Research VPN interface naming conventions:
   - WireGuard, OpenVPN, Tailscale, L2TP, PPTP patterns
   - Validation against real-world VPN clients
3. Confirm `golang.org/x/sys` and `golang.org/x/net` API stability:
   - Review API documentation for socket options
   - Check for deprecation notices

**Output**: `research.md` documenting:
- Platform-specific socket option availability matrix
- Comprehensive VPN/Docker interface naming patterns
- golang.org/x API stability assessment

---

### Phase 1: Design & Contracts (2 hours)

**Objective**: Define data models, API contracts, and developer quick reference

**Tasks**:
1. **Data Model** (`data-model.md`):
   - `SocketConfig`: Platform, SO_REUSEADDR, SO_REUSEPORT, kernel version
   - `InterfaceFilter`: Explicit list, custom filter, default rules
   - `RateLimitEntry`: Source IP, query count, window start, cooldown expiry
   - `MulticastSocket`: Interface, FD, multicast group, TTL, loopback

2. **API Contracts** (`contracts/querier-options.md`):
   - `WithInterfaces([]net.Interface)`: Explicit interface selection
   - `WithInterfaceFilter(func(net.Interface) bool)`: Custom filtering
   - `WithRateLimit(bool)`: Enable/disable rate limiting
   - `WithRateLimitThreshold(int)`: Queries/second threshold (default 100)
   - `WithRateLimitCooldown(time.Duration)`: Cooldown duration (default 60s)

3. **Quickstart** (`quickstart.md`):
   - Default usage (smart interface selection, rate limiting enabled)
   - Explicit interface selection example
   - Rate limiting configuration example
   - VPN interface override example

**Output**:
- `data-model.md`
- `contracts/querier-options.md`
- `quickstart.md`

---

### Phase 2: Tasks (NOT in this command - use `/speckit.tasks`)

**Note**: Phase 2 (task generation) is handled by `/speckit.tasks` command separately after plan approval.

Expected task breakdown (for reference):
- **Phase 0**: Baseline (capture current metrics)
- **Phase 1 (RED)**: Write socket option tests (Linux, macOS, Windows)
- **Phase 1 (GREEN)**: Implement platform-specific socket files
- **Phase 1 (REFACTOR)**: Integrate socket options into UDPv4Transport
- **Phase 2 (RED)**: Write interface filtering tests
- **Phase 2 (GREEN)**: Implement DefaultInterfaces() + filtering
- **Phase 3 (RED)**: Write rate limiter tests
- **Phase 3 (GREEN)**: Implement rate limiter
- **Phase 4 (RED)**: Write source IP validation tests
- **Phase 4 (GREEN)**: Implement source IP filtering
- **Phase 5**: Integration tests (Avahi, VPN, storm simulation)
- **Phase 6**: Documentation, completion validation

## Next Steps

1. ✅ **Review this plan** with stakeholders
2. ⏭️ **Execute Phase 0**: Run research tasks (1 hour) to generate `research.md`
3. ⏭️ **Execute Phase 1**: Generate design artifacts (2 hours)
4. ⏭️ **Generate tasks**: Run `/speckit.tasks` to create `tasks.md` from this plan
5. ⏭️ **Implementation**: Execute tasks in TDD cycles (RED → GREEN → REFACTOR)

## References

- **Feature Spec**: [spec.md](./spec.md)
- **F-Specs**: F-9 (Socket), F-10 (Interface), F-11 (Security)
- **Foundation**: M1-Refactoring transport interface (ADR-001, ADR-002)
- **ROADMAP**: Target completion 2025-11-15 (25 hour estimate)
- **Constitution**: v1.1.0 (Principle V justifies golang.org/x dependencies)
