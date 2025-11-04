package responder

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
)

// ConflictDetector handles name conflict detection and resolution per RFC 6762 §8.
//
// RFC 6762 §8.2: When simultaneous probes occur, use lexicographic tie-breaking
// R002 Decision: Use bytes.Compare for lexicographic comparison of RDATA
//
// T019: Implement ConflictDetector with lexicographic tie-breaking
type ConflictDetector struct {
	// Stateless for now - no internal state needed
}

// NewConflictDetector creates a new conflict detector.
//
// Returns:
//   - *ConflictDetector: A new conflict detector instance
//
// T019: Initialize ConflictDetector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{}
}

// DetectConflict checks if two services conflict (same instance name).
//
// RFC 6762 §8.1: Conflict occurs when same name is used
//
// Parameters:
//   - ourService: The service we're trying to register
//   - theirService: The conflicting service
//
// Returns:
//   - bool: true if services conflict (same InstanceName)
//
// T019: Implement conflict detection
func (cd *ConflictDetector) DetectConflict(ourService, theirService *Service) bool {
	if ourService == nil || theirService == nil {
		return false
	}

	// Conflict if same instance name
	return ourService.InstanceName == theirService.InstanceName
}

// CompareProbes compares probe records for tie-breaking per RFC 6762 §8.2.1.
//
// RFC 6762 §8.2.1: "The two records are compared and the lexicographically
// later data wins."
//
// Parameters:
//   - ourData: Our probe record data (RDATA)
//   - theirData: Their probe record data (RDATA)
//
// Returns:
//   - bool: true if we win (our data is lexicographically later), false if they win
//
// R002 Decision: Use bytes.Compare for lexicographic comparison
// T019: Implement tie-breaking logic
func (cd *ConflictDetector) CompareProbes(ourData, theirData []byte) bool {
	// bytes.Compare returns:
	//   -1 if ourData < theirData (they win)
	//    0 if ourData == theirData (identical, no conflict)
	//    1 if ourData > theirData (we win)

	cmp := bytes.Compare(ourData, theirData)

	// We win if our data is lexicographically later (cmp > 0)
	// If identical (cmp == 0), no conflict - both can proceed
	return cmp > 0
}

// CompareMultipleRecords compares multiple probe records pairwise per RFC 6762 §8.2.1.
//
// RFC 6762 §8.2.1: "When a host is probing for a set of records with the same name,
// the host's records and the tiebreaker records from the message are each sorted
// into order, and then compared pairwise."
//
// Parameters:
//   - ourRecords: Our probe records (e.g., SRV + TXT)
//   - theirRecords: Their probe records
//
// Returns:
//   - bool: true if we win, false if they win
//
// T019: Handle services with multiple records
func (cd *ConflictDetector) CompareMultipleRecords(ourRecords, theirRecords [][]byte) bool {
	// Compare pairwise until a difference is found
	minLen := len(ourRecords)
	if len(theirRecords) < minLen {
		minLen = len(theirRecords)
	}

	for i := 0; i < minLen; i++ {
		cmp := bytes.Compare(ourRecords[i], theirRecords[i])
		if cmp > 0 {
			return true // We win
		} else if cmp < 0 {
			return false // They win
		}
		// cmp == 0: records match, continue to next pair
	}

	// If all compared records match, the list with more records wins
	// RFC 6762 §8.2.1: "the list with records remaining is deemed to have won"
	return len(ourRecords) > len(theirRecords)
}

// Rename generates a new instance name with incremented suffix.
//
// RFC 6762 §8.2: On conflict, rename service (e.g., "MyApp" → "MyApp (2)")
//
// Parameters:
//   - instanceName: Current instance name
//
// Returns:
//   - string: New instance name with incremented suffix
//
// Examples:
//   - "My Printer" → "My Printer (2)"
//   - "My Printer (2)" → "My Printer (3)"
//   - "My Printer (9)" → "My Printer (10)"
//
// T019: Implement renaming with suffix increment
func (cd *ConflictDetector) Rename(instanceName string) string {
	// Regex to match " (N)" suffix at end of name
	re := regexp.MustCompile(`^(.*)\s+\((\d+)\)$`)

	matches := re.FindStringSubmatch(instanceName)
	if matches != nil {
		// Already has suffix - increment it
		baseName := matches[1]
		// Error impossible: regex (\d+) ensures matches[2] contains only digits
		currentNum, _ := strconv.Atoi(matches[2]) // nosemgrep: beacon-error-swallowing
		newNum := currentNum + 1
		return fmt.Sprintf("%s (%d)", baseName, newNum)
	}

	// No suffix - add " (2)"
	return fmt.Sprintf("%s (2)", instanceName)
}
