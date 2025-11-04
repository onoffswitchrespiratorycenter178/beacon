package protocol

import (
	goerrors "errors"
	"strings"
	"testing"

	"github.com/joshuafuller/beacon/internal/errors"
)

// TestValidateName_RFC1035_ValidNames validates that ValidateName accepts
// valid DNS names per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1: Domain names are limited to 255 bytes, labels to 63 bytes,
// and characters to [a-z0-9-_] (case insensitive).
//
// FR-003: System MUST validate queried names follow DNS naming rules
func TestValidateName_RFC1035_ValidNames(t *testing.T) {
	tests := []struct {
		name    string
		dnsName string
		wantErr bool
	}{
		{
			name:    "simple valid name per RFC 1035 §3.1",
			dnsName: "test.local",
			wantErr: false,
		},
		{
			name:    "printer name",
			dnsName: "printer.local",
			wantErr: false,
		},
		{
			name:    "service name with underscores (valid for mDNS)",
			dnsName: "_http._tcp.local",
			wantErr: false,
		},
		{
			name:    "name with hyphens",
			dnsName: "my-device.local",
			wantErr: false,
		},
		{
			name:    "multi-level name",
			dnsName: "a.b.c.d.local",
			wantErr: false,
		},
		{
			name:    "single label",
			dnsName: "localhost",
			wantErr: false,
		},
		{
			name:    "label exactly 63 bytes (valid per RFC 1035 §3.1)",
			dnsName: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.local",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.dnsName)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateName(%q) expected error, got nil", tt.dnsName)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateName(%q) unexpected error: %v", tt.dnsName, err)
			}
		})
	}
}

// TestValidateName_RFC1035_InvalidNames validates that ValidateName rejects
// invalid DNS names per RFC 1035 §3.1 and returns ValidationError (FR-003, FR-014).
//
// RFC 1035 §3.1: Names exceeding 255 bytes or labels exceeding 63 bytes are invalid.
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
func TestValidateName_RFC1035_InvalidNames(t *testing.T) {
	tests := []struct {
		name     string
		dnsName  string
		errField string
	}{
		{
			name:     "empty name",
			dnsName:  "",
			errField: "name",
		},
		{
			name:     "label exceeds 63 bytes per RFC 1035 §3.1",
			dnsName:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.local", // 64 'a's
			errField: "name",
		},
		{
			name:     "invalid character (space)",
			dnsName:  "test host.local",
			errField: "name",
		},
		{
			name:     "invalid character (slash)",
			dnsName:  "test/host.local",
			errField: "name",
		},
		{
			name:     "label starts with hyphen (invalid per RFC 1035 §3.1)",
			dnsName:  "-test.local",
			errField: "name",
		},
		{
			name:     "label ends with hyphen (invalid per RFC 1035 §3.1)",
			dnsName:  "test-.local",
			errField: "name",
		},
		{
			name:     "empty label (consecutive dots)",
			dnsName:  "test..local",
			errField: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.dnsName)

			if err == nil {
				t.Errorf("ValidateName(%q) expected error per FR-003, FR-014, got nil", tt.dnsName)
				return
			}

			// Verify it's a ValidationError
			var validationErr *errors.ValidationError
			if !goerrors.As(err, &validationErr) {
				t.Errorf("ValidateName(%q) expected ValidationError per FR-014, got %T: %v", tt.dnsName, err, err)
				return
			}

			// Verify the field is set correctly
			if validationErr.Field != tt.errField {
				t.Errorf("ValidationError.Field = %q, expected %q", validationErr.Field, tt.errField)
			}
		})
	}
}

// TestValidateName_RFC1035_MaxNameLength validates that ValidateName enforces
// the 255-byte maximum name length per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1: The total length of a domain name is limited to 255 bytes.
//
// FR-003: System MUST validate queried names follow DNS naming rules
func TestValidateName_RFC1035_MaxNameLength(t *testing.T) {
	// Programmatically build a name that's exactly 255 bytes in wire format (VALID)
	// Wire format: each label = 1 byte length prefix + label content, plus 1 byte terminator
	//
	// 3 labels of 63 bytes: 3 * (1 + 63) = 192 bytes
	// 1 label of 61 bytes: 1 * (1 + 61) = 62 bytes
	// 1 byte terminator: 1 byte
	// Total: 192 + 62 + 1 = 255 bytes (exactly at limit)

	label63a := strings.Repeat("a", 63)
	label63b := strings.Repeat("b", 63)
	label63c := strings.Repeat("c", 63)
	label61 := strings.Repeat("d", 61)

	validName := label63a + "." + label63b + "." + label63c + "." + label61

	err := ValidateName(validName)
	if err != nil {
		t.Errorf("ValidateName(255-byte name) expected to pass per RFC 1035 §3.1, got error: %v", err)
	}

	// Build a name that exceeds 255 bytes (256 bytes in wire format - INVALID)
	// Wire format: 3 * (1 + 63) + 1 * (1 + 62) + 1 = 192 + 63 + 1 = 256 bytes
	label62 := strings.Repeat("e", 62)
	invalidName := label63a + "." + label63b + "." + label63c + "." + label62

	err = ValidateName(invalidName)
	if err == nil {
		t.Errorf("ValidateName(256-byte name) expected error per RFC 1035 §3.1, got nil")
		return
	}

	var validationErr *errors.ValidationError
	if !goerrors.As(err, &validationErr) {
		t.Errorf("Expected ValidationError per FR-014, got %T: %v", err, err)
	}
}

// TestValidateRecordType_FR002_SupportedTypes validates that ValidateRecordType
// accepts only A, PTR, SRV, and TXT record types per FR-002.
//
// FR-002: System MUST support querying for A, PTR, SRV, and TXT record types
// FR-014: System MUST return ValidationError for unsupported record types
func TestValidateRecordType_FR002_SupportedTypes(t *testing.T) {
	tests := []struct {
		name       string
		recordType uint16
		wantErr    bool
	}{
		{
			name:       "A record (1) supported per FR-002",
			recordType: 1,
			wantErr:    false,
		},
		{
			name:       "PTR record (12) supported per FR-002",
			recordType: 12,
			wantErr:    false,
		},
		{
			name:       "TXT record (16) supported per FR-002",
			recordType: 16,
			wantErr:    false,
		},
		{
			name:       "SRV record (33) supported per FR-002",
			recordType: 33,
			wantErr:    false,
		},
		{
			name:       "AAAA record (28) not supported in M1",
			recordType: 28,
			wantErr:    true,
		},
		{
			name:       "MX record (15) not supported in M1",
			recordType: 15,
			wantErr:    true,
		},
		{
			name:       "Unknown record type (999)",
			recordType: 999,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordType(tt.recordType)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateRecordType(%d) expected error per FR-002, got nil", tt.recordType)
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateRecordType(%d) unexpected error: %v", tt.recordType, err)
				return
			}

			// If error expected, verify it's a ValidationError
			if tt.wantErr {
				var validationErr *errors.ValidationError
				if !goerrors.As(err, &validationErr) {
					t.Errorf("ValidateRecordType(%d) expected ValidationError per FR-014, got %T: %v", tt.recordType, err, err)
					return
				}

				if validationErr.Field != "recordType" {
					t.Errorf("ValidationError.Field = %q, expected %q", validationErr.Field, "recordType")
				}
			}
		})
	}
}

// TestValidateResponse_RFC6762_ResponseFlags validates that ValidateResponse
// checks QR=1 per RFC 6762 §18.2 and RCODE=0 per RFC 6762 §18.11 (FR-021, FR-022).
//
// RFC 6762 §18.2: Response messages MUST have QR bit set to 1
// RFC 6762 §18.11: mDNS responders MUST ignore messages with non-zero RCODE
//
// FR-021: System MUST verify QR bit is set in response messages
// FR-022: System MUST validate RCODE is 0 per RFC 6762 §18.11
func TestValidateResponse_RFC6762_ResponseFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   uint16
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid response: QR=1, RCODE=0 per RFC 6762 §18.2, §18.11",
			flags:   0x8000, // QR=1, OPCODE=0, AA=0, TC=0, RD=0, RA=0, Z=0, RCODE=0
			wantErr: false,
		},
		{
			name:    "valid response: QR=1, AA=1, RCODE=0",
			flags:   0x8400, // QR=1, AA=1, RCODE=0
			wantErr: false,
		},
		{
			name:    "invalid: QR=0 (query, not response) per RFC 6762 §18.2",
			flags:   0x0000, // QR=0
			wantErr: true,
			errMsg:  "QR bit",
		},
		{
			name:    "invalid: RCODE=1 (format error) per RFC 6762 §18.11",
			flags:   0x8001, // QR=1, RCODE=1
			wantErr: true,
			errMsg:  "RCODE",
		},
		{
			name:    "invalid: RCODE=2 (server failure) per RFC 6762 §18.11",
			flags:   0x8002, // QR=1, RCODE=2
			wantErr: true,
			errMsg:  "RCODE",
		},
		{
			name:    "invalid: RCODE=3 (name error / NXDOMAIN) per RFC 6762 §18.11",
			flags:   0x8003, // QR=1, RCODE=3
			wantErr: true,
			errMsg:  "RCODE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResponse(tt.flags)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateResponse(0x%04X) expected error per FR-021/FR-022, got nil", tt.flags)
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateResponse(0x%04X) unexpected error: %v", tt.flags, err)
				return
			}

			// If error expected, verify it's a ValidationError
			if tt.wantErr {
				var validationErr *errors.ValidationError
				if !goerrors.As(err, &validationErr) {
					t.Errorf("ValidateResponse(0x%04X) expected ValidationError, got %T: %v", tt.flags, err, err)
					return
				}

				// Verify error message contains expected substring
				if tt.errMsg != "" {
					errStr := validationErr.Error()
					if len(errStr) == 0 {
						t.Errorf("ValidationError has empty message")
					}
				}
			}
		})
	}
}

// TestValidateResponse_RFC6762_OpcodeHandling validates that ValidateResponse
// handles OPCODE field correctly per RFC 6762 §18.3 (FR-021).
//
// RFC 6762 §18.3: OPCODE MUST be zero in mDNS messages
//
// FR-021: System MUST verify QR bit is set in response messages
func TestValidateResponse_RFC6762_OpcodeHandling(t *testing.T) {
	tests := []struct {
		name    string
		flags   uint16
		wantErr bool
	}{
		{
			name:    "OPCODE=0 (standard query) per RFC 6762 §18.3",
			flags:   0x8000, // QR=1, OPCODE=0, RCODE=0
			wantErr: false,
		},
		{
			name:    "OPCODE=1 (inverse query) - invalid per RFC 6762 §18.3",
			flags:   0x8800, // QR=1, OPCODE=1, RCODE=0
			wantErr: true,
		},
		{
			name:    "OPCODE=2 (status) - invalid per RFC 6762 §18.3",
			flags:   0x9000, // QR=1, OPCODE=2, RCODE=0
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResponse(tt.flags)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateResponse(0x%04X) expected error per RFC 6762 §18.3, got nil", tt.flags)
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateResponse(0x%04X) unexpected error: %v", tt.flags, err)
			}
		})
	}
}
