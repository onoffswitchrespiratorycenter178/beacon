package network

import (
	"net"
	"testing"
)

// TestDefaultInterfaces_ExcludesVPN verifies VPN interfaces are excluded by default.
// Per F-10 REQ-F10-6: Prevent mDNS query leakage to VPN provider.
func TestDefaultInterfaces_ExcludesVPN(t *testing.T) {
	// This test verifies that DefaultInterfaces() excludes VPN interfaces
	// matching the 6 primary patterns from research.md:
	// utun*, tun*, ppp*, wg*, tailscale*, wireguard*

	// Get actual system interfaces
	ifaces, err := DefaultInterfaces()
	if err != nil {
		t.Fatalf("DefaultInterfaces() returned error: %v", err)
	}

	// Verify NO VPN interfaces in result
	vpnPatterns := []string{"utun", "tun", "ppp", "wg", "tailscale", "wireguard"}
	for _, iface := range ifaces {
		for _, pattern := range vpnPatterns {
			if len(iface.Name) >= len(pattern) && iface.Name[:len(pattern)] == pattern {
				t.Errorf("DefaultInterfaces() included VPN interface %q (pattern: %s)", iface.Name, pattern)
			}
		}
	}
}

// TestDefaultInterfaces_ExcludesDocker verifies Docker interfaces are excluded by default.
// Per F-10 REQ-F10-7: Avoid wasting resources on isolated container networks.
func TestDefaultInterfaces_ExcludesDocker(t *testing.T) {
	// This test verifies that DefaultInterfaces() excludes Docker interfaces
	// matching the 3 primary patterns from research.md:
	// docker0 (exact), veth*, br-*

	// Get actual system interfaces
	ifaces, err := DefaultInterfaces()
	if err != nil {
		t.Fatalf("DefaultInterfaces() returned error: %v", err)
	}

	// Verify NO Docker interfaces in result
	for _, iface := range ifaces {
		// Check exact match: docker0
		if iface.Name == "docker0" {
			t.Errorf("DefaultInterfaces() included Docker interface %q", iface.Name)
		}

		// Check prefix: veth*
		if len(iface.Name) >= 4 && iface.Name[:4] == "veth" {
			t.Errorf("DefaultInterfaces() included Docker veth interface %q", iface.Name)
		}

		// Check prefix: br-*
		if len(iface.Name) >= 3 && iface.Name[:3] == "br-" {
			t.Errorf("DefaultInterfaces() included Docker bridge interface %q", iface.Name)
		}
	}
}

// TestDefaultInterfaces_ExcludesLoopback verifies loopback is excluded.
// Per F-10 REQ-F10-3: mDNS is for network discovery, not localhost.
func TestDefaultInterfaces_ExcludesLoopback(t *testing.T) {
	// This test verifies that DefaultInterfaces() excludes loopback interfaces
	// (net.FlagLoopback set)

	// Get actual system interfaces
	ifaces, err := DefaultInterfaces()
	if err != nil {
		t.Fatalf("DefaultInterfaces() returned error: %v", err)
	}

	// Verify NO loopback interfaces in result
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			t.Errorf("DefaultInterfaces() included loopback interface %q", iface.Name)
		}
	}
}

// TestDefaultInterfaces_RequiresUpAndMulticast verifies only UP+MULTICAST interfaces are included.
// Per F-10 REQ-F10-4, REQ-F10-5: Interface must be operational and support multicast.
func TestDefaultInterfaces_RequiresUpAndMulticast(t *testing.T) {
	// This test verifies that DefaultInterfaces() includes ONLY interfaces
	// with both FlagUp and FlagMulticast set

	// Get actual system interfaces
	ifaces, err := DefaultInterfaces()
	if err != nil {
		t.Fatalf("DefaultInterfaces() returned error: %v", err)
	}

	// Verify ALL interfaces in result have UP+MULTICAST flags
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			t.Errorf("DefaultInterfaces() included DOWN interface %q (flags: %v)", iface.Name, iface.Flags)
		}

		if iface.Flags&net.FlagMulticast == 0 {
			t.Errorf("DefaultInterfaces() included non-MULTICAST interface %q (flags: %v)", iface.Name, iface.Flags)
		}
	}
}

// TestIsVPN verifies VPN pattern detection.
// Patterns: utun*, tun*, ppp*, wg*, tailscale*, wireguard*
// Per FR-017: VPN interface exclusion (6 patterns, 95%+ coverage)
func TestIsVPN(t *testing.T) {
	tests := []struct {
		name      string
		ifaceName string
		want      bool
	}{
		{"macOS OpenVPN", "utun0", true},
		{"Linux OpenVPN", "tun0", true},
		{"PPTP", "ppp0", true},
		{"WireGuard", "wg0", true},
		{"Tailscale", "tailscale0", true},
		{"Regular Ethernet", "eth0", false},
		{"WiFi", "wlan0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVPN(tt.ifaceName)
			if got != tt.want {
				t.Errorf("isVPN(%q) = %v, want %v", tt.ifaceName, got, tt.want)
			}
		})
	}
}

// TestIsDocker verifies Docker pattern detection.
// Patterns: docker0, veth*, br-*
// Per FR-018: Docker interface exclusion (3 patterns, 100% coverage)
func TestIsDocker(t *testing.T) {
	tests := []struct {
		name      string
		ifaceName string
		want      bool
	}{
		{"Docker bridge", "docker0", true},
		{"Virtual ethernet", "veth1a2b3c4", true},
		{"Custom bridge", "br-abc123", true},
		{"Regular Ethernet", "eth0", false},
		{"WiFi", "wlan0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDocker(tt.ifaceName)
			if got != tt.want {
				t.Errorf("isDocker(%q) = %v, want %v", tt.ifaceName, got, tt.want)
			}
		})
	}
}
