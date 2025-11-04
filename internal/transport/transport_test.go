package transport_test

import (
	"testing"

	"github.com/joshuafuller/beacon/internal/transport"
)

// TDD - RED Phase: Tests for Transport interface
// These tests are written FIRST, before implementation exists
// Expected: COMPILATION ERRORS (transport.Transport doesn't exist yet)

// T010: Contract test - Transport interface compiles with Send/Receive/Close methods
// NOTE: This test verifies the interface exists and has the correct method signatures
func TestTransportInterface_HasRequiredMethods(_ *testing.T) {
	// Verify Transport interface exists and can be implemented
	// This test passes if the interface compiles with the expected method signatures
	var _ transport.Transport = (*transport.MockTransport)(nil)
	var _ transport.Transport = (*transport.UDPv4Transport)(nil)
}
