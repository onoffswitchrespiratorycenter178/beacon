// Package records manages DNS resource records with TTL tracking.
package records

import (
	"time"

	"github.com/joshuafuller/beacon/internal/protocol"
)

// RecordTTL represents a DNS record with TTL tracking.
//
// RFC 6762 §10: TTL values decrease over time from creation timestamp.
//
// T017: Implement TTL calculation (GetRemainingTTL, IsExpired)
type RecordTTL struct {
	RecordType protocol.RecordType
	TTL        uint32    // Initial TTL in seconds
	CreatedAt  time.Time // Creation timestamp for TTL calculation
}

// NewRecordTTL creates a new record with TTL tracking.
//
// Parameters:
//   - rt: The DNS record type (A, PTR, SRV, TXT)
//   - ttl: The initial TTL value in seconds
//
// Returns:
//   - *RecordTTL: A new record with CreatedAt set to current time
//
// T017: Create records with timestamp for TTL calculation
func NewRecordTTL(rt protocol.RecordType, ttl uint32) *RecordTTL {
	return &RecordTTL{
		RecordType: rt,
		TTL:        ttl,
		CreatedAt:  time.Now(),
	}
}

// GetRemainingTTL returns the remaining TTL in seconds.
//
// Calculates: TTL - elapsed_time
// Returns 0 if TTL has expired (elapsed >= TTL)
//
// RFC 6762 §10: TTL values decrease over time
//
// T017: Implement TTL calculation
func (r *RecordTTL) GetRemainingTTL() uint32 {
	elapsed := uint32(time.Since(r.CreatedAt).Seconds())

	// If elapsed time >= TTL, record has expired
	if elapsed >= r.TTL {
		return 0
	}

	// Return remaining TTL
	return r.TTL - elapsed
}

// IsExpired returns true if the record has expired (TTL reached zero).
//
// RFC 6762 §10: Records expire when TTL reaches zero
//
// T017: Implement expiration check
func (r *RecordTTL) IsExpired() bool {
	elapsed := time.Since(r.CreatedAt)
	return elapsed >= time.Duration(r.TTL)*time.Second
}

// GetTTLForRecordType returns the appropriate TTL for a record type per RFC 6762 §10.
//
// RFC 6762 §10:
//   - Service records (SRV, TXT, PTR): 120 seconds
//   - Hostname records (A, AAAA): 4500 seconds (75 minutes)
//
// Parameters:
//   - rt: The DNS record type
//
// Returns:
//   - uint32: The recommended TTL in seconds
//
// T017: Map record types to TTL values per RFC 6762 §10
func GetTTLForRecordType(rt protocol.RecordType) uint32 {
	switch rt {
	case protocol.RecordTypeA:
		// A records use TTLHostname (4500s) per RFC 6762 §10
		return protocol.TTLHostname

	case protocol.RecordTypeSRV, protocol.RecordTypeTXT, protocol.RecordTypePTR:
		// Service records use TTLService (120s) per RFC 6762 §10
		return protocol.TTLService

	default:
		// Default to service TTL for unknown types
		return protocol.TTLService
	}
}
