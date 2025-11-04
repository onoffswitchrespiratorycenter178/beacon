# Data Model: Basic mDNS Querier (M1)

**Feature**: 002-mdns-querier
**Date**: 2025-11-01
**Status**: Complete

---

## Overview

This document defines the data entities for M1 Basic mDNS Querier. Entities are organized by layer per F-2 Package Structure:
- **Public API Layer** (`beacon/querier/`): User-facing entities
- **Protocol Layer** (`internal/message/`, `internal/protocol/`): DNS message entities
- **Transport Layer** (`internal/network/`): Network I/O entities

---

## Public API Entities (`beacon/querier/`)

### Querier

**Purpose**: Core component for executing one-shot mDNS queries.

**Attributes**:
- `socket`: UDP multicast socket (`*net.UDPConn`)
- `timeout`: Query timeout duration (`time.Duration`, default: 1 second per FR-007)
- `responseChan`: Channel for receiving parsed responses (`chan *DNSMessage`)
- `cancelFunc`: Context cancellation function (`context.CancelFunc`)
- `wg`: Wait group for goroutine lifecycle tracking (`sync.WaitGroup`)

**Operations**:
- `New(options ...Option) (*Querier, error)`: Constructor with functional options (per F-5)
- `Query(ctx context.Context, name string, recordType RecordType) (*Response, error)`: Execute one-shot query
- `Close() error`: Graceful shutdown (FR-018)

**State Transitions**:
```
Created → Querying → Query Complete → Created (can query again)
        ↘ Closed (terminal state)
```

**Validation Rules** (from FR requirements):
- Querier MUST be created before Query (enforced by constructor pattern)
- Query MUST validate name (FR-003: ≤255 bytes, valid characters)
- Query MUST validate recordType (FR-002: A, PTR, SRV, TXT only)
- Close MUST be called before program exit (resource cleanup per F-7)

**Relationships**:
- Querier → Response (1:N, one querier can execute many queries)
- Querier → ResourceRecord (1:N, via Response)

---

### Query (Internal, not exposed in public API)

**Purpose**: Represents a single in-flight query operation.

**Attributes**:
- `name`: DNS name being queried (`string`, e.g., "printer.local")
- `recordType`: DNS record type (`RecordType`, e.g., A, PTR, SRV, TXT)
- `timeout`: Query-specific timeout (`time.Duration`)
- `ctx`: Query context for cancellation (`context.Context`)

**Operations**:
- `execute() (*Response, error)`: Execute query lifecycle (send → collect → parse)

**Validation Rules**:
- `name` MUST be valid DNS name per FR-003 (≤255 bytes, labels ≤63 bytes, valid characters)
- `recordType` MUST be one of: A (1), PTR (12), SRV (33), TXT (16) per FR-002
- `timeout` MUST be 100ms ≤ timeout ≤ 10s per FR-007

---

### Response

**Purpose**: Contains parsed DNS resource records from mDNS responses.

**Attributes**:
- `Records`: List of resource records from Answer section (`[]ResourceRecord`)

**Operations**:
- `GetRecordsByType(recordType RecordType) []ResourceRecord`: Filter records by type
- `GetRecordsByName(name string) []ResourceRecord`: Filter records by name

**Validation Rules**:
- Records MUST be from Answer section only (M1: Authority/Additional sections ignored per FR-010)
- Records MUST have passed RFC validation (FR-011, FR-021, FR-022)

**Relationships**:
- Response → ResourceRecord (1:N, one response contains multiple records)

---

### ResourceRecord

**Purpose**: Represents a single DNS resource record (answer to a query).

**Attributes**:
- `Name`: DNS name (`string`, e.g., "printer.local")
- `Type`: Record type (`RecordType`, e.g., A, PTR, SRV, TXT)
- `Class`: DNS class (`uint16`, always IN=1 for mDNS per RFC 6762)
- `TTL`: Time-to-live in seconds (`uint32`, used in M2 for caching)
- `Data`: Type-specific data (`interface{}`, varies by record type)

**Type-Specific Data**:
- **A Record**: `net.IP` (IPv4 address, 4 bytes)
- **PTR Record**: `string` (pointer to service instance name)
- **SRV Record**: `*SRVData` struct (priority, weight, port, target)
- **TXT Record**: `[]string` (key=value pairs)

**Operations**:
- `AsA() (net.IP, error)`: Type-assert Data as A record
- `AsPTR() (string, error)`: Type-assert Data as PTR record
- `AsSRV() (*SRVData, error)`: Type-assert Data as SRV record
- `AsTXT() ([]string, error)`: Type-assert Data as TXT record

**Validation Rules**:
- `Name` MUST be valid DNS name (≤255 bytes)
- `Type` MUST be one of supported types (A, PTR, SRV, TXT per FR-002)
- `Class` MUST be IN (1) for mDNS
- `TTL` can be 0 for M1 (caching deferred to M2)
- `Data` MUST match `Type` (e.g., Type=A implies Data is `net.IP`)

---

### RecordType (Enum)

**Purpose**: Enumerates supported DNS record types.

**Values**:
- `RecordTypeA = 1` (IPv4 address)
- `RecordTypePTR = 12` (pointer to service instance)
- `RecordTypeSRV = 33` (service location: host, port)
- `RecordTypeTXT = 16` (metadata key-value pairs)

**Validation Rules**:
- Only these 4 types are supported in M1 (FR-002)
- AAAA (28) for IPv6 deferred to M4 (returns ValidationError in M1 per FR-014)

---

### SRVData

**Purpose**: Service record data (RFC 2782).

**Attributes**:
- `Priority`: Service priority (`uint16`, lower value = higher priority)
- `Weight`: Load balancing weight (`uint16`)
- `Port`: Service port number (`uint16`)
- `Target`: Target hostname (`string`, e.g., "myserver.local")

**Validation Rules**:
- `Target` MUST be valid DNS name
- `Port` MUST be 1-65535
- `Priority` and `Weight` can be 0

---

## Protocol Layer Entities (`internal/message/`, `internal/protocol/`)

### DNSMessage

**Purpose**: Wire format representation of DNS message (RFC 1035).

**Attributes**:
- `Header`: DNS header (`DNSHeader`)
- `Questions`: Question section (`[]Question`)
- `Answers`: Answer section (`[]Answer`)
- `Authorities`: Authority section (`[]Answer`, ignored in M1)
- `Additionals`: Additional section (`[]Answer`, ignored in M1)

**Operations**:
- `Encode() ([]byte, error)`: Encode to wire format (for sending queries)
- `Decode([]byte) (*DNSMessage, error)`: Decode from wire format (for parsing responses)

**Validation Rules**:
- Message MUST be ≥12 bytes (header size)
- Message MUST be ≤9000 bytes (mDNS max per RFC 6762)
- Header MUST have valid flags per RFC 6762 §18

**Relationships**:
- DNSMessage → DNSHeader (1:1)
- DNSMessage → Question (1:N)
- DNSMessage → Answer (1:N)

---

### DNSHeader

**Purpose**: DNS message header (RFC 1035 §4.1.1).

**Attributes**:
- `ID`: Transaction ID (`uint16`, random for queries)
- `Flags`: Header flags (`uint16`, bit-packed: QR, OPCODE, AA, TC, RD, RA, Z, RCODE)
- `QDCount`: Question count (`uint16`)
- `ANCount`: Answer count (`uint16`)
- `NSCount`: Authority count (`uint16`)
- `ARCount`: Additional count (`uint16`)

**Operations**:
- `IsQuery() bool`: Check if QR bit = 0 (query)
- `IsResponse() bool`: Check if QR bit = 1 (response)
- `GetRCODE() uint8`: Extract response code (bits 12-15)

**Validation Rules** (per FR-020, FR-021):
- **Query**: QR=0, OPCODE=0, AA=0, TC=0, RD=0, Z=0, RCODE=0
- **Response**: QR=1 (per FR-021), RCODE=0 (per FR-022, ignore non-zero RCODE)

---

### Question

**Purpose**: DNS question section (RFC 1035 §4.1.2).

**Attributes**:
- `QNAME`: Queried name (`string`, encoded as labels)
- `QTYPE`: Queried type (`uint16`, e.g., A=1, PTR=12)
- `QCLASS`: Queried class (`uint16`, IN=1 or IN+QU=0x8001)

**Validation Rules**:
- `QNAME` MUST be ≤255 bytes total, labels ≤63 bytes
- `QTYPE` MUST be one of supported types (A, PTR, SRV, TXT)
- `QCLASS` MUST be IN (0x0001) for M1 (QU bit = 0 per FR-001)

---

### Answer

**Purpose**: DNS answer/authority/additional section (RFC 1035 §4.1.3).

**Attributes**:
- `NAME`: Record name (`string`)
- `TYPE`: Record type (`uint16`)
- `CLASS`: Record class (`uint16`, IN=1 or IN+cache-flush=0x8001)
- `TTL`: Time-to-live (`uint32`)
- `RDLENGTH`: Resource data length (`uint16`)
- `RDATA`: Resource data (type-specific, `[]byte`)

**Operations**:
- `ParseRDATA() (interface{}, error)`: Parse RDATA based on TYPE (A → net.IP, PTR → string, etc.)

**Validation Rules**:
- `NAME` MUST be valid DNS name (can be compressed per RFC 1035 §4.1.4)
- `TYPE` MUST be one of supported types (A, PTR, SRV, TXT)
- `CLASS` MUST be IN (1) or IN+cache-flush (0x8001, M1 ignores cache-flush bit)
- `RDLENGTH` MUST match actual RDATA length

---

## Transport Layer Entities (`internal/network/`)

### Socket (Internal struct, not exported)

**Purpose**: UDP multicast socket wrapper.

**Attributes**:
- `conn`: UDP connection (`*net.UDPConn`)
- `multicastAddr`: Multicast address (`*net.UDPAddr`, 224.0.0.251:5353)

**Operations**:
- `send([]byte) error`: Send message to multicast group (FR-005)
- `receive(timeout time.Duration) ([]byte, error)`: Receive message with timeout (FR-006)
- `close() error`: Close socket (FR-017)

**Validation Rules**:
- Socket MUST bind to 224.0.0.251:5353 (FR-004)
- Socket MUST set read buffer to 65536 bytes (64KB) for multicast traffic
- Socket MUST close on Querier.Close() (FR-017, F-7 resource management)

---

## Error Entities (`internal/errors/`)

Per F-3 Error Handling Strategy:

### NetworkError

**Purpose**: Socket and I/O failures (FR-013).

**Attributes**:
- `message`: Error description (`string`)
- `cause`: Underlying error (`error`)

**Examples**:
- "failed to bind to 224.0.0.251:5353: permission denied (requires root or CAP_NET_RAW)"
- "failed to send query: network unreachable"
- "no network interfaces available"

---

### ValidationError

**Purpose**: Invalid query inputs (FR-014).

**Attributes**:
- `message`: Error description (`string`)
- `field`: Invalid field name (`string`, e.g., "name", "recordType")
- `value`: Invalid value (`interface{}`)

**Examples**:
- "name cannot be empty"
- "name exceeds maximum length (255 bytes)"
- "invalid characters in hostname"
- "unsupported record type: AAAA"

---

### WireFormatError

**Purpose**: Malformed DNS packets (FR-015).

**Attributes**:
- `message`: Error description (`string`)
- `offset`: Byte offset where error occurred (`int`)

**Examples**:
- "message too short: expected 12 bytes, got 5"
- "invalid compression pointer: offset 500 exceeds message length 200"
- "label length exceeds 63 bytes"
- "too many compression jumps (possible loop)"

---

## Entity Relationships

```
Querier (Public API)
  ├─ Query (internal) → validates inputs, executes lifecycle
  ├─ Response → contains ResourceRecords
  │   └─ ResourceRecord (A, PTR, SRV, TXT) → type-specific data
  │
  └─ Internal Dependencies:
      ├─ DNSMessage (Protocol Layer) → wire format
      │   ├─ DNSHeader → flags, counts
      │   ├─ Question → QNAME, QTYPE, QCLASS
      │   └─ Answer → NAME, TYPE, CLASS, TTL, RDATA
      │
      └─ Socket (Transport Layer) → UDP multicast I/O
```

---

## Validation Summary

**From Functional Requirements**:
- **FR-002**: RecordType limited to A, PTR, SRV, TXT
- **FR-003**: Name validation (≤255 bytes, labels ≤63 bytes, valid characters)
- **FR-007**: Timeout validation (100ms ≤ timeout ≤ 10s)
- **FR-011**: Response message format validation (≥12 bytes, valid sections)
- **FR-012**: Name compression validation (detect loops, validate offsets)
- **FR-020**: Query header validation (QR=0, OPCODE=0, etc.)
- **FR-021**: Response header validation (QR=1)
- **FR-022**: Response RCODE validation (ignore RCODE != 0)

---

**Data Model Status**: ✅ **COMPLETE**
**Next Phase**: API Contracts (`contracts/querier-api.md`)
