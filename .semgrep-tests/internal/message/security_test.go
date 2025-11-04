// Test file for F-11 Security Architecture rules
// This file intentionally contains security violations
package message

// ============================================================================
// F-11: Unsafe in Parser (beacon-unsafe-in-parser)
// ============================================================================

// SHOULD TRIGGER: beacon-unsafe-in-parser
import "unsafe"

func BadUseUnsafe(data []byte) {
	// Unsafe package forbidden in parsing code (security risk)
	_ = unsafe.Pointer(&data[0])
}

// ============================================================================
// F-11: Panic on Network Input (beacon-panic-on-network-input)
// ============================================================================

func ParseMessage(data []byte) error {
	if len(data) < 12 {
		// SHOULD TRIGGER: beacon-panic-on-network-input
		panic("message too short") // WRONG! Should return WireFormatError
	}
	return nil
}

func ParseMessageCorrect(data []byte) error {
	if len(data) < 12 {
		// Correct! Return error instead of panic
		return &WireFormatError{
			Op:      "parse message",
			Field:   "header",
			Message: "message too short",
		}
	}
	return nil
}

type WireFormatError struct {
	Op      string
	Field   string
	Message string
}

func (e *WireFormatError) Error() string {
	return e.Op + ": " + e.Field + ": " + e.Message
}
