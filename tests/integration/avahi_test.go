package integration

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/responder"
)

// TestAvahiBrowse_ServiceDiscoverable_RED tests end-to-end service discovery with Avahi.
//
// TDD Phase: RED - This test will FAIL until full User Story 1 is implemented
//
// Success Criteria (SC-001): Register a service and verify it appears in Avahi Browse within 2 seconds
//
// RFC 6762 §8: Full probing + announcing cycle (~1.75s)
// Plus network propagation: Total should be <2 seconds
//
// Prerequisites:
//   - Avahi daemon running on test system
//   - avahi-browse command available
//
// T030: Integration test for service discovery
func TestAvahiBrowse_ServiceDiscoverable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if avahi-browse is available
	if _, err := exec.LookPath("avahi-browse"); err != nil {
		t.Skip("avahi-browse not found, skipping integration test")
	}

	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register a test service
	service := &responder.Service{
		InstanceName: "Beacon Integration Test",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
		TXTRecords: map[string]string{
			"version": "1.0",
			"test":    "integration",
		},
	}

	start := time.Now()
	err = r.Register(service)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	registerDuration := time.Since(start)

	t.Logf("Service registered in %v", registerDuration)

	// Give additional time for network propagation (100ms)
	time.Sleep(100 * time.Millisecond)

	// Browse for the service using avahi-browse
	// Command: avahi-browse -t _http._tcp --resolve --parsable
	cmd := exec.CommandContext(ctx, "avahi-browse", "-t", "_http._tcp", "--resolve", "--parsable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("avahi-browse error = %v, output: %s", err, output)
	}

	outputStr := string(output)
	t.Logf("avahi-browse output:\n%s", outputStr)

	// Parse avahi-browse output for our service
	// Format: =;interface;protocol;name;type;domain;host;address;port;txt...
	lines := strings.Split(outputStr, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, service.InstanceName) {
			found = true
			t.Logf("Found service in avahi-browse: %s", line)

			// Verify service details in the line
			fields := strings.Split(line, ";")
			if len(fields) < 10 {
				t.Errorf("avahi-browse line has %d fields, want ≥10", len(fields))
				continue
			}

			// Field 3: Service instance name
			if !strings.Contains(fields[3], service.InstanceName) {
				t.Errorf("Service name = %q, want %q", fields[3], service.InstanceName)
			}

			// Field 4: Service type
			if !strings.Contains(fields[4], "_http._tcp") {
				t.Errorf("Service type = %q, want _http._tcp", fields[4])
			}

			// Field 8: Port
			if !strings.Contains(fields[8], "8080") {
				t.Errorf("Service port = %q, want 8080", fields[8])
			}

			break
		}
	}

	if !found {
		t.Errorf("Service %q not found in avahi-browse output after %v", service.InstanceName, time.Since(start))
	}

	// Verify total time <2 seconds per SC-001
	totalTime := time.Since(start)
	if totalTime > 2*time.Second {
		t.Errorf("Service discovery took %v, want <2s (SC-001)", totalTime)
	} else {
		t.Logf("Service discoverable in %v (SC-001 satisfied)", totalTime)
	}
}

// TestAvahiBrowse_MultipleServices_RED tests registering multiple services concurrently.
//
// TDD Phase: RED
//
// FR-030: System MUST support concurrent service registrations
// T030: Test multiple concurrent services
func TestAvahiBrowse_MultipleServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if avahi-browse is available
	if _, err := exec.LookPath("avahi-browse"); err != nil {
		t.Skip("avahi-browse not found, skipping integration test")
	}

	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register multiple services
	services := []*responder.Service{
		{
			InstanceName: "Beacon HTTP Service",
			ServiceType:  "_http._tcp.local",
			Port:         8080,
		},
		{
			InstanceName: "Beacon Printer",
			ServiceType:  "_printer._tcp.local",
			Port:         9100,
		},
		{
			InstanceName: "Beacon SSH",
			ServiceType:  "_ssh._tcp.local",
			Port:         22,
		},
	}

	// Register all services
	for _, svc := range services {
		err := r.Register(svc)
		if err != nil {
			t.Fatalf("Register(%q) error = %v, want nil", svc.InstanceName, err)
		}
	}

	// Give time for announcements to propagate
	time.Sleep(200 * time.Millisecond)

	// Browse for all registered services
	for _, svc := range services {
		// Extract service type without .local suffix for avahi-browse
		serviceType := strings.TrimSuffix(svc.ServiceType, ".local")

		// G204: test code with controlled input (avahi-browse command and arguments are hardcoded)
		cmd := exec.CommandContext(ctx, "avahi-browse", "-t", serviceType, "--resolve", "--parsable") //nolint:gosec // G204: test code with controlled input
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("avahi-browse for %s error = %v, output: %s", serviceType, err, output)
			continue
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, svc.InstanceName) {
			t.Errorf("Service %q not found in avahi-browse output", svc.InstanceName)
		} else {
			t.Logf("Service %q is discoverable", svc.InstanceName)
		}
	}
}

// TestAvahiBrowse_Unregister_RED tests service unregistration with goodbye packets.
//
// TDD Phase: RED
//
// RFC 6762 §10.1: Goodbye packets (TTL=0) remove service from browsers
// FR-014: System MUST send goodbye packets on unregistration
// T030: Test unregistration removes service from Avahi
func TestAvahiBrowse_Unregister(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if avahi-browse is available
	if _, err := exec.LookPath("avahi-browse"); err != nil {
		t.Skip("avahi-browse not found, skipping integration test")
	}

	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	service := &responder.Service{
		InstanceName: "Beacon Unregister Test",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	// Register service
	err = r.Register(service)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Verify service is discoverable
	time.Sleep(200 * time.Millisecond)
	cmd := exec.CommandContext(ctx, "avahi-browse", "-t", "_http._tcp", "--resolve", "--parsable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("avahi-browse error = %v", err)
	}

	if !strings.Contains(string(output), service.InstanceName) {
		t.Fatalf("Service %q not found before unregister", service.InstanceName)
	}

	t.Logf("Service %q is discoverable before unregister", service.InstanceName)

	// Unregister service
	err = r.Unregister(service.InstanceName)
	if err != nil {
		t.Fatalf("Unregister() error = %v, want nil", err)
	}

	// Wait for goodbye packets to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify service is no longer discoverable
	cmd = exec.CommandContext(ctx, "avahi-browse", "-t", "_http._tcp", "--resolve", "--parsable")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("avahi-browse error = %v", err)
	}

	// Service should NOT appear in output after unregister + goodbye
	if strings.Contains(string(output), service.InstanceName) {
		t.Errorf("Service %q still discoverable after Unregister (goodbye packets not sent?)", service.InstanceName)
	} else {
		t.Logf("Service %q successfully removed from Avahi after Unregister", service.InstanceName)
	}
}

// TestAvahiBrowse_TXTRecords_RED tests TXT record propagation.
//
// TDD Phase: RED
//
// RFC 6763 §6: TXT records carry service metadata
// T030: Verify TXT records appear in Avahi
func TestAvahiBrowse_TXTRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if avahi-browse is available
	if _, err := exec.LookPath("avahi-browse"); err != nil {
		t.Skip("avahi-browse not found, skipping integration test")
	}

	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	service := &responder.Service{
		InstanceName: "Beacon TXT Test",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
		TXTRecords: map[string]string{
			"version": "2.0",
			"path":    "/api/v2",
			"secure":  "true",
		},
	}

	err = r.Register(service)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Browse with --resolve to get TXT records
	cmd := exec.CommandContext(ctx, "avahi-browse", "-t", "_http._tcp", "--resolve", "--parsable")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("avahi-browse error = %v", err)
	}

	outputStr := string(output)
	t.Logf("avahi-browse output:\n%s", outputStr)

	// Find our service line
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, service.InstanceName) {
			// TXT records appear in field 9+ (semicolon-separated)
			// Format: =;interface;protocol;name;type;domain;host;address;port;txt...

			// Check if TXT record key-value pairs appear in the line
			for key, value := range service.TXTRecords {
				expectedTXT := key + "=" + value
				if !strings.Contains(line, expectedTXT) {
					t.Errorf("TXT record %q not found in avahi-browse output", expectedTXT)
				} else {
					t.Logf("TXT record %q found in avahi-browse", expectedTXT)
				}
			}
			return
		}
	}

	t.Errorf("Service %q not found in avahi-browse output", service.InstanceName)
}
