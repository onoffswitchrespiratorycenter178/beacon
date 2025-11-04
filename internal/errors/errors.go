// Package errors defines error types for the Beacon mDNS querier.
//
// This package implements the error handling requirements from F-3 (Error Handling)
// and provides structured error types for network, validation, and wire format errors.
//
// Architecture: Per F-3, all errors include:
//   - Operation context (what operation failed)
//   - Root cause (underlying error if any)
//   - Actionable message (how to fix the problem)
//
// Requirements:
//   - FR-013: NetworkError for socket creation, binding, or I/O failures
//   - FR-014: ValidationError for invalid query names or unsupported record types
//   - FR-015: WireFormatError for malformed response packets
//   - NFR-006: Error messages MUST include actionable context
package errors

import (
	"fmt"
)

// NetworkError represents network-related failures such as socket creation,
// binding, or I/O operations.
//
// This error type is returned when the system cannot establish or use network
// resources required for mDNS queries.
//
// FR-013: System MUST return NetworkError for socket creation, binding, or I/O failures
type NetworkError struct {
	// Operation describes what network operation failed (e.g., "bind socket", "send query")
	Operation string

	// Err is the underlying error from the network stack
	Err error

	// Details provides additional context for troubleshooting
	Details string
}

// Error implements the error interface for NetworkError.
//
// NFR-006: Error messages MUST include actionable context
func (e *NetworkError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("network error during %s: %v (%s)", e.Operation, e.Err, e.Details)
	}
	return fmt.Sprintf("network error during %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying error, enabling error chain inspection with errors.Is/As.
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// ValidationError represents validation failures for query inputs such as
// invalid names, unsupported record types, or out-of-range parameters.
//
// This error type is returned when the caller provides invalid input to the querier API.
//
// FR-014: System MUST return ValidationError for invalid query names or unsupported record types
type ValidationError struct {
	// Field identifies which input field failed validation (e.g., "name", "recordType", "timeout")
	Field string

	// Value is the invalid value that was provided (if safe to include)
	Value interface{}

	// Message describes why the validation failed
	Message string
}

// Error implements the error interface for ValidationError.
//
// NFR-006: Error messages MUST include actionable context
func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error for %s: %s (value: %v)", e.Field, e.Message, e.Value)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// WireFormatError represents errors parsing DNS wire format messages, including
// malformed packets, invalid compression pointers, or truncated data.
//
// This error type is returned when a received mDNS response cannot be parsed
// according to RFC 1035/6762 wire format specifications.
//
// FR-015: System MUST return WireFormatError for malformed response packets
type WireFormatError struct {
	// Operation describes what parsing operation failed (e.g., "parse header", "decompress name")
	Operation string

	// Offset indicates the byte offset in the message where the error occurred (if known)
	Offset int

	// Message describes why the wire format is invalid
	Message string

	// Err is the underlying error (if any)
	Err error
}

// Error implements the error interface for WireFormatError.
//
// NFR-006: Error messages MUST include actionable context
func (e *WireFormatError) Error() string {
	if e.Offset >= 0 {
		if e.Err != nil {
			return fmt.Sprintf("wire format error during %s at offset %d: %s (underlying: %v)", e.Operation, e.Offset, e.Message, e.Err)
		}
		return fmt.Sprintf("wire format error during %s at offset %d: %s", e.Operation, e.Offset, e.Message)
	}

	if e.Err != nil {
		return fmt.Sprintf("wire format error during %s: %s (underlying: %v)", e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("wire format error during %s: %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error, enabling error chain inspection with errors.Is/As.
func (e *WireFormatError) Unwrap() error {
	return e.Err
}
