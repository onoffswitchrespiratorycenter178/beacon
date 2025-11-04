//go:build linux

package transport

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// setSocketOptions configures platform-specific socket options for Linux.
// Sets SO_REUSEADDR and SO_REUSEPORT (if kernel >= 3.9) to enable
// coexistence with Avahi and systemd-resolved on port 5353.
//
// Per F-9 REQ-F9-2: SO_REUSEPORT required for multi-daemon coexistence.
// Per research.md: Linux kernel 3.9+ supports SO_REUSEPORT.
func setSocketOptions(fd uintptr) error {
	// SO_REUSEADDR: Allow binding to address already in use (POSIX standard)
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	// SO_REUSEPORT: Allow multiple sockets to bind to same port (Linux 3.9+)
	// This is THE critical option for Avahi/systemd-resolved coexistence
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		// Fall back gracefully if kernel doesn't support SO_REUSEPORT
		// (Old kernels <3.9 will fail with ENOPROTOOPT)
		if err != unix.ENOPROTOOPT {
			return fmt.Errorf("failed to set SO_REUSEPORT: %w", err)
		}
		// Log warning but continue (SO_REUSEADDR only)
		// Querier initialization will detect kernel version and warn user
	}

	return nil
}

// getKernelVersion returns the Linux kernel version string for logging/debugging.
// Format: "3.10.0-1160.el7.x86_64"
func getKernelVersion() string {
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return "unknown"
	}

	// Convert [65]int8 to string
	release := make([]byte, 0, 65)
	for _, b := range uname.Release {
		if b == 0 {
			break
		}
		release = append(release, byte(b))
	}

	return string(release)
}

// Control function for net.ListenConfig on Linux.
// This is called by UDPv4Transport during socket creation.
func platformControl(_, _ string, c syscall.RawConn) error {
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
