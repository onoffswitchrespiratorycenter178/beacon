//go:build darwin

package transport

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// setSocketOptions configures platform-specific socket options for macOS.
// Sets SO_REUSEADDR and SO_REUSEPORT to enable coexistence with Bonjour (mDNSResponder).
//
// Per F-9 REQ-F9-2: SO_REUSEPORT required for multi-daemon coexistence.
// Per research.md: macOS supports SO_REUSEPORT on all versions (BSD semantics).
func setSocketOptions(fd uintptr) error {
	// SO_REUSEADDR: Allow binding to address already in use (BSD standard)
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	// SO_REUSEPORT: Allow multiple sockets to bind to same port (BSD, always available)
	// macOS has native support - no kernel version check needed
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return fmt.Errorf("failed to set SO_REUSEPORT: %w", err)
	}

	return nil
}

// getKernelVersion returns empty string on macOS (not applicable).
// macOS uses Darwin kernel versioning which doesn't map cleanly to SO_REUSEPORT support.
// All macOS versions support SO_REUSEPORT, so version check unnecessary.
func getKernelVersion() string {
	return "" // Not applicable on macOS
}

// Control function for net.ListenConfig on macOS.
// This is called by UDPv4Transport during socket creation.
func platformControl(network, address string, c syscall.RawConn) error {
	var sockoptErr error
	err := c.Control(func(fd uintptr) {
		sockoptErr = setSocketOptions(fd)
	})
	if err != nil {
		return fmt.Errorf("raw conn control failed: %w", err)
	}
	return sockoptErr
}

// PlatformControl returns the platform-specific control function for net.ListenConfig.
// This is the public API for other packages to use socket options.
func PlatformControl(network, address string, c syscall.RawConn) error {
	return platformControl(network, address, c)
}
