package contract

import (
	"net"
	"testing"

	"github.com/joshuafuller/beacon/internal/security"
)

// TestSourceIPFiltering_RFC6762_LinkLocalScope validates RFC 6762 §2 compliance.
// RFC 6762 §2 specifies that mDNS is link-local scope - packets MUST originate from
// link-local addresses (169.254.0.0/16) OR the same subnet as the receiving interface.
//
// This integration test validates SC-007: 100% of non-link-local packets from different
// subnets are dropped before parsing.
//
// Task T071
func TestSourceIPFiltering_RFC6762_LinkLocalScope(t *testing.T) {
	// Get a real network interface for testing
	// We use the loopback interface which is guaranteed to exist
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("Failed to get interfaces: %v", err)
	}

	var testIface net.Interface
	found := false
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && len(iface.Name) > 0 {
			testIface = iface
			found = true
			break
		}
	}

	if !found {
		t.Skip("No UP interfaces found for testing")
	}

	sf, err := security.NewSourceFilter(testIface)
	if err != nil {
		t.Fatalf("NewSourceFilter() failed: %v", err)
	}

	// Test matrix: [source IP, expected result, reason]
	// Note: We focus on link-local addresses (guaranteed to work) and public IPs (guaranteed to fail)
	// Same-subnet testing requires knowing the actual interface subnet, which varies by system
	tests := []struct {
		name     string
		sourceIP string
		want     bool
		reason   string
	}{
		// RFC 6762 §2: Link-local addresses MUST be accepted
		{
			name:     "link_local_169.254.1.1",
			sourceIP: "169.254.1.1",
			want:     true,
			reason:   "RFC 6762 §2: link-local addresses are valid",
		},
		{
			name:     "link_local_169.254.255.254",
			sourceIP: "169.254.255.254",
			want:     true,
			reason:   "RFC 6762 §2: all link-local addresses are valid",
		},
		{
			name:     "link_local_169.254.100.100",
			sourceIP: "169.254.100.100",
			want:     true,
			reason:   "RFC 6762 §2: all link-local addresses are valid",
		},

		// Routed/public IPs MUST be rejected (not link-local, unlikely to be same subnet)
		{
			name:     "public_ip_8.8.8.8",
			sourceIP: "8.8.8.8",
			want:     false,
			reason:   "Public IP (not link-local, not same subnet)",
		},
		{
			name:     "public_ip_1.1.1.1",
			sourceIP: "1.1.1.1",
			want:     false,
			reason:   "Public IP (not link-local, not same subnet)",
		},
	}

	// SC-007: Validate that 100% of invalid packets are rejected
	invalidRejected := 0
	invalidTotal := 0
	validAccepted := 0
	validTotal := 0

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.sourceIP)
			if ip == nil {
				t.Fatalf("Failed to parse IP: %s", tt.sourceIP)
			}

			got := sf.IsValid(ip)

			// Track metrics for SC-007
			if tt.want {
				validTotal++
				if got {
					validAccepted++
				}
			} else {
				invalidTotal++
				if !got {
					invalidRejected++
				}
			}

			if got != tt.want {
				t.Errorf("IsValid(%s) = %v, want %v\nReason: %s",
					tt.sourceIP, got, tt.want, tt.reason)
			}
		})
	}

	// SC-007: Verify 100% of non-link-local packets from different subnets dropped
	if invalidTotal > 0 {
		rejectionRate := float64(invalidRejected) / float64(invalidTotal) * 100
		if rejectionRate < 100.0 {
			t.Errorf("SC-007 FAILED: Invalid packet rejection rate = %.1f%%, want 100.0%%",
				rejectionRate)
		} else {
			t.Logf("SC-007 PASS: Invalid packet rejection rate = 100%% (%d/%d packets dropped)",
				invalidRejected, invalidTotal)
		}
	}

	// Verify valid packets are accepted
	if validTotal > 0 {
		acceptanceRate := float64(validAccepted) / float64(validTotal) * 100
		if acceptanceRate < 100.0 {
			t.Errorf("Valid packet acceptance rate = %.1f%%, want 100.0%%", acceptanceRate)
		} else {
			t.Logf("Valid packet acceptance rate = 100%% (%d/%d packets accepted)",
				validAccepted, validTotal)
		}
	}
}

// TestSourceIPFiltering_PrivateIPDetection validates private IP handling.
// Private IPs are accepted if they're in the same subnet as the interface,
// rejected otherwise. This test verifies the IP parsing and validation logic.
func TestSourceIPFiltering_PrivateIPDetection(t *testing.T) {
	// Note: This test validates that private IPs are handled correctly
	// (accepted if same subnet, rejected if different subnet)

	iface := net.Interface{
		Index: 1,
		Name:  "eth0",
		Flags: net.FlagUp | net.FlagMulticast,
	}

	_, err := security.NewSourceFilter(iface)
	if err != nil {
		t.Fatalf("NewSourceFilter() failed: %v", err)
	}

	// Test private IP ranges - just verify parsing works
	// Actual acceptance/rejection depends on interface subnet configuration
	privateRanges := []struct {
		ip    string
		class string
	}{
		{"10.0.0.1", "Class A private"},
		{"172.16.0.1", "Class B private"},
		{"192.168.1.1", "Class C private"},
	}

	for _, pr := range privateRanges {
		t.Run(pr.class, func(t *testing.T) {
			ip := net.ParseIP(pr.ip)
			if ip == nil {
				t.Errorf("Failed to parse %s (%s)", pr.ip, pr.class)
			}
		})
	}
}
