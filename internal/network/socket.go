// Package network implements UDP multicast socket operations for mDNS.
package network

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/ipv4"

	"github.com/joshuafuller/beacon/internal/errors"
	"github.com/joshuafuller/beacon/internal/protocol"
	"github.com/joshuafuller/beacon/internal/transport"
)

// CreateSocket creates a UDP multicast socket bound to mDNS port 5353
// per RFC 6762 §5 (FR-004).
//
// M1.1 Update (FR-001 through FR-010):
// - Uses net.ListenConfig with platform-specific socket options (SO_REUSEADDR + SO_REUSEPORT)
// - Enables coexistence with Avahi/Bonjour/systemd-resolved
// - Uses golang.org/x/net/ipv4 for proper multicast group membership
// - Sets TTL=255 per RFC 6762 §11
// - Enables multicast loopback for local testing
//
// RFC 6762 §5: mDNS uses UDP port 5353 and multicast address 224.0.0.251
// RFC 6762 §11: Multicast DNS messages MUST be sent with TTL=255
//
// FR-001: System MUST use net.ListenConfig with Control function
// FR-002: System MUST set SO_REUSEADDR on all platforms
// FR-003: System MUST set SO_REUSEPORT on Linux (kernel >= 3.9) and macOS
// FR-004: System MUST replace ListenMulticastUDP with ListenConfig pattern
// FR-005: System MUST use golang.org/x/net/ipv4 to join multicast group 224.0.0.251
// FR-006: System MUST set multicast TTL to 255
// FR-007: System MUST enable multicast loopback
//
// Returns:
//   - conn: UDP connection bound to mDNS port with multicast configured
//   - error: NetworkError if socket creation fails
func CreateSocket() (net.PacketConn, error) {
	ctx := context.Background()

	// Step 1: Create ListenConfig with platform-specific socket options
	// This sets SO_REUSEADDR (all platforms) and SO_REUSEPORT (Linux/macOS)
	// BEFORE binding to enable coexistence with Avahi/Bonjour
	lc := net.ListenConfig{
		Control: transport.PlatformControl, // Platform-specific socket options
	}

	// Step 2: Listen on port 5353 (bind to 0.0.0.0:5353)
	// Note: We bind to 0.0.0.0, NOT the multicast address
	// (ListenMulticastUDP had bugs, see Go issues #73484, #34728)
	conn, err := lc.ListenPacket(ctx, "udp4", fmt.Sprintf("0.0.0.0:%d", protocol.Port))
	if err != nil {
		return nil, &errors.NetworkError{
			Operation: "create socket",
			Err:       err,
			Details:   fmt.Sprintf("failed to bind to port %d (is Avahi/Bonjour running without SO_REUSEPORT?)", protocol.Port),
		}
	}

	// Step 3: Wrap in ipv4.PacketConn for multicast control
	p := ipv4.NewPacketConn(conn)

	// Step 4: Join multicast group 224.0.0.251 on all interfaces
	// Per RFC 6762 §5: Must join group to receive multicast packets
	multicastGroup := net.IPv4(224, 0, 0, 251)
	ifaces, err := net.Interfaces()
	if err != nil {
		_ = conn.Close() // Ignore error, already returning primary error
		return nil, &errors.NetworkError{
			Operation: "enumerate interfaces",
			Err:       err,
			Details:   "failed to get network interfaces for multicast join",
		}
	}

	// Join multicast group on all UP+MULTICAST interfaces
	// M1.1: This joins on ALL interfaces; M1.2 will add filtering
	joinedCount := 0
	for _, iface := range ifaces {
		// Skip down interfaces and non-multicast interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		// G601: Create a copy of iface to avoid implicit memory aliasing in loop
		// The loop variable iface is reused, so we must copy before taking its address
		ifaceCopy := iface
		// Join multicast group on this interface
		if err := p.JoinGroup(&ifaceCopy, &net.UDPAddr{IP: multicastGroup}); err != nil {
			// Log but don't fail - interface might not support multicast
			// In production, we'd use a logger here
			continue
		}
		joinedCount++
	}

	if joinedCount == 0 {
		_ = conn.Close() // Ignore error, already returning primary error
		return nil, &errors.NetworkError{
			Operation: "join multicast group",
			Err:       fmt.Errorf("no interfaces available"),
			Details:   "failed to join 224.0.0.251 on any interface",
		}
	}

	// Step 5: Set multicast TTL to 255 per RFC 6762 §11
	if err := p.SetMulticastTTL(255); err != nil {
		_ = conn.Close() // Ignore error, already returning primary error
		return nil, &errors.NetworkError{
			Operation: "set multicast TTL",
			Err:       err,
			Details:   "failed to set TTL=255",
		}
	}

	// Step 6: Enable multicast loopback (receive own packets)
	// Required for some mDNS behavior and local testing
	if err := p.SetMulticastLoopback(true); err != nil {
		_ = conn.Close() // Ignore error, already returning primary error
		return nil, &errors.NetworkError{
			Operation: "set multicast loopback",
			Err:       err,
			Details:   "failed to enable loopback",
		}
	}

	// Step 7: Configure socket buffer
	if udpConn, ok := conn.(*net.UDPConn); ok {
		if err := udpConn.SetReadBuffer(65536); err != nil {
			_ = conn.Close() // Ignore error, already returning primary error
			return nil, &errors.NetworkError{
				Operation: "configure socket",
				Err:       err,
				Details:   "failed to set read buffer size",
			}
		}
	}

	return conn, nil
}

// SendQuery sends an mDNS query to the multicast group per RFC 6762 §5 (FR-005).
//
// RFC 6762 §5: Queries are sent to 224.0.0.251:5353
//
// FR-005: System MUST send queries to multicast group 224.0.0.251:5353
// FR-013: System MUST return NetworkError for transmission failures
//
// Parameters:
//   - conn: UDP connection created by CreateSocket
//   - query: DNS query message in wire format
//
// Returns:
//   - error: NetworkError if transmission fails
func SendQuery(conn net.PacketConn, query []byte) error {
	// Resolve mDNS multicast destination
	addr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(protocol.MulticastAddrIPv4, strconv.Itoa(protocol.Port)))
	if err != nil {
		return &errors.NetworkError{
			Operation: "resolve destination",
			Err:       err,
			Details:   "failed to resolve multicast destination",
		}
	}

	// Send query to multicast group
	n, err := conn.WriteTo(query, addr)
	if err != nil {
		return &errors.NetworkError{
			Operation: "send query",
			Err:       err,
			Details:   fmt.Sprintf("failed to send %d bytes to %s", len(query), addr),
		}
	}

	// Verify full message was sent
	if n != len(query) {
		return &errors.NetworkError{
			Operation: "send query",
			Err:       fmt.Errorf("partial write: %d/%d bytes", n, len(query)),
			Details:   "incomplete transmission",
		}
	}

	return nil
}

// ReceiveResponse receives an mDNS response with timeout per FR-006.
//
// FR-006: System MUST receive responses with configurable timeout
// FR-013: System MUST return NetworkError for timeout or receive errors
//
// Parameters:
//   - conn: UDP connection created by CreateSocket
//   - timeout: Maximum time to wait for response
//
// Returns:
//   - response: DNS response message in wire format
//   - error: NetworkError if timeout occurs or receive fails
func ReceiveResponse(conn net.PacketConn, timeout time.Duration) ([]byte, error) {
	// Set read deadline
	deadline := time.Now().Add(timeout)
	err := conn.SetReadDeadline(deadline)
	if err != nil {
		return nil, &errors.NetworkError{
			Operation: "set read timeout",
			Err:       err,
			Details:   fmt.Sprintf("failed to set timeout %v", timeout),
		}
	}

	// Allocate buffer for DNS message (max 512 bytes for mDNS per RFC 6762 §17)
	// In practice, mDNS messages can be larger due to additional records
	buffer := make([]byte, 9000) // Support jumbo frames

	// Read response
	n, _, err := conn.ReadFrom(buffer)
	if err != nil {
		// Check if it's a timeout error
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, &errors.NetworkError{
				Operation: "receive response",
				Err:       err,
				Details:   fmt.Sprintf("timeout after %v", timeout),
			}
		}

		return nil, &errors.NetworkError{
			Operation: "receive response",
			Err:       err,
			Details:   "failed to read from socket",
		}
	}

	// Return only the bytes actually read
	return buffer[:n], nil
}

// CloseSocket closes the UDP connection per FR-017.
//
// FR-017: System MUST close socket after query completion
//
// Parameters:
//   - conn: UDP connection to close
//
// Returns:
//   - error: nil (close errors are logged but not returned in M1)
func CloseSocket(conn net.PacketConn) error {
	if conn == nil {
		return nil // Gracefully handle nil connection
	}

	err := conn.Close()
	if err != nil {
		// In M1, we log but don't fail on close errors
		// Production systems might want to return NetworkError here
		return nil
	}

	return nil
}
