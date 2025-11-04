package message

import (
	goerrors "errors"
	"strings"
	"testing"

	"github.com/joshuafuller/beacon/internal/errors"
)

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
		wantOff  int
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
			wantOff:  12,
		},
		{
			name: "compressed pointer per RFC 1035 §4.1.4",
			data: []byte{
				// Offset 0: "example.local\x00"
				0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
				0x05, 'l', 'o', 'c', 'a', 'l',
				0x00,
				// Offset 15: "test" + pointer to "local" at offset 8
				0x04, 't', 'e', 's', 't',
				0xC0, 0x08, // Compression pointer: 11000000 00001000 (points to offset 8)
			},
			offset:   15,
			expected: "test.local",
			wantOff:  22, // After length+label (1+4=5 bytes) and 2-byte pointer (15+5+2=22)
		},
		{
			name: "compression loop detection per RFC 1035 §4.1.4",
			data: []byte{
				0xC0, 0x00, // Pointer to self (self-reference rejected immediately)
			},
			offset: 0,
			errMsg: "invalid compression pointer",
		},
		{
			name: "root name (empty)",
			data: []byte{
				0x00, // Zero-length label (root)
			},
			offset:   0,
			expected: "",
			wantOff:  1,
		},
		{
			name: "single label",
			data: []byte{
				0x04, 't', 'e', 's', 't',
				0x00,
			},
			offset:   0,
			expected: "test",
			wantOff:  6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, newOffset, err := ParseName(tt.data, tt.offset)

			if tt.errMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}

			if newOffset != tt.wantOff {
				t.Errorf("expected offset %d, got %d", tt.wantOff, newOffset)
			}
		})
	}
}

// TestParseName_RFC1035_LabelLength validates that ParseName enforces the
// maximum label length of 63 bytes per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1 states: "Labels must be 63 octets or less."
//
// FR-003: System MUST validate queried names follow DNS naming rules (labels ≤63 bytes)
func TestParseName_RFC1035_LabelLength(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		errMsg string
	}{
		{
			name: "label exactly 63 bytes (valid per RFC 1035 §3.1)",
			data: func() []byte {
				// Create a 63-byte label
				data := []byte{63}
				for i := 0; i < 63; i++ {
					data = append(data, 'a')
				}
				data = append(data, 0) // Terminator
				return data
			}(),
			errMsg: "", // No error expected
		},
		{
			name: "label 64 bytes (exceeds maximum per RFC 1035 §3.1)",
			data: []byte{
				64, // Length byte = 64 (exceeds 63)
				'a', 'a', 'a', 'a', 'a', 'a', 'a', 'a',
			},
			errMsg: "exceeds maximum 63 bytes per RFC 1035 §3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseName(tt.data, 0)

			if tt.errMsg != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestParseName_RFC1035_NameLength validates that ParseName enforces the
// maximum name length of 255 bytes per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1 states: "The total number of octets that represent a domain
// name is limited to 255."
//
// FR-003: System MUST validate queried names follow DNS naming rules (total name ≤255 bytes)
func TestParseName_RFC1035_NameLength(t *testing.T) {
	// Create a name that exceeds 255 bytes
	var data []byte
	// Add labels until we exceed 255 bytes
	for i := 0; i < 50; i++ { // 50 labels of 5 bytes each = 300 bytes
		data = append(data, 5, 'l', 'a', 'b', 'e', 'l')
	}
	data = append(data, 0) // Terminator

	_, _, err := ParseName(data, 0)

	if err == nil {
		t.Error("expected error for name exceeding 255 bytes per RFC 1035 §3.1, got nil")
		return
	}

	if !strings.Contains(err.Error(), "exceeds maximum 255 bytes per RFC 1035 §3.1") {
		t.Errorf("expected error about 255 byte limit, got: %v", err)
	}
}

// TestParseName_TruncatedMessage validates that ParseName returns WireFormatError
// when the message is truncated (FR-015).
//
// FR-015: System MUST return WireFormatError for malformed response packets
func TestParseName_TruncatedMessage(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		offset int
		errMsg string
	}{
		{
			name:   "truncated label",
			data:   []byte{0x05, 't', 'e'}, // Says 5 bytes, only has 2
			offset: 0,
			errMsg: "truncated label",
		},
		{
			name:   "truncated compression pointer",
			data:   []byte{0xC0}, // Compression pointer needs 2 bytes, only has 1
			offset: 0,
			errMsg: "truncated compression pointer",
		},
		{
			name:   "offset out of bounds",
			data:   []byte{0x04, 't', 'e', 's', 't', 0x00},
			offset: 100,
			errMsg: "offset out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseName(tt.data, tt.offset)

			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errMsg)
				return
			}

			// Verify it's a WireFormatError per FR-015
			var wireErr *errors.WireFormatError
			if !goerrors.As(err, &wireErr) {
				t.Errorf("expected WireFormatError per FR-015, got %T", err)
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestEncodeName_RFC1035_BasicEncoding validates that EncodeName correctly
// encodes DNS names per RFC 1035 §3.1 (FR-001, FR-003).
//
// RFC 1035 §3.1: "Domain names in messages are expressed in terms of a sequence
// of labels. Each label is represented as a one octet length field followed by
// that number of octets."
//
// FR-001: System MUST construct valid mDNS query messages per RFC 6762
// FR-003: System MUST validate queried names follow DNS naming rules
func TestEncodeName_RFC1035_BasicEncoding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:  "simple name per RFC 1035 §3.1",
			input: "test.local",
			expected: []byte{
				0x04, 't', 'e', 's', 't',
				0x05, 'l', 'o', 'c', 'a', 'l',
				0x00,
			},
		},
		{
			name:  "root name",
			input: "",
			expected: []byte{
				0x00,
			},
		},
		{
			name:  "root name with dot",
			input: ".",
			expected: []byte{
				0x00,
			},
		},
		{
			name:  "name with trailing dot",
			input: "test.local.",
			expected: []byte{
				0x04, 't', 'e', 's', 't',
				0x05, 'l', 'o', 'c', 'a', 'l',
				0x00,
			},
		},
		{
			name:  "service name with underscore",
			input: "_http._tcp.local",
			expected: []byte{
				0x05, '_', 'h', 't', 't', 'p',
				0x04, '_', 't', 'c', 'p',
				0x05, 'l', 'o', 'c', 'a', 'l',
				0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeName(tt.input)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if i >= len(tt.expected) {
					break
				}
				if result[i] != tt.expected[i] {
					t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestEncodeName_RFC1035_Validation validates that EncodeName rejects invalid
// names per RFC 1035 §3.1 (FR-003, FR-014).
//
// RFC 1035 §3.1: Labels must be ≤63 bytes, total name ≤255 bytes, valid characters.
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
func TestEncodeName_RFC1035_Validation(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{
			name:   "empty label (consecutive dots)",
			input:  "test..local",
			errMsg: "empty label",
		},
		{
			name:   "label exceeds 63 bytes per RFC 1035 §3.1",
			input:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.local", // 64 'a's
			errMsg: "exceeds maximum length 63 bytes per RFC 1035 §3.1",
		},
		{
			name:   "invalid character (space)",
			input:  "test host.local",
			errMsg: "invalid character",
		},
		{
			name:   "hyphen at start of label",
			input:  "-test.local",
			errMsg: "hyphen cannot be first or last character",
		},
		{
			name:   "hyphen at end of label",
			input:  "test-.local",
			errMsg: "hyphen cannot be first or last character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeName(tt.input)

			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.errMsg)
				return
			}

			// Verify it's a ValidationError per FR-014
			var valErr *errors.ValidationError
			if !goerrors.As(err, &valErr) {
				t.Errorf("expected ValidationError per FR-014, got %T", err)
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestEncodeName_MaxNameLength validates that EncodeName enforces the 255-byte
// limit per RFC 1035 §3.1 (FR-003).
//
// RFC 1035 §3.1: "The total number of octets that represent a domain name is
// limited to 255."
//
// FR-003: System MUST validate queried names follow DNS naming rules (total name ≤255 bytes)
func TestEncodeName_MaxNameLength(t *testing.T) {
	// Create a name that will exceed 255 bytes when encoded
	// Each label: 1 byte length + 63 bytes data = 64 bytes
	// 4 labels = 256 bytes + 1 terminator = 257 bytes (exceeds 255)
	var labels []string
	for i := 0; i < 4; i++ {
		label := strings.Repeat("a", 63) // Max label size
		labels = append(labels, label)
	}
	name := strings.Join(labels, ".")

	_, err := EncodeName(name)

	if err == nil {
		t.Error("expected error for name exceeding 255 bytes per RFC 1035 §3.1, got nil")
		return
	}

	if !strings.Contains(err.Error(), "exceeds maximum 255 bytes per RFC 1035 §3.1") {
		t.Errorf("expected error about 255 byte limit, got: %v", err)
	}
}

// TestParseEncodeName_Roundtrip validates that ParseName and EncodeName are
// inverse operations for valid names.
func TestParseEncodeName_Roundtrip(t *testing.T) {
	tests := []string{
		"test.local",
		"printer.local",
		"_http._tcp.local",
		"my-device.local",
		"a.b.c.d.local",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			// Encode
			encoded, err := EncodeName(name)
			if err != nil {
				t.Fatalf("EncodeName failed: %v", err)
			}

			// Parse
			decoded, _, err := ParseName(encoded, 0)
			if err != nil {
				t.Fatalf("ParseName failed: %v", err)
			}

			// Verify roundtrip
			if decoded != name {
				t.Errorf("roundtrip failed: encoded %q, decoded %q", name, decoded)
			}
		})
	}
}
