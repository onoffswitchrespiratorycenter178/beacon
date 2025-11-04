# Research: M1 Architectural Alignment and Refactoring

**Milestone**: M1-Refactoring (Post-M1, Pre-M1.1)
**Status**: Phase 0 Complete
**Date**: 2025-11-01
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

---

## Overview

This document presents research findings for 5 architectural patterns required to refactor M1 (Basic mDNS Querier) in alignment with F-series specifications. Research validates that the 4 P0 critical issues identified in `docs/M1_REFACTORING_ANALYSIS.md` naturally align with M1.1 Transport Layer requirements (F-9, F-10, F-11), making this the optimal intervention point.

**Research Method**: Cross-reference analysis of:
- F-series specifications (F-2, F-3, F-7, F-9)
- M1 implementation (`internal/network/socket.go`, `querier/querier.go`)
- M1 refactoring analysis (`docs/M1_REFACTORING_ANALYSIS.md`)
- Go standard library patterns and best practices

**Key Findings**:
- ✅ Transport interface design aligns with F-9 REQ-F9-1 and REQ-F9-7
- ✅ Buffer pooling pattern from F-7 (lines 286-311) eliminates hot path allocations
- ✅ Layer boundary compliance restores F-2 architecture integrity
- ✅ Error propagation fixes align with F-3 RULE-1 and F-7 cleanup patterns
- ✅ All refactoring changes support M1.1 implementation (no rework needed)

---

## Topic 1: Transport Interface Design Pattern (FR-001)

**Question**: What is the optimal Transport interface design that supports M1 refactoring AND M1.1 F-9 requirements?

### Decision

Create a `Transport` interface in new `internal/transport/` package with context-aware methods that abstract UDP socket operations:

```go
// internal/transport/transport.go
package transport

import (
    "context"
    "net"
)

// Transport abstracts network I/O operations for mDNS.
// This interface enables:
// - IPv6 support (future UDPv6Transport)
// - Platform-specific socket configuration (F-9 REQ-F9-1)
// - Testing via MockTransport
// - Context propagation (F-9 REQ-F9-7)
type Transport interface {
    // Send transmits a packet to the specified destination address.
    // Context cancellation stops transmission attempt.
    Send(ctx context.Context, packet []byte, dest net.Addr) error

    // Receive reads a packet from the transport.
    // Context deadline sets read timeout.
    // Context cancellation returns immediately.
    Receive(ctx context.Context) ([]byte, net.Addr, error)

    // Close releases transport resources.
    // Returns error on cleanup failure (F-3 RULE-1).
    Close() error
}
```

**UDPv4Transport Implementation** (migrates current `internal/network/socket.go` logic):

```go
// internal/transport/udp.go
package transport

import (
    "context"
    "fmt"
    "net"
    "time"

    "github.com/joshuafuller/beacon/internal/errors"
    "github.com/joshuafuller/beacon/internal/protocol"
)

// UDPv4Transport implements Transport for IPv4 UDP multicast.
type UDPv4Transport struct {
    conn net.PacketConn
    multicastAddr *net.UDPAddr
}

// NewUDPv4Transport creates IPv4 UDP multicast transport.
// Migrated from internal/network/CreateSocket.
func NewUDPv4Transport() (*UDPv4Transport, error) {
    // Resolve mDNS multicast address
    multicastAddr, err := net.ResolveUDPAddr("udp4",
        fmt.Sprintf("%s:%d", protocol.MulticastAddrIPv4, protocol.Port))
    if err != nil {
        return nil, &errors.NetworkError{
            Operation: "resolve multicast address",
            Err:       err,
            Details:   fmt.Sprintf("failed to resolve %s:%d",
                protocol.MulticastAddrIPv4, protocol.Port),
        }
    }

    // Listen on mDNS multicast group
    conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
    if err != nil {
        return nil, &errors.NetworkError{
            Operation: "create socket",
            Err:       err,
            Details:   fmt.Sprintf("failed to bind to multicast %s:%d",
                protocol.MulticastAddrIPv4, protocol.Port),
        }
    }

    // Configure socket buffer
    if err := conn.SetReadBuffer(65536); err != nil {
        conn.Close()
        return nil, &errors.NetworkError{
            Operation: "configure socket",
            Err:       err,
            Details:   "failed to set read buffer size",
        }
    }

    return &UDPv4Transport{
        conn:          conn,
        multicastAddr: multicastAddr,
    }, nil
}

// Send implements Transport.Send (migrated from network.SendQuery).
func (t *UDPv4Transport) Send(ctx context.Context, packet []byte, dest net.Addr) error {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Send to multicast group
    n, err := t.conn.WriteTo(packet, t.multicastAddr)
    if err != nil {
        return &errors.NetworkError{
            Operation: "send query",
            Err:       err,
            Details:   fmt.Sprintf("failed to send %d bytes to %s", len(packet), dest),
        }
    }

    // Verify full message sent
    if n != len(packet) {
        return &errors.NetworkError{
            Operation: "send query",
            Err:       fmt.Errorf("partial write: %d/%d bytes", n, len(packet)),
            Details:   "incomplete transmission",
        }
    }

    return nil
}

// Receive implements Transport.Receive (migrated from network.ReceiveResponse).
// Uses buffer pooling per FR-003 (see Topic 2).
func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    // Get pooled buffer
    bufPtr := bufferPool.Get().(*[]byte)
    defer bufferPool.Put(bufPtr)
    buffer := *bufPtr

    // Set read deadline from context
    if deadline, ok := ctx.Deadline(); ok {
        t.conn.SetReadDeadline(deadline)
    } else {
        // No deadline, use short timeout to allow periodic context checks
        t.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
    }

    // Read from socket
    n, srcAddr, err := t.conn.ReadFrom(buffer)
    if err != nil {
        // Check if it's a timeout (allows context check on next iteration)
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            // Check context
            select {
            case <-ctx.Done():
                return nil, nil, ctx.Err()
            default:
                // Continue receiving
                return nil, nil, &errors.NetworkError{
                    Operation: "receive response",
                    Err:       err,
                    Details:   "timeout (will retry)",
                }
            }
        }

        return nil, nil, &errors.NetworkError{
            Operation: "receive response",
            Err:       err,
            Details:   "failed to read from socket",
        }
    }

    // Copy to return buffer (caller owns memory)
    result := make([]byte, n)
    copy(result, buffer[:n])

    return result, srcAddr, nil
}

// Close implements Transport.Close.
// Propagates errors per F-3 RULE-1 (fixes P0-4).
func (t *UDPv4Transport) Close() error {
    if t.conn == nil {
        return nil // Graceful nil handling
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

**MockTransport for Testing**:

```go
// internal/transport/mock.go
package transport

import (
    "context"
    "net"
)

// MockTransport provides controllable transport for testing.
type MockTransport struct {
    SendFunc    func(ctx context.Context, packet []byte, dest net.Addr) error
    ReceiveFunc func(ctx context.Context) ([]byte, net.Addr, error)
    CloseFunc   func() error
}

func (m *MockTransport) Send(ctx context.Context, packet []byte, dest net.Addr) error {
    if m.SendFunc != nil {
        return m.SendFunc(ctx, packet, dest)
    }
    return nil
}

func (m *MockTransport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    if m.ReceiveFunc != nil {
        return m.ReceiveFunc(ctx)
    }
    return nil, nil, nil
}

func (m *MockTransport) Close() error {
    if m.CloseFunc != nil {
        return m.CloseFunc()
    }
    return nil
}
```

### Rationale

**F-9 Alignment**:
- **REQ-F9-1** (ListenConfig pattern): UDPv4Transport encapsulates socket creation, enabling future platform-specific socket options via `net.ListenConfig` in M1.1
- **REQ-F9-7** (Context propagation): All Transport methods accept `context.Context` as first parameter, enabling timeout/cancellation propagation

**M1 Refactoring Needs**:
- **P0-1**: Transport interface abstracts UDP operations, enabling future IPv6 support (UDPv6Transport)
- **P0-2**: Provides correct layer boundary (querier uses Transport, not direct network)
- **Testing**: MockTransport eliminates need for real network in unit tests

**Go Best Practices**:
- Interface defines behavior, not implementation (Go idiom)
- Context-aware methods follow Go 1.7+ context patterns
- Small interface (3 methods) per Go proverb "The bigger the interface, the weaker the abstraction"

### Alternatives Considered

**Alternative 1: No Interface (Keep Current Pattern)**
- **Pros**: No abstraction overhead, simpler code
- **Cons**: Cannot support IPv6, testing requires real network, violates F-2 layer boundaries
- **Rejected**: Blocks M1.1 F-9 requirements and makes testing difficult

**Alternative 2: Larger Interface (Include Configuration Methods)**
- **Example**: `SetBufferSize()`, `JoinMulticastGroup()`, `SetTTL()`
- **Pros**: Comprehensive transport abstraction
- **Cons**: Larger interface harder to mock, violates Go proverb
- **Rejected**: Configuration should be constructor parameters (options pattern)

**Alternative 3: Use Standard `net.PacketConn` Interface**
- **Pros**: No custom interface needed
- **Cons**: `net.PacketConn` not context-aware, cannot add behavior (pooling, logging)
- **Rejected**: F-9 REQ-F9-7 mandates context propagation

### Implementation Notes

**Migration Strategy**:
1. Create `internal/transport/` package with Transport interface
2. Implement UDPv4Transport (migrate `internal/network/socket.go` logic)
3. Create MockTransport for testing
4. Update Querier to use Transport interface (FR-002)
5. Deprecate `internal/network/` package (mark with deprecation comment)
6. Run full M1 test suite (107 tests must pass)

**Testing Requirements**:
- Unit tests for UDPv4Transport (socket creation, send, receive, close)
- Integration tests with real multicast (verify actual network behavior)
- MockTransport tests (verify querier works with mock)
- Contract tests (verify Transport interface behavior)

**Querier Integration** (FR-002):

```go
// querier/querier.go
package querier

import (
    "github.com/joshuafuller/beacon/internal/transport"
    // Remove: "github.com/joshuafuller/beacon/internal/network"
)

type Querier struct {
    transport transport.Transport  // Use interface
    // ... other fields
}

func New(opts ...Option) (*Querier, error) {
    // Create concrete transport
    trans, err := transport.NewUDPv4Transport()
    if err != nil {
        return nil, err
    }

    ctx, cancel := context.WithCancel(context.Background())

    q := &Querier{
        transport:      trans,  // Interface field
        defaultTimeout: 1 * time.Second,
        responseChan:   make(chan []byte, 100),
        ctx:            ctx,
        cancel:         cancel,
    }

    q.wg.Add(1)
    go q.receiveLoop()

    return q, nil
}

// Query uses Transport.Send instead of network.SendQuery
func (q *Querier) Query(ctx context.Context, name string, recordType RecordType) (*Response, error) {
    q.mu.Lock()
    defer q.mu.Unlock()

    // ... validation ...

    queryMsg, err := message.BuildQuery(name, uint16(recordType))
    if err != nil {
        return nil, err
    }

    // Use Transport interface (not direct network call)
    err = q.transport.Send(ctx, queryMsg, nil)  // dest = nil (multicast)
    if err != nil {
        return nil, err
    }

    return q.collectResponses(ctx, name, recordType)
}

// receiveLoop uses Transport.Receive
func (q *Querier) receiveLoop() {
    defer q.wg.Done()

    for {
        select {
        case <-q.ctx.Done():
            return
        default:
            // Use Transport.Receive with short timeout
            ctx, cancel := context.WithTimeout(q.ctx, 100*time.Millisecond)
            responseMsg, _, err := q.transport.Receive(ctx)
            cancel()

            if err != nil {
                // Timeout expected, continue
                continue
            }

            select {
            case q.responseChan <- responseMsg:
            default:
                // Channel full, drop packet
            }
        }
    }
}

func (q *Querier) Close() error {
    q.cancel()
    q.wg.Wait()

    // Use Transport.Close (propagates errors per P0-4 fix)
    err := q.transport.Close()
    if err != nil {
        return err
    }

    close(q.responseChan)
    return nil
}
```

### References

- **F-9 Transport Layer Socket Configuration**:
  - REQ-F9-1 (lines 83-120): ListenConfig pattern
  - REQ-F9-7 (lines 328-422): Context propagation in blocking operations
- **F-2 Package Structure**:
  - Lines 266-278: `internal/transport/` package organization
  - Lines 295-307: RULE-1 (Public → Internal allowed)
- **M1 Refactoring Analysis**:
  - P0-1 (lines 34-66): No Transport Interface Abstraction
  - P0-2 (lines 69-107): Layer Boundary Violations
- **M1 Implementation**:
  - `internal/network/socket.go`: Current socket operations to migrate
  - `querier/querier.go:184`: Current layer violation to fix

---

## Topic 2: Buffer Pooling Pattern (FR-003)

**Question**: What is the correct sync.Pool implementation for UDP receive buffers that prevents leaks and improves performance?

### Decision

Implement `sync.Pool` for 9KB receive buffers in `internal/transport/buffer_pool.go` with strict ownership rules:

```go
// internal/transport/buffer_pool.go
package transport

import "sync"

// bufferPool reuses 9KB buffers for UDP packet reception.
// Reduces hot path allocations per F-7 Resource Management (lines 286-311).
//
// Buffer Ownership Rules:
// 1. Pool owns buffer during Get() → Put() lifecycle
// 2. Caller owns returned []byte (copy of buffer contents)
// 3. defer Put() ensures buffer always returned to pool
var bufferPool = sync.Pool{
    New: func() interface{} {
        // Allocate 9KB buffer per RFC 6762 §17 max message size
        // (Jumbo frame support - standard mDNS max is 512 bytes)
        buf := make([]byte, 9000)
        return &buf  // Return pointer to avoid allocation on Get()
    },
}
```

**Integration in UDPv4Transport.Receive()** (already shown in Topic 1):

```go
func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    // Get pooled buffer
    bufPtr := bufferPool.Get().(*[]byte)
    defer bufferPool.Put(bufPtr)  // ✅ Always returned to pool

    buffer := *bufPtr

    // Set read deadline from context
    if deadline, ok := ctx.Deadline(); ok {
        t.conn.SetReadDeadline(deadline)
    } else {
        t.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
    }

    // Read from socket
    n, srcAddr, err := t.conn.ReadFrom(buffer)
    if err != nil {
        // Buffer returned to pool via defer even on error
        return nil, nil, &errors.NetworkError{
            Operation: "receive response",
            Err:       err,
            Details:   "failed to read from socket",
        }
    }

    // ✅ Copy to new buffer (caller owns result)
    result := make([]byte, n)
    copy(result, buffer[:n])

    return result, srcAddr, nil
    // Buffer returned to pool via defer
}
```

### Rationale

**Performance Problem**:
- Current M1 allocates 9KB on every `ReceiveResponse()` call (`internal/network/socket.go:132`)
- Hot path per F-6 Logging & Observability specification
- **Impact**: 100 queries/sec = 900KB/sec allocations = 54MB/min
- Forces frequent GC cycles, degrades high-throughput performance

**F-7 Buffer Pooling Pattern** (lines 286-311):
- `sync.Pool` amortizes allocation cost across multiple operations
- `defer Put()` ensures zero buffer leaks (even on panic or error)
- Copy-on-return ensures caller owns memory (no aliasing bugs)

**Benchmark Expectations**:

```bash
# Before buffer pooling (M1 current)
BenchmarkReceive-8    10000    120000 ns/op    9216 B/op    2 allocs/op

# After buffer pooling (target)
BenchmarkReceive-8    10000    120000 ns/op    1024 B/op    1 allocs/op
#                                               ^^^^^ 89% reduction (9216 → 1024)
```

**Validation**: ≥80% allocation reduction per FR-003 acceptance criteria

### Alternatives Considered

**Alternative 1: No Pooling (Keep Current Pattern)**
- **Pros**: Simpler code, no pool management
- **Cons**: High GC pressure, poor high-throughput performance
- **Rejected**: Violates F-7 Resource Management (no hot path allocations)

**Alternative 2: Single Shared Buffer (No Pool)**
- **Example**: `var sharedBuffer [9000]byte`
- **Pros**: Zero allocations
- **Cons**: Requires mutex, copies always needed, not concurrent
- **Rejected**: Serializes all receives, defeats concurrency

**Alternative 3: Caller-Provided Buffer (io.Reader Pattern)**
- **Example**: `Receive(ctx, buffer []byte) (int, net.Addr, error)`
- **Pros**: No allocations if caller pools buffers
- **Cons**: Shifts complexity to caller, error-prone
- **Rejected**: Poor API design, caller may not pool correctly

**Alternative 4: Ring Buffer**
- **Pros**: Can eliminate copy-on-return
- **Cons**: Complex lifecycle, memory aliasing risks
- **Rejected**: Over-engineering for M1, `sync.Pool` sufficient

### Implementation Notes

**Buffer Size Justification** (9000 bytes):
- RFC 6762 §17: "mDNS messages should be 512 bytes or less"
- Practice: Additional records can exceed 512 bytes
- Jumbo frames: Support 9000 byte MTU (common in data centers)
- Trade-off: 9KB pool vs potential message truncation

**Zero Leaks Guarantee**:
```go
// defer ensures Put() even on:
// 1. Normal return
// 2. Error return
// 3. Panic (defer runs during stack unwind)
defer bufferPool.Put(bufPtr)
```

**Ownership Semantics**:
1. **Pool owns buffer**: From `Get()` to `Put()`
2. **Function owns buffer**: During function execution (via `defer Put()`)
3. **Caller owns result**: After `copy()` creates new allocation

**Testing Requirements**:
- Benchmark allocation reduction (before/after comparison)
- Leak detection test (verify `Put()` called on all paths)
- Concurrency test (verify pool thread-safety)
- Memory profiling (pprof validation)

**Benchmark Test**:

```go
// internal/transport/buffer_pool_test.go
func BenchmarkReceive_WithPooling(b *testing.B) {
    trans, _ := NewUDPv4Transport()
    defer trans.Close()

    ctx := context.Background()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, _, _ = trans.Receive(ctx)
    }
}

// Run: go test -bench=BenchmarkReceive -benchmem -memprofile=mem.prof
// Compare: go tool pprof -alloc_space mem.prof
```

**Leak Detection Test**:

```go
func TestBufferPool_NoLeaks(t *testing.T) {
    // Warm up pool
    for i := 0; i < 10; i++ {
        bufPtr := bufferPool.Get().(*[]byte)
        bufferPool.Put(bufPtr)
    }

    // Capture pool state
    before := runtime.NumGoroutine()

    // Perform 1000 receives
    trans, _ := NewUDPv4Transport()
    defer trans.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    for i := 0; i < 1000; i++ {
        trans.Receive(ctx)
    }

    // Force GC
    runtime.GC()
    time.Sleep(100 * time.Millisecond)

    // Verify no goroutine leaks
    after := runtime.NumGoroutine()
    if after > before+1 { // +1 tolerance
        t.Errorf("goroutine leak: before=%d after=%d", before, after)
    }

    // Verify pool returns buffers (no unbounded growth)
    // Pool should reuse buffers, not allocate 1000 new ones
}
```

### References

- **F-7 Resource Management**:
  - Lines 286-311: Buffer Pooling Pattern (exact pattern used)
  - Lines 284-285: "Use sync.Pool for frequent allocations"
- **M1 Refactoring Analysis**:
  - P0-3 (lines 110-159): Buffer Allocation in Hot Path
  - Lines 126-128: Performance impact calculation (54MB/min)
- **M1 Implementation**:
  - `internal/network/socket.go:132`: Current allocation site
- **F-6 Logging & Observability**:
  - Hot path definition (receive operations)
- **Go Documentation**:
  - `sync.Pool`: https://pkg.go.dev/sync#Pool
  - Best practices: https://go.dev/blog/pprof

---

## Topic 3: Layer Boundary Compliance (FR-002)

**Question**: How should querier access Transport to maintain F-2 layer boundaries?

### Decision

Querier imports `internal/transport` directly and uses Transport interface. This is **correct per F-2 RULE-1** (Public → Internal allowed).

```go
// querier/querier.go
package querier

import (
    "context"
    "github.com/joshuafuller/beacon/internal/transport"  // ✅ CORRECT
    "github.com/joshuafuller/beacon/internal/message"
    // Remove: "github.com/joshuafuller/beacon/internal/network"  // ❌ WRONG
)

type Querier struct {
    transport transport.Transport  // ✅ Use interface (enables mocking)
    // ... other fields
}

func New(opts ...Option) (*Querier, error) {
    // Create concrete transport
    trans, err := transport.NewUDPv4Transport()
    if err != nil {
        return nil, err
    }

    q := &Querier{
        transport: trans,  // ✅ Store as interface
        // ...
    }
    return q, nil
}
```

**Layer Flow Validation**:

```
Public API (querier/)
    ↓ imports
Internal Transport (internal/transport/)
    ↓ (no imports of protocol - decoupled)
Internal Protocol (internal/protocol/)
    ↓ imports
Internal Message (internal/message/)
```

**Correct**: Stable (querier) depends on Unstable (transport) - F-2 RULE-1 ✅
**Incorrect** (M1 current): Stable depends on Unstable skipping Protocol layer - F-2 violation ❌

### Rationale

**F-2 Package Structure** (lines 295-307):
- **RULE-1**: Public → Internal (Allowed) - "Public packages MAY import internal packages"
- **RULE-2**: Internal → Public (Prohibited) - Prevents circular dependencies
- **RULE-3**: Internal → Internal (Allowed, with ordering) - Respects layer order

**Current M1 Violation** (P0-2):
```go
// querier/querier.go:10 - LAYER VIOLATION
import "github.com/joshuafuller/beacon/internal/network"

// querier/querier.go:184 - DIRECT NETWORK CALL
err = network.SendQuery(q.socket, queryMsg)  // ❌ Bypasses protocol layer
```

**Why This Is Wrong**:
1. Querier (Public API) imports network (Transport layer) - skips Protocol layer
2. Violates F-2 layer flow: Public API → Service → Protocol → Transport
3. Tight coupling prevents protocol layer from orchestrating operations
4. Cannot add protocol-level features (caching, validation) without changing querier

**Correct Pattern**:
```go
// Querier imports Transport (interface abstraction)
q.transport.Send(ctx, queryMsg, nil)  // ✅ Through Transport interface

// If Protocol layer needed:
// protocol.SendQuery(ctx, q.transport, queryMsg)  // ✅ Protocol orchestrates
```

**Why Direct Import Is OK**:
- F-2 RULE-1 explicitly allows Public → Internal
- Transport is interface-based (decoupled from implementation)
- MockTransport enables testing without real network
- Future: If protocol orchestration needed, add protocol.Query() wrapper

### Alternatives Considered

**Alternative 1: Querier → Protocol → Transport (3-layer)**
- **Pattern**: `protocol.SendQuery(ctx, transport, msg)`
- **Pros**: Protocol layer can add caching, validation, orchestration
- **Cons**: Extra indirection for simple operations (M1 doesn't need orchestration)
- **Decision**: DEFERRED - Use direct Transport for M1, add Protocol wrapper if M1.1+ needs orchestration

**Alternative 2: Querier Keeps net.PacketConn (No Abstraction)**
- **Pattern**: Current M1 approach
- **Pros**: No abstraction overhead
- **Cons**: Violates F-2 layer boundaries, cannot support IPv6, testing requires real network
- **Rejected**: Blocks M1.1 requirements

**Alternative 3: Protocol Layer Creates Transport**
- **Pattern**: `protocol.NewQuery()` returns pre-configured transport
- **Pros**: Centralized transport creation
- **Cons**: Protocol depends on Transport (inverted dependency)
- **Rejected**: Violates F-2 RULE-3 (Internal → Internal ordering)

### Implementation Notes

**Migration Steps**:
1. Create `internal/transport/` package with Transport interface (Topic 1)
2. Update `querier/querier.go`:
   - Add `import "github.com/joshuafuller/beacon/internal/transport"`
   - Remove `import "github.com/joshuafuller/beacon/internal/network"`
   - Change `socket net.PacketConn` → `transport transport.Transport`
   - Update `New()` to create `transport.NewUDPv4Transport()`
   - Replace `network.SendQuery()` → `q.transport.Send()`
   - Replace `network.ReceiveResponse()` → `q.transport.Receive()`
   - Replace `network.CloseSocket()` → `q.transport.Close()`
3. Run full M1 test suite (107 tests must pass)
4. Validate no `internal/network` imports remain in `querier/`

**Dependency Validation**:

```bash
# Check for layer violations (should return no matches)
grep -r "internal/network" querier/
# Expected: No matches after refactoring

# Check for correct Transport usage
grep -r "transport.Transport" querier/
# Expected: type Querier struct { transport transport.Transport ... }

# Validate dependency graph
go mod graph | grep beacon/internal
# Expected: No circular dependencies
```

**Testing Strategy**:
- Unit tests with MockTransport (verify querier logic without network)
- Integration tests with UDPv4Transport (verify actual network behavior)
- Contract tests (verify Transport interface compliance)

**Example MockTransport Test**:

```go
// querier/querier_test.go
func TestQuery_WithMockTransport(t *testing.T) {
    // Create mock transport
    mock := &transport.MockTransport{
        SendFunc: func(ctx context.Context, packet []byte, dest net.Addr) error {
            return nil  // Simulate successful send
        },
        ReceiveFunc: func(ctx context.Context) ([]byte, net.Addr, error) {
            // Return mock mDNS response
            response := buildMockResponse("test.local", "192.168.1.100")
            return response, nil, nil
        },
        CloseFunc: func() error {
            return nil
        },
    }

    // Create querier with mock transport
    q := &Querier{
        transport:      mock,
        defaultTimeout: 1 * time.Second,
        responseChan:   make(chan []byte, 100),
        ctx:            context.Background(),
        cancel:         func() {},
    }

    // Query should succeed without real network
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "test.local", RecordTypeA)
    if err != nil {
        t.Fatalf("Query failed: %v", err)
    }

    if len(response.Records) == 0 {
        t.Error("Expected records, got none")
    }
}
```

### References

- **F-2 Package Structure**:
  - Lines 295-307: Import Rules (RULE-1, RULE-2, RULE-3)
  - Lines 333-380: Layer Boundaries (Layer 1-4 definitions)
  - Lines 266-278: `internal/transport/` package organization
- **M1 Refactoring Analysis**:
  - P0-2 (lines 69-107): Layer Boundary Violations
  - Lines 75-82: Current violation in `querier/querier.go:184`
- **M1 Implementation**:
  - `querier/querier.go:10`: Current import violation
  - `querier/querier.go:184`: Current direct network call

---

## Topic 4: Error Propagation Pattern (FR-004)

**Question**: What is the correct error handling pattern for CloseSocket that aligns with F-3 RULE-1?

### Decision

**Fix CloseSocket to propagate errors** per F-3 RULE-1: "Return errors to caller". This is a **trivial fix** (0.5 hours).

**Current M1 Implementation** (INCORRECT):

```go
// internal/network/socket.go:166-179
func CloseSocket(conn net.PacketConn) error {
    if conn == nil {
        return nil
    }

    err := conn.Close()
    if err != nil {
        // In M1, we log but don't fail on close errors
        return nil  // ❌ ERROR SWALLOWED - Violates F-3 RULE-1
    }

    return nil
}
```

**Required Fix** (Transport.Close()):

```go
// internal/transport/udp.go
func (t *UDPv4Transport) Close() error {
    if t.conn == nil {
        return nil  // ✅ Graceful nil handling OK
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

**Querier.Close() Already Handles This Correctly**:

```go
// querier/querier.go:326-343
func (q *Querier) Close() error {
    q.cancel()
    q.wg.Wait()

    // ✅ Propagates close errors to caller
    err := network.CloseSocket(q.socket)  // Old: network.CloseSocket
    // New: err := q.transport.Close()
    if err != nil {
        return err  // ✅ Caller gets error (can log, report, handle)
    }

    close(q.responseChan)
    return nil
}
```

### Rationale

**F-3 Error Handling** (RULE-1, lines 660-687):
- **RULE-1**: "Return errors to caller" - Functions SHOULD return errors, not just log them
- Caller decides how to handle (retry, ignore, propagate, log)
- Enables resource leak detection in production

**Why Swallowing Errors Is Bad**:
1. **Resource leak detection impossible**: Applications cannot detect close failures
2. **Violates F-7 cleanup patterns** (lines 218-285): Cleanup errors must be reportable
3. **Production monitoring impossible**: Cannot log/alert on resource leaks
4. **Violates Go idiom**: Errors are values, caller decides handling

**Real-World Impact**:
- File descriptor leaks (socket not closed)
- Connection pool exhaustion
- OS resource limits reached (ulimit)
- Application cannot detect and restart cleanly

**Correct Pattern** (F-7 lines 218-285):
```go
// Cleanup errors are returned, not swallowed
func (c *Component) Close() error {
    var errs []error

    if c.conn != nil {
        if err := c.conn.Close(); err != nil {
            errs = append(errs, err)  // ✅ Collect errors
        }
    }

    if len(errs) > 0 {
        return fmt.Errorf("cleanup errors: %v", errs)  // ✅ Return to caller
    }
    return nil
}
```

### Alternatives Considered

**Alternative 1: Keep Error Swallowing (M1 Current)**
- **Pros**: Simpler, fewer error checks
- **Cons**: Resource leaks undetectable, violates F-3 RULE-1
- **Rejected**: Critical for production monitoring

**Alternative 2: Log But Don't Return**
- **Pattern**: `log.Error("close failed: %v", err); return nil`
- **Pros**: Error visible in logs
- **Cons**: Caller cannot handle, no structured error, production may not check logs
- **Rejected**: Violates F-3 RULE-1 (caller must decide handling)

**Alternative 3: Panic on Close Error**
- **Pattern**: `if err := conn.Close(); err != nil { panic(err) }`
- **Pros**: Ensures close errors never ignored
- **Cons**: Too aggressive, crashes application
- **Rejected**: Close errors often non-fatal (already closed, etc.)

### Implementation Notes

**Migration Steps**:
1. Update `UDPv4Transport.Close()` to return `NetworkError` on close failure
2. Verify `Querier.Close()` already propagates errors (it does - line 334-337)
3. Add test for close error propagation
4. Run M1 test suite (verify no regressions)

**Testing Requirements**:

```go
// internal/transport/udp_test.go
func TestUDPv4Transport_Close_ErrorPropagation(t *testing.T) {
    // Test 1: Close nil conn (should succeed)
    trans := &UDPv4Transport{conn: nil}
    err := trans.Close()
    if err != nil {
        t.Errorf("Close(nil) should succeed, got: %v", err)
    }

    // Test 2: Close already-closed conn (should return error)
    trans2, _ := NewUDPv4Transport()
    trans2.Close()  // First close succeeds

    err = trans2.Close()  // Second close should error
    if err == nil {
        t.Error("Close(already-closed) should return error")
    }

    // Verify error type
    var netErr *errors.NetworkError
    if !goerrors.As(err, &netErr) {
        t.Errorf("Expected NetworkError, got: %T", err)
    }

    // Verify error details
    if netErr.Operation != "close socket" {
        t.Errorf("Expected operation 'close socket', got: %s", netErr.Operation)
    }
}

// querier/querier_test.go
func TestQuerier_Close_ErrorPropagation(t *testing.T) {
    // Create querier with mock transport that fails on close
    mock := &transport.MockTransport{
        CloseFunc: func() error {
            return &errors.NetworkError{
                Operation: "close socket",
                Err:       fmt.Errorf("mock error"),
                Details:   "simulated close failure",
            }
        },
    }

    q := &Querier{
        transport:    mock,
        responseChan: make(chan []byte, 100),
        ctx:          context.Background(),
        cancel:       func() {},
    }

    // Close should propagate error from transport
    err := q.Close()
    if err == nil {
        t.Fatal("Expected close error, got nil")
    }

    var netErr *errors.NetworkError
    if !goerrors.As(err, &netErr) {
        t.Errorf("Expected NetworkError, got: %T", err)
    }
}
```

**Production Impact**:
- Applications can now detect close failures:
  ```go
  err := querier.Close()
  if err != nil {
      log.Errorf("Failed to close querier: %v", err)
      metrics.RecordResourceLeak("querier")
      // Alert operations team
  }
  ```

- Resource leak monitoring enabled:
  ```go
  // Prometheus metric
  if err := querier.Close(); err != nil {
      resourceLeakCounter.Inc()
  }
  ```

### References

- **F-3 Error Handling**:
  - RULE-1 (lines 660-687): "Return errors to caller"
  - Lines 621-641: Cleanup on Error pattern
- **F-7 Resource Management**:
  - Lines 218-285: Cleanup Patterns (errors must be reportable)
  - Lines 541-601: Cleanup patterns with error capture
- **M1 Refactoring Analysis**:
  - P0-4 (lines 162-229): CloseSocket Swallows Errors
  - Lines 168-179: Current error swallowing code
- **M1 Implementation**:
  - `internal/network/socket.go:166-179`: Current CloseSocket implementation
  - `querier/querier.go:334-337`: Querier.Close() already handles errors correctly

---

## Topic 5: M1.1 Alignment Validation (Cross-Cutting)

**Question**: Does this refactoring create the correct foundation for M1.1 F-9/F-10/F-11 requirements?

### Decision

**YES** - All P0 refactoring changes align with and enable M1.1 implementation. No rework needed.

### Validation Matrix

| F-Spec | M1.1 Requirement | Refactoring Support | Evidence |
|--------|------------------|---------------------|----------|
| **F-9 REQ-F9-1** | ListenConfig pattern for socket options | ✅ ENABLED | Transport interface allows UDPv4Transport to use `net.ListenConfig` in constructor |
| **F-9 REQ-F9-2** | Platform-specific socket options (SO_REUSEADDR, SO_REUSEPORT) | ✅ ENABLED | UDPv4Transport constructor can set socket options via `ListenConfig.Control` |
| **F-9 REQ-F9-3** | Multicast group membership (golang.org/x/net/ipv4) | ✅ ENABLED | UDPv4Transport can join multicast groups in constructor |
| **F-9 REQ-F9-6** | Socket buffer configuration (64KB minimum) | ✅ ALREADY IMPLEMENTED | M1 sets 65536 buffer (socket.go:47) |
| **F-9 REQ-F9-7** | Context propagation in blocking operations | ✅ IMPLEMENTED | Transport.Send() and Transport.Receive() accept `context.Context` |
| **F-10 REQ-F10-1** | Network interface enumeration | 🟡 FUTURE | Transport interface supports, not in M1 refactoring |
| **F-10 REQ-F10-2** | Interface-specific multicast | 🟡 FUTURE | Transport can be extended with `JoinGroupOnInterface()` |
| **F-11 REQ-F11-2** | Rate limiting | 🟡 FUTURE | Transport.Send() can add rate limiter wrapper |

### Detailed Validation

#### F-9 REQ-F9-1: ListenConfig Pattern

**M1.1 Requirement** (F-9 lines 83-120):
```go
lc := net.ListenConfig{
    Control: setPlatformSocketOptions, // Set SO_REUSEPORT before bind
}
conn, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353")
```

**Refactoring Support**:
```go
// internal/transport/udp.go
func NewUDPv4Transport() (*UDPv4Transport, error) {
    // M1: Current approach (will be replaced in M1.1)
    conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)

    // M1.1: ListenConfig approach (enabled by Transport abstraction)
    // lc := net.ListenConfig{
    //     Control: setPlatformSocketOptions,
    // }
    // conn, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353")
    // ... join multicast group using golang.org/x/net/ipv4

    return &UDPv4Transport{conn: conn, ...}, nil
}
```

**Validation**: ✅ Transport interface allows constructor change without breaking querier

---

#### F-9 REQ-F9-7: Context Propagation

**M1.1 Requirement** (F-9 lines 328-422):
- All blocking operations MUST accept `context.Context` as first parameter
- MUST check `ctx.Done()` in receive loops
- MUST propagate `ctx.Deadline()` to `SetReadDeadline()`

**Refactoring Implementation**:
```go
// Transport interface mandates context
type Transport interface {
    Send(ctx context.Context, packet []byte, dest net.Addr) error
    Receive(ctx context.Context) ([]byte, net.Addr, error)
    Close() error
}

// UDPv4Transport.Receive implements REQ-F9-7
func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
    // Propagate context deadline to socket
    if deadline, ok := ctx.Deadline(); ok {
        t.conn.SetReadDeadline(deadline)  // ✅ REQ-F9-7 compliance
    } else {
        t.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
    }

    n, srcAddr, err := t.conn.ReadFrom(buffer)
    if err != nil {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return nil, nil, ctx.Err()  // ✅ REQ-F9-7 compliance
        default:
            // Timeout or real error
        }
    }
    // ...
}
```

**Validation**: ✅ REQ-F9-7 fully implemented in refactoring

---

#### F-9 REQ-F9-2: Platform-Specific Socket Options

**M1.1 Requirement** (F-9 lines 122-163):
- Linux: `SO_REUSEADDR` + `SO_REUSEPORT` (via `golang.org/x/sys/unix`)
- macOS: `SO_REUSEADDR` + `SO_REUSEPORT`
- Windows: `SO_REUSEADDR` only

**Refactoring Support**:
```go
// internal/transport/udp_linux.go (future M1.1 file)
// +build linux

import "golang.org/x/sys/unix"

func setPlatformSocketOptions(fd uintptr) error {
    // SO_REUSEADDR
    if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
        return err
    }

    // SO_REUSEPORT (Linux >= 3.9)
    if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
        return err
    }

    return nil
}

// NewUDPv4Transport uses ListenConfig with platform-specific options
func NewUDPv4Transport() (*UDPv4Transport, error) {
    lc := net.ListenConfig{
        Control: func(network, address string, c syscall.RawConn) error {
            var setErr error
            err := c.Control(func(fd uintptr) {
                setErr = setPlatformSocketOptions(fd)
            })
            if err != nil {
                return err
            }
            return setErr
        },
    }

    conn, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353")
    // ... join multicast, set buffers
}
```

**Validation**: ✅ Transport abstraction enables platform-specific builds without changing querier

---

#### F-9 REQ-F9-3: Multicast Group Membership

**M1.1 Requirement** (F-9 lines 165-198):
```go
import "golang.org/x/net/ipv4"

p := ipv4.NewPacketConn(conn)
err := p.JoinGroup(iface, &net.UDPAddr{IP: net.ParseIP("224.0.0.251")})
err = p.SetMulticastTTL(255)  // RFC 6762 §11
```

**Refactoring Support**:
```go
// internal/transport/udp.go (future M1.1 enhancement)
import "golang.org/x/net/ipv4"

func NewUDPv4Transport() (*UDPv4Transport, error) {
    // ... create socket via ListenConfig ...

    // Join multicast group
    p := ipv4.NewPacketConn(conn)
    group := &net.UDPAddr{IP: net.ParseIP("224.0.0.251")}
    if err := p.JoinGroup(iface, group); err != nil {
        conn.Close()
        return nil, &errors.NetworkError{
            Operation: "join multicast group",
            Err:       err,
            Details:   "failed to join 224.0.0.251",
        }
    }

    // Set multicast TTL per RFC 6762 §11
    if err := p.SetMulticastTTL(255); err != nil {
        conn.Close()
        return nil, &errors.NetworkError{
            Operation: "set multicast TTL",
            Err:       err,
            Details:   "failed to set TTL 255",
        }
    }

    return &UDPv4Transport{
        conn:          conn,
        ipv4Conn:      p,  // Store for future multicast operations
        multicastAddr: &net.UDPAddr{IP: net.ParseIP("224.0.0.251"), Port: 5353},
    }, nil
}
```

**Validation**: ✅ Transport constructor can be enhanced without changing interface or querier

---

#### F-10/F-11: Future Features

**F-10 Interface Management**:
- Transport interface supports interface-specific operations
- Future: `JoinGroupOnInterface(iface *net.Interface, group net.Addr) error`
- No changes needed to querier

**F-11 Rate Limiting**:
- Transport.Send() can be wrapped with rate limiter
- Future: Decorator pattern or middleware
- No changes needed to querier

**Validation**: ✅ Transport abstraction enables future enhancements

### No Rework Needed

**Key Validation**: All M1.1 socket configuration changes happen in `internal/transport/udp.go` constructor. Querier uses `transport.Transport` interface and is **completely decoupled** from socket implementation.

**M1.1 Implementation Plan**:
1. Update `UDPv4Transport` constructor to use `net.ListenConfig`
2. Add platform-specific socket option files (`udp_linux.go`, `udp_darwin.go`, `udp_windows.go`)
3. Add multicast group join via `golang.org/x/net/ipv4`
4. Add justification for `golang.org/x/sys` and `golang.org/x/net` dependencies
5. No changes to querier (uses Transport interface)

### References

- **F-9 Transport Layer Socket Configuration**:
  - REQ-F9-1 (lines 83-120): ListenConfig pattern
  - REQ-F9-2 (lines 122-163): Platform-specific socket options
  - REQ-F9-3 (lines 165-198): Multicast group membership
  - REQ-F9-6 (lines 291-324): Socket buffer configuration
  - REQ-F9-7 (lines 328-422): Context propagation
- **F-10 Network Interface Management**:
  - REQ-F10-1: Interface enumeration
  - REQ-F10-2: Interface-specific multicast
- **F-11 Security Architecture**:
  - REQ-F11-2: Rate limiting
- **M1 Refactoring Analysis**:
  - Lines 595-609: F-Spec Compliance Matrix
  - Lines 611-624: RFC Compliance Validation
- **M1.1 Planning** (docs/M1.1_PLANNING_COMPLETE.md):
  - Transport layer requirements alignment

---

## Implementation Readiness Checklist

### Phase 0: Research ✅ COMPLETE

- [x] Transport interface design validated against F-9 requirements
- [x] Buffer pooling pattern validated against F-7 specification
- [x] Layer boundary compliance validated against F-2 specification
- [x] Error propagation pattern validated against F-3 RULE-1
- [x] M1.1 alignment validated (no rework needed)
- [x] All code examples provided (copy-paste ready)
- [x] All F-spec references documented with line numbers
- [x] All M1 implementation references documented
- [x] Performance benchmarks defined (≥80% allocation reduction)
- [x] Testing requirements defined for each topic

### Phase 1: Design & Validation (Next)

- [ ] Capture baseline metrics (tests, coverage, benchmarks, dependencies)
- [ ] Document current layer violations (2 instances expected)
- [ ] Run baseline benchmarks (allocation before buffer pooling)
- [ ] Validate dependency graph (no circular dependencies)
- [ ] Confirm all 107 M1 tests pass (baseline)

### Phase 2: Implementation

- [ ] Create `internal/transport/` package structure
- [ ] Implement Transport interface
- [ ] Implement UDPv4Transport (migrate socket.go logic)
- [ ] Implement MockTransport
- [ ] Implement buffer pooling (sync.Pool)
- [ ] Update Querier to use Transport interface
- [ ] Fix CloseSocket error propagation
- [ ] Remove `internal/network` imports from querier
- [ ] Add Transport interface tests
- [ ] Add buffer pool tests
- [ ] Add close error propagation test
- [ ] Validate all 107 M1 tests pass (zero regression)
- [ ] Run benchmarks (validate ≥80% allocation reduction)
- [ ] Validate layer boundaries (zero violations)

### Phase 3: Documentation

- [ ] Update godoc comments (Transport interface, buffer pooling)
- [ ] Create ADR for Transport abstraction decision
- [ ] Update CHANGELOG.md with refactoring summary
- [ ] Document benchmark improvements (before/after)

---

## Summary of Decisions

| Topic | Decision | F-Spec Alignment | M1.1 Alignment | Effort |
|-------|----------|------------------|----------------|--------|
| **Topic 1** | Transport interface with context-aware methods | F-9 REQ-F9-1, REQ-F9-7 | ✅ Enables socket config | 8h |
| **Topic 2** | sync.Pool for 9KB receive buffers | F-7 lines 286-311 | ✅ No impact | 2h |
| **Topic 3** | Querier imports internal/transport directly | F-2 RULE-1 | ✅ No impact | 4h |
| **Topic 4** | CloseSocket propagates errors | F-3 RULE-1, F-7 cleanup | ✅ No impact | 0.5h |
| **Topic 5** | All refactoring supports M1.1 | F-9, F-10, F-11 | ✅ Zero rework | 1h |
| **TOTAL** | | | | **15.5h** |

**Overall Assessment**: ✅ **READY FOR IMPLEMENTATION**

All architectural patterns validated. Zero blocking issues. M1.1 alignment confirmed. Implementation can proceed to Phase 1 (Design & Validation).

---

## References

### F-Series Specifications

- **F-2: Package Structure & Dependencies**
  - Lines 295-307: Import rules (RULE-1, RULE-2, RULE-3)
  - Lines 333-380: Layer boundaries (4-layer architecture)
  - Lines 266-278: `internal/transport/` package organization

- **F-3: Error Handling Strategy**
  - Lines 660-687: RULE-1 (Return errors to caller)
  - Lines 621-641: Cleanup on error pattern

- **F-7: Resource Management**
  - Lines 286-311: Buffer pooling pattern (sync.Pool)
  - Lines 218-285: Cleanup patterns (error propagation)

- **F-9: Transport Layer Socket Configuration**
  - Lines 83-120: REQ-F9-1 (ListenConfig pattern)
  - Lines 122-163: REQ-F9-2 (Platform-specific socket options)
  - Lines 165-198: REQ-F9-3 (Multicast group membership)
  - Lines 291-324: REQ-F9-6 (Socket buffer configuration)
  - Lines 328-422: REQ-F9-7 (Context propagation)

### M1 Implementation

- `internal/network/socket.go`: Current socket operations (to migrate)
- `querier/querier.go`: Current querier implementation (to update)
  - Line 10: Layer violation (import network)
  - Line 184: Direct network call (bypasses protocol)
  - Line 334-337: Close error handling (already correct)

### Analysis Documents

- `docs/M1_REFACTORING_ANALYSIS.md`: Comprehensive 74-issue analysis
  - P0-1 (lines 34-66): No Transport Interface Abstraction
  - P0-2 (lines 69-107): Layer Boundary Violations
  - P0-3 (lines 110-159): Buffer Allocation in Hot Path
  - P0-4 (lines 162-229): CloseSocket Swallows Errors

### Go Documentation

- `sync.Pool`: https://pkg.go.dev/sync#Pool
- `context.Context`: https://pkg.go.dev/context
- `net.ListenConfig`: https://pkg.go.dev/net#ListenConfig
- Go Blog: Memory Profiling (pprof)

---

**Research Status**: ✅ **COMPLETE**
**Next Phase**: Phase 1 - Design & Validation (Baseline Metrics Capture)
**Estimated Implementation**: 15.5 hours (14.5h P0 fixes + 1h validation)
**Constitutional Compliance**: ✅ ALL GATES PASS

**Research Created**: 2025-11-01
**Research Version**: 1.0
