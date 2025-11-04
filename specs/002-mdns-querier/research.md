# Research Document: Basic mDNS Querier (M1)

**Feature**: 002-mdns-querier
**Date**: 2025-11-01
**Status**: Complete

---

## Overview

This document captures research findings for implementing M1 Basic mDNS Querier. All research topics derived from technical requirements in spec.md and architecture patterns from F-series specifications.

---

## Research Topic 1: RFC 6762 Query Construction

**Required for**: `internal/message/builder.go` (FR-001, FR-020)

### Decision: RFC 6762 §18.1 Query Message Format

**Research Findings**:

Per RFC 6762 §18.1, mDNS query messages MUST have specific header field values:

```
DNS Header for Queries:
- QR (Query/Response):     0 (this is a query, not a response)
- OPCODE:                  0 (standard query)
- AA (Authoritative Answer): 0 (not authoritative in query)
- TC (Truncation):         0 (query not truncated)
- RD (Recursion Desired):  0 (mDNS does not use recursion)
- Z:                       0 (reserved, must be zero)
- RCODE:                   0 (no error in query)

Question Section:
- QNAME: DNS name (e.g., "printer.local")
- QTYPE: Record type (A=1, PTR=12, SRV=33, TXT=16)
- QCLASS: IN (1) | QU bit in high bit (0x8001 for unicast response, 0x0001 for multicast)
```

**QU Bit Decision** (RFC 6762 §5.4):
- QU bit = 0 (0x0001): Standard multicast response expected (M1 approach)
- QU bit = 1 (0x8001): Unicast response preferred (M3: Advanced Queries)

**Implementation Pattern**:
```go
type DNSHeader struct {
    ID      uint16  // Transaction ID (random)
    Flags   uint16  // QR=0, OPCODE=0, AA=0, TC=0, RD=0, Z=0, RCODE=0
    QDCount uint16  // Question count (1 for single query)
    ANCount uint16  // Answer count (0 for queries)
    NSCount uint16  // Authority count (0 for queries)
    ARCount uint16  // Additional count (0 for M1, used in M2 for Known-Answer)
}

// Build query message
func BuildQuery(name string, recordType RecordType) ([]byte, error) {
    header := DNSHeader{
        ID:      rand.Uint16(),  // Random transaction ID
        Flags:   0x0000,          // All flags zero (per §18.1)
        QDCount: 1,               // One question
        ANCount: 0, NSCount: 0, ARCount: 0,
    }
    // Encode header + question section to wire format
}
```

**Rationale**: RFC 6762 §18.1 mandates these exact header values for mDNS queries. Any deviation breaks protocol compliance (Constitution Principle I).

**Alternatives Considered**:
- ❌ Using third-party DNS library (e.g., `miekg/dns`): Adds external dependency, may not enforce RFC 6762 specifics
- ✅ Custom implementation with `encoding/binary`: No dependencies, full control over RFC compliance

---

## Research Topic 2: DNS Wire Format Parsing

**Required for**: `internal/message/parser.go` (FR-009)

### Decision: RFC 1035 Wire Format with `encoding/binary`

**Research Findings**:

DNS messages use big-endian (network byte order) binary encoding (RFC 1035 §4.1):

```
Header (12 bytes):
  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      ID                       |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|QR|  OPCODE |AA|TC|RD|RA| Z|AD|CD|   RCODE    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    QDCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ANCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    NSCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ARCOUNT                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+

Question Section (variable length):
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     QNAME                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     QTYPE                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     QCLASS                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+

Answer/Authority/Additional Sections (variable length):
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      NAME                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      TYPE                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     CLASS                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                      TTL                      |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                   RDLENGTH                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--|
|                     RDATA                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

**Implementation Pattern**:
```go
import "encoding/binary"

func ParseMessage(data []byte) (*DNSMessage, error) {
    if len(data) < 12 {
        return nil, WireFormatError("message too short")
    }

    header := DNSHeader{
        ID:      binary.BigEndian.Uint16(data[0:2]),
        Flags:   binary.BigEndian.Uint16(data[2:4]),
        QDCount: binary.BigEndian.Uint16(data[4:6]),
        ANCount: binary.BigEndian.Uint16(data[6:8]),
        NSCount: binary.BigEndian.Uint16(data[8:10]),
        ARCount: binary.BigEndian.Uint16(data[10:12]),
    }

    // Parse Question, Answer, Authority, Additional sections
    offset := 12
    // ... (see name.go for name parsing with compression support)
}
```

**Rationale**: `encoding/binary` is standard library, zero dependencies, handles big-endian encoding correctly.

**Alternatives Considered**:
- ❌ Manual byte shifting: Error-prone, harder to maintain
- ✅ `encoding/binary.BigEndian`: Idiomatic Go, proven reliability

---

## Research Topic 3: DNS Name Compression

**Required for**: `internal/message/name.go` (FR-012)

### Decision: RFC 1035 §4.1.4 Compression Pointers

**Research Findings**:

DNS names can be compressed using pointers to avoid repeating domain labels (RFC 1035 §4.1.4):

```
Label Format:
+--+--+--+--+--+--+--+--+
| 0|     Label Length    |  (0-63 bytes, high 2 bits = 00)
+--+--+--+--+--+--+--+--+
|   Label bytes ...      |
+--+--+--+--+--+--+--+--+

Compression Pointer:
+--+--+--+--+--+--+--+--+
| 1  1|    Offset         |  (14-bit offset, high 2 bits = 11)
+--+--+--+--+--+--+--+--+

Example: "myprinter.local" and "yourprinter.local" sharing ".local"
myprinter (9 bytes) + 0x00 = "myprinter\0local\0"
yourprinter (11 bytes) + pointer to offset of "local" = "yourprinter\xC0\x0A"
```

**Algorithm**:
1. Read first byte
2. If high 2 bits == 00: Normal label (length + bytes)
3. If high 2 bits == 11: Compression pointer (14-bit offset into message)
4. Follow pointer to resolve label, detect cycles (max 255 jumps)

**Implementation Pattern**:
```go
func ParseName(data []byte, offset int) (string, int, error) {
    var labels []string
    originalOffset := offset
    jumpCount := 0
    const maxJumps = 255  // Prevent infinite loops

    for {
        if offset >= len(data) {
            return "", 0, WireFormatError("name extends beyond message")
        }

        length := data[offset]

        // Check for compression pointer (high 2 bits = 11)
        if length&0xC0 == 0xC0 {
            if offset+1 >= len(data) {
                return "", 0, WireFormatError("incomplete compression pointer")
            }
            pointer := binary.BigEndian.Uint16(data[offset:offset+2]) & 0x3FFF
            offset = int(pointer)
            jumpCount++
            if jumpCount > maxJumps {
                return "", 0, WireFormatError("too many compression jumps")
            }
            continue
        }

        // Normal label (length 0-63)
        if length == 0 {
            break  // End of name
        }
        if length > 63 {
            return "", 0, WireFormatError("label length exceeds 63")
        }

        offset++
        if offset+int(length) > len(data) {
            return "", 0, WireFormatError("label extends beyond message")
        }

        labels = append(labels, string(data[offset:offset+int(length)]))
        offset += int(length)
    }

    return strings.Join(labels, "."), originalOffset, nil
}
```

**Rationale**: RFC 1035 §4.1.4 compression is mandatory for DNS compliance. Responders will use compression; parser must handle it.

**Alternatives Considered**:
- ❌ Ignoring compression: Protocol violation, fails with real responders
- ✅ Full compression support: RFC-compliant, handles all responses

---

## Research Topic 4: UDP Multicast in Go

**Required for**: `internal/network/socket.go` (FR-005, FR-006)

### Decision: `net.ListenMulticastUDP` with `net.UDPAddr`

**Research Findings**:

Go's `net` package provides multicast UDP support via `ListenMulticastUDP`:

```go
import "net"

const (
    mDNSAddress = "224.0.0.251"  // IPv4 multicast group (RFC 6762)
    mDNSPort    = 5353           // mDNS port (RFC 6762)
)

// Bind to multicast group on default interface
func createSocket() (*net.UDPConn, error) {
    addr := &net.UDPAddr{
        IP:   net.ParseIP(mDNSAddress),
        Port: mDNSPort,
    }

    // Listen on multicast group (receives all mDNS traffic)
    conn, err := net.ListenMulticastUDP("udp4", nil, addr)
    if err != nil {
        return nil, NetworkError("failed to bind multicast socket", err)
    }

    // Set socket options for multicast
    if err := conn.SetReadBuffer(65536); err != nil {  // 64KB buffer
        conn.Close()
        return nil, NetworkError("failed to set read buffer", err)
    }

    return conn, nil
}

// Send query to multicast group
func sendQuery(conn *net.UDPConn, message []byte) error {
    addr := &net.UDPAddr{
        IP:   net.ParseIP(mDNSAddress),
        Port: mDNSPort,
    }

    _, err := conn.WriteToUDP(message, addr)
    if err != nil {
        return NetworkError("failed to send query", err)
    }
    return nil
}

// Receive responses (with timeout)
func receiveResponses(conn *net.UDPConn, timeout time.Duration) ([]byte, error) {
    buffer := make([]byte, 9000)  // Max DNS message size (RFC 6762)

    conn.SetReadDeadline(time.Now().Add(timeout))
    n, _, err := conn.ReadFromUDP(buffer)
    if err != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            return nil, nil  // Timeout is expected
        }
        return nil, NetworkError("failed to receive response", err)
    }

    return buffer[:n], nil
}
```

**Platform-Specific Requirements**:
- **Linux**: Requires root privileges or `CAP_NET_RAW` capability to bind to port 5353
- **Multicast Routing**: Requires default interface to have multicast route (`ip route` shows multicast entry)

**Rationale**: `net.ListenMulticastUDP` is standard library, handles multicast group join/leave, supports IPv4 (M1) and IPv6 (M4 future).

**Alternatives Considered**:
- ❌ Raw sockets: Requires more privileges, harder to use
- ✅ `ListenMulticastUDP`: Idiomatic Go, handles multicast correctly

---

## Research Topic 5: Context-based Cancellation

**Required for**: F-4 concurrency patterns (FR-008, FR-017, FR-018)

### Decision: `context.Context` with Timeout and Cancellation

**Research Findings**:

Per F-4 Concurrency Model and Go best practices:

```go
import "context"

// Query with timeout and cancellation support
func (q *Querier) Query(ctx context.Context, name string, recordType RecordType) (*Response, error) {
    // Create timeout context (default 1 second per FR-007)
    queryCtx, cancel := context.WithTimeout(ctx, q.timeout)
    defer cancel()

    // Send query
    if err := q.sendQuery(name, recordType); err != nil {
        return nil, err
    }

    // Collect responses until timeout or cancellation
    responses := []ResourceRecord{}
    for {
        select {
        case <-queryCtx.Done():
            // Timeout or cancellation
            if ctx.Err() == context.Canceled {
                return nil, ctx.Err()  // User cancelled
            }
            return &Response{Records: responses}, nil  // Timeout (expected)

        case resp := <-q.responseChan:
            // Process response
            records, err := parseResponse(resp)
            if err != nil {
                // Log and continue (FR-016: don't fail on malformed packets)
                log.Debug("malformed packet: %v", err)
                continue
            }
            responses = append(responses, records...)
        }
    }
}

// Graceful shutdown (FR-018)
func (q *Querier) Close() error {
    q.cancelFunc()  // Cancel all active queries

    // Wait for goroutines to finish (with timeout)
    done := make(chan struct{})
    go func() {
        q.wg.Wait()  // sync.WaitGroup for goroutine tracking
        close(done)
    }()

    select {
    case <-done:
        return nil  // Clean shutdown
    case <-time.After(5 * time.Second):
        return errors.New("shutdown timeout: some goroutines did not terminate")
    }
}
```

**Patterns**:
- **Timeout**: `context.WithTimeout(parent, duration)` for query deadline (FR-007)
- **Cancellation**: User can cancel parent context to abort query (FR-008)
- **Cleanup**: `defer cancel()` ensures context resources are released (F-7)
- **Graceful Shutdown**: `sync.WaitGroup` + context cancellation + timeout (FR-018)

**Rationale**: `context.Context` is Go standard for cancellation and timeouts. F-4 mandates context propagation for all I/O operations.

**Alternatives Considered**:
- ❌ Manual timeout with `time.After`: Doesn't support user cancellation
- ✅ `context.Context`: Idiomatic Go, supports timeout + cancellation + propagation

---

## Research Topic 6: Go Fuzz Testing

**Required for**: `tests/fuzz/parser_fuzz_test.go` (NFR-003)

### Decision: Go 1.18+ Native Fuzz Testing

**Research Findings**:

Go 1.18+ supports native fuzz testing for binary parsers:

```go
import "testing"

func FuzzMessageParser(f *testing.F) {
    // Seed corpus with known good messages
    f.Add([]byte{
        // Valid DNS header (12 bytes)
        0x00, 0x01,  // ID
        0x00, 0x00,  // Flags
        0x00, 0x01,  // QDCount
        0x00, 0x00,  // ANCount
        0x00, 0x00,  // NSCount
        0x00, 0x00,  // ARCount
        // Question: "test.local" A IN
        0x04, 't', 'e', 's', 't',
        0x05, 'l', 'o', 'c', 'a', 'l',
        0x00,
        0x00, 0x01,  // Type A
        0x00, 0x01,  // Class IN
    })

    // Fuzz with random data
    f.Fuzz(func(t *testing.T, data []byte) {
        // Parse should never panic (NFR-003: zero crashes)
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("parser panicked: %v", r)
            }
        }()

        // Parse message (may return error for invalid data)
        _, err := ParseMessage(data)

        // Test passes if no panic occurred
        // Error is expected for malformed data
        _ = err
    })
}

// Run fuzz test
// go test -fuzz=FuzzMessageParser -fuzztime=10s
// go test -fuzz=FuzzMessageParser -fuzztime=10000x  // 10,000 iterations (per NFR-003)
```

**Fuzz Coverage**:
- Malformed headers (wrong size, invalid flags)
- Truncated messages (incomplete sections)
- Invalid name compression (circular pointers, out-of-bounds)
- Oversized labels (>63 bytes)
- Invalid record types

**Rationale**: Native Go fuzzing finds edge cases that manual tests miss. NFR-003 requires 10,000 random packets without crashes.

**Alternatives Considered**:
- ❌ Manual test cases: Can't cover all edge cases
- ❌ go-fuzz (external tool): Go 1.18+ has native support
- ✅ Native Go fuzz testing: Built-in, fast, no dependencies

---

## Summary of Decisions

| Research Topic | Decision | Rationale |
|----------------|----------|-----------|
| RFC 6762 Query Construction | Use RFC 6762 §18.1 header format with `encoding/binary` | RFC compliance (Constitution Principle I), no dependencies |
| DNS Wire Format Parsing | Use RFC 1035 format with `encoding/binary.BigEndian` | Standard library, big-endian handling |
| DNS Name Compression | Implement RFC 1035 §4.1.4 compression pointer algorithm | RFC compliance, handles real responder messages |
| UDP Multicast | Use `net.ListenMulticastUDP` with 224.0.0.251:5353 | Standard library, multicast support, idiomatic Go |
| Context Cancellation | Use `context.Context` with timeout and cancellation | F-4 compliance, Go best practice |
| Fuzz Testing | Use Go 1.18+ native fuzz testing | Built-in, no dependencies, finds edge cases |

**No Unresolved Unknowns**: All technical decisions are based on RFCs, F-series specifications, and Go standard library patterns. No external dependencies required for M1.

---

**Research Status**: ✅ **COMPLETE**
**Next Phase**: Phase 1 (Data Model, API Contracts, Quickstart)
