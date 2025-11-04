// Package network provides network interface filtering and management.
package network

import (
	"net"
)

// DefaultInterfaces returns network interfaces suitable for mDNS multicast,
// excluding VPN interfaces, Docker interfaces, loopback, and down interfaces.
//
// This function implements the "smart defaults" behavior per F-10 (Interface Management):
// - Excludes VPN interfaces (utun*, tun*, ppp*, wg*, tailscale*, wireguard*)
// - Excludes Docker interfaces (docker0, veth*, br-*)
// - Excludes loopback interfaces
// - Excludes down interfaces
// - Includes only interfaces with multicast support
//
// Users can override this behavior via WithInterfaces() or WithInterfaceFilter()
// functional options.
//
// Implements:
//   - FR-013: System MUST implement DefaultInterfaces() function
//   - FR-014: Include only UP interfaces (net.FlagUp)
//   - FR-015: Include only MULTICAST interfaces (net.FlagMulticast)
//   - FR-016: Exclude loopback interfaces (net.FlagLoopback)
//   - FR-017: Exclude VPN interfaces (6 patterns)
//   - FR-018: Exclude Docker interfaces (3 patterns)
func DefaultInterfaces() ([]net.Interface, error) {
	// Get all system interfaces
	allIfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Filter interfaces based on requirements
	filtered := make([]net.Interface, 0, len(allIfaces))
	for _, iface := range allIfaces {
		// FR-014: Skip DOWN interfaces (must be UP)
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// FR-015: Skip non-MULTICAST interfaces
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}

		// FR-016: Skip LOOPBACK interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// FR-017: Skip VPN interfaces (6 patterns)
		if isVPN(iface.Name) {
			continue
		}

		// FR-018: Skip Docker interfaces (3 patterns)
		if isDocker(iface.Name) {
			continue
		}

		// Interface passed all filters - include it
		filtered = append(filtered, iface)
	}

	return filtered, nil
}

// isVPN returns true if the interface name matches known VPN naming patterns.
// Patterns cover 95%+ of VPN clients (per research.md).
//
// Recognized patterns:
//   - utun*      - macOS system VPNs, Tunnelblick, OpenVPN
//   - tun*       - Linux OpenVPN, generic TUN devices
//   - ppp*       - PPTP, L2TP tunnels
//   - wg*        - WireGuard (standard naming)
//   - tailscale* - Tailscale VPN
//   - wireguard* - WireGuard (alternative naming)
func isVPN(name string) bool {
	vpnPrefixes := []string{"utun", "tun", "ppp", "wg", "tailscale", "wireguard"}
	for _, prefix := range vpnPrefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isDocker returns true if the interface name matches Docker interface patterns.
// Patterns cover 100% of Docker networking (per research.md).
//
// Recognized patterns:
//   - docker0  - Default Docker bridge (exact match)
//   - veth*    - Virtual ethernet pairs (container connections)
//   - br-*     - Custom Docker bridge networks
func isDocker(name string) bool {
	// Exact match: docker0
	if name == "docker0" {
		return true
	}

	// Prefix matches
	dockerPrefixes := []string{"veth", "br-"}
	for _, prefix := range dockerPrefixes {
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
