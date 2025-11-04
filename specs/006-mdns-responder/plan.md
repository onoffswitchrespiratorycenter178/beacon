# Implementation Plan: mDNS Responder

**Branch**: `006-mdns-responder` | **Date**: 2025-11-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-mdns-responder/spec.md`

---

## Summary

Implement mDNS Responder functionality enabling service registration, name conflict detection via probing, and query response per RFC 6762. This builds on the M1.1 Foundation (query-only) to add bidirectional mDNS capabilities. The responder will allow applications to advertise services on the local network, automatically resolve name conflicts, and respond to queries from other devices. Key technical challenges include implementing the Probing→Announcing→Established state machine, tie-breaking logic for simultaneous probes, TTL-aware response generation, and known-answer suppression to minimize network traffic.

---

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Standard library + `golang.org/x/sys` (platform-specific socket options from M1.1), `golang.org/x/net` (multicast group management from M1.1)
**Storage**: In-memory (registered services, resource record sets with TTLs)
**Testing**: Go standard testing, integration tests with Avahi/Bonjour, Apple Bonjour Conformance Test (BCT), fuzz testing
**Target Platform**: Linux (primary), macOS, Windows (code-complete from M1.1)
**Project Type**: Library (Go package)
**Performance Goals**: <100ms query response latency (90th percentile), support ≥100 concurrent service registrations
**Constraints**:
- RFC 6762 compliance (probing timing: 3 queries × 250ms, announcing: 2 packets × 1s)
- 9000 byte packet size limit (RFC 6762 §17)
- TTL constraints (120s service records, 75min hostname records)
- Thread-safe concurrent registration/query handling

**Scale/Scope**:
- Support 100+ services per responder instance
- Coexist with 50+ other mDNS services on network
- Handle 100+ queries/sec per service without degradation

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: RFC Compliant ✅ PASS

**Validation**:
- Spec references RFC 6762 sections throughout (§8.1 Probing, §8.3 Announcing, §10 TTL, §7.1 Known-Answer Suppression)
- All MUST requirements from RFC 6762 responder sections are captured in FRs
- Apple BCT (Bonjour Conformance Test) validation required (SC-003)
- No deviations from RFC MUST requirements

**Evidence**: FR-006 through FR-024 implement RFC 6762 §8 (Probing and Announcing), FR-019 through FR-021 implement §10 (TTL Handling), FR-017 implements §7.1 (Known-Answer Suppression)

### Principle II: Spec-Driven Development ✅ PASS

**Validation**:
- Complete specification exists: `specs/006-mdns-responder/spec.md`
- 5 prioritized user stories with acceptance scenarios
- 35 functional requirements organized by category
- 12 success criteria with measurable outcomes
- Specification validated via requirements checklist

**Evidence**: spec.md includes User Scenarios, Requirements, Success Criteria, Assumptions, Dependencies sections as mandated by spec-template.md

### Principle III: Test-Driven Development ✅ PASS

**Validation**:
- TDD approach planned: RED (write tests for state machine, probing, conflict detection) → GREEN (implement until tests pass) → REFACTOR
- Coverage target: ≥80% (SC-005)
- Race detector required: `go test -race` (SC-008, NFR-005)
- Integration tests with Avahi/Bonjour planned (SC-004)
- Apple BCT test suite integration (SC-003)

**Evidence**: Success criteria SC-005 (≥80% coverage), SC-008 (zero races), SC-003 (BCT pass), SC-004 (Avahi/Bonjour interoperability)

### Principle IV: Phased Approach ✅ PASS

**Validation**:
- Milestone M2 follows M1 (Querier) and M1.1 (Foundation)
- Builds on existing architecture (M1.1 transport, interfaces, security)
- User stories prioritized (P1: Registration & Conflict Resolution, P2: Query Response & Multi-Service, P3: Cache Coherency)
- Incremental testability: each user story independently testable

**Evidence**: User stories include "Independent Test" criteria, dependencies section lists M1.1 prerequisites, phased implementation via P1→P2→P3 priorities

### Principle V: Dependencies and Supply Chain ✅ PASS

**Validation**:
- Uses existing `golang.org/x/sys` and `golang.org/x/net` from M1.1 (already justified for socket options and multicast management)
- No new external dependencies required
- Standard library for all other functionality (state machines, TTL management, conflict detection)

**Evidence**: FR-028 requires using M1.1 transport layer (which already uses golang.org/x/sys for SO_REUSEPORT), FR-029 requires M1.1 interface management (which uses golang.org/x/net for multicast groups)

### Principle VI: Open Source ✅ PASS

**Validation**:
- MIT License (existing)
- Public development (GitHub)
- Community contributions welcome (testing on macOS/Windows needed)

### Principle VII: Maintained ✅ PASS

**Validation**:
- Active development (Foundation just completed)
- Follows Spec Kit methodology
- Documentation maintained (compliance dashboard, FR matrix)

### Principle VIII: Excellence ✅ PASS

**Validation**:
- Production-ready quality targets (Apple BCT, Avahi/Bonjour interoperability)
- Zero crashes, zero races (fuzz tested, race detector)
- Comprehensive edge case handling (7 edge cases identified in spec)
- Performance benchmarks (known-answer suppression 30% reduction)

**Overall Constitution Compliance**: ✅ **PASS** - No violations, all principles satisfied

---

## Project Structure

### Documentation (this feature)

```text
specs/006-mdns-responder/
├── spec.md              # Feature specification (COMPLETE)
├── plan.md              # This file (IN PROGRESS)
├── research.md          # Phase 0 output (PENDING)
├── data-model.md        # Phase 1 output (PENDING)
├── quickstart.md        # Phase 1 output (PENDING)
├── contracts/           # Phase 1 output (PENDING)
│   ├── responder-api.md # Responder API contract
│   └── state-machine.md # State machine transitions contract
├── checklists/
│   └── requirements.md  # Requirements validation checklist (COMPLETE)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Single Go library project (existing from M1/M1.1)

responder/                     # NEW - Public responder API
├── responder.go              # Responder type, Register/Unregister methods
├── responder_test.go         # Unit tests
├── service.go                # Service type (instance name, type, port, TXT)
├── service_test.go           # Service unit tests
└── options.go                # Functional options (WithHostname, etc.)

internal/
├── state/                    # NEW - State machine implementation
│   ├── machine.go           # State machine (Probing → Announcing → Established)
│   ├── machine_test.go      # State machine unit tests
│   ├── prober.go            # Probing logic (3 queries × 250ms, conflict detection)
│   ├── prober_test.go       # Prober unit tests
│   ├── announcer.go         # Announcing logic (2 unsolicited announcements × 1s)
│   └── announcer_test.go    # Announcer unit tests
│
├── responder/                # NEW - Internal responder logic
│   ├── registry.go          # Service registry (thread-safe map)
│   ├── registry_test.go     # Registry unit tests
│   ├── conflict.go          # Conflict detector, tie-breaking logic
│   ├── conflict_test.go     # Conflict detection tests
│   ├── response_builder.go # Response packet builder (PTR, SRV, TXT, A records)
│   └── response_builder_test.go
│
├── records/                  # NEW - Resource record management
│   ├── record_set.go        # Resource record set (PTR, SRV, TXT, A with TTLs)
│   ├── record_set_test.go   # Record set tests
│   ├── ttl.go               # TTL management (creation timestamp, elapsed calculation)
│   └── ttl_test.go          # TTL tests
│
├── transport/                # EXISTING from M1.1 - extend for responses
│   ├── transport.go         # Transport interface (already exists)
│   ├── udp.go               # UDPv4Transport - extend Send for responses
│   └── udp_test.go          # Extended tests for response sending
│
├── message/                  # EXISTING from M1 - extend for response building
│   ├── builder.go           # Extend for response messages (AA=1, QR=1)
│   ├── builder_test.go      # Response builder tests
│   └── parser.go            # Already handles responses - no changes needed
│
└── protocol/                 # EXISTING from M1 - add response constants
    └── constants.go         # Add TTL constants (120s, 75min)

tests/
├── contract/                 # RFC compliance tests
│   ├── rfc6762_probing_test.go    # §8.1 probing compliance
│   ├── rfc6762_announcing_test.go # §8.3 announcing compliance
│   └── rfc6762_ttl_test.go        # §10 TTL handling compliance
│
├── integration/              # Real network tests
│   ├── avahi_test.go        # Avahi coexistence/interoperability (Linux)
│   ├── bonjour_test.go      # Bonjour coexistence (macOS)
│   ├── conflict_test.go     # Multi-instance conflict resolution
│   └── query_response_test.go # Query→response integration
│
└── fuzz/                     # Fuzz testing (extend existing)
    └── responder_test.go    # Fuzz service registration, response building
```

**Structure Decision**: Extends existing M1/M1.1 single Go library structure with new `responder/` public package and `internal/state/`, `internal/responder/`, `internal/records/` packages. Existing `internal/transport/`, `internal/message/`, `internal/protocol/` packages are reused and extended where necessary. Maintains clean architecture with strict layer boundaries per F-2.

---

## Complexity Tracking

*No violations to justify - all Constitution principles pass without exceptions.*

---

## Phase 0: Research & Discovery

### Research Tasks

#### R001: RFC 6762 §8 Probing & Announcing Implementation Patterns

**Goal**: Understand best practices for implementing the probing/announcing state machine

**Questions**:
- How do production implementations (Avahi, Bonjour) handle the 3-probe sequence with 250ms intervals?
- What are common pitfalls in conflict detection during simultaneous probing?
- How should the state machine handle queries received during probing vs. announcing states?

**Method**:
- Read RFC 6762 §8.1 (Probing), §8.2 (Tiebreaking), §8.3 (Announcing) in detail
- Review Avahi source code (C) for probing state machine patterns
- Review Apple's mDNSResponder (Bonjour) source code for announcing logic
- Check Apple BCT test cases for edge cases

**Deliverable**: Decision on state machine architecture (FSM with timeout channels, goroutine-per-service, or event-driven)

---

#### R002: Tie-Breaking Algorithm for Simultaneous Probes

**Goal**: Understand RFC 6762 §8.2.1 lexicographic comparison for conflict resolution

**Questions**:
- What is the exact byte-by-byte comparison algorithm for tie-breaking?
- How do you compare SRV record data (priority, weight, port, target) lexicographically?
- What happens if two probes have identical record data?

**Method**:
- Read RFC 6762 §8.2.1 ("Simultaneous Probe Tiebreaking")
- Review RFC 1035 §3.3.14 (SRV record wire format)
- Test with two Beacon instances probing the same name simultaneously

**Deliverable**: Decision on tie-breaking implementation (byte-slice comparison, record field comparison, or wire-format comparison)

---

#### R003: Known-Answer Suppression Implementation

**Goal**: Optimize response generation by suppressing duplicate records per RFC 6762 §7.1

**Questions**:
- How do you parse the known-answer section from a query?
- What is the TTL threshold for suppression (must be > half of correct TTL)?
- How do you efficiently check if a record matches a known-answer?

**Method**:
- Read RFC 6762 §7.1 ("Known-Answer Suppression")
- Check if M1 parser already extracts answer section from queries (it does for responses)
- Review efficiency: O(n×m) comparison vs. hash-based lookup

**Deliverable**: Decision on known-answer matching algorithm (record equality check, hash-based deduplication)

---

#### R004: TTL Management and Expiry Tracking

**Goal**: Implement RFC 6762 §10 TTL handling for service records and hostname records

**Questions**:
- How do you track TTL expiry for registered services (120s for services, 75min for hostnames)?
- Should TTL be decremented in real-time or calculated on-demand when building responses?
- How do you handle TTL=0 goodbye packets vs. normal unregistration?

**Method**:
- Read RFC 6762 §10 ("Resource Record TTL Values and Cache Coherency")
- Review how existing querier handles TTL in responses (M1 parser extracts TTL)
- Consider trade-offs: background goroutine for expiry vs. lazy calculation

**Deliverable**: Decision on TTL tracking mechanism (creation timestamp + elapsed calculation vs. background expiry goroutine)

---

#### R005: Response Aggregation and Packet Size Limits

**Goal**: Fit multiple resource records into a single response packet while respecting 9000-byte limit (RFC 6762 §17)

**Questions**:
- How do you calculate wire format size before encoding the packet?
- What happens if PTR + SRV + TXT + A records exceed 9000 bytes?
- Should additional records be omitted or split into multiple response packets?

**Method**:
- Read RFC 6762 §17 ("Multicast DNS Message Size")
- Review M1 message builder for size calculation logic (does it exist?)
- Check RFC 6762 §6 ("Multicast DNS Responses") for additional record guidelines

**Deliverable**: Decision on size-overflow handling (omit additional records, split into multiple responses, or truncate TXT records)

---

#### R006: Thread Safety for Concurrent Service Registration

**Goal**: Ensure concurrent Register/Unregister/Query handling is safe

**Questions**:
- What concurrency primitives should protect the service registry (mutex, RWMutex, sync.Map)?
- How do you isolate state machines for each service (goroutine-per-service, single coordinator goroutine)?
- How do you prevent race conditions between probing goroutines and query handlers?

**Method**:
- Review Go concurrency best practices (mutexes vs. channels)
- Test with `go test -race` under concurrent load
- Consider F-2 architecture boundaries (single lock vs. per-service locks)

**Deliverable**: Decision on concurrency model (single registry lock, per-service locks, or lock-free channel-based coordination)

---

### Research Consolidation

**Output**: `research.md` file with all decisions documented using format:

```markdown
## R001: State Machine Architecture

**Decision**: Event-driven state machine with goroutine-per-service and timeout channels

**Rationale**: Avahi uses event loop with timers; Bonjour uses dispatch queues. Go's goroutines + select on timeout channels provide equivalent functionality with cleaner code. Each service gets its own goroutine running the state machine, avoiding shared state.

**Alternatives Considered**:
- Single coordinator goroutine with event queue: More complex, harder to test
- FSM library (e.g., looplab/fsm): Overkill for 3-state machine, adds dependency
```

(Repeat for R002-R006)

---

## Phase 1: Design & Contracts

### D001: Data Model (`data-model.md`)

**Entities** (from spec.md Key Entities section):

#### Service Instance

**Attributes**:
- `InstanceName` (string): e.g., "MyApp", "MyApp (2)"
- `ServiceType` (string): e.g., "_http._tcp.local"
- `Port` (uint16): e.g., 8080
- `TXTRecords` ([]string): key=value metadata, e.g., ["version=1.0", "path=/api"]
- `Hostname` (string): e.g., "mydevice.local"
- `State` (enum): Probing | Announcing | Established
- `CreatedAt` (time.Time): For TTL calculation
- `ProbesSent` (int): Count of probes sent (0-3)

**Relationships**:
- Has one ResourceRecordSet
- Managed by one Responder

**Validation Rules** (from FRs):
- FR-002: ServiceType must match `_[a-z0-9-]+\._tcp\.local` or `_[a-z0-9-]+\._udp\.local`
- FR-003: InstanceName + ServiceType + "local" = fully-qualified name
- Port must be 1-65535

**State Transitions**:
- Probing → Announcing: After 3 successful probes (no conflicts)
- Announcing → Established: After 2 announcements (1 second apart)
- Any state → Probing: On conflict detected (rename and restart)
- Any state → [deleted]: On Unregister (send goodbye packets)

---

#### Resource Record Set

**Attributes**:
- `PTR` (DNSRecord): Points from service type to instance name
- `SRV` (DNSRecord): Contains hostname, port, priority=0, weight=0
- `TXT` (DNSRecord): Contains TXT records (metadata)
- `A` (DNSRecord): Contains IPv4 address of hostname
- `AAAA` (DNSRecord): Contains IPv6 address (future - M3)
- `TTL` (map[RecordType]time.Duration): Per-record TTL (120s for services, 75min for hostname)
- `CreatedAt` (time.Time): Timestamp for TTL expiry calculation

**Relationships**:
- Owned by one Service Instance

**Validation Rules**:
- FR-019: Default TTL 120s for PTR/SRV/TXT, 75 minutes for A/AAAA
- FR-020: TTL decrements based on elapsed time since CreatedAt
- FR-021: TTL=0 for goodbye packets

---

#### State Machine

**States**:
- `Probing`: Sending 3 queries for proposed name, waiting for conflicts
- `Announcing`: Broadcasting 2 unsolicited announcements, 1 second apart
- `Established`: Responding to queries for registered service

**Events**:
- `StartProbe`: Begin probing sequence
- `ProbeTimeout`: 250ms elapsed, send next probe
- `ConflictDetected`: Received response during probing → rename and restart
- `ProbingComplete`: All 3 probes sent, no conflicts → transition to Announcing
- `AnnouncementTimeout`: 1 second elapsed, send next announcement
- `AnnouncingComplete`: 2 announcements sent → transition to Established
- `QueryReceived`: Incoming query for this service
- `Unregister`: Application requests unregistration → send goodbye packets

**Transitions** (from FRs):
- FR-022: Probing → Announcing (on ProbingComplete)
- FR-022: Announcing → Established (on AnnouncingComplete)
- FR-023: Probing + QueryReceived → queue response until Established
- FR-023: Announcing + QueryReceived → respond immediately
- FR-024: Probing + ConflictDetected → rename, restart Probing

---

#### Conflict Detector

**Purpose**: Detect name conflicts during probing and apply tie-breaking logic per RFC 6762 §8.2.1

**Inputs**:
- Proposed service name
- Our probe record data (SRV: priority, weight, port, target)
- Received probe record data (from simultaneous probe)

**Outputs**:
- Conflict: true/false
- Winner: us/them (if simultaneous probe)

**Algorithm** (RFC 6762 §8.2.1):
1. If ANY response received for proposed name during probing → conflict (FR-007)
2. If simultaneous probe (query from another prober):
   - Compare record data lexicographically (byte-by-byte wire format)
   - Lower lexicographic value wins
   - If we lose: conflict, rename and restart (FR-009)
   - If we win: continue probing
3. If identical record data: both continue (extremely rare)

---

#### Response Builder

**Purpose**: Construct mDNS response packets with appropriate records

**Inputs**:
- Query (parsed DNS message): QNAME, QTYPE, known-answers section
- Registered services (from registry)

**Outputs**:
- Response packet ([]byte): DNS wire format with AA=1, QR=1

**Logic**:
- FR-012: If query matches service type (PTR query) → include PTR, SRV, TXT, A records
- FR-013: For PTR queries, include SRV, TXT, A in additional section (reduce round-trips)
- FR-016: If QU bit set → send unicast response, else multicast
- FR-017: Apply known-answer suppression (omit records if in query's answer section with TTL > half correct TTL)
- FR-018: Aggregate multiple records into single packet, respect 9000-byte limit
- FR-020: Decrement TTL based on elapsed time since record creation

---

### D002: API Contracts (`contracts/responder-api.md`)

**Public Responder API** (from spec User Stories):

```go
// Package responder provides mDNS Responder functionality for service registration
package responder

// Responder represents an mDNS responder instance
type Responder struct {
	// internal fields (not exposed)
}

// New creates a new mDNS responder
//
// Options:
// - WithHostname(string): Set custom hostname (default: os.Hostname())
// - WithInterfaces([]net.Interface): Use specific interfaces (reuse M1.1 option)
// - WithInterfaceFilter(func): Custom interface filter (reuse M1.1 option)
//
// Returns error if:
// - No suitable network interfaces found
// - Transport layer fails to initialize
func New(opts ...Option) (*Responder, error)

// Register registers an mDNS service
//
// Parameters:
// - service: Service instance with name, type, port, TXT records
//
// Behavior (from User Story 1):
// 1. Validates service type format (FR-002)
// 2. Starts probing (3 queries × 250ms) - FR-006
// 3. Detects conflicts, renames if needed (FR-007, FR-009)
// 4. Announces service (2 packets × 1s) - FR-011
// 5. Transitions to Established state - FR-022
//
// Returns error if:
// - Service type invalid
// - Name unavailable after 10 rename attempts (FR-032)
// - Responder already closed
func (r *Responder) Register(ctx context.Context, service *Service) error

// UpdateService updates TXT records for an already-registered service
//
// Does NOT re-probe (FR-004)
// Sends announcement with new TXT records
func (r *Responder) UpdateService(instanceName string, txtRecords []string) error

// Unregister unregisters a service
//
// Sends goodbye packets (TTL=0) - FR-005, FR-014, FR-021
// Removes service from registry
func (r *Responder) Unregister(instanceName string) error

// Close closes the responder
//
// Sends goodbye packets for all registered services (FR-014)
// Shuts down transport layer
func (r *Responder) Close() error

// Service represents an mDNS service instance
type Service struct {
	InstanceName string   // e.g., "MyApp"
	ServiceType  string   // e.g., "_http._tcp.local"
	Port         uint16   // e.g., 8080
	TXTRecords   []string // e.g., ["version=1.0", "path=/api"]
	// Hostname is set by Responder (defaults to os.Hostname())
}
```

**Example Usage**:

```go
// From User Story 1: Service Registration
resp, err := responder.New()
if err != nil {
	log.Fatal(err)
}
defer resp.Close()

service := &responder.Service{
	InstanceName: "MyApp",
	ServiceType:  "_http._tcp.local",
	Port:         8080,
	TXTRecords:   []string{"version=1.0", "path=/api"},
}

// Blocks until probing completes (~750ms) or context canceled
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := resp.Register(ctx, service); err != nil {
	log.Fatalf("Failed to register service: %v", err)
}

// Service is now discoverable (User Story 1, Acceptance Scenario 1)
```

---

### D003: State Machine Contract (`contracts/state-machine.md`)

**State Machine Specification**:

```
States: Probing, Announcing, Established

Initial State: Probing (on Register call)

Transitions:

1. Probing → Announcing
   Trigger: All 3 probes sent (750ms elapsed), no conflicts detected
   Actions: Reset probe counter, start announcing timer
   Guard: probesSent == 3 && !conflictDetected

2. Announcing → Established
   Trigger: 2 announcements sent (1 second elapsed)
   Actions: Mark service as established, stop announcement timer
   Guard: announcementsSent == 2

3. Any → Probing (Conflict Restart)
   Trigger: Conflict detected during probing
   Actions: Rename service (append " (2)"), reset state, restart probing
   Guard: conflictDetected == true

4. Any → [Deleted] (Unregister)
   Trigger: Unregister() called
   Actions: Send goodbye packets (TTL=0), remove from registry
   Guard: none

Events:

- ProbeTimeout (250ms timer): Send next probe, increment probesSent
- AnnouncementTimeout (1s timer): Send next announcement, increment announcementsSent
- QueryReceived:
  - If Probing: Queue response until Established
  - If Announcing or Established: Respond immediately
- ConflictDetected: Trigger conflict restart transition
```

**Implementation Sketch** (from R001 research decision):

```go
// Goroutine-per-service state machine
func (r *Responder) runStateMachine(svc *Service) {
	state := Probing
	probesSent := 0
	announcementsSent := 0

	probeTicker := time.NewTicker(250 * time.Millisecond)
	announceTicker := time.NewTicker(1 * time.Second)
	defer probeTicker.Stop()
	defer announceTicker.Stop()

	for {
		select {
		case <-probeTicker.C:
			if state == Probing {
				r.sendProbe(svc)
				probesSent++
				if probesSent >= 3 {
					state = Announcing
					announcementsSent = 0
				}
			}
		case <-announceTicker.C:
			if state == Announcing {
				r.sendAnnouncement(svc)
				announcementsSent++
				if announcementsSent >= 2 {
					state = Established
				}
			}
		case query := <-r.queryQueue:
			r.handleQuery(query, svc, state)
		case <-r.closeChan:
			r.sendGoodbye(svc)
			return
		}
	}
}
```

---

### D004: Quickstart Guide (`quickstart.md`)

**Purpose**: Help developers quickly understand how to use the mDNS Responder API

**Content**:

```markdown
# mDNS Responder Quickstart

## Register a Service

Register an HTTP service on port 8080:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/joshuafuller/beacon/responder"
)

func main() {
	// Create responder (uses default interfaces from M1.1)
	resp, err := responder.New()
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Close()

	// Define service
	service := &responder.Service{
		InstanceName: "MyWebServer",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
		TXTRecords:   []string{"path=/", "version=1.0"},
	}

	// Register (blocks ~750ms for probing)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := resp.Register(ctx, service); err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	log.Println("Service registered and discoverable!")

	// Keep running to respond to queries
	select {} // Block forever (or until signal)
}
```

## Handle Name Conflicts

If another service named "MyWebServer" exists, Beacon automatically renames to "MyWebServer (2)":

```go
// No code changes needed - conflict resolution is automatic (FR-009)
// The registered service will be "MyWebServer (2)" if "MyWebServer" is taken
```

## Unregister a Service

Gracefully remove a service from the network:

```go
// Sends goodbye packets (TTL=0) per RFC 6762 §10.1
if err := resp.Unregister("MyWebServer"); err != nil {
	log.Printf("Unregister failed: %v", err)
}

// Or close responder to unregister all services
resp.Close()
```

## Advanced: Custom Hostname

Override the default hostname (os.Hostname()):

```go
resp, err := responder.New(
	responder.WithHostname("custom-device.local"),
)
```

## Advanced: Multiple Services

Register multiple services on different ports:

```go
httpService := &responder.Service{
	InstanceName: "MyApp HTTP",
	ServiceType:  "_http._tcp.local",
	Port:         8080,
}

sshService := &responder.Service{
	InstanceName: "MyApp SSH",
	ServiceType:  "_ssh._tcp.local",
	Port:         22,
}

resp.Register(ctx, httpService)
resp.Register(ctx, sshService) // Independent probing/announcing
```
```

---

### D005: Update Agent Context

Run the agent context update script to add M2 technology to Claude's context:

```bash
.specify/scripts/bash/update-agent-context.sh claude
```

This will update `CLAUDE.md` with:
- New package: `responder/` (public API)
- New internal packages: `internal/state/`, `internal/responder/`, `internal/records/`
- Extended packages: `internal/transport/` (response sending), `internal/message/` (response building)
- M2 testing approach (Apple BCT, Avahi/Bonjour interoperability)

---

## Phase 2: Task Breakdown (To be generated by `/speckit.tasks`)

*This section will be populated by running `/speckit.tasks` after this plan is complete. Tasks will be organized by user story priority (P1, P2, P3) following TDD methodology.*

**Expected Task Structure**:
- Setup tasks (directories, contracts validation)
- P1 tasks: Service Registration + Name Conflict Resolution (User Stories 1-2)
- P2 tasks: Response to Queries + Multi-Service (User Stories 3, 5)
- P3 tasks: Cache Coherency (User Story 4)
- Integration tasks (Apple BCT, Avahi/Bonjour tests)
- Final validation tasks (coverage, RFC compliance)

---

## Risk Analysis

### High Risk

1. **Apple BCT Availability** (Mitigation: Document test cases from RFC 6762, create manual test suite)
2. **macOS/Windows Testing** (Mitigation: Code-complete with build tags, rely on community testing)
3. **Tie-Breaking Edge Cases** (Mitigation: Fuzz test with two Beacon instances probing simultaneously)

### Medium Risk

1. **TTL Precision** (Mitigation: Unit tests for TTL calculation, integration tests with real queries)
2. **Packet Size Overflow** (Mitigation: Size calculation before encoding, tests with 20+ TXT records)
3. **State Machine Races** (Mitigation: `go test -race`, stress tests with 100 concurrent registrations)

### Low Risk

1. **Known-Answer Suppression Efficiency** (Mitigation: Benchmark tests, profile if needed)
2. **Goodbye Packet Delivery** (Mitigation: Integration tests verify service disappears within 1s)

---

## Success Validation

**Criteria from spec.md mapped to validation approach**:

- SC-001 (2s discoverability) → Integration test: Register service, query from Avahi within 2s
- SC-002 (100% conflict detection) → Integration test: Run two Beacon instances with same name
- SC-003 (Apple BCT pass) → Run BCT test suite, document results
- SC-004 (Avahi/Bonjour coexist) → Integration test: Beacon + Avahi on same host, both respond
- SC-005 (≥80% coverage) → `go test -cover`, enforce in CI
- SC-006 (<100ms response) → Benchmark test: Query → response latency
- SC-007 (1s goodbye) → Integration test: Unregister, verify service disappears in browser
- SC-008 (100 concurrent) → Stress test: Register 100 services concurrently with `-race`
- SC-009 (30% suppression) → Benchmark: Repeated queries with/without known-answers
- SC-010 (50+ services) → Interoperability test: Beacon on network with 50+ Avahi services
- SC-011 (70% RFC compliance) → Update RFC_COMPLIANCE_MATRIX.md, recalculate percentage
- SC-012 (All MUST implemented) → Checklist: §8.1, §8.3, §10.1 requirements

---

**Plan Version**: 1.0
**Status**: Phase 0 Research Required
**Next Command**: Research tasks → `/speckit.plan` (auto-continues to Phase 1)
