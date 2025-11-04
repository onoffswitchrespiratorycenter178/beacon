# Feature Specification: M1.1 Architectural Hardening

**Feature Branch**: `004-m1-1-architectural-hardening`
**Created**: 2025-11-01
**Status**: Draft
**Input**: User description: "M1.1 Architectural Hardening: Socket configuration, interface management, and security features per F-9, F-10, F-11 specifications"

## User Scenarios & Testing

### User Story 1 - System Daemon Coexistence (Priority: P1)

A developer integrates Beacon into their Linux application on a server where Avahi is already running for system-wide service discovery. The application needs to perform its own mDNS queries for application-specific services without conflicts.

**Why this priority**: This is the #1 production adoption blocker. Without Avahi/Bonjour coexistence, Beacon cannot be used in most enterprise environments or on macOS/Linux systems with system daemons running. Enterprise customers cannot disable system services.

**Independent Test**: Can be fully tested by installing Avahi on Linux (or checking macOS with Bonjour running), starting a Beacon querier, and verifying: (1) Beacon starts without "address already in use" errors, (2) Beacon receives mDNS responses, (3) System daemon continues functioning normally. Delivers immediate value - Beacon works on production systems.

**Acceptance Scenarios**:

1. **Given** Avahi is running on Linux binding port 5353, **When** application creates a Beacon querier, **Then** querier initializes successfully without "address already in use" error
2. **Given** Beacon querier is active on macOS, **When** Bonjour service advertises a service, **Then** Beacon receives and parses the advertisement correctly
3. **Given** systemd-resolved is active on port 5353, **When** application starts Beacon, **Then** both services coexist without port conflicts
4. **Given** Beacon is running, **When** system daemon (Avahi/Bonjour) sends mDNS queries, **Then** Beacon receives relevant multicast traffic

---

### User Story 2 - VPN Privacy Protection (Priority: P2)

A user connects to a corporate VPN while using an application with Beacon. The application performs mDNS queries to discover local network printers. Without proper interface filtering, mDNS queries leak to the VPN provider's network, potentially exposing the user's location and device activity.

**Why this priority**: Privacy violation affecting current M1 users. Enterprises require VPN traffic isolation for compliance (GDPR, HIPAA). Users expect local network discovery to stay local. This is a compliance and privacy risk, not just a functional issue.

**Independent Test**: Can be tested by connecting to a VPN (any VPN client), creating a Beacon querier with default settings, and verifying: (1) Queries are NOT sent to the VPN interface (packet capture shows no traffic on utun*/tun*), (2) Queries ARE sent to physical network interfaces (Wi-Fi, Ethernet), (3) Application discovers local services but not VPN-side services. Delivers value - user privacy protected by default.

**Acceptance Scenarios**:

1. **Given** user has active VPN connection (utun0, tun0, ppp0), **When** application uses Beacon with default settings, **Then** mDNS queries are NOT sent to VPN interface
2. **Given** user is on Wi-Fi with Docker containers running, **When** Beacon initializes with default interface filtering, **Then** Docker virtual interfaces (docker0, veth*, br-*) are excluded from binding
3. **Given** application explicitly specifies Wi-Fi interface only, **When** Beacon sends queries, **Then** queries are sent ONLY to specified interface (not VPN, not Docker)
4. **Given** user disconnects VPN mid-session, **When** Beacon continues operating, **Then** queries continue on physical interfaces (network change detection deferred to M1.2)

---

### User Story 3 - Multicast Storm Protection (Priority: P3)

A developer deploys Beacon in an IoT environment where a buggy device sends 1000+ mDNS queries per second (real-world example: Hubitat Home Automation bug, 2020). Without rate limiting, the Beacon application experiences resource exhaustion, high CPU usage, and potential crashes.

**Why this priority**: Resilience and production stability. Lower priority than P1/P2 because it's a defense-in-depth measure protecting against abnormal conditions, whereas P1/P2 address fundamental functionality and compliance. However, still critical for production deployments.

**Independent Test**: Can be tested by simulating a multicast storm (send 1000 queries/sec from a test tool), creating a Beacon querier with rate limiting enabled, and verifying: (1) Beacon detects the storm, (2) Cooldown is applied to the flooding source IP, (3) Application remains responsive (CPU usage bounded), (4) Legitimate traffic from other sources continues to be processed. Delivers value - application stability under attack.

**Acceptance Scenarios**:

1. **Given** a device sends 1000 queries/second to multicast group, **When** Beacon receives this traffic, **Then** rate limiter detects flooding source and applies 60-second cooldown
2. **Given** rate limiter has applied cooldown to source IP 192.168.1.100, **When** packets arrive from 192.168.1.100 during cooldown, **Then** packets are silently dropped without processing
3. **Given** rate limiter detects flooding, **When** flooding source stops for 60 seconds, **Then** cooldown expires and traffic from that source is processed again
4. **Given** legitimate device sends 50 queries/second (below threshold), **When** Beacon processes traffic, **Then** no rate limiting is applied (threshold is 100 queries/sec)
5. **Given** rate limiter is disabled via configuration, **When** application creates querier, **Then** all traffic is processed without rate limiting

---

### User Story 4 - Link-Local Scope Enforcement (Priority: P3)

An attacker or misconfigured device sends spoofed mDNS responses from a routed (non-link-local) IP address. Beacon validates that the source IP is link-local or on the same subnet as the receiving interface, silently dropping invalid traffic before parsing.

**Why this priority**: Defense-in-depth security measure. RFC 6762 §2 specifies link-local scope, but violations typically indicate misconfiguration rather than active attacks. Lower priority because the protocol itself limits scope via multicast TTL=255. However, important for RFC compliance and CPU waste reduction.

**Independent Test**: Can be tested by crafting mDNS response packets with non-link-local source IPs (e.g., 8.8.8.8), sending them to the multicast group (requires packet crafting tool), and verifying: (1) Beacon receives the packet, (2) Packet is dropped before parsing (log shows drop), (3) CPU is not wasted on parsing invalid packets. Delivers value - RFC compliance and efficiency.

**Acceptance Scenarios**:

1. **Given** packet arrives from link-local IPv4 source (169.254.x.x), **When** Beacon validates source, **Then** packet is accepted for parsing
2. **Given** packet arrives from same subnet as receiving interface (e.g., 192.168.1.50 on 192.168.1.0/24 interface), **When** Beacon validates source, **Then** packet is accepted
3. **Given** packet arrives from routed IP (e.g., 8.8.8.8), **When** Beacon validates source, **Then** packet is silently dropped and logged at debug level
4. **Given** packet arrives from private IP on different subnet (e.g., 10.0.0.5 when interface is 192.168.1.0/24), **When** Beacon validates source, **Then** packet is dropped (not same subnet, not link-local)

---

### Edge Cases

- **What happens when socket option SO_REUSEPORT is not supported on older Linux kernels (< 3.9)?**
  - System logs warning at initialization
  - Coexistence with system daemons NOT guaranteed
  - Application continues with SO_REUSEADDR only (multicast still works, but may conflict with other listeners)

- **What happens when application explicitly selects a VPN interface (user override)?**
  - Beacon binds to VPN interface as specified (user intent respected)
  - Warning logged: "Binding to VPN interface may violate link-local scope"
  - mDNS queries sent to VPN network (user-requested behavior)

- **What happens when no suitable interfaces are found by DefaultInterfaces()?**
  - Querier initialization fails with clear error: "No suitable interfaces found (all interfaces filtered)"
  - Error includes list of interfaces evaluated and why each was rejected

- **What happens when rate limiter is under heavy load (10,000 sources tracked)?**
  - Cooldown map is bounded (max 10,000 entries)
  - Oldest cooldown entries are evicted (LRU policy)
  - Warning logged if map reaches 80% capacity

- **How does system handle invalid platform-specific socket options?**
  - Platform-specific code wrapped in build tags (compile-time safety)
  - If socket option fails at runtime, error propagated with context
  - Example: Windows attempting SO_REUSEPORT (not supported) returns clear error

- **What happens when multicast group join fails?**
  - Socket initialization fails immediately (cannot receive multicast traffic)
  - Error message specifies interface and group address
  - User can retry with different interface or configuration

## Requirements

### Functional Requirements

#### Socket Configuration (F-9)

- **FR-001**: System MUST use `net.ListenConfig` with Control function to set socket options BEFORE bind() call
- **FR-002**: System MUST set SO_REUSEADDR on Linux, macOS, and Windows for multicast binding
- **FR-003**: System MUST set SO_REUSEPORT on Linux (kernel >= 3.9) and macOS for port sharing with system daemons
- **FR-004**: System MUST NOT use `net.ListenMulticastUDP()` due to Go Issues #73484 and #34728
- **FR-005**: System MUST join multicast group 224.0.0.251 using `golang.org/x/net/ipv4` package per RFC 6762 §5
- **FR-006**: System MUST set multicast TTL to 255 per RFC 6762 §11
- **FR-007**: System MUST enable multicast loopback for local testing
- **FR-008**: System MUST detect Linux kernel version and log warning if < 3.9 (SO_REUSEPORT not guaranteed)
- **FR-009**: System MUST use platform-specific files (build tags) for socket option configuration
- **FR-010**: System MUST propagate socket initialization errors with context (interface, operation, cause)

#### Interface Management (F-10)

- **FR-011**: System MUST provide `WithInterfaces([]net.Interface)` functional option for explicit interface selection
- **FR-012**: System MUST provide `WithInterfaceFilter(func(net.Interface) bool)` functional option for custom filtering
- **FR-013**: System MUST implement `DefaultInterfaces()` function that returns filtered list of suitable interfaces
- **FR-014**: `DefaultInterfaces()` MUST include only interfaces with `net.FlagUp` set (active interfaces)
- **FR-015**: `DefaultInterfaces()` MUST include only interfaces with `net.FlagMulticast` set (multicast-capable)
- **FR-016**: `DefaultInterfaces()` MUST exclude loopback interfaces (127.0.0.1, ::1)
- **FR-017**: `DefaultInterfaces()` MUST exclude VPN interfaces by pattern: utun*, tun*, ppp*, wg*, tailscale*, wireguard*
- **FR-018**: `DefaultInterfaces()` MUST exclude Docker interfaces by pattern: docker0, veth*, br-*
- **FR-019**: System MUST validate each interface exists and is suitable for mDNS before binding
- **FR-020**: System MUST log interface selection decisions at debug level (interface evaluated, filter decision, reason)
- **FR-021**: System MUST log selected interfaces at info level (user-visible confirmation)
- **FR-022**: When no interfaces match filters, system MUST fail initialization with error listing rejected interfaces

#### Security (F-11)

- **FR-023**: System MUST validate source IP is link-local (169.254.0.0/16) OR same subnet as receiving interface
- **FR-024**: System MUST silently drop packets with non-link-local source IPs BEFORE parsing
- **FR-025**: System MUST log dropped packets at debug level with source IP and reason
- **FR-026**: System MUST implement per-source-IP rate limiting with sliding window (1 second)
- **FR-027**: Rate limiter MUST have configurable threshold (default: 100 queries/second)
- **FR-028**: Rate limiter MUST have configurable cooldown duration (default: 60 seconds)
- **FR-029**: Rate limiter MUST drop packets from flooding sources during cooldown without parsing
- **FR-030**: Rate limiter MUST log first violation at warn level, subsequent at debug level
- **FR-031**: Rate limiter MUST periodically clean up expired cooldown entries (every 5 minutes)
- **FR-032**: Rate limiter MUST bound cooldown map size (max 10,000 entries, LRU eviction)
- **FR-033**: System MUST provide `WithRateLimit(bool)` option to enable/disable rate limiting (default: enabled)
- **FR-034**: System MUST reject packets larger than 9000 bytes per RFC 6762 §17
- **FR-035**: System MUST never panic on malformed packets (already implemented in M1, validated in fuzzing)

### Key Entities

- **SocketConfig**: Represents platform-specific socket configuration
  - Attributes: Platform (Linux/macOS/Windows), SO_REUSEADDR enabled, SO_REUSEPORT enabled (if supported), kernel version (Linux only)
  - Relationships: Created per interface binding, validated before socket creation

- **InterfaceFilter**: Represents interface selection logic
  - Attributes: Explicit interface list (if provided), Custom filter function (if provided), Default filter rules (VPN patterns, Docker patterns, loopback exclusion)
  - Relationships: Applied during querier initialization, determines which interfaces receive sockets

- **RateLimitEntry**: Tracks query rate per source IP
  - Attributes: Source IP address, Query count in current window, Window start time, Cooldown expiry time (if in cooldown)
  - Relationships: Stored in rate limiter map, evicted after cooldown expiry or when map reaches capacity

- **MulticastSocket**: Represents a bound socket joined to multicast group
  - Attributes: Interface (net.Interface), Socket file descriptor, Multicast group joined (224.0.0.251), TTL (255), Loopback enabled
  - Relationships: One per selected interface, managed by transport layer

## Success Criteria

### Measurable Outcomes

- **SC-001**: Application successfully initializes Beacon querier on Linux system with Avahi running, without "address already in use" errors (100% success rate in integration tests)
- **SC-002**: Application successfully initializes Beacon querier on macOS with Bonjour running, coexisting without port conflicts (100% success rate in integration tests)
- **SC-003**: Beacon with default configuration does NOT send mDNS queries to VPN interfaces (utun*, tun*, ppp*) when VPN is active (0% VPN interface binding in automated tests)
- **SC-004**: Beacon with default configuration does NOT bind to Docker virtual interfaces (docker0, veth*, br-*) when Docker is running (0% Docker interface binding in automated tests)
- **SC-005**: Beacon survives simulated multicast storm of 1000 queries/second without crashing or resource exhaustion (CPU usage remains below 20%, memory growth bounded to 10MB)
- **SC-006**: Rate limiter applies cooldown within 1 second of detecting flooding source (>100 queries/sec threshold)
- **SC-007**: Beacon drops packets from non-link-local source IPs before parsing, reducing CPU waste (0% parsing of invalid packets in tests)
- **SC-008**: All M1 tests continue passing after M1.1 changes (zero regression - 9/9 packages PASS)
- **SC-009**: Test coverage maintained at ≥80% after adding M1.1 features
- **SC-010**: Platform-specific socket option tests pass on Linux, macOS, and Windows (100% pass rate on each platform in CI)
- **SC-011**: Integration test with Avahi demonstrates successful coexistence (Avahi continues advertising services, Beacon receives advertisements)

## Assumptions

1. **Platform Support**: Linux kernel >= 3.9 is the baseline for SO_REUSEPORT support. Older kernels receive a logged warning but Beacon attempts to continue with SO_REUSEADDR only.

2. **VPN Detection**: VPN interface patterns (utun*, tun*, ppp*, wg*, tailscale*) cover 95%+ of VPN clients in the wild. Custom VPN configurations may require explicit interface selection via `WithInterfaces()`.

3. **System Daemon Behavior**: Avahi, systemd-resolved, and Bonjour respect SO_REUSEPORT and share port 5353 cooperatively. This is validated behavior in production deployments.

4. **Rate Limiting Threshold**: 100 queries/second threshold strikes a balance between protecting against attacks (Hubitat bug sent 1000+/sec) and allowing legitimate high-volume use cases (multiple services being browsed simultaneously).

5. **Link-Local Enforcement**: Non-link-local source IPs indicate misconfiguration or spoofing. Legitimate mDNS traffic uses link-local or same-subnet IPs per RFC 6762 §2.

6. **Network Stability**: During M1.1, network interface list is determined at initialization and does not change. Dynamic network change detection (interface added/removed, IP address changes) is deferred to M1.2 per roadmap.

7. **Dependencies**: `golang.org/x/sys` and `golang.org/x/net` are justified per Constitution v1.1.0 Principle V because Go standard library lacks necessary socket option control and multicast group management.

8. **Testing Environment**: Integration tests with Avahi/Bonjour require those daemons to be running on test systems. CI pipeline runs on Linux (Avahi available), macOS tests performed manually or in macOS CI runner.

## Dependencies

### Internal Dependencies

- **F-2 (Package Structure)**: Socket configuration lives in `internal/network/`, platform-specific files follow build tag conventions
- **F-3 (Error Handling)**: Socket errors wrapped in `errors.NetworkError` with operation and context
- **F-7 (Resource Management)**: Rate limiter implements bounded map (10,000 entries max) to prevent memory exhaustion
- **F-8 (Testing Strategy)**: Platform-specific tests use build tags, integration tests validate Avahi/Bonjour coexistence
- **M1-Refactoring**: Transport interface abstraction provides clean extension point for socket configuration

### External Dependencies

- **golang.org/x/sys**: Platform-specific socket options (SO_REUSEADDR, SO_REUSEPORT)
  - Linux: `golang.org/x/sys/unix`
  - macOS: `golang.org/x/sys/unix`
  - Windows: `golang.org/x/sys/windows` or standard `syscall`
  - **Justification**: Go standard library does not expose socket option control needed for ListenConfig.Control pattern

- **golang.org/x/net**: Multicast group membership
  - Package: `golang.org/x/net/ipv4` (IPv4 multicast)
  - Package: `golang.org/x/net/ipv6` (IPv6 multicast, deferred to M2)
  - **Justification**: Go standard library `net` package lacks multicast group join/leave control

### System Dependencies

- **Linux**: Kernel >= 3.9 recommended for full SO_REUSEPORT support (warning logged on older kernels)
- **macOS**: No specific version requirements (SO_REUSEPORT supported in all modern versions)
- **Windows**: No SO_REUSEPORT (not needed - SO_REUSEADDR behaves differently, sufficient for port sharing)

### Testing Dependencies

- **Avahi** (Linux integration tests): `sudo apt-get install avahi-daemon`
- **Bonjour** (macOS integration tests): Pre-installed on macOS
- **systemd-resolved** (optional Linux integration test): Pre-installed on Ubuntu 18.04+

## Out of Scope

The following are explicitly OUT OF SCOPE for M1.1 and will be addressed in later milestones:

### M1.2 - Network Change Detection

- Dynamic interface monitoring (interfaces added/removed during runtime)
- IP address change detection (DHCP renew, manual configuration)
- Automatic reconnection on network changes
- **Rationale**: M1.1 focuses on architectural foundation. Dynamic network handling adds complexity deferred to dedicated milestone.

### M2 - IPv6 Support

- IPv6 multicast group (FF02::FB)
- Dual-stack operation (IPv4 + IPv6 simultaneously)
- IPv6-specific socket options
- **Rationale**: F-9 architecture supports future IPv6, but M1.1 implements IPv4 only to minimize scope.

### M1.1 - Not Included

- Responder functionality (still query-only, responder is M2)
- Service registration/advertising (DNS-SD publishing is M3)
- Long-lived queries (RFC 6762 §5.2, post-v1.0 enhancement)
- Advanced caching strategies (basic cache remains, optimizations in M4)

## Notes

- M1.1 builds on the transport interface abstraction from M1-Refactoring (003). The `Transport` interface defined in M1-Refactoring provides a clean extension point for the socket configuration work in M1.1.

- This specification prioritizes user stories by production impact: (P1) Coexistence blocks enterprise adoption, (P2) Privacy affects compliance, (P3) Resilience protects against abnormal conditions.

- Each user story is independently testable and delivers standalone value, following specification-driven development principles.

- The roadmap estimates 25 hours total effort for M1.1 (12 hours socket/interface, 3.5 hours security, 6 hours specs, 4 hours testing). Target completion: 2025-11-15.

- All F-spec requirements (F-9, F-10, F-11) are mapped to functional requirements in this spec. FR-001 through FR-010 implement F-9, FR-011 through FR-022 implement F-10, FR-023 through FR-035 implement F-11.
