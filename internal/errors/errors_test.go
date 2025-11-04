package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestNetworkError_Error validates that NetworkError.Error() returns
// actionable error messages per NFR-006.
//
// NFR-006: Error messages MUST include actionable context
func TestNetworkError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *NetworkError
		want    string
		wantAll []string // All strings that must be present in error message
	}{
		{
			name: "with details",
			err: &NetworkError{
				Operation: "bind socket",
				Err:       fmt.Errorf("permission denied"),
				Details:   "requires root or CAP_NET_RAW",
			},
			wantAll: []string{"network error", "bind socket", "permission denied", "requires root or CAP_NET_RAW"},
		},
		{
			name: "without details",
			err: &NetworkError{
				Operation: "send query",
				Err:       fmt.Errorf("network unreachable"),
			},
			wantAll: []string{"network error", "send query", "network unreachable"},
		},
		{
			name: "socket creation failure",
			err: &NetworkError{
				Operation: "create socket",
				Err:       fmt.Errorf("address family not supported"),
				Details:   "IPv6 not enabled on this system",
			},
			wantAll: []string{"network error", "create socket", "address family not supported", "IPv6 not enabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("NetworkError.Error() missing expected substring:\ngot:  %q\nwant: %q", got, want)
				}
			}
		})
	}
}

// TestNetworkError_Unwrap validates that NetworkError.Unwrap() returns
// the underlying error for error chain inspection.
func TestNetworkError_Unwrap(t *testing.T) {
	underlying := fmt.Errorf("connection refused")
	err := &NetworkError{
		Operation: "connect",
		Err:       underlying,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("NetworkError.Unwrap() = %v, want %v", unwrapped, underlying)
	}

	// Verify errors.Is works
	if !errors.Is(err, underlying) {
		t.Error("errors.Is(NetworkError, underlying) = false, want true")
	}
}

// TestValidationError_Error validates that ValidationError.Error() returns
// actionable error messages per NFR-006.
//
// NFR-006: Error messages MUST include actionable context
// FR-014: System MUST return ValidationError for invalid query names or unsupported record types
func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantAll []string // All strings that must be present in error message
	}{
		{
			name: "with value",
			err: &ValidationError{
				Field:   "name",
				Value:   "",
				Message: "name cannot be empty",
			},
			wantAll: []string{"validation error", "name", "name cannot be empty", "value:"},
		},
		{
			name: "without value",
			err: &ValidationError{
				Field:   "timeout",
				Message: "timeout must be between 100ms and 10s",
			},
			wantAll: []string{"validation error", "timeout", "timeout must be between 100ms and 10s"},
		},
		{
			name: "unsupported record type",
			err: &ValidationError{
				Field:   "recordType",
				Value:   28, // AAAA (IPv6, not supported in M1)
				Message: "unsupported record type: AAAA",
			},
			wantAll: []string{"validation error", "recordType", "unsupported record type: AAAA", "28"},
		},
		{
			name: "invalid hostname characters",
			err: &ValidationError{
				Field:   "name",
				Value:   "host name with spaces.local",
				Message: "invalid characters in hostname",
			},
			wantAll: []string{"validation error", "name", "invalid characters in hostname", "host name with spaces.local"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("ValidationError.Error() missing expected substring:\ngot:  %q\nwant: %q", got, want)
				}
			}
		})
	}
}

// TestWireFormatError_Error validates that WireFormatError.Error() returns
// actionable error messages per NFR-006.
//
// NFR-006: Error messages MUST include actionable context
// FR-015: System MUST return WireFormatError for malformed response packets
func TestWireFormatError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *WireFormatError
		wantAll []string // All strings that must be present in error message
	}{
		{
			name: "with offset and underlying error",
			err: &WireFormatError{
				Operation: "parse header",
				Offset:    12,
				Message:   "truncated message",
				Err:       fmt.Errorf("unexpected EOF"),
			},
			wantAll: []string{"wire format error", "parse header", "offset 12", "truncated message", "unexpected EOF"},
		},
		{
			name: "with offset only",
			err: &WireFormatError{
				Operation: "decompress name",
				Offset:    48,
				Message:   "invalid compression pointer",
			},
			wantAll: []string{"wire format error", "decompress name", "offset 48", "invalid compression pointer"},
		},
		{
			name: "without offset",
			err: &WireFormatError{
				Operation: "validate response",
				Offset:    -1,
				Message:   "QR bit is 0, expected 1 per RFC 6762 §18.2",
			},
			wantAll: []string{"wire format error", "validate response", "QR bit is 0", "RFC 6762 §18.2"},
		},
		{
			name: "compression loop detection",
			err: &WireFormatError{
				Operation: "decompress name",
				Offset:    24,
				Message:   "too many compression jumps (possible loop)",
				Err:       fmt.Errorf("exceeded 256 jumps"),
			},
			wantAll: []string{"wire format error", "decompress name", "offset 24", "too many compression jumps", "exceeded 256 jumps"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("WireFormatError.Error() missing expected substring:\ngot:  %q\nwant: %q", got, want)
				}
			}
		})
	}
}

// TestWireFormatError_Unwrap validates that WireFormatError.Unwrap() returns
// the underlying error for error chain inspection.
func TestWireFormatError_Unwrap(t *testing.T) {
	underlying := fmt.Errorf("buffer underflow")
	err := &WireFormatError{
		Operation: "read field",
		Offset:    10,
		Message:   "not enough bytes",
		Err:       underlying,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("WireFormatError.Unwrap() = %v, want %v", unwrapped, underlying)
	}

	// Verify errors.Is works
	if !errors.Is(err, underlying) {
		t.Error("errors.Is(WireFormatError, underlying) = false, want true")
	}
}

// TestWireFormatError_NoUnderlyingError validates that Unwrap returns nil
// when there is no underlying error.
func TestWireFormatError_NoUnderlyingError(t *testing.T) {
	err := &WireFormatError{
		Operation: "validate",
		Message:   "invalid value",
	}

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("WireFormatError.Unwrap() = %v, want nil", unwrapped)
	}
}

// TestNetworkError_AsError validates that NetworkError can be used as error interface.
func TestNetworkError_AsError(t *testing.T) {
	var err error = &NetworkError{
		Operation: "test",
		Err:       fmt.Errorf("test error"),
	}

	if err.Error() == "" {
		t.Error("NetworkError.Error() returned empty string")
	}

	// Verify errors.As works
	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Error("errors.As(error, *NetworkError) = false, want true")
	}
}

// TestValidationError_AsError validates that ValidationError can be used as error interface.
func TestValidationError_AsError(t *testing.T) {
	var err error = &ValidationError{
		Field:   "test",
		Message: "test message",
	}

	if err.Error() == "" {
		t.Error("ValidationError.Error() returned empty string")
	}

	// Verify errors.As works
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Error("errors.As(error, *ValidationError) = false, want true")
	}
}

// TestWireFormatError_AsError validates that WireFormatError can be used as error interface.
func TestWireFormatError_AsError(t *testing.T) {
	var err error = &WireFormatError{
		Operation: "test",
		Message:   "test message",
	}

	if err.Error() == "" {
		t.Error("WireFormatError.Error() returned empty string")
	}

	// Verify errors.As works
	var wireErr *WireFormatError
	if !errors.As(err, &wireErr) {
		t.Error("errors.As(error, *WireFormatError) = false, want true")
	}
}
