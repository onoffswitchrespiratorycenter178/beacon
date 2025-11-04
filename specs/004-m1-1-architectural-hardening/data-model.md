# Data Model: M1.1 Architectural Hardening

**Phase 1 Design** | **Date**: 2025-11-01
**Branch**: `004-m1-1-architectural-hardening`

## Overview

This document defines the data structures and entities for M1.1 socket configuration, interface filtering, and security features. All types are internal implementation details (not exported in public API).

---

## 1. SocketConfig (Platform-Specific Socket Options)

**Package**: `internal/transport`
**Purpose**: Platform-specific socket option configuration for SO_REUSEADDR and SO_REUSEPORT

### Structure

```go
// SocketConfig holds platform-specific socket configuration.
// This type is NOT exported (internal implementation detail).
type socketConfig struct {
    // Platform is the runtime platform (linux, darwin, windows)
    platform string

    // reuseAddr indicates whether SO_REUSEADDR is set
    reuseAddr bool

    // reusePort indicates whether SO_REUSEPORT is set
    // Windows: always false (SO_REUSEPORT not supported)
    // Linux/macOS: true if kernel supports it
    reusePort bool

    // kernelVersion is the Linux kernel version (for logging/debugging)
    // Format: "3.10.0-1160.el7.x86_64"
    // Only populated on Linux (empty on macOS/Windows)
    kernelVersion string
}
```

### Platform-Specific Behavior

| Platform | SO_REUSEADDR | SO_REUSEPORT | Notes |
|----------|--------------|--------------|-------|
| **Linux** | ✅ Always set | ✅ Set if kernel 3.9+ | Fall back to SO_REUSEADDR only if SO_REUSEPORT fails |
| **macOS** | ✅ Always set | ✅ Always set | BSD semantics (slightly different from Linux) |
| **Windows** | ✅ Always set | ❌ Not supported | Windows SO_REUSEADDR has different semantics (see research.md) |

### Lifecycle

1. **Creation**: Created in `internal/transport/socket_{linux,darwin,windows}.go` via build tags
2. **Validation**: No validation needed (platform-specific files ensure correctness)
3. **Usage**: Passed to `net.ListenConfig.Control` function to set socket options before `bind()`

### Related Functions

- `setSocketOptions(fd uintptr) error` — Platform-specific function (build tags)
- `getKernelVersion() string` — Linux-only helper (returns empty string on macOS/Windows)

---

## 2. InterfaceFilter (Interface Selection Logic)

**Package**: `internal/network`
**Purpose**: Filter network interfaces for mDNS querying (exclude VPN/Docker, select physical interfaces)

### Structure

```go
// InterfaceFilter represents interface selection logic.
// Users can provide explicit list OR custom filter function.
type interfaceFilter struct {
    // explicit is a user-provided explicit list of interfaces
    // If non-nil, ONLY these interfaces are used (ignores filter function)
    explicit []net.Interface

    // filter is a user-provided custom filter function
    // If explicit is nil, this function determines which interfaces to use
    // Default: DefaultInterfaces() logic (exclude VPN/Docker)
    filter func(net.Interface) bool
}
```

### Filter Priority

1. **Explicit list** (`explicit != nil`): Use ONLY these interfaces, skip all filtering
2. **Custom filter** (`filter != nil`): Apply custom function
3. **Default**: Use `DefaultInterfaces()` (exclude VPN, Docker, loopback, down interfaces)

### Default Filter Logic (DefaultInterfaces)

```go
func DefaultInterfaces() ([]net.Interface, error) {
    allIfaces, _ := net.Interfaces()
    var filtered []net.Interface
    for _, iface := range allIfaces {
        // Include if: UP + MULTICAST + NOT loopback + NOT VPN + NOT Docker
        if iface.Flags&net.FlagUp == 0 { continue }
        if iface.Flags&net.FlagMulticast == 0 { continue }
        if iface.Flags&net.FlagLoopback != 0 { continue }
        if isVPN(iface.Name) { continue }
        if isDocker(iface.Name) { continue }
        filtered = append(filtered, iface)
    }
    return filtered, nil
}
```

### VPN Detection Patterns

**Function**: `isVPN(name string) bool`

**Patterns** (95%+ coverage):
- `utun*` — macOS system VPNs, Tunnelblick, OpenVPN
- `tun*` — Linux OpenVPN, generic TUN devices
- `ppp*` — PPTP, L2TP tunnels
- `wg*` — WireGuard (standard naming)
- `tailscale*` — Tailscale VPN
- `wireguard*` — WireGuard (alternative naming)

**Rationale**: Privacy protection (prevent mDNS queries from leaking to VPN provider per F-10 REQ-F10-6)

### Docker Detection Patterns

**Function**: `isDocker(name string) bool`

**Patterns** (100% coverage):
- `docker0` — Default Docker bridge (exact match)
- `veth*` — Virtual ethernet pairs (container connections)
- `br-*` — Custom Docker bridge networks

**Rationale**: Performance (avoid wasting CPU on isolated container networks)

### Lifecycle

1. **Creation**: Created in `querier.New()` from functional options
2. **Resolution**: Resolved once during `Querier` initialization (not per-query)
3. **Caching**: Interface list cached in `Querier` struct (immutable after creation)

---

## 3. RateLimitEntry (Per-Source Rate Limiting)

**Package**: `internal/security`
**Purpose**: Track query rate per source IP to detect/prevent multicast storms

### Structure

```go
// RateLimitEntry tracks query rate for a single source IP.
type RateLimitEntry struct {
    // sourceIP is the source IP address (key in RateLimiter map)
    sourceIP string

    // queryCount is the number of queries in the current sliding window
    queryCount int

    // windowStart is the start of the current 1-second sliding window
    windowStart time.Time

    // cooldownExpiry is when the cooldown period ends (if in cooldown)
    // If zero, source is not in cooldown
    // If non-zero and > time.Now(), drop all packets from this source
    cooldownExpiry time.Time

    // lastSeen is the last time a query was received from this source
    // Used for LRU eviction when map exceeds 10,000 entries
    lastSeen time.Time
}
```

### State Machine

```text
NORMAL → RATE_LIMITED → COOLDOWN → NORMAL
  ↑                                    |
  └────────────────────────────────────┘
```

**States**:
1. **NORMAL**: `queryCount < threshold` (default: 100 qps) — Accept packets
2. **RATE_LIMITED**: `queryCount >= threshold` — Start cooldown
3. **COOLDOWN**: `time.Now() < cooldownExpiry` (default: 60s) — Drop all packets
4. **NORMAL**: `time.Now() >= cooldownExpiry` — Reset to NORMAL

### Sliding Window Algorithm

```text
Time: ────────────[1s window]────────────>
      ↑           ↑                ↑
      windowStart t                t+1s

Query at t:
  - If (t - windowStart) > 1s: Reset window, queryCount = 1
  - Else: queryCount++
  - If queryCount > threshold: cooldownExpiry = t + 60s
```

### Configuration

**Defaults** (configurable via functional options):
- **Threshold**: 100 queries/second (detects Hubitat bug, allows legitimate high-volume)
- **Cooldown**: 60 seconds (long enough to detect persistent storms, short enough to recover from transient issues)
- **Window**: 1 second (sliding window for rate calculation)
- **Max Entries**: 10,000 (prevent memory exhaustion)

### LRU Eviction

When map size exceeds 10,000 entries:
1. Sort entries by `lastSeen` (oldest first)
2. Remove oldest 1,000 entries (10% eviction)
3. Log eviction for debugging

**Rationale**: Bounded memory (10,000 entries ≈ 200KB), handles high-churn scenarios

---

## 4. RateLimiter (Global Rate Limiter)

**Package**: `internal/security`
**Purpose**: Manage per-source-IP rate limiting with bounded map

### Structure

```go
// RateLimiter manages per-source-IP rate limiting.
type RateLimiter struct {
    // threshold is the max queries/second per source IP (default: 100)
    threshold int

    // cooldown is the duration to drop packets after threshold exceeded (default: 60s)
    cooldown time.Duration

    // maxEntries is the max number of source IPs tracked (default: 10,000)
    maxEntries int

    // sources maps source IP → RateLimitEntry
    // Protected by mu (concurrent reads allowed, writes exclusive)
    sources map[string]*RateLimitEntry

    // mu protects sources map (sync.RWMutex for concurrent reads)
    mu sync.RWMutex

    // evictionCount tracks number of LRU evictions (for metrics/logging)
    evictionCount uint64
}
```

### API Methods

#### Allow(sourceIP string) bool

**Purpose**: Check if a query from `sourceIP` should be allowed

**Algorithm**:
```go
func (rl *RateLimiter) Allow(sourceIP string) bool {
    rl.mu.RLock()
    entry, exists := rl.sources[sourceIP]
    rl.mu.RUnlock()

    if !exists {
        // First query from this source - create entry
        rl.mu.Lock()
        rl.sources[sourceIP] = &RateLimitEntry{
            sourceIP:    sourceIP,
            queryCount:  1,
            windowStart: time.Now(),
            lastSeen:    time.Now(),
        }
        rl.mu.Unlock()
        return true
    }

    // Check cooldown
    if !entry.cooldownExpiry.IsZero() && time.Now().Before(entry.cooldownExpiry) {
        return false // In cooldown, drop packet
    }

    // Update sliding window
    now := time.Now()
    if now.Sub(entry.windowStart) > 1*time.Second {
        // Reset window
        entry.queryCount = 1
        entry.windowStart = now
    } else {
        // Increment count
        entry.queryCount++
    }

    entry.lastSeen = now

    // Check threshold
    if entry.queryCount > rl.threshold {
        entry.cooldownExpiry = now.Add(rl.cooldown)
        return false // Exceeded threshold, start cooldown
    }

    return true
}
```

**Thread Safety**: Uses `sync.RWMutex` for concurrent reads (hot path), exclusive writes

#### Evict()

**Purpose**: Periodic cleanup to prevent unbounded memory growth

**Trigger**: Called when `len(sources) > maxEntries`

**Algorithm**: LRU eviction (remove 10% of oldest entries by `lastSeen`)

---

## 5. SourceFilter (Link-Local Source Validation)

**Package**: `internal/security`
**Purpose**: Validate source IP is link-local OR same subnet as receiving interface (RFC 6762 §2)

### Structure

```go
// SourceFilter validates source IPs before parsing packets.
type SourceFilter struct {
    // iface is the receiving interface (used for subnet check)
    iface net.Interface

    // ifaceAddrs is the cached list of interface addresses
    // Pre-computed during initialization (avoids syscall per packet)
    ifaceAddrs []net.IPNet
}
```

### API Methods

#### IsValid(srcIP net.IP) bool

**Purpose**: Check if `srcIP` is valid for mDNS (link-local or same subnet)

**Algorithm**:
```go
func (sf *SourceFilter) IsValid(srcIP net.IP) bool {
    // Check 1: IPv4 link-local (169.254.0.0/16)
    if srcIP.To4() != nil {
        if srcIP[0] == 169 && srcIP[1] == 254 {
            return true // Link-local per RFC 3927
        }
    }

    // Check 2: Same subnet as interface
    for _, ipnet := range sf.ifaceAddrs {
        if ipnet.Contains(srcIP) {
            return true // Same subnet
        }
    }

    // Check 3: Private IP on same subnet (10.x, 172.16-31.x, 192.168.x)
    // (Covers most local networks)
    for _, ipnet := range sf.ifaceAddrs {
        if isPrivate(srcIP) && ipnet.Contains(srcIP) {
            return true
        }
    }

    return false // Not link-local, not same subnet
}
```

**RFC 6762 §2 Compliance**: mDNS is link-local scope (not routable). Source IPs outside link-local indicate misconfiguration or spoofing.

---

## 6. MulticastSocket (Interface + Socket Pair)

**Package**: `internal/transport`
**Purpose**: Represents a socket bound to a specific interface for multicast send/receive

### Structure

```go
// MulticastSocket represents a UDP socket bound to a specific interface
// for mDNS multicast communication.
type multicastSocket struct {
    // iface is the network interface this socket is bound to
    iface net.Interface

    // conn is the underlying UDP connection (multicast-enabled)
    conn net.PacketConn

    // p is the IPv4 packet connection (for multicast group management)
    // Wraps conn via ipv4.NewPacketConn(conn)
    p *ipv4.PacketConn

    // group is the multicast group address (224.0.0.251)
    group *net.UDPAddr

    // ttl is the multicast TTL (255 per RFC 6762 §11)
    ttl int

    // loopback indicates whether multicast loopback is enabled
    // (true = receive own packets, required for some mDNS behavior)
    loopback bool
}
```

### Lifecycle

#### 1. Creation (per interface)

```go
func createMulticastSocket(iface net.Interface) (*multicastSocket, error) {
    // Step 1: Create UDP socket with ListenConfig + socket options
    lc := net.ListenConfig{
        Control: func(network, address string, c syscall.RawConn) error {
            return c.Control(func(fd uintptr) {
                setSocketOptions(fd) // Platform-specific (build tags)
            })
        },
    }
    conn, err := lc.ListenPacket(ctx, "udp4", "0.0.0.0:5353")

    // Step 2: Wrap in ipv4.PacketConn for multicast control
    p := ipv4.NewPacketConn(conn)

    // Step 3: Join multicast group (224.0.0.251)
    group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251)}
    p.JoinGroup(&iface, group)

    // Step 4: Set TTL and loopback
    p.SetMulticastTTL(255)       // RFC 6762 §11
    p.SetMulticastLoopback(true) // Receive own packets

    return &multicastSocket{
        iface:    iface,
        conn:     conn,
        p:        p,
        group:    group,
        ttl:      255,
        loopback: true,
    }, nil
}
```

#### 2. Send (multicast query)

```go
func (ms *multicastSocket) Send(msg []byte) error {
    // Write to multicast group via ipv4.PacketConn
    _, err := ms.p.WriteTo(msg, nil, ms.group)
    return err
}
```

#### 3. Receive (multicast responses)

```go
func (ms *multicastSocket) Receive(buf []byte) (int, net.Addr, error) {
    // Read from socket (receives multicast packets on this interface)
    n, _, addr, err := ms.p.ReadFrom(buf)
    return n, addr, err
}
```

#### 4. Close

```go
func (ms *multicastSocket) Close() error {
    // Leave multicast group
    ms.p.LeaveGroup(&ms.iface, ms.group)
    // Close underlying connection
    return ms.conn.Close()
}
```

### Why One Socket Per Interface?

**Reason**: Multicast group membership is per-interface in IPv4. To send/receive on multiple interfaces, we need one socket per interface.

**Alternative (REJECTED)**: Single socket listening on 0.0.0.0:5353, joined to multicast group on all interfaces
- **Problem**: Cannot control which interface sends queries (kernel chooses)
- **Problem**: Cannot filter VPN/Docker interfaces effectively

---

## Entity Relationships

```text
Querier (public API)
├── InterfaceFilter (internal/network)
│   ├── explicit: []net.Interface
│   └── filter: func(net.Interface) bool
│       └── DefaultInterfaces() → isVPN(), isDocker()
│
├── MulticastSocket[] (internal/transport)
│   ├── iface: net.Interface
│   ├── conn: net.PacketConn
│   │   └── Created via ListenConfig + setSocketOptions()
│   │       └── SocketConfig (platform-specific)
│   │           ├── SO_REUSEADDR (Linux/macOS/Windows)
│   │           └── SO_REUSEPORT (Linux/macOS only)
│   └── p: *ipv4.PacketConn
│       ├── JoinGroup(224.0.0.251)
│       ├── SetMulticastTTL(255)
│       └── SetMulticastLoopback(true)
│
├── RateLimiter (internal/security)
│   ├── sources: map[string]*RateLimitEntry
│   ├── threshold: int (100 qps)
│   ├── cooldown: time.Duration (60s)
│   └── maxEntries: int (10,000)
│
└── SourceFilter[] (internal/security, one per interface)
    ├── iface: net.Interface
    └── ifaceAddrs: []net.IPNet
        └── IsValid(srcIP) → link-local OR same subnet
```

---

## Configuration Flow

### User API → Internal Entities

```text
User calls:
  q, _ := querier.New(
      querier.WithInterfaces([]net.Interface{eth0}),     // Explicit list
      querier.WithRateLimit(true),                       // Enable rate limiting
      querier.WithRateLimitThreshold(50),                // 50 qps (stricter)
  )

Internal Initialization:
  1. InterfaceFilter created:
     - explicit = []net.Interface{eth0}
     - filter = nil (explicit list takes priority)

  2. Resolve interfaces:
     - Use eth0 only (explicit list)

  3. For each interface (eth0):
     a. Create MulticastSocket:
        - setSocketOptions(fd) → SO_REUSEADDR + SO_REUSEPORT
        - Join 224.0.0.251
        - Set TTL=255, loopback=true

     b. Create SourceFilter:
        - Cache eth0 addresses (e.g., 192.168.1.0/24)

  4. Create RateLimiter:
     - threshold = 50 (user override)
     - cooldown = 60s (default)
     - maxEntries = 10,000 (default)

Query Execution:
  1. Send query on eth0 MulticastSocket
  2. Receive responses:
     a. SourceFilter.IsValid(srcIP) → Check link-local
     b. RateLimiter.Allow(srcIP) → Check rate limit
     c. If valid, parse packet
```

---

## Memory Footprint

| Entity | Count | Size per Instance | Total | Notes |
|--------|-------|------------------|-------|-------|
| InterfaceFilter | 1 | ~100 bytes | ~100 B | Immutable after init |
| MulticastSocket | ~2-4 (typical) | ~200 bytes | ~800 B | One per interface |
| SourceFilter | ~2-4 (typical) | ~150 bytes | ~600 B | One per interface |
| RateLimiter | 1 | ~48 bytes (struct) | ~48 B | Excluding sources map |
| RateLimitEntry | 0-10,000 (bounded) | ~80 bytes | ~800 KB (max) | LRU eviction at 10,000 |
| **Total** | | | **~802 KB (max)** | Dominated by rate limiter map |

**Worst Case**: 10,000 rate limiter entries ≈ 800 KB (acceptable for server applications)
**Typical Case**: 10-100 entries ≈ 8 KB (negligible)

---

## Performance Characteristics

### Hot Path (per packet received)

```text
1. SourceFilter.IsValid(srcIP)         — O(n) where n = # subnets (typically 1-2)
2. RateLimiter.Allow(srcIP)           — O(1) map lookup + O(1) update
3. message.ParseMessage(packet)        — O(m) where m = packet size
```

**Critical Path**: SourceFilter + RateLimiter must be fast (executed per packet)
- SourceFilter: Pre-compute interface addresses (no syscall)
- RateLimiter: RWMutex for concurrent reads (hot path)

### Initialization (Querier.New)

```text
1. Resolve interfaces                  — O(n) where n = # interfaces
2. Create MulticastSockets             — O(k) where k = # selected interfaces
3. Join multicast groups               — O(k) syscalls
```

**Target**: <100ms total (acceptable startup time)

---

## Next Steps

Proceed to `contracts/querier-options.md` to define the public API functional options.
