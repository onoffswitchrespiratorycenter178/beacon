# Test Comment Guide: RFC References

**Feature**: 002-mdns-querier
**Purpose**: Ensure all tests explicitly reference RFC sections for traceability and compliance validation

---

## Test Comment Format

All tests that validate RFC requirements MUST include:
1. **RFC Section Reference**: Exact section number (e.g., RFC 6762 §18.2)
2. **Requirement ID**: Link to FR/NFR requirement (e.g., FR-020)
3. **Expected Behavior**: What the RFC mandates (MUST/SHOULD/MAY)
4. **Test Validation**: How the test verifies compliance

---

## Examples

### Example 1: Query Header Field Validation (T043, T021)

**File**: `tests/contract/rfc_test.go` or `internal/message/message_test.go`

```go
// TestBuildQuery_RFC6762_HeaderFields validates that mDNS query messages
// have correct header field values per RFC 6762 §18 (FR-020).
//
// RFC 6762 Requirements:
//   §18.2 - QR bit MUST be zero in query messages
//   §18.3 - OPCODE MUST be zero (standard query)
//   §18.4 - AA bit MUST be zero in query messages
//   §18.5 - TC bit clear means no additional Known Answers
//   §18.6 - RD bit SHOULD be zero (M1 enforces MUST for simplicity)
//
// FR-020: System MUST set DNS header fields per RFC 6762 §18
func TestBuildQuery_RFC6762_HeaderFields(t *testing.T) {
    query, err := BuildQuery("test.local", RecordTypeA)
    if err != nil {
        t.Fatalf("BuildQuery failed: %v", err)
    }

    header := parseHeader(query)

    // RFC 6762 §18.2: QR bit MUST be zero in queries
    if header.Flags&0x8000 != 0 {
        t.Errorf("QR bit is 1, expected 0 per RFC 6762 §18.2")
    }

    // RFC 6762 §18.3: OPCODE MUST be zero
    opcode := (header.Flags >> 11) & 0x0F
    if opcode != 0 {
        t.Errorf("OPCODE is %d, expected 0 per RFC 6762 §18.3", opcode)
    }

    // RFC 6762 §18.4: AA bit MUST be zero in queries
    if header.Flags&0x0400 != 0 {
        t.Errorf("AA bit is 1, expected 0 per RFC 6762 §18.4")
    }

    // RFC 6762 §18.5: TC bit clear (no Known Answers in M1)
    if header.Flags&0x0200 != 0 {
        t.Errorf("TC bit is 1, expected 0 per RFC 6762 §18.5")
    }

    // RFC 6762 §18.6: RD bit SHOULD be zero (M1 enforces as MUST)
    if header.Flags&0x0100 != 0 {
        t.Errorf("RD bit is 1, expected 0 per RFC 6762 §18.6")
    }
}
```

### Example 2: Response Validation (T044, T031)

**File**: `tests/contract/rfc_test.go` or `internal/protocol/validator_test.go`

```go
// TestValidateResponse_RFC6762_QRBit validates that responses have QR=1
// per RFC 6762 §18.2 (FR-021).
//
// RFC 6762 §18.2 states: "In response messages the QR bit MUST be one."
//
// FR-021: System MUST validate received responses have QR=1
func TestValidateResponse_RFC6762_QRBit(t *testing.T) {
    tests := []struct {
        name      string
        flags     uint16
        expectErr bool
        errMsg    string
    }{
        {
            name:      "valid response QR=1",
            flags:     0x8000, // QR=1, all other bits 0
            expectErr: false,
        },
        {
            name:      "invalid query QR=0 per RFC 6762 §18.2",
            flags:     0x0000, // QR=0
            expectErr: true,
            errMsg:    "QR bit is 0, expected 1 per RFC 6762 §18.2",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msg := &DNSMessage{
                Header: DNSHeader{Flags: tt.flags},
            }

            err := ValidateResponse(msg)

            if tt.expectErr && err == nil {
                t.Error("expected error, got nil")
            }
            if !tt.expectErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if tt.expectErr && err != nil && !strings.Contains(err.Error(), "RFC 6762 §18.2") {
                t.Errorf("error missing RFC reference, got: %v", err)
            }
        })
    }
}

// TestValidateResponse_RFC6762_RCODE validates that non-zero RCODE responses
// are silently ignored per RFC 6762 §18.11 (FR-022).
//
// RFC 6762 §18.11 states: "Multicast DNS messages received with non-zero
// Response Codes MUST be silently ignored."
//
// FR-022: System MUST ignore responses with RCODE != 0
func TestValidateResponse_RFC6762_RCODE(t *testing.T) {
    tests := []struct {
        name   string
        rcode  uint16
        ignore bool
    }{
        {
            name:   "RCODE=0 (no error) per RFC 6762 §18.11",
            rcode:  0x0000,
            ignore: false,
        },
        {
            name:   "RCODE=1 (format error) - ignore per RFC 6762 §18.11",
            rcode:  0x0001,
            ignore: true,
        },
        {
            name:   "RCODE=2 (server failure) - ignore per RFC 6762 §18.11",
            rcode:  0x0002,
            ignore: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msg := &DNSMessage{
                Header: DNSHeader{
                    Flags: 0x8000 | tt.rcode, // QR=1, RCODE=tt.rcode
                },
            }

            err := ValidateResponse(msg)

            if tt.ignore && err == nil {
                t.Error("expected error (to ignore response), got nil")
            }
            if !tt.ignore && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
```

### Example 3: Name Compression (T019, T017)

**File**: `internal/message/name_test.go`

```go
// TestParseName_RFC1035_Compression validates DNS name compression per
// RFC 1035 §4.1.4 (FR-012).
//
// RFC 1035 §4.1.4 defines message compression using pointers (high 2 bits = 11).
// RFC 6762 §18.14 states: "implementations SHOULD use name compression wherever
// possible... [RFC1035]."
//
// FR-012: System MUST decompress DNS names per RFC 1035 §4.1.4
func TestParseName_RFC1035_Compression(t *testing.T) {
    tests := []struct {
        name     string
        data     []byte
        offset   int
        expected string
        errMsg   string
    }{
        {
            name: "uncompressed name per RFC 1035 §4.1.4",
            data: []byte{
                // "test.local\x00"
                0x04, 't', 'e', 's', 't',
                0x05, 'l', 'o', 'c', 'a', 'l',
                0x00,
            },
            offset:   0,
            expected: "test.local",
        },
        {
            name: "compressed pointer per RFC 1035 §4.1.4",
            data: []byte{
                // Offset 0: "example.local\x00"
                0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
                0x05, 'l', 'o', 'c', 'a', 'l',
                0x00,
                // Offset 16: "test" + pointer to "local" at offset 8
                0x04, 't', 'e', 's', 't',
                0xC0, 0x08, // Compression pointer: 11000000 00001000 (points to offset 8)
            },
            offset:   16,
            expected: "test.local",
        },
        {
            name: "compression loop detection per RFC 1035 §4.1.4",
            data: []byte{
                0xC0, 0x00, // Pointer to self (infinite loop)
            },
            offset: 0,
            errMsg: "too many compression jumps (possible loop)",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, _, err := ParseName(tt.data, tt.offset)

            if tt.errMsg != "" {
                if err == nil {
                    t.Errorf("expected error containing %q, got nil", tt.errMsg)
                } else if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
                }
            } else {
                if err != nil {
                    t.Errorf("unexpected error: %v", err)
                }
                if result != tt.expected {
                    t.Errorf("expected %q, got %q", tt.expected, result)
                }
            }
        })
    }
}
```

---

## Test Comment Requirements

### For All RFC Compliance Tests:

1. **Function Comment**:
   ```go
   // Test<Function>_RFC<Number>_<Feature> validates <what> per RFC <number> §<section> (FR-###).
   //
   // RFC <number> §<section> states: "<exact quote from RFC if short>"
   //
   // FR-###: <requirement text>
   ```

2. **Inline Comments** (for each assertion):
   ```go
   // RFC <number> §<section>: <specific MUST/SHOULD/MAY requirement>
   if condition {
       t.Errorf("<what failed>, expected <value> per RFC <number> §<section>")
   }
   ```

3. **Error Messages** MUST include RFC section:
   ```go
   return fmt.Errorf("QR bit is 0, expected 1 per RFC 6762 §18.2")
   ```

### For Unit Tests (Non-RFC):

1. **Function Comment**:
   ```go
   // Test<Function>_<Scenario> validates <what> (FR-###).
   //
   // FR-###: <requirement text>
   ```

2. **Error Messages**:
   ```go
   t.Errorf("expected <value>, got <actual>")
   ```

---

## Checklist for Test Implementation

When implementing tests for tasks T043, T044, T021, T031, T019, etc.:

- [ ] Function comment includes RFC section reference
- [ ] Function comment includes FR/NFR requirement ID
- [ ] Function comment includes exact RFC quote (if short)
- [ ] Each assertion has inline comment with RFC section
- [ ] Error messages include RFC section for failures
- [ ] Test names follow `Test<Function>_RFC<Number>_<Feature>` pattern
- [ ] Table-driven tests include RFC section in test case names

---

## Benefits

This approach ensures:
1. ✅ **Traceability**: Clear path from RFC → FR → Test → Implementation
2. ✅ **Constitution Compliance**: Principle I (RFC Compliant) rigorously enforced
3. ✅ **Maintainability**: Future developers can trace requirements to RFC source
4. ✅ **Auditability**: External reviewers can verify RFC compliance
5. ✅ **Documentation**: Tests serve as executable RFC compliance documentation

---

**Status**: ✅ **MANDATORY** for all M1 implementation tests
**Next Phase**: Use this guide during `/speckit.implement` for T043, T044, T021, T031, T019, etc.
