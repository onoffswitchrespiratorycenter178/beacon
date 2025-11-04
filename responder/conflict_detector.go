package responder

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/joshuafuller/beacon/internal/message"
)

// ConflictDetector implements RFC 6762 §8.2 Simultaneous Probe Tiebreaking.
//
// It determines whether an incoming mDNS probe conflicts with our own probe
// by performing lexicographic comparison of resource records per the RFC algorithm:
//
//  1. Compare record class (excluding cache-flush bit) - numerically greater wins
//  2. Compare record type - numerically greater wins
//  3. Compare RDATA bytewise (as UNSIGNED 0-255 values) - greater byte wins
//  4. If one record runs out of data, the longer record wins
//  5. If records are identical, there is no conflict (fault tolerance)
//
// RFC 6762 §8.2: "When a host that is probing for a record sees another host
// issue a query for the same record, it consults the Authority Section of that
// query. If it finds any resource record(s) there which answers the query, then
// it compares the data of that (those) resource record(s) with its own tentative
// data... The two records are compared and the lexicographically later data wins."
//
// This implementation is stateless and safe for concurrent use by multiple
// Prober instances (F-4: Concurrency and Context Management).
//
// Task: T054-T058 (GREEN phase)
// PRIMARY TECHNICAL AUTHORITY: RFC 6762 §8.2
type ConflictDetector struct {
	// Stateless - no fields needed
}

// DetectConflict checks if incomingRecord conflicts with ourRecord per RFC 6762 §8.2.
//
// Returns:
//   - (true, nil):  Conflict detected - we lose tie-break and MUST defer
//   - (false, nil): No conflict - either different names, we win tie-break, or identical records
//   - (_, error):   Error validating or comparing records
//
// A conflict occurs when:
//  1. Both records have the same name (case-insensitive per DNS)
//  2. We lose the lexicographic comparison (their data > our data)
//
// No conflict occurs when:
//  1. Records have different names (not competing for same name)
//  2. We win the lexicographic comparison (our data > their data)
//  3. Records are identical (fault tolerance per RFC §8.2)
//
// RFC 6762 §8.2: "If the host finds that its own data is lexicographically
// earlier, then it defers to the winning host by waiting one second, and then
// begins probing for this record again."
//
// Examples:
//
//	// No conflict - different names
//	ourRecord: myservice.local A 192.168.1.100
//	incoming:  otherservice.local A 192.168.1.50
//	→ (false, nil)
//
//	// Conflict - same name, we lose
//	ourRecord: myservice.local A 192.168.1.50
//	incoming:  myservice.local A 192.168.1.100
//	→ (true, nil)
//
//	// No conflict - same name, we win
//	ourRecord: myservice.local A 192.168.1.100
//	incoming:  myservice.local A 192.168.1.50
//	→ (false, nil)
//
//	// No conflict - identical (fault tolerance)
//	ourRecord: myservice.local A 192.168.1.100
//	incoming:  myservice.local A 192.168.1.100
//	→ (false, nil)
//
// Safe for concurrent use by multiple goroutines (pure function, no shared state).
//
// Task: T055-T058
func (cd *ConflictDetector) DetectConflict(ourRecord, incomingRecord message.ResourceRecord) (bool, error) {
	// FR-003: Validate input records
	if err := cd.validateRecord(ourRecord); err != nil {
		return false, fmt.Errorf("invalid ourRecord: %w", err)
	}
	if err := cd.validateRecord(incomingRecord); err != nil {
		return false, fmt.Errorf("invalid incomingRecord: %w", err)
	}

	// RFC 6762 §8.2: Only records with the same name can conflict
	// DNS names are case-insensitive per RFC 1035 §2.3.3
	if !strings.EqualFold(ourRecord.Name, incomingRecord.Name) {
		return false, nil // Different names - no conflict
	}

	// RFC 6762 §8.2: Compare records lexicographically
	// Returns: -1 if we lose, 0 if tie, +1 if we win
	cmp := cd.lexicographicCompare(ourRecord, incomingRecord)

	if cmp < 0 {
		// We lose tie-break - CONFLICT (we must defer)
		return true, nil
	}

	// We win (cmp > 0) or tie (cmp == 0) - NO CONFLICT
	// RFC 6762 §8.2: "If both lists run out of records at the same time without
	// any difference being found, then this indicates that two devices are
	// advertising identical sets of records, as is sometimes done for fault
	// tolerance, and there is, in fact, no conflict."
	return false, nil
}

// validateRecord checks if a resource record is valid for conflict detection.
//
// Per F-3 (Error Handling), we return typed errors for invalid input.
//
// Task: T058
func (cd *ConflictDetector) validateRecord(record message.ResourceRecord) error {
	if record.Name == "" {
		return fmt.Errorf("empty name")
	}
	if record.Data == nil {
		return fmt.Errorf("nil data")
	}
	return nil
}

// lexicographicCompare compares two resource records per RFC 6762 §8.2.
//
// Returns:
//   - -1 if ourRecord < incomingRecord (we lose, must defer)
//   - 0  if ourRecord == incomingRecord (tie, no conflict)
//   - +1 if ourRecord > incomingRecord (we win, they defer)
//
// RFC 6762 §8.2 comparison algorithm:
//  1. Compare class (excluding cache-flush bit) - numerically greater wins
//  2. Compare type - numerically greater wins
//  3. Compare RDATA bytewise (UNSIGNED 0-255) - greater byte wins
//  4. If one runs out of data, longer wins
//
// RFC 6762 §8.2: "The determination of 'lexicographically later' is performed
// by first comparing the record class (excluding the cache-flush bit described
// in Section 10.2), then the record type, then raw comparison of the binary
// content of the rdata without regard for meaning or structure."
//
// Task: T056-T057
func (cd *ConflictDetector) lexicographicCompare(ourRecord, incomingRecord message.ResourceRecord) int {
	// Step 1: Compare class (excluding cache-flush bit)
	// RFC 6762 §10.2: Cache-flush bit is bit 15 of the class field
	ourClass := uint16(ourRecord.Class) & 0x7FFF // Clear bit 15
	theirClass := uint16(incomingRecord.Class) & 0x7FFF

	if ourClass < theirClass {
		return -1 // They win (numerically greater class)
	}
	if ourClass > theirClass {
		return +1 // We win (numerically greater class)
	}

	// Step 2: Compare type
	ourType := uint16(ourRecord.Type)
	theirType := uint16(incomingRecord.Type)

	if ourType < theirType {
		return -1 // They win (numerically greater type)
	}
	if ourType > theirType {
		return +1 // We win (numerically greater type)
	}

	// Step 3: Compare RDATA bytewise
	// RFC 6762 §8.2: "The bytes of the raw uncompressed rdata are compared in
	// turn, interpreting the bytes as eight-bit UNSIGNED values, until a byte
	// is found whose value is greater than that of its counterpart (in which
	// case, the rdata whose byte has the greater value is deemed lexicographically
	// later) or one of the resource records runs out of rdata (in which case, the
	// resource record which still has remaining data first is deemed lexicographically
	// later)."
	//
	// CRITICAL: "Note that it is vital that the bytes are interpreted as UNSIGNED
	// values in the range 0-255, or the wrong outcome may result."
	//
	// Example from RFC: 169.254.200.50 wins over 169.254.99.200
	// (byte 200 > byte 99, even though 200 as signed would be -56)
	cmp := bytes.Compare(ourRecord.Data, incomingRecord.Data)

	// bytes.Compare returns:
	// - -1 if ourRecord.Data < incomingRecord.Data (we lose)
	// -  0 if ourRecord.Data == incomingRecord.Data (tie)
	// - +1 if ourRecord.Data > incomingRecord.Data (we win)
	//
	// bytes.Compare uses UNSIGNED byte comparison, which is exactly what RFC requires
	return cmp
}
