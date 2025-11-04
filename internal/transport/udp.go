package transport

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/joshuafuller/beacon/internal/errors"
	"github.com/joshuafuller/beacon/internal/protocol"
)

// UDPv4Transport implements Transport interface for IPv4 UDP multicast.
//
// This implementation:
// - Migrates logic from internal/network/socket.go (CreateSocket, SendQuery, ReceiveResponse)
// - Adds context support for cancellation and deadlines (F-9 REQ-F9-7)
// - Fixes error propagation in Close() (FR-004)
//
// T020: Migrate internal/network/socket.go CreateSocket logic to make T011 pass
type UDPv4Transport struct {
	conn net.PacketConn
}

// NewUDPv4Transport creates a UDP multicast transport bound to mDNS port 5353.
//
// This migrates CreateSocket() from internal/network/socket.go:24-58.
//
// RFC 6762 §5: mDNS uses UDP port 5353 and multicast address 224.0.0.251
//
// FR-004: System MUST use mDNS port 5353 and multicast address 224.0.0.251
// FR-013: System MUST return NetworkError for socket creation failures
//
// Returns:
//   - *UDPv4Transport: Configured transport ready for Send/Receive
//   - error: NetworkError if socket creation fails
//
// T021: Socket creation, multicast join
func NewUDPv4Transport() (*UDPv4Transport, error) {
	// Resolve mDNS multicast address
	multicastAddr, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(protocol.MulticastAddrIPv4, strconv.Itoa(protocol.Port)))
	if err != nil {
		return nil, &errors.NetworkError{
			Operation: "resolve multicast address",
			Err:       err,
			Details:   fmt.Sprintf("failed to resolve %s:%d", protocol.MulticastAddrIPv4, protocol.Port),
		}
	}

	// Listen on mDNS multicast group
	// This binds to the multicast address and joins the group automatically
	// Connection ownership transferred to UDPv4Transport, closed via t.Close() method
	//
	// NOTE: ListenMulticastUDP is acceptable here for M1 (IPv4-only).
	// F-9 REQ-F9-1 requires platform-specific sockets for M2 (IPv6 + SO_REUSEPORT).
	// This will be replaced during M2 implementation with proper socket creation.
	conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr) // nosemgrep: beacon-socket-close-check, beacon-listen-multicast-udp
	if err != nil {
		return nil, &errors.NetworkError{
			Operation: "create socket",
			Err:       err,
			Details:   fmt.Sprintf("failed to bind to multicast %s:%d", protocol.MulticastAddrIPv4, protocol.Port),
		}
	}

	// Configure socket buffer
	err = conn.SetReadBuffer(65536) // 64KB buffer for DNS messages
	if err != nil {
		_ = conn.Close() // Ignore error, already returning primary error
		return nil, &errors.NetworkError{
			Operation: "configure socket",
			Err:       err,
			Details:   "failed to set read buffer size",
		}
	}

	return &UDPv4Transport{conn: conn}, nil
}

// Send transmits a packet to the specified destination address.
//
// This migrates SendQuery() from internal/network/socket.go:73-104.
//
// RFC 6762 §5: Queries are sent to 224.0.0.251:5353
//
// FR-005: System MUST send queries to multicast group 224.0.0.251:5353
// FR-013: System MUST return NetworkError for transmission failures
//
// T022: Migrate internal/network SendQuery logic, make T013 pass
func (t *UDPv4Transport) Send(ctx context.Context, packet []byte, dest net.Addr) error {
	// Check context cancellation before sending
	select {
	case <-ctx.Done():
		return &errors.NetworkError{
			Operation: "send query",
			Err:       ctx.Err(),
			Details:   "context canceled before send",
		}
	default:
	}

	// Send query to destination
	n, err := t.conn.WriteTo(packet, dest)
	if err != nil {
		return &errors.NetworkError{
			Operation: "send query",
			Err:       err,
			Details:   fmt.Sprintf("failed to send %d bytes to %s", len(packet), dest),
		}
	}

	// Verify full message was sent
	if n != len(packet) {
		return &errors.NetworkError{
			Operation: "send query",
			Err:       fmt.Errorf("partial write: %d/%d bytes", n, len(packet)),
			Details:   "incomplete transmission",
		}
	}

	return nil
}

// Receive waits for an incoming packet, respecting context cancellation/deadline.
//
// This migrates ReceiveResponse() from internal/network/socket.go:118-155
// with context support added for F-9 REQ-F9-7.
//
// FR-006: System MUST receive responses with configurable timeout
// FR-013: System MUST return NetworkError for timeout or receive errors
// F-9 REQ-F9-7: Context propagation (M1.1 alignment)
//
// T023: Migrate internal/network ReceiveResponse, add ctx.Done() checking to make T014-T015 pass
func (t *UDPv4Transport) Receive(ctx context.Context) ([]byte, net.Addr, error) {
	// Check context cancellation before receive
	select {
	case <-ctx.Done():
		return nil, nil, &errors.NetworkError{
			Operation: "receive response",
			Err:       ctx.Err(),
			Details:   "context canceled before receive",
		}
	default:
	}

	// Propagate context deadline to socket (F-9 REQ-F9-7)
	if deadline, ok := ctx.Deadline(); ok {
		err := t.conn.SetReadDeadline(deadline)
		if err != nil {
			return nil, nil, &errors.NetworkError{
				Operation: "set read timeout",
				Err:       err,
				Details:   fmt.Sprintf("failed to set deadline %v", deadline),
			}
		}
	}

	// T053: Get buffer from pool (FR-003 buffer pooling optimization)
	// This eliminates hot path allocations (9KB/receive → near-zero after warmup)
	bufPtr := GetBuffer()
	defer PutBuffer(bufPtr) // T053: Return buffer to pool on function exit

	buffer := *bufPtr

	// Read response
	n, srcAddr, err := t.conn.ReadFrom(buffer)
	if err != nil {
		// Check if it's a timeout error
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil, &errors.NetworkError{
				Operation: "receive response",
				Err:       err,
				Details:   "timeout",
			}
		}

		return nil, nil, &errors.NetworkError{
			Operation: "receive response",
			Err:       err,
			Details:   "failed to read from socket",
		}
	}

	// T054: Return copy to caller (pool owns buffer, caller owns result)
	// This ensures caller can use result after buffer is returned to pool
	result := make([]byte, n)
	copy(result, buffer[:n])
	return result, srcAddr, nil
}

// Close releases network resources.
//
// This migrates CloseSocket() from internal/network/socket.go:166-179
// with FIX for FR-004: propagate errors instead of swallowing them.
//
// FR-017: System MUST close socket after query completion
// FR-004 FIX: Return errors to caller (was swallowing errors at line 172-176)
//
// T024: Migrate internal/network CloseSocket, FIX error propagation to make T016 pass (FR-004)
func (t *UDPv4Transport) Close() error {
	if t.conn == nil {
		return nil // Gracefully handle nil connection
	}

	err := t.conn.Close()
	if err != nil {
		// FR-004 FIX: Propagate error to caller (don't swallow it)
		return &errors.NetworkError{
			Operation: "close socket",
			Err:       err,
			Details:   "failed to close UDP connection",
		}
	}

	return nil
}
