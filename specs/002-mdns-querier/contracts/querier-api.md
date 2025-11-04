# API Contract: beacon/querier Package

**Feature**: 002-mdns-querier (M1 Basic mDNS Querier)
**Package**: `beacon/querier`
**Date**: 2025-11-01
**Status**: Complete

---

## Overview

This document defines the public API contract for the `beacon/querier` package. This is the primary user-facing API for M1 Basic mDNS Querier.

**Package Purpose**: Enable one-shot mDNS queries for discovering hosts and services on the local network.

**Import Path**: `github.com/joshuafuller/beacon/querier` (example, adjust based on actual module path)

---

## Package-Level Functions

### `func New(options ...Option) (*Querier, error)`

**Purpose**: Create a new Querier instance with optional configuration.

**Parameters**:
- `options ...Option`: Variadic functional options (per F-5 Configuration & Defaults)

**Returns**:
- `*Querier`: Configured querier instance
- `error`: NetworkError if socket creation fails (e.g., permission denied, no network interfaces)

**Behavior**:
1. Create UDP multicast socket on 224.0.0.251:5353 (FR-004)
2. Apply functional options (e.g., `WithTimeout`)
3. Set default timeout to 1 second if not specified (FR-007)
4. Start goroutine for receiving responses (F-4 concurrency pattern)
5. Return querier ready for Query() calls

**Error Conditions**:
- Returns `NetworkError` if:
  - Cannot bind to port 5353 (requires root/CAP_NET_RAW)
  - No network interfaces available
  - Multicast not supported on default interface

**Example**:
```go
// Default querier (1 second timeout)
q, err := querier.New()
if err != nil {
    log.Fatal(err)
}
defer q.Close()

// Custom timeout querier
q, err := querier.New(querier.WithTimeout(500 * time.Millisecond))
```

**Concurrency**: Safe to call from multiple goroutines. Each call creates independent querier.

**Resource Management**: Caller MUST call `Close()` when done (per F-7).

---

## Functional Options (per F-5)

### `func WithTimeout(timeout time.Duration) Option`

**Purpose**: Configure query timeout duration.

**Parameters**:
- `timeout time.Duration`: Query timeout (valid range: 100ms to 10 seconds per FR-007)

**Returns**:
- `Option`: Functional option for `New()`

**Behavior**:
- Sets timeout for all queries executed by this querier
- If timeout < 100ms, uses 100ms (minimum)
- If timeout > 10s, uses 10s (maximum)

**Example**:
```go
q, err := querier.New(querier.WithTimeout(2 * time.Second))
```

**Validation**: Timeout clamped to 100ms-10s range (no error returned for out-of-range values).

---

## Querier Type

### `type Querier struct`

**Purpose**: Core mDNS querier instance.

**Fields**: All fields are unexported (internal implementation per F-2 encapsulation).

**Methods**: See below.

---

## Querier Methods

### `func (q *Querier) Query(ctx context.Context, name string, recordType RecordType) (*Response, error)`

**Purpose**: Execute one-shot mDNS query for specified name and record type.

**Parameters**:
- `ctx context.Context`: Context for cancellation and timeout (per F-4)
- `name string`: DNS name to query (e.g., "printer.local", "_http._tcp.local")
- `recordType RecordType`: DNS record type (A, PTR, SRV, TXT)

**Returns**:
- `*Response`: Parsed response with resource records (may be empty if no responders)
- `error`: ValidationError, NetworkError, or WireFormatError

**Behavior**:
1. Validate inputs (FR-003, FR-014):
   - `name` not empty, ≤255 bytes, valid characters
   - `recordType` one of: A, PTR, SRV, TXT
2. Construct mDNS query message per RFC 6762 §18.1 (FR-001, FR-020)
3. Send query to 224.0.0.251:5353 multicast group (FR-005)
4. Listen for responses until timeout or context cancellation (FR-006, FR-008)
5. Parse responses and extract resource records (FR-009, FR-010)
6. Validate responses per RFC 6762 (FR-021, FR-022):
   - QR=1 (response bit set)
   - RCODE=0 (ignore error responses)
7. Aggregate all valid resource records into Response
8. Return Response (empty if no responders, non-empty if responders found)

**Error Conditions**:
- Returns `ValidationError` if:
  - `name` is empty → "name cannot be empty"
  - `name` exceeds 255 bytes → "name exceeds maximum length (255 bytes)"
  - `name` has invalid characters → "invalid characters in hostname"
  - `recordType` is unsupported (e.g., AAAA in M1) → "unsupported record type: AAAA"
- Returns `NetworkError` if:
  - Send fails → "failed to send query: [underlying error]"
  - Network interface down → "network interface down"
- Returns `WireFormatError` if:
  - Response parsing fails → "malformed packet: [details]" (logged, query continues per FR-016)
- Returns `context.Canceled` if:
  - Context cancelled by user → propagates ctx.Err()

**Timeout Behavior**:
- If no responses received before timeout: Returns empty Response (not an error per FR-006)
- Timeout defaults to querier's configured timeout (from `New()` or `WithTimeout`)
- User can override with context timeout:
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
  defer cancel()
  resp, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
  ```

**Example**:
```go
// Query for A record (IPv4 address)
ctx := context.Background()
resp, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
if err != nil {
    log.Fatal(err)
}

// Check if any devices responded
if len(resp.Records) == 0 {
    fmt.Println("No responders found")
} else {
    for _, record := range resp.Records {
        if ip, err := record.AsA(); err == nil {
            fmt.Printf("Found device at %s\n", ip)
        }
    }
}

// Query for service instances (PTR records)
resp, err = q.Query(ctx, "_http._tcp.local", querier.RecordTypePTR)
```

**Concurrency**: Safe to call from multiple goroutines. Each Query() is independent.

**Performance**: Processing overhead <100ms per NFR-001 (from response receipt to parsed records available).

---

### `func (q *Querier) Close() error`

**Purpose**: Gracefully shutdown querier and release all resources.

**Parameters**: None

**Returns**:
- `error`: Error if shutdown timeout exceeded (goroutines didn't terminate within 5 seconds)

**Behavior**:
1. Cancel context for all active queries (stops receiving goroutine)
2. Close UDP socket (FR-017)
3. Wait for all goroutines to terminate (uses `sync.WaitGroup`)
4. Return nil if clean shutdown within 5 seconds
5. Return error if goroutines don't terminate (should not happen in normal operation)

**Error Conditions**:
- Returns error if shutdown timeout exceeded (indicates resource leak, should never happen if F-7 followed)

**Example**:
```go
q, err := querier.New()
if err != nil {
    log.Fatal(err)
}
defer q.Close()  // Always close querier when done

// Use querier...
```

**Concurrency**: Safe to call from any goroutine. Multiple Close() calls are safe (idempotent after first call).

**Resource Management**: Frees all resources (socket, goroutines) per F-7 requirements. No leaks per FR-017.

---

## Response Type

### `type Response struct`

**Purpose**: Contains DNS resource records returned from mDNS query.

**Fields**:
- `Records []ResourceRecord`: List of resource records from Answer section

**Methods**:
- *(Future)* `GetRecordsByType(recordType RecordType) []ResourceRecord`
- *(Future)* `GetRecordsByName(name string) []ResourceRecord`

**Example**:
```go
resp, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d records\n", len(resp.Records))
for _, record := range resp.Records {
    fmt.Printf("  %s: %v\n", record.Name, record.Data)
}
```

---

## ResourceRecord Type

### `type ResourceRecord struct`

**Purpose**: Represents a single DNS resource record.

**Fields**:
- `Name string`: DNS name (e.g., "printer.local")
- `Type RecordType`: Record type (A, PTR, SRV, TXT)
- `Class uint16`: DNS class (always IN=1 for mDNS)
- `TTL uint32`: Time-to-live in seconds (M1: informational only, caching in M2)
- `Data interface{}`: Type-specific data (see Type-Specific Methods)

**Type-Specific Methods**:

#### `func (r *ResourceRecord) AsA() (net.IP, error)`
- **Purpose**: Extract IPv4 address from A record
- **Returns**: `net.IP` (4 bytes) or error if wrong type
- **Example**: `ip, err := record.AsA()`

#### `func (r *ResourceRecord) AsPTR() (string, error)`
- **Purpose**: Extract pointer name from PTR record
- **Returns**: Service instance name or error if wrong type
- **Example**: `instance, err := record.AsPTR()`

#### `func (r *ResourceRecord) AsSRV() (*SRVData, error)`
- **Purpose**: Extract service location from SRV record
- **Returns**: `*SRVData` (priority, weight, port, target) or error if wrong type
- **Example**: `srv, err := record.AsSRV()`

#### `func (r *ResourceRecord) AsTXT() ([]string, error)`
- **Purpose**: Extract metadata from TXT record
- **Returns**: List of key=value strings or error if wrong type
- **Example**: `txt, err := record.AsTXT()`

**Example**:
```go
for _, record := range resp.Records {
    switch record.Type {
    case querier.RecordTypeA:
        if ip, err := record.AsA(); err == nil {
            fmt.Printf("A: %s -> %s\n", record.Name, ip)
        }
    case querier.RecordTypePTR:
        if ptr, err := record.AsPTR(); err == nil {
            fmt.Printf("PTR: %s -> %s\n", record.Name, ptr)
        }
    case querier.RecordTypeSRV:
        if srv, err := record.AsSRV(); err == nil {
            fmt.Printf("SRV: %s -> %s:%d\n", record.Name, srv.Target, srv.Port)
        }
    case querier.RecordTypeTXT:
        if txt, err := record.AsTXT(); err == nil {
            fmt.Printf("TXT: %s -> %v\n", record.Name, txt)
        }
    }
}
```

---

## RecordType Enum

### `type RecordType int`

**Purpose**: Enumerates supported DNS record types.

**Constants**:
```go
const (
    RecordTypeA   RecordType = 1   // IPv4 address
    RecordTypePTR RecordType = 12  // Pointer (service instance)
    RecordTypeSRV RecordType = 33  // Service location (host, port)
    RecordTypeTXT RecordType = 16  // Metadata (key=value pairs)
)
```

**Methods**:
- `func (rt RecordType) String() string`: Human-readable name ("A", "PTR", "SRV", "TXT")

**Example**:
```go
recordType := querier.RecordTypeA
fmt.Println(recordType.String())  // "A"
```

---

## SRVData Type

### `type SRVData struct`

**Purpose**: Service record data (RFC 2782).

**Fields**:
- `Priority uint16`: Service priority (lower = higher priority)
- `Weight uint16`: Load balancing weight
- `Port uint16`: Service port number
- `Target string`: Target hostname (e.g., "myserver.local")

**Example**:
```go
srv, err := record.AsSRV()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Connect to %s:%d (priority %d)\n", srv.Target, srv.Port, srv.Priority)
```

---

## Error Types (per F-3)

All errors returned by this package implement the `error` interface and can be type-asserted to specific error types:

### `NetworkError`
- **Returned by**: `New()`, `Query()`
- **Cause**: Socket creation, binding, or I/O failures
- **Example Messages**:
  - "failed to bind to 224.0.0.251:5353: permission denied (requires root or CAP_NET_RAW)"
  - "failed to send query: network unreachable"

### `ValidationError`
- **Returned by**: `Query()`
- **Cause**: Invalid inputs (name, recordType)
- **Example Messages**:
  - "name cannot be empty"
  - "name exceeds maximum length (255 bytes)"
  - "unsupported record type: AAAA"

### `WireFormatError`
- **Returned by**: *(Not typically returned to user, logged internally per FR-016)*
- **Cause**: Malformed DNS packets (logged and discarded)
- **Example Messages** (in logs):
  - "malformed packet: message too short"
  - "malformed packet: invalid compression pointer"

**Error Handling Example**:
```go
resp, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
if err != nil {
    switch err.(type) {
    case *querier.NetworkError:
        fmt.Println("Network issue:", err)
    case *querier.ValidationError:
        fmt.Println("Invalid input:", err)
    default:
        fmt.Println("Unknown error:", err)
    }
    return
}
```

---

## Usage Examples

### Basic Host Discovery (User Story P1)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier
    q, err := querier.New()
    if err != nil {
        log.Fatal(err)
    }
    defer q.Close()

    // Query for printer
    ctx := context.Background()
    resp, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
    if err != nil {
        log.Fatal(err)
    }

    // Print results
    if len(resp.Records) == 0 {
        fmt.Println("Printer not found")
    } else {
        for _, record := range resp.Records {
            if ip, err := record.AsA(); err == nil {
                fmt.Printf("Printer found at %s\n", ip)
            }
        }
    }
}
```

### Service Discovery (User Story P2)

```go
// Discover all HTTP services
resp, err := q.Query(ctx, "_http._tcp.local", querier.RecordTypePTR)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d HTTP services:\n", len(resp.Records))
for _, record := range resp.Records {
    if instanceName, err := record.AsPTR(); err == nil {
        fmt.Printf("  - %s\n", instanceName)

        // Query SRV record for connection details
        srvResp, err := q.Query(ctx, instanceName, querier.RecordTypeSRV)
        if err == nil && len(srvResp.Records) > 0 {
            if srv, err := srvResp.Records[0].AsSRV(); err == nil {
                fmt.Printf("    Connect to: %s:%d\n", srv.Target, srv.Port)
            }
        }

        // Query TXT record for metadata
        txtResp, err := q.Query(ctx, instanceName, querier.RecordTypeTXT)
        if err == nil && len(txtResp.Records) > 0 {
            if txt, err := txtResp.Records[0].AsTXT(); err == nil {
                fmt.Printf("    Metadata: %v\n", txt)
            }
        }
    }
}
```

### Custom Timeout

```go
// Create querier with 500ms timeout
q, err := querier.New(querier.WithTimeout(500 * time.Millisecond))
if err != nil {
    log.Fatal(err)
}
defer q.Close()

// Or override per-query with context
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
resp, err := q.Query(ctx, "slow-device.local", querier.RecordTypeA)
```

### Error Handling (User Story P3)

```go
resp, err := q.Query(ctx, "device.local", querier.RecordTypeA)
if err != nil {
    switch e := err.(type) {
    case *querier.NetworkError:
        fmt.Printf("Network error: %s\n", e)
        // Check network interface, firewall, permissions
    case *querier.ValidationError:
        fmt.Printf("Invalid input: %s\n", e)
        // Fix input (name, recordType)
    default:
        fmt.Printf("Unknown error: %s\n", err)
    }
    return
}

// Check if any devices responded
if len(resp.Records) == 0 {
    fmt.Println("No devices responded (timeout expired)")
} else {
    fmt.Printf("Found %d devices\n", len(resp.Records))
}
```

---

## Contract Testing

Per F-8 Testing Strategy, contract tests verify API behavior:

**Test Coverage**:
- ✅ `New()` creates querier with default timeout
- ✅ `WithTimeout()` sets custom timeout
- ✅ `Query()` validates inputs (name, recordType)
- ✅ `Query()` returns ValidationError for invalid inputs
- ✅ `Query()` returns NetworkError for network failures
- ✅ `Query()` returns empty Response on timeout (not error)
- ✅ `Query()` aggregates multiple responses from different responders
- ✅ `Query()` respects context cancellation
- ✅ `Close()` releases all resources (no leaks)
- ✅ `ResourceRecord.AsA()`, `AsPTR()`, `AsSRV()`, `AsTXT()` type-assert correctly
- ✅ RecordType constants match DNS standard values (A=1, PTR=12, SRV=33, TXT=16)

---

**API Contract Status**: ✅ **COMPLETE**
**Next Phase**: Quickstart documentation (`quickstart.md`)
