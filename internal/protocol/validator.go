// Package protocol implements mDNS protocol validation and constants.
package protocol

import (
	"fmt"
	"strings"

	"github.com/joshuafuller/beacon/internal/errors"
)

// ValidateName validates a DNS name per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1 DNS Naming Rules:
//   - Total name length: ≤255 bytes
//   - Label length: ≤63 bytes
//   - Valid characters: [a-z0-9-_] (case insensitive)
//   - Labels MUST NOT start or end with hyphen
//   - Empty labels are invalid (no consecutive dots)
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
//
// Parameters:
//   - name: The DNS name to validate (e.g., "test.local", "_http._tcp.local")
//
// Returns:
//   - error: ValidationError if name is invalid, nil if valid
func ValidateName(name string) error {
	// Empty name is invalid (root name "." is handled as empty string)
	if name == "" {
		return &errors.ValidationError{
			Field:   "name",
			Value:   name,
			Message: "name cannot be empty",
		}
	}

	// Remove trailing dot if present (canonical form)
	name = strings.TrimSuffix(name, ".")

	// Split into labels and validate each
	labels := strings.Split(name, ".")

	// Calculate wire format length (255 bytes max per RFC 1035 §3.1)
	// Wire format: each label has 1 byte length prefix + label content, plus 1 byte terminator
	wireLength := 1 // terminator
	for _, label := range labels {
		wireLength += 1 + len(label) // length prefix + label content
	}

	if wireLength > MaxNameLength {
		return &errors.ValidationError{
			Field:   "name",
			Value:   name,
			Message: fmt.Sprintf("name exceeds maximum length %d bytes (wire format: %d bytes) per RFC 1035 §3.1", MaxNameLength, wireLength),
		}
	}
	for i, label := range labels {
		if err := validateLabel(label, i); err != nil {
			// Wrap with ValidationError including the full name
			return &errors.ValidationError{
				Field:   "name",
				Value:   name,
				Message: err.Error(),
			}
		}
	}

	return nil
}

// validateLabel validates a single DNS label per RFC 1035 §3.1.
//
// RFC 1035 §3.1: Labels are limited to 63 bytes and valid characters.
func validateLabel(label string, position int) error {
	// Empty label (consecutive dots)
	if label == "" {
		return fmt.Errorf("empty label at position %d (consecutive dots)", position)
	}

	// Label length check (63 bytes max per RFC 1035 §3.1)
	if len(label) > MaxLabelLength {
		return fmt.Errorf("label %q exceeds maximum length 63 bytes per RFC 1035 §3.1", label)
	}

	// Label MUST NOT start with hyphen (per RFC 1035 §3.1)
	if strings.HasPrefix(label, "-") {
		return fmt.Errorf("label %q starts with hyphen (invalid per RFC 1035 §3.1)", label)
	}

	// Label MUST NOT end with hyphen (per RFC 1035 §3.1)
	if strings.HasSuffix(label, "-") {
		return fmt.Errorf("label %q ends with hyphen (invalid per RFC 1035 §3.1)", label)
	}

	// Validate characters: [a-zA-Z0-9-_]
	// Note: Underscore (_) is technically not in RFC 1035, but is allowed in mDNS
	// service names (e.g., "_http._tcp.local")
	for i, ch := range label {
		if !isValidDNSChar(ch) {
			return fmt.Errorf("invalid character %q in label %q (position %d)", ch, label, i)
		}
	}

	return nil
}

// isValidDNSChar checks if a character is valid in a DNS label.
//
// Valid characters per RFC 1035 §3.1: [a-zA-Z0-9-]
// mDNS extension: Underscore (_) is allowed for service names per RFC 6763
func isValidDNSChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' ||
		ch == '_' // mDNS service names (e.g., "_http._tcp.local")
}

// ValidateRecordType validates that the record type is supported per FR-002.
//
// FR-002: System MUST support querying for A, PTR, SRV, and TXT record types
// FR-014: System MUST return ValidationError for unsupported record types
//
// M1 Supported Types:
//   - A (1): IPv4 address
//   - PTR (12): Pointer (service discovery)
//   - TXT (16): Text strings (service metadata)
//   - SRV (33): Service location
//
// Parameters:
//   - recordType: The DNS record type to validate
//
// Returns:
//   - error: ValidationError if type is unsupported, nil if supported
func ValidateRecordType(recordType uint16) error {
	if !RecordType(recordType).IsSupported() {
		return &errors.ValidationError{
			Field:   "recordType",
			Value:   recordType,
			Message: fmt.Sprintf("unsupported record type %d (M1 supports A=1, PTR=12, TXT=16, SRV=33 per FR-002)", recordType),
		}
	}
	return nil
}

// ValidateResponse validates DNS response flags per RFC 6762 §18 (FR-021, FR-022).
//
// RFC 6762 Requirements:
//
//	§18.2: Response messages MUST have QR bit set to 1
//	§18.3: OPCODE MUST be zero (standard query)
//	§18.11: mDNS responders MUST ignore messages with non-zero RCODE
//
// FR-021: System MUST verify QR bit is set in response messages
// FR-022: System MUST validate RCODE is 0 per RFC 6762 §18.11
//
// Parameters:
//   - flags: The DNS header flags field (16 bits)
//
// Returns:
//   - error: ValidationError if response is invalid, nil if valid
func ValidateResponse(flags uint16) error {
	// RFC 6762 §18.2: QR bit MUST be 1 in responses
	qr := (flags & FlagQR) >> 15
	if qr != 1 {
		return &errors.ValidationError{
			Field:   "flags",
			Value:   flags,
			Message: fmt.Sprintf("QR bit is %d, expected 1 per RFC 6762 §18.2 (flags: 0x%04X)", qr, flags),
		}
	}

	// RFC 6762 §18.3: OPCODE MUST be 0 (standard query)
	opcode := (flags >> 11) & 0x0F
	if opcode != OpcodeQuery {
		return &errors.ValidationError{
			Field:   "flags",
			Value:   flags,
			Message: fmt.Sprintf("OPCODE is %d, expected %d per RFC 6762 §18.3 (flags: 0x%04X)", opcode, OpcodeQuery, flags),
		}
	}

	// RFC 6762 §18.11: RCODE MUST be 0 (no error)
	// mDNS responders MUST ignore messages with non-zero RCODE
	rcode := flags & 0x000F
	if rcode != RCodeNoError {
		return &errors.ValidationError{
			Field:   "flags",
			Value:   flags,
			Message: fmt.Sprintf("RCODE is %d, expected %d per RFC 6762 §18.11 (flags: 0x%04X)", rcode, RCodeNoError, flags),
		}
	}

	return nil
}
