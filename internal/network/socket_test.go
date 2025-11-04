package network

import (
	goerrors "errors"
	"net"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/internal/errors"
	"github.com/joshuafuller/beacon/internal/protocol"
)

// TestCreateSocket_RFC6762_MulticastBind validates that CreateSocket binds to
// mDNS multicast address per RFC 6762 §5 (FR-004).
//
// RFC 6762 §5: mDNS uses multicast address 224.0.0.251 on port 5353
//
// FR-004: System MUST use mDNS port 5353 and multicast address 224.0.0.251
func TestCreateSocket_RFC6762_MulticastBind(t *testing.T) {
	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed per FR-004: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Verify the connection is UDP
	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		t.Fatalf("CreateSocket() returned %T, expected *net.UDPConn", conn)
	}

	// Verify local address is bound to port 5353
	localAddr := udpConn.LocalAddr().(*net.UDPAddr)
	if localAddr.Port != protocol.Port {
		t.Errorf("Socket bound to port %d, expected %d per RFC 6762 §5", localAddr.Port, protocol.Port)
	}
}

// TestCreateSocket_ErrorHandling validates that CreateSocket returns
// NetworkError for socket creation failures (FR-013).
//
// FR-013: System MUST return NetworkError for socket creation failures
func TestCreateSocket_ErrorHandling(t *testing.T) {
	// Note: This test is difficult to trigger without OS-level interference
	// In normal conditions, CreateSocket should succeed
	// We verify the error type when it does fail

	conn, err := CreateSocket()
	if err != nil {
		// If error occurs, verify it's a NetworkError
		var networkErr *errors.NetworkError
		if !goerrors.As(err, &networkErr) {
			t.Errorf("CreateSocket() error is %T, expected NetworkError per FR-013", err)
		}
		return
	}

	// If successful, clean up
	if conn != nil {
		_ = conn.Close() // Test cleanup, error not critical
	}
}

// TestSendQuery_RFC6762_MulticastTransmit validates that SendQuery transmits
// to mDNS multicast group per RFC 6762 §5 (FR-005).
//
// RFC 6762 §5: Queries are sent to multicast group 224.0.0.251:5353
//
// FR-005: System MUST send queries to multicast group 224.0.0.251:5353
func TestSendQuery_RFC6762_MulticastTransmit(t *testing.T) {
	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Build a simple query message
	query := []byte{
		// Header
		0x00, 0x00, // ID
		0x00, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "test.local" A IN
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x01, // QTYPE = A
		0x00, 0x01, // QCLASS = IN
	}

	err = SendQuery(conn, query)
	if err != nil {
		t.Errorf("SendQuery() failed per FR-005: %v", err)
	}
}

// TestSendQuery_NetworkError validates that SendQuery returns NetworkError
// for transmission failures (FR-013).
//
// FR-013: System MUST return NetworkError for transmission failures
func TestSendQuery_NetworkError(t *testing.T) {
	// Create a closed connection to trigger network error
	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed: %v", err)
	}
	_ = conn.Close() // Close immediately to trigger error, ignore close error

	query := []byte{0x00, 0x00, 0x00, 0x00}

	err = SendQuery(conn, query)
	if err == nil {
		t.Errorf("SendQuery() on closed connection expected error per FR-013, got nil")
		return
	}

	// Verify it's a NetworkError
	var networkErr *errors.NetworkError
	if !goerrors.As(err, &networkErr) {
		t.Errorf("SendQuery() error is %T, expected NetworkError per FR-013", err)
	}
}

// TestReceiveResponse_RFC6762_Timeout validates that ReceiveResponse respects
// timeout per FR-006.
//
// FR-006: System MUST receive responses with configurable timeout
func TestReceiveResponse_RFC6762_Timeout(t *testing.T) {
	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Set a short timeout (100ms)
	timeout := 100 * time.Millisecond

	// Attempt to receive with no data sent (should timeout, unless there's mDNS traffic)
	start := time.Now()
	response, err := ReceiveResponse(conn, timeout)
	elapsed := time.Since(start)

	// Case 1: We received actual mDNS traffic (valid - socket is working)
	if err == nil && len(response) > 0 {
		t.Logf("ReceiveResponse() got real mDNS traffic (%d bytes) - socket is working correctly", len(response))
		return
	}

	// Case 2: Timeout occurred (expected in isolated environment)
	if err != nil {
		// Verify the timeout was respected (should be ~100ms, allow 50ms variance)
		if elapsed < timeout || elapsed > timeout+50*time.Millisecond {
			t.Errorf("ReceiveResponse() timeout took %v, expected ~%v per FR-006", elapsed, timeout)
		}

		// Verify it's a NetworkError
		var networkErr *errors.NetworkError
		if !goerrors.As(err, &networkErr) {
			t.Errorf("ReceiveResponse() timeout error is %T, expected NetworkError per FR-013", err)
		}
		return
	}

	// Case 3: No error but also no data (unexpected)
	t.Errorf("ReceiveResponse() returned no error but also no data")
}

// TestReceiveResponse_ValidMessage validates that ReceiveResponse returns
// complete DNS messages (FR-006).
//
// FR-006: System MUST receive responses with configurable timeout
func TestReceiveResponse_ValidMessage(t *testing.T) {
	// Note: This test requires an actual mDNS responder on the network
	// In unit tests, we verify that ReceiveResponse can read data
	// Integration tests will validate end-to-end message exchange

	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Send a query to potentially trigger a response
	query := []byte{
		// Header
		0x00, 0x00, // ID
		0x00, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "_services._dns-sd._udp.local" PTR IN
		0x09, '_', 's', 'e', 'r', 'v', 'i', 'c', 'e', 's',
		0x07, '_', 'd', 'n', 's', '-', 's', 'd',
		0x04, '_', 'u', 'd', 'p',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x0C, // QTYPE = PTR
		0x00, 0x01, // QCLASS = IN
	}

	err = SendQuery(conn, query)
	if err != nil {
		t.Logf("SendQuery() failed: %v (expected in isolated test environment)", err)
	}

	// Try to receive with a short timeout (1 second)
	timeout := 1 * time.Second
	response, err := ReceiveResponse(conn, timeout)

	// In isolated test environments, timeout is expected
	if err != nil {
		t.Logf("ReceiveResponse() timed out: %v (expected in isolated test environment)", err)
		return
	}

	// If we got a response, verify it's at least 12 bytes (minimum DNS header)
	if len(response) < 12 {
		t.Errorf("ReceiveResponse() returned %d bytes, expected at least 12 bytes per FR-006", len(response))
	}
}

// TestCloseSocket_Cleanup validates that CloseSocket properly releases
// resources per FR-017.
//
// FR-017: System MUST close socket after query completion
func TestCloseSocket_Cleanup(t *testing.T) {
	conn, err := CreateSocket()
	if err != nil {
		t.Fatalf("CreateSocket() failed: %v", err)
	}

	err = CloseSocket(conn)
	if err != nil {
		t.Errorf("CloseSocket() failed per FR-017: %v", err)
	}

	// Verify connection is closed by attempting to send (should fail)
	query := []byte{0x00, 0x00}
	err = SendQuery(conn, query)
	if err == nil {
		t.Errorf("SendQuery() on closed connection should fail, got nil")
	}
}

// TestCloseSocket_NilConnection validates that CloseSocket handles nil
// connections gracefully (defensive programming).
func TestCloseSocket_NilConnection(t *testing.T) {
	err := CloseSocket(nil)
	if err != nil {
		t.Errorf("CloseSocket(nil) should handle nil gracefully, got error: %v", err)
	}
}
