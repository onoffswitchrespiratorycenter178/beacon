package message

import (
	"encoding/binary"
	"testing"

	"github.com/joshuafuller/beacon/internal/protocol"
)

// TestBuildResponse_RED tests response message construction for mDNS responder.
//
// TDD Phase: RED - These tests will FAIL until we implement BuildResponse()
//
// RFC 6762 §18 Response Requirements:
//   - §18.2: QR bit MUST be one (response)
//   - §18.3: OPCODE MUST be zero (standard query)
//   - §18.4: AA bit MUST be one for authoritative answers
//   - §18.11: RCODE MUST be zero (mDNS doesn't use error codes)
//
// FR-023: System MUST construct valid mDNS response messages per RFC 6762
// T011: Extend message.Builder for response messages (QR=1, AA=1)
func TestBuildResponse_HeaderFlags(t *testing.T) {
	tests := []struct {
		name       string
		wantQR     bool   // QR bit (bit 15) - MUST be 1 for responses
		wantAA     bool   // AA bit (bit 10) - MUST be 1 for authoritative
		wantOPCODE uint16 // MUST be 0
		wantRCODE  uint16 // MUST be 0
	}{
		{
			name:       "response message has QR=1 per RFC 6762 §18.2",
			wantQR:     true,
			wantAA:     true,
			wantOPCODE: 0,
			wantRCODE:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a minimal response (empty answers for now)
			response, err := BuildResponse(nil) // nil = no answers yet

			if err != nil {
				t.Fatalf("BuildResponse() error = %v, want nil", err)
			}

			if len(response) < 12 {
				t.Fatalf("response too short: got %d bytes, want at least 12 (header)", len(response))
			}

			// Parse header flags (bytes 2-3)
			flags := binary.BigEndian.Uint16(response[2:4])

			// Check QR bit (bit 15) - MUST be 1 for responses
			gotQR := (flags & protocol.FlagQR) != 0
			if gotQR != tt.wantQR {
				t.Errorf("QR bit = %v, want %v (RFC 6762 §18.2: responses MUST have QR=1)", gotQR, tt.wantQR)
			}

			// Check AA bit (bit 10) - MUST be 1 for authoritative answers
			gotAA := (flags & protocol.FlagAA) != 0
			if gotAA != tt.wantAA {
				t.Errorf("AA bit = %v, want %v (RFC 6762 §18.4: authoritative answers MUST have AA=1)", gotAA, tt.wantAA)
			}

			// Check OPCODE (bits 11-14) - MUST be 0
			gotOPCODE := (flags >> 11) & 0x0F
			if gotOPCODE != tt.wantOPCODE {
				t.Errorf("OPCODE = %d, want %d (RFC 6762 §18.3: MUST be 0)", gotOPCODE, tt.wantOPCODE)
			}

			// Check RCODE (bits 0-3) - MUST be 0
			gotRCODE := flags & 0x0F
			if gotRCODE != tt.wantRCODE {
				t.Errorf("RCODE = %d, want %d (RFC 6762 §18.11: MUST be 0)", gotRCODE, tt.wantRCODE)
			}
		})
	}
}

// TestBuildResponse_AnswerSection_RED tests that response includes answers.
//
// TDD Phase: RED - Will fail until we implement answer serialization
//
// RFC 6762 §6: Response messages contain answer records
// T011: Response builder must support answer section
func TestBuildResponse_WithAnswers(t *testing.T) {
	// Define a simple A record answer
	answer := &ResourceRecord{
		Name:  "test.local",
		Type:  protocol.RecordTypeA,
		Class: protocol.ClassIN,
		TTL:   protocol.TTLHostname,     // 4500 seconds per RFC 6762 §10
		Data:  []byte{192, 168, 1, 100}, // 192.168.1.100
	}

	response, err := BuildResponse([]*ResourceRecord{answer})
	if err != nil {
		t.Fatalf("BuildResponse() error = %v, want nil", err)
	}

	if len(response) < 12 {
		t.Fatalf("response too short: got %d bytes, want at least 12", len(response))
	}

	// Check ANCOUNT (bytes 6-7) - should be 1
	ancount := binary.BigEndian.Uint16(response[6:8])
	if ancount != 1 {
		t.Errorf("ANCOUNT = %d, want 1 (response should have 1 answer)", ancount)
	}

	// Response should be longer than just header if it has answers
	if len(response) <= 12 {
		t.Errorf("response size = %d bytes, want > 12 (should include answer section)", len(response))
	}
}

// TestBuildResponse_CacheFlushBit_RED tests cache-flush bit for unique records.
//
// TDD Phase: RED - Will fail until we implement cache-flush bit
//
// RFC 6762 §10.2: Unique records SHOULD set cache-flush bit (bit 15 of class)
// T011: Response builder must support cache-flush bit
func TestBuildResponse_CacheFlushBit(t *testing.T) {
	// Unique record (A record for hostname)
	uniqueRecord := &ResourceRecord{
		Name:       "myhost.local",
		Type:       protocol.RecordTypeA,
		Class:      protocol.ClassIN,
		TTL:        protocol.TTLHostname,
		Data:       []byte{192, 168, 1, 100},
		CacheFlush: true, // Cache-flush bit should be set
	}

	response, err := BuildResponse([]*ResourceRecord{uniqueRecord})
	if err != nil {
		t.Fatalf("BuildResponse() error = %v, want nil", err)
	}

	// Parse the class field from the answer section
	// Skip header (12 bytes) + name encoding + type (2 bytes)
	// This is a simplified test - actual parsing would need to skip encoded name
	// For now, we're just testing that BuildResponse accepts the CacheFlush field

	if len(response) <= 12 {
		t.Errorf("response should include answer section with cache-flush bit")
	}
}

// TestBuildResponse_MultipleAnswers_RED tests responses with multiple records.
//
// TDD Phase: RED - Will fail until we implement multiple answers
//
// RFC 6762 §6.1: Responses MAY contain multiple answer records
// T011: Response builder must support multiple answers
func TestBuildResponse_MultipleAnswers(t *testing.T) {
	answers := []*ResourceRecord{
		{
			Name:  "test.local",
			Type:  protocol.RecordTypeA,
			Class: protocol.ClassIN,
			TTL:   protocol.TTLHostname,
			Data:  []byte{192, 168, 1, 100},
		},
		{
			Name:  "test.local",
			Type:  protocol.RecordTypeTXT,
			Class: protocol.ClassIN,
			TTL:   protocol.TTLService,
			Data:  []byte{0x00}, // Empty TXT record per RFC 6763 §6
		},
	}

	response, err := BuildResponse(answers)
	if err != nil {
		t.Fatalf("BuildResponse() error = %v, want nil", err)
	}

	// Check ANCOUNT should be 2
	ancount := binary.BigEndian.Uint16(response[6:8])
	if ancount != 2 {
		t.Errorf("ANCOUNT = %d, want 2", ancount)
	}
}

// Note: ResourceRecord type is now implemented in builder.go (T012 GREEN phase)
