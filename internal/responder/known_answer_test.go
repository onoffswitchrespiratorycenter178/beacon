package responder

import (
	"testing"

	"github.com/joshuafuller/beacon/internal/message"
	"github.com/joshuafuller/beacon/internal/protocol"
)

// TestApplyKnownAnswerSuppression_TTLThreshold_RED tests known-answer suppression
// per RFC 6762 §7.1 TTL threshold rules.
//
// TDD Phase: RED - This test will FAIL until we implement ApplyKnownAnswerSuppression()
//
// RFC 6762 §7.1: "A Multicast DNS responder MUST NOT answer a Multicast DNS query
// if the answer it would give is already included in the Answer Section with an RR
// TTL at least half the correct value."
//
// FR-035: System MUST suppress responses when known-answer TTL ≥50% of true TTL
// T087: Test known-answer suppression with TTL ≥50%
func TestApplyKnownAnswerSuppression_TTLThreshold(t *testing.T) {
	rb := NewResponseBuilder()

	// Our PTR record (we would normally send this)
	ourRecord := &message.ResourceRecord{
		Name:  "_http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   120, // True TTL = 120 seconds
		Data:  []byte{7, 'M', 'y', ' ', 'A', 'p', 'p', 's', 14, '_', 'h', 't', 't', 'p', '.', '_', 't', 'c', 'p', '.', 'l', 'o', 'c', 'a', 'l', 0},
	}

	tests := []struct {
		name           string
		knownTTL       uint32
		shouldSuppress bool
		description    string
	}{
		{
			name:           "TTL at 100% - suppress",
			knownTTL:       120, // 100% of true TTL
			shouldSuppress: true,
			description:    "RFC 6762 §7.1: ≥50% TTL means suppress",
		},
		{
			name:           "TTL at 75% - suppress",
			knownTTL:       90, // 75% of true TTL
			shouldSuppress: true,
			description:    "RFC 6762 §7.1: ≥50% TTL means suppress",
		},
		{
			name:           "TTL at exactly 50% - suppress",
			knownTTL:       60, // Exactly 50% of true TTL
			shouldSuppress: true,
			description:    "RFC 6762 §7.1: ≥50% TTL means suppress (boundary case)",
		},
		{
			name:           "TTL at 49% - do NOT suppress",
			knownTTL:       59, // 49.16% of true TTL (just under threshold)
			shouldSuppress: false,
			description:    "RFC 6762 §7.1: <50% TTL means respond to refresh cache",
		},
		{
			name:           "TTL at 25% - do NOT suppress",
			knownTTL:       30, // 25% of true TTL
			shouldSuppress: false,
			description:    "RFC 6762 §7.1: <50% TTL means respond to refresh cache",
		},
		{
			name:           "TTL at 1% - do NOT suppress",
			knownTTL:       1, // Near expiration
			shouldSuppress: false,
			description:    "RFC 6762 §7.1: <50% TTL means respond to refresh cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Known-answer from query (same record, different TTL)
			knownAnswer := &message.ResourceRecord{
				Name:  ourRecord.Name,
				Type:  ourRecord.Type,
				Class: ourRecord.Class,
				TTL:   tt.knownTTL, // Querier's cached TTL
				Data:  ourRecord.Data,
			}

			// Apply known-answer suppression
			shouldInclude := rb.ApplyKnownAnswerSuppression(ourRecord, []*message.ResourceRecord{knownAnswer})

			// Verify suppression behavior
			if tt.shouldSuppress {
				if shouldInclude {
					t.Errorf("%s: shouldInclude = true, want false (should suppress)", tt.description)
				}
			} else {
				if !shouldInclude {
					t.Errorf("%s: shouldInclude = false, want true (should respond)", tt.description)
				}
			}
		})
	}
}

// TestApplyKnownAnswerSuppression_MismatchedRDATA_RED tests that mismatched
// RDATA does NOT trigger suppression.
//
// TDD Phase: RED
//
// RFC 6762 §7.1: Suppression only applies when the answer matches exactly.
// Different RDATA means different answer, so responder MUST respond.
//
// T088: Test known-answer with mismatched RDATA (don't suppress)
func TestApplyKnownAnswerSuppression_MismatchedRDATA(t *testing.T) {
	rb := NewResponseBuilder()

	// Our PTR record
	ourRecord := &message.ResourceRecord{
		Name:  "_http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   120,
		Data:  []byte{7, 'M', 'y', ' ', 'A', 'p', 'p', 's', 14, '_', 'h', 't', 't', 'p', '.', '_', 't', 'c', 'p', '.', 'l', 'o', 'c', 'a', 'l', 0},
	}

	// Known-answer with DIFFERENT RDATA (different service instance)
	knownAnswer := &message.ResourceRecord{
		Name:  ourRecord.Name,
		Type:  ourRecord.Type,
		Class: ourRecord.Class,
		TTL:   120,                                                                                                                                           // Even at 100% TTL
		Data:  []byte{9, 'O', 't', 'h', 'e', 'r', ' ', 'A', 'p', 'p', 14, '_', 'h', 't', 't', 'p', '.', '_', 't', 'c', 'p', '.', 'l', 'o', 'c', 'a', 'l', 0}, // Different instance name
	}

	// Apply known-answer suppression
	shouldInclude := rb.ApplyKnownAnswerSuppression(ourRecord, []*message.ResourceRecord{knownAnswer})

	// MUST NOT suppress - different RDATA means different answer
	if !shouldInclude {
		t.Error("shouldInclude = false, want true (mismatched RDATA should NOT suppress)")
	}
}

// TestApplyKnownAnswerSuppression_NoKnownAnswers_RED tests behavior when
// query has no known-answer section.
//
// TDD Phase: RED
//
// RFC 6762 §7.1: If no known-answers provided, respond normally.
//
// T088: Edge case - empty known-answer list
func TestApplyKnownAnswerSuppression_NoKnownAnswers(t *testing.T) {
	rb := NewResponseBuilder()

	ourRecord := &message.ResourceRecord{
		Name:  "_http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   120,
		Data:  []byte{7, 'M', 'y', ' ', 'A', 'p', 'p', 's', 14, '_', 'h', 't', 't', 'p', '.', '_', 't', 'c', 'p', '.', 'l', 'o', 'c', 'a', 'l', 0},
	}

	// No known answers
	shouldInclude := rb.ApplyKnownAnswerSuppression(ourRecord, []*message.ResourceRecord{})

	// MUST include - no suppression without known-answers
	if !shouldInclude {
		t.Error("shouldInclude = false, want true (no known-answers means no suppression)")
	}
}

// TestApplyKnownAnswerSuppression_DifferentType_RED tests that known-answers
// for different record types don't suppress.
//
// TDD Phase: RED
//
// RFC 6762 §7.1: Known-answer must match Name, Type, Class, and RDATA.
//
// T088: Edge case - different record type
func TestApplyKnownAnswerSuppression_DifferentType(t *testing.T) {
	rb := NewResponseBuilder()

	// Our PTR record
	ourRecord := &message.ResourceRecord{
		Name:  "_http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   120,
		Data:  []byte{7, 'M', 'y', ' ', 'A', 'p', 'p', 's', 14, '_', 'h', 't', 't', 'p', '.', '_', 't', 'c', 'p', '.', 'l', 'o', 'c', 'a', 'l', 0},
	}

	// Known-answer with DIFFERENT TYPE (SRV instead of PTR)
	knownAnswer := &message.ResourceRecord{
		Name:  ourRecord.Name,
		Type:  protocol.RecordTypeSRV, // Different type
		Class: ourRecord.Class,
		TTL:   120,
		Data:  ourRecord.Data,
	}

	// Apply known-answer suppression
	shouldInclude := rb.ApplyKnownAnswerSuppression(ourRecord, []*message.ResourceRecord{knownAnswer})

	// MUST NOT suppress - different type
	if !shouldInclude {
		t.Error("shouldInclude = false, want true (different type should NOT suppress)")
	}
}
