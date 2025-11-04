# M1 Functional Requirements (FR-M1-001 through FR-M1-022)

**Source**: `specs/002-mdns-querier/checklists/requirements.md`
**Milestone**: M1 (Basic mDNS Querier)
**Task**: T005 (R002) - Extract and convert to milestone-prefixed IDs

---

## Query Construction (4 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-001 | System MUST construct valid mDNS query messages | ✅ Implemented | internal/message/builder.go (BuildQuery) | RFC 6762 §5.1 | tests/integration/TestQuery |
| FR-M1-002 | System MUST support querying for A, PTR, SRV, and TXT record types | ✅ Implemented | querier/querier.go (Query method), internal/message/builder.go | RFC 6762 §5 | tests/integration/TestQueryRecordTypes |
| FR-M1-003 | System MUST validate queried names follow DNS naming rules | ✅ Implemented | querier/querier.go (Query validation) | RFC 1035 §2.3.1 | tests/unit/TestQueryValidation |
| FR-M1-004 | System MUST use mDNS port 5353 and multicast address 224.0.0.251 | ✅ Implemented | internal/protocol/constants.go, network/socket.go | RFC 6762 §3 | tests/integration/TestQueryNetworkParams |

## Query Execution (4 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-005 | System MUST send mDNS queries to the multicast group | ✅ Implemented | network/socket.go (SendQuery) | RFC 6762 §5.3 | tests/integration/TestQueryTransmission |
| FR-M1-006 | System MUST listen for mDNS responses on port 5353 | ✅ Implemented | network/socket.go (ListenForResponses) | RFC 6762 §6 | tests/integration/TestQueryReceive |
| FR-M1-007 | System MUST accept configurable query timeout (default: 1 second, range: 100ms to 10 seconds) | ✅ Implemented | querier/options.go (WithTimeout) | - | tests/unit/TestQueryTimeout |
| FR-M1-008 | System MUST support context-based cancellation | ✅ Implemented | querier/querier.go (accepts context.Context) | - | tests/integration/TestQueryCancellation |

## Response Handling (4 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-009 | System MUST parse mDNS response messages per RFC 6762 | ✅ Implemented | internal/message/parser.go (ParseMessage) | RFC 6762 §6 | tests/contract/TestParseResponse |
| FR-M1-010 | System MUST extract Answer, Authority, and Additional sections | ✅ Implemented | internal/message/parser.go (ParseMessage) | RFC 1035 §4.1 | tests/unit/TestParseMessageSections |
| FR-M1-011 | System MUST validate response message format and discard malformed packets | ✅ Implemented | internal/message/parser.go (validation), internal/errors/errors.go (WireFormatError) | RFC 6762 §18.3 | tests/fuzz/FuzzParseMessage |
| FR-M1-012 | System MUST decompress DNS names per RFC 1035 §4.1.4 | ✅ Implemented | internal/message/name.go (ParseName - compression pointer handling) | RFC 1035 §4.1.4 | tests/unit/TestNameDecompression |

## Error Handling (4 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-013 | System MUST return NetworkError for socket creation, binding, or I/O failures | ✅ Implemented | internal/errors/errors.go (NetworkError type) | - | tests/integration/TestNetworkErrors |
| FR-M1-014 | System MUST return ValidationError for invalid query names or unsupported record types | ✅ Implemented | internal/errors/errors.go (ValidationError type) | - | tests/unit/TestValidationErrors |
| FR-M1-015 | System MUST return WireFormatError for malformed response packets | ✅ Implemented | internal/errors/errors.go (WireFormatError type) | RFC 6762 §18.3 | tests/fuzz/FuzzParseMessage |
| FR-M1-016 | System MUST log malformed packets at DEBUG level | ✅ Implemented | internal/message/parser.go (debug logging on parse errors) | - | tests/unit/TestMalformedPacketLogging |

## Resource Management (3 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-017 | System MUST clean up all sockets and goroutines when query completes | ✅ Implemented | querier/querier.go (defer cleanup, context cancellation) | - | tests/integration/TestResourceCleanup |
| FR-M1-018 | System MUST support graceful shutdown | ✅ Implemented | querier/querier.go (context cancellation propagation) | - | tests/integration/TestGracefulShutdown |
| FR-M1-019 | System MUST pass `go test -race` with zero race conditions | ✅ Implemented | All packages | - | CI: `go test ./... -race` |

## RFC 6762 Compliance (3 FRs)

| FR-ID | Description | Status | Implementation | RFC Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1-020 | System MUST set DNS header fields per RFC 6762 §18.1 (QR=0, OPCODE=0, AA=0, TC=0, RD=0) | ✅ Implemented | internal/message/builder.go (BuildQuery) | RFC 6762 §18.1 | tests/contract/TestQueryHeaderFormat |
| FR-M1-021 | System MUST validate received responses have QR=1 (response bit set) | ✅ Implemented | internal/message/parser.go (ParseMessage validation) | RFC 6762 §18.1 | tests/contract/TestResponseValidation |
| FR-M1-022 | System MUST ignore responses with RCODE != 0 (error responses) | ✅ Implemented | internal/message/parser.go (RCODE check) | RFC 1035 §4.1.1 | tests/unit/TestRCODEHandling |

---

## Summary

- **Total FRs**: 22
- **Status**: All ✅ Implemented (M1 complete)
- **Functional Areas**:
  - Query Construction: 4 FRs
  - Query Execution: 4 FRs
  - Response Handling: 4 FRs
  - Error Handling: 4 FRs
  - Resource Management: 3 FRs
  - RFC Compliance: 3 FRs

---

**Generated**: 2025-11-02
**Next**: Use this data in T031 (aggregate M1 FRs into FR matrix)
