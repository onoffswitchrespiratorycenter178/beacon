//go:build windows

package transport

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// setSocketOptions configures platform-specific socket options for Windows.
// Sets SO_REUSEADDR only (Windows does NOT support SO_REUSEPORT).
//
// Per F-9 REQ-F9-3: Windows SO_REUSEADDR has different semantics than POSIX.
// Per research.md: Windows SO_REUSEADDR allows multiple binds to same port (similar to BSD SO_REUSEPORT).
//
// WARNING: Windows SO_REUSEADDR behavior differs from POSIX:
// - POSIX SO_REUSEADDR: Allows binding to TIME_WAIT sockets
// - Windows SO_REUSEADDR: Allows multiple processes to bind to same port (like POSIX SO_REUSEPORT)
//
// This means Beacon CAN coexist with other mDNS applications on Windows,
// but the semantics are slightly different from Linux/macOS.
func setSocketOptions(fd uintptr) error {
	// SO_REUSEADDR: Windows-specific behavior (allows port sharing)
	// This is the ONLY socket option we can use on Windows for coexistence
	if err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("failed to set SO_REUSEADDR: %w", err)
	}

	// SO_REUSEPORT does not exist on Windows - do not attempt to set it
	// The golang.org/x/sys/windows package doesn't even define SO_REUSEPORT constant

	return nil
}

// getKernelVersion returns empty string on Windows (not applicable).
// Windows doesn't have a "kernel version" in the same sense as Linux.
// Socket option support is Windows version-dependent, but SO_REUSEADDR
// is supported on all modern Windows versions (XP+).
func getKernelVersion() string {
	return "" // Not applicable on Windows
}

// Control function for net.ListenConfig on Windows.
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
