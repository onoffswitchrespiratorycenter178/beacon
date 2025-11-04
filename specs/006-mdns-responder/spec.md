# Feature Specification: mDNS Responder

**Feature Branch**: `006-mdns-responder`
**Created**: 2025-11-02
**Status**: Draft
**Input**: User description: "Implement mDNS Responder functionality for service registration, probing, and announcing per RFC 6762"

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Service Registration (Priority: P1)

A developer wants to register their application as an mDNS service so that other devices on the local network can discover it. The service needs a unique name, service type, port number, and optional metadata (TXT records).

**Why this priority**: Core MVP functionality - without service registration, there's no responder capability. This is the foundation that all other stories build upon.

**Independent Test**: Can be fully tested by registering a service (e.g., `_http._tcp.local`) and verifying it appears in another mDNS browser (like Avahi, Bonjour Browser, or Beacon's own querier). Delivers immediate value: basic service advertisement.

**Acceptance Scenarios**:

1. **Given** no services are registered, **When** application registers a service with name "MyApp", type "_http._tcp.local", port 8080, **Then** the service is discoverable via mDNS queries
2. **Given** a service is successfully registered, **When** the service updates its TXT records (metadata), **Then** browsers see the updated information within the TTL window
3. **Given** a service is registered, **When** the application gracefully shuts down, **Then** a goodbye packet is sent and the service disappears from browsers within 1 second

---

### User Story 2 - Name Conflict Resolution (Priority: P1)

When registering a service, the name might already be in use by another device on the network. The system must detect this conflict during probing and automatically select an alternative name.

**Why this priority**: Critical for RFC compliance and avoiding broken service discovery. Without conflict resolution, services can silently fail or cause network confusion. Required by Apple BCT.

**Independent Test**: Can be tested by running two instances of Beacon trying to register the same service name. The second instance should detect the conflict and rename itself (e.g., "MyApp" → "MyApp (2)"). Delivers value: reliable, automatic conflict handling.

**Acceptance Scenarios**:

1. **Given** a service named "MyApp" exists on the network, **When** a new instance tries to register "MyApp", **Then** the new instance probes, detects the conflict, and registers as "MyApp (2)"
2. **Given** multiple services try to register the same name simultaneously, **When** they all probe concurrently, **Then** tie-breaking logic ensures each gets a unique name deterministically
3. **Given** a conflict is detected during probing, **When** the alternative name is also taken, **Then** the system continues trying sequential numbers until finding an available name

---

### User Story 3 - Response to Queries (Priority: P2)

When other devices query for services (e.g., "_http._tcp.local" PTR queries), the responder must send appropriate answers including PTR, SRV, TXT, and A/AAAA records.

**Why this priority**: Completes the query/response cycle but depends on Story 1 (registration) being functional. Without registration, there's nothing to respond about.

**Independent Test**: Can be tested by registering a service and sending mDNS queries from another device. Verify the responder sends correct PTR (service enumeration), SRV (host:port), TXT (metadata), and A (IPv4 address) records. Delivers value: full discoverability.

**Acceptance Scenarios**:

1. **Given** a service "_http._tcp.local" is registered, **When** a device sends a PTR query for "_http._tcp.local", **Then** the responder sends PTR, SRV, TXT, and A records in the response
2. **Given** a query has the QU bit set (unicast response requested), **When** the responder processes it, **Then** the response is sent via unicast, not multicast
3. **Given** a query includes known-answers, **When** the responder processes it, **Then** duplicate records matching known-answers are suppressed from the response

---

### User Story 4 - Cache Coherency (Priority: P3)

To minimize network traffic, the responder implements known-answer suppression and respects TTL values. When a query includes records the querier already knows about, those records are not re-sent.

**Why this priority**: Optimization for network efficiency, not critical for basic functionality. Can be added after core query/response works.

**Independent Test**: Can be tested by sending queries with known-answer sections and verifying the responder doesn't re-send those records. Measure network traffic reduction. Delivers value: reduced bandwidth usage.

**Acceptance Scenarios**:

1. **Given** a query includes known-answer records matching our service, **When** the responder processes the query, **Then** those records are omitted from the response
2. **Given** a resource record has a TTL of 120 seconds, **When** 60 seconds elapse, **Then** the responder still advertises the record with remaining TTL of 60 seconds
3. **Given** a service is about to be unregistered, **When** sending goodbye packets, **Then** all records are sent with TTL=0

---

### User Story 5 - Multi-service Support (Priority: P2)

An application may want to advertise multiple services simultaneously (e.g., both HTTP and SSH), each with different ports and metadata.

**Why this priority**: Common real-world need (servers often expose multiple services), but can be deferred until single-service registration works perfectly.

**Independent Test**: Can be tested by registering 2-3 services with different types and verifying each is independently discoverable and queryable. Delivers value: realistic deployment scenarios.

**Acceptance Scenarios**:

1. **Given** no services are registered, **When** application registers both "_http._tcp.local" and "_ssh._tcp.local", **Then** both services are independently discoverable
2. **Given** multiple services are registered, **When** one service is updated or unregistered, **Then** other services remain unaffected
3. **Given** 5 services are registered, **When** a PTR query for "_services._dns-sd._udp.local" is received, **Then** all 5 service types are listed in the response

---

### Edge Cases

- **Rapid service churn**: What happens when a service is registered, unregistered, and re-registered within a few seconds? (TTL handling, cache confusion)
- **Network interface changes**: How does the responder handle a network interface going down and coming back up? (Re-announce, re-probe?)
- **Malformed queries**: How does the responder handle queries with invalid DNS encoding or non-link-local source addresses? (Leverage M1.1 source filtering)
- **Probe timeout**: What happens if no response is received during the probing phase after 250ms × 3 probes? (Assume name is available, proceed to announcing)
- **Concurrent registration**: How does the system handle 10+ services being registered simultaneously from different threads? (Thread-safety, state machine isolation)
- **TTL edge case**: What happens when a record's TTL reaches 0 while a response is being constructed? (Omit from response, mark as stale)
- **Maximum packet size**: How does the responder handle a service with 20+ TXT records exceeding 9000 bytes? (Split responses, omit additional records, or truncate per RFC 6762 §17)

---

## Requirements *(mandatory)*

### Functional Requirements

#### Service Registration

- **FR-001**: System MUST allow registration of mDNS services with service type, instance name, port number, and optional TXT records
- **FR-002**: System MUST validate service type format (e.g., "_http._tcp.local", not "http.local") per RFC 6763 §7
- **FR-003**: System MUST assign a unique fully-qualified service name (instance + type + "local") to each registered service
- **FR-004**: System MUST support updating TXT records for an already-registered service without re-probing
- **FR-005**: System MUST allow unregistering a service, sending goodbye packets (TTL=0) on all registered records

#### Probing & Name Conflict Detection

- **FR-006**: System MUST perform probing by sending 3 queries for the proposed name, spaced 250ms apart, before announcing (RFC 6762 §8.1)
- **FR-007**: System MUST detect conflicts if ANY response is received during probing for the proposed name
- **FR-008**: System MUST implement tie-breaking logic when simultaneous probes occur (compare record data lexicographically per RFC 6762 §8.2.1)
- **FR-009**: System MUST rename services automatically upon conflict detection (e.g., "MyApp" → "MyApp (2)" → "MyApp (3)")
- **FR-010**: System MUST support concurrent probing for multiple services simultaneously without interference

#### Announcing & Continuous Advertisement

- **FR-011**: System MUST send unsolicited announcements after successful probing: 2 announcements, 1 second apart (RFC 6762 §8.3)
- **FR-012**: System MUST respond to queries with PTR, SRV, TXT, and A/AAAA records for registered services
- **FR-013**: System MUST include additional records (SRV, TXT, A) when responding to PTR queries to reduce round-trips
- **FR-014**: System MUST send goodbye packets (TTL=0 for all records) when a service is unregistered or the application exits

#### Query Response Behavior

- **FR-015**: System MUST respond to multicast queries matching registered service types or names
- **FR-016**: System MUST support unicast responses when the QU bit is set in the query (RFC 6762 §5.4)
- **FR-017**: System MUST implement known-answer suppression: omit records from responses if they appear in the query's answer section with TTL > half the correct TTL (RFC 6762 §7.1)
- **FR-018**: System MUST aggregate multiple records into a single response packet when possible, respecting the 9000-byte limit (RFC 6762 §6)

#### TTL & Cache Management

- **FR-019**: System MUST assign appropriate TTLs to resource records: 120 seconds default for services, 75 minutes for hostnames (RFC 6762 §10)
- **FR-020**: System MUST decrement TTL in responses based on elapsed time since record creation
- **FR-021**: System MUST send goodbye packets (TTL=0) at least 1 second before service termination when gracefully shutting down

#### State Machine

- **FR-022**: System MUST implement a state machine with states: Probing → Announcing → Established
- **FR-023**: System MUST handle queries received during Probing state (respond after successful probing) and Announcing state (respond immediately)
- **FR-024**: System MUST transition from Probing to Announcing only after all 3 probe queries receive no conflicting responses

#### Multi-Service Support

- **FR-025**: System MUST support registering multiple services simultaneously within a single responder instance
- **FR-026**: System MUST isolate state machines for each service (one service's conflict doesn't affect others)
- **FR-027**: System MUST respond to "_services._dns-sd._udp.local" PTR queries with a list of all registered service types (RFC 6763 §9)

#### Platform & Architectural Integration

- **FR-028**: System MUST use the existing M1.1 transport layer (ListenConfig, SO_REUSEPORT) to coexist with Avahi/Bonjour
- **FR-029**: System MUST use the existing M1.1 interface management (DefaultInterfaces, WithInterfaces) to select network interfaces
- **FR-030**: System MUST apply the existing M1.1 rate limiting to responses (prevent multicast storm when responding)
- **FR-031**: System MUST leverage the existing M1.1 source IP filtering to reject non-link-local queries

#### Error Handling & Robustness

- **FR-032**: System MUST handle registration failures gracefully (e.g., name unavailable after 10 rename attempts → return error to caller)
- **FR-033**: System MUST continue operating if one service fails to register (isolation)
- **FR-034**: System MUST NOT panic on malformed queries (leverage M1 parser robustness)
- **FR-035**: System MUST be thread-safe for concurrent registration/unregistration/query handling

### Key Entities

- **Service Instance**: Represents a registered mDNS service with attributes: instance name (e.g., "MyApp"), service type (e.g., "_http._tcp.local"), port number, TXT records (metadata), hostname, and current state (Probing/Announcing/Established)

- **Resource Record Set**: A collection of DNS resource records (PTR, SRV, TXT, A, AAAA) associated with a service instance, each with a TTL and creation timestamp

- **State Machine**: Tracks the lifecycle of a service registration from Probing (name conflict detection) → Announcing (unsolicited advertisement) → Established (responding to queries)

- **Probe Query**: A query sent during the Probing state to detect name conflicts, containing the proposed service name

- **Conflict Detector**: Logic to detect simultaneous probe conflicts and apply tie-breaking rules based on lexicographic comparison of record data

- **Response Builder**: Constructs mDNS response packets with appropriate records (PTR, SRV, TXT, A), respecting known-answer suppression and 9000-byte limit

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Registered services are discoverable by standard mDNS browsers (Avahi Browse, macOS Bonjour Browser, DNS-SD Browser on iOS) within 2 seconds of registration
- **SC-002**: System successfully detects and resolves name conflicts 100% of the time (no duplicate names on network after probing)
- **SC-003**: System passes all Apple Bonjour Conformance Test (BCT) test cases related to responder functionality
- **SC-004**: Responder coexists with Avahi and Bonjour on the same host without port conflicts (verified via integration tests on Linux and macOS)
- **SC-005**: Test coverage remains ≥80% after M2 implementation
- **SC-006**: Registered services respond to queries within 100ms (measured from query send to response received)
- **SC-007**: Goodbye packets are sent within 1 second of service unregistration, causing browsers to remove the service immediately
- **SC-008**: System handles 100 concurrent service registrations without crashes or data races (verified via `go test -race`)
- **SC-009**: Known-answer suppression reduces response packet count by at least 30% in repeated query scenarios (benchmark test)
- **SC-010**: Responder operates correctly on networks with 50+ mDNS services without degradation (interoperability test)

### Compliance Targets

- **SC-011**: RFC 6762 compliance increases from current 52.8% to at least 70% (covers §8 Probing, §9 Responding, §12 Special Characteristics, §13 Conflict Resolution)
- **SC-012**: All RFC 6762 "MUST" requirements related to responder functionality are implemented (§8.1 probing, §8.3 announcing, §10.1 goodbye packets)

---

## Assumptions

1. **IPv4 Only (M2 Scope)**: This milestone focuses on IPv4 responder functionality. IPv6 (FF02::FB) will be added in a future milestone.
2. **Single Hostname**: Each responder instance represents a single hostname (typically the device's hostname). Multiple service instances can be registered under that hostname.
3. **Default TTLs**: Standard TTL values will be used: 120 seconds for service records (PTR, SRV, TXT), 75 minutes for hostname A records, per RFC 6762 §10 recommendations.
4. **Apple BCT Availability**: Apple Bonjour Conformance Test suite is available for testing (either via macOS or documented test cases).
5. **Avahi/Bonjour Behavior**: Assumes Avahi (Linux) and Bonjour (macOS) follow RFC 6762 for interoperability testing.
6. **Thread Safety Model**: Uses Go's standard concurrency primitives (mutexes, channels) for thread-safe service registration/unregistration.
7. **Packet Size Limit**: Respects RFC 6762 §17 maximum packet size of 9000 bytes. Services with excessive TXT records will need to handle truncation or split responses.
8. **Network Stability**: Assumes network interfaces remain stable during service registration. M1.1 provides static interface selection (WithInterfaces, DefaultInterfaces with VPN exclusion); dynamic interface change handling (interface up/down events, IP address changes) is deferred to M3 or later.
9. **No DNS-SD Browsing API**: This milestone implements the responder (answering queries). The high-level DNS-SD browsing API (continuous monitoring) is deferred to M3.

---

## Dependencies

### Prerequisites (M1.1 Foundation)

- ✅ M1.1 Foundation complete (see FR-028 through FR-031 for integration requirements)
- ✅ M1 DNS message builder and parser (query/response wire format)

### External Dependencies

- RFC 6762 (Multicast DNS) - primary technical authority
- RFC 6763 (DNS-Based Service Discovery) - service type naming conventions
- Apple Bonjour Conformance Test (BCT) - validation tool

### Out of Scope (Future Milestones)

- IPv6 support (FF02::FB multicast group) - deferred to M2.1 or M3
- DNS-SD continuous browsing API - deferred to M3
- Wide-area service discovery - deferred to M5
- Record caching for received queries - not required for responder (queriers cache, responders don't)
- Response deduplication across multiple interfaces - deferred to M3

---

## Non-Functional Requirements

### Performance

- **NFR-001**: Service registration completes within 1 second (excluding 750ms probing delay)
- **NFR-002**: Query response latency <100ms (90th percentile)
- **NFR-003**: Support ≥100 concurrent service registrations without memory leaks or performance degradation

### Reliability

- **NFR-004**: Zero crashes on malformed queries (inherit M1 fuzz test robustness)
- **NFR-005**: Zero race conditions (verified with `go test -race`)
- **NFR-006**: Graceful degradation if probing fails (timeout after 750ms, assume name available)

### Compatibility

- **NFR-007**: Interoperate with Avahi 0.8+ (Linux)
- **NFR-008**: Interoperate with macOS Bonjour (tested on macOS 12+)
- **NFR-009**: Pass Apple BCT test suite (100% pass rate on responder tests)

### Observability

- **NFR-010**: Log key state transitions (Probing → Announcing → Established) at INFO level
- **NFR-011**: Log name conflicts and resolutions at WARN level
- **NFR-012**: Provide debug logging for query/response packet details (opt-in)

---

## Open Questions

*None at this time. Specification is complete based on RFC 6762 requirements and ROADMAP guidance. Implementation details will be determined during planning phase.*

---

**Specification Version**: 1.0
**Ready for**: Planning (`/speckit.plan`)
