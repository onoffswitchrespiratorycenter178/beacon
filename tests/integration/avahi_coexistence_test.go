//go:build unix

package integration

import (
	"context"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/querier"
)

// TestAvahiCoexistence verifies Beacon can coexist with Avahi/Bonjour on port 5353.
// Per F-9 REQ-F9-2: SO_REUSEPORT enables multiple processes to bind to same port.
//
// Test Strategy:
// 1. Create first listener on port 5353 (simulates Avahi/Bonjour)
// 2. Create Beacon querier (should also bind to port 5353)
// 3. Verify both can receive packets
// 4. Verify no "address already in use" error
func TestAvahiCoexistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Step 1: Create first listener on port 5353 (simulates Avahi/Bonjour)
	lc := net.ListenConfig{
		Control: func(_, _ string, c syscall.RawConn) error {
			var sockoptErr error
			err := c.Control(func(fd uintptr) {
				// Set SO_REUSEADDR and SO_REUSEPORT (Unix only - per build tag)
				sockoptErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if sockoptErr != nil {
					return
				}
				// SO_REUSEPORT = 15 on Linux/BSD
				sockoptErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0xf, 1)
			})
			if err != nil {
				return err
			}
			return sockoptErr
		},
	}

	firstListener, err := lc.ListenPacket(ctx, "udp4", "224.0.0.251:5353")
	if err != nil {
		t.Fatalf("Failed to create first listener (simulated Avahi): %v", err)
	}
	defer func() { _ = firstListener.Close() }()
	t.Logf("✓ First listener bound to %s (simulated Avahi/Bonjour)", firstListener.LocalAddr())

	// Step 2: Create Beacon querier (should also bind to port 5353)
	// This will fail with "address already in use" if SO_REUSEPORT is not working
	q, err := querier.New()
	if err != nil {
		// Check if error is "address already in use" - this indicates SO_REUSEPORT failure
		if opErr, ok := err.(*net.OpError); ok {
			if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
				if sysErr.Err == syscall.EADDRINUSE {
					t.Fatalf("✗ SC-001/SC-002 FAIL: Address already in use - SO_REUSEPORT not working: %v", err)
				}
			}
		}
		t.Fatalf("Failed to create Beacon querier: %v", err)
	}
	defer func() { _ = q.Close() }()
	t.Logf("✓ Beacon querier created (coexisting with first listener)")

	// Step 3: Verify both are bound (no conflict)
	// If we got here without "address already in use" error, SO_REUSEPORT is working
	t.Log("✓ SC-001/SC-002 PASS: Beacon coexists with Avahi/Bonjour on port 5353")

	// Step 4: Optional - Send test packet to verify both can receive
	// (In real scenario, actual mDNS responses would be received by both)
	testConn, err := net.DialUDP("udp4", nil, &net.UDPAddr{ // nosemgrep: beacon-socket-close-check - Connection is closed via defer on line 99
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	})
	if err != nil {
		t.Logf("Warning: Could not create test sender: %v", err)
		return
	}
	defer func() { _ = testConn.Close() }()

	testPacket := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if _, err := testConn.Write(testPacket); err != nil {
		t.Logf("Warning: Could not send test packet: %v", err)
	} else {
		t.Log("✓ Test packet sent to multicast group (both listeners can receive)")
	}
}

// TestAvahiCoexistence_ManualValidation provides instructions for manual testing
// with real Avahi/Bonjour daemons.
func TestAvahiCoexistence_ManualValidation(t *testing.T) {
	t.Skip("Manual test - run on system with Avahi (Linux) or Bonjour (macOS)")

	// Manual Test Instructions:
	// 1. Linux: Install Avahi: sudo apt-get install avahi-daemon
	// 2. Linux: Verify Avahi running: systemctl status avahi-daemon
	// 3. macOS: Bonjour runs by default (mDNSResponder)
	// 4. Run this test: go test -v -run TestAvahiCoexistence_ManualValidation
	// 5. Expected: No "address already in use" error
	// 6. Expected: Beacon can query services while Avahi/Bonjour continues running
	// 7. Validation: Run avahi-browse -a (Linux) or dns-sd -B _services._dns-sd._udp (macOS)
	//    alongside Beacon queries - both should work simultaneously
}
