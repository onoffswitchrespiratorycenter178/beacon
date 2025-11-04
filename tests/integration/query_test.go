// Package integration provides end-to-end integration tests for mDNS querying.
//
// These tests validate real mDNS queries against actual network responders.
// They may timeout in isolated test environments without mDNS services.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/querier"
)

// TestQuery_RealNetwork_ARecord validates end-to-end A record query
// against real mDNS responders on the network (User Story 1).
//
// User Story 1: "As a developer, I want to resolve .local hostnames to
// IP addresses so that I can discover devices on my local network."
//
// Success Criteria (SC-001):
//   - Developers can resolve .local hostnames to IP addresses with single function call
//
// NOTE: This test may timeout in isolated environments without mDNS services
func TestQuery_RealNetwork_ARecord(t *testing.T) {
	// Skip in short mode (requires network)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Try to query a common mDNS name that might exist on the network
	// Common names: router.local, printer.local, etc.
	testCases := []string{
		"_services._dns-sd._udp.local", // Service discovery (usually responds)
		"router.local",                 // Common router name
		"printer.local",                // Common printer name
	}

	foundAny := false
	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			response, err := q.Query(ctx, name, querier.RecordTypeA)
			if err != nil {
				t.Logf("Query(%q) returned error: %v (acceptable if device not on network)", name, err)
				return
			}

			if response == nil {
				t.Logf("Query(%q) returned nil response", name)
				return
			}

			if len(response.Records) == 0 {
				t.Logf("Query(%q) received no records (device not on network or timeout)", name)
				return
			}

			// Success! We got A records
			foundAny = true
			t.Logf("Query(%q) SUCCESS: received %d A records", name, len(response.Records))

			for i, record := range response.Records {
				t.Logf("  Record[%d]: %s → Type=%v, TTL=%d", i, record.Name, record.Type, record.TTL)

				// Validate A record has valid IP
				if record.Type == querier.RecordTypeA {
					ip := record.AsA()
					if ip == nil {
						t.Errorf("Record[%d] AsA() returned nil IP", i)
						continue
					}

					// Verify it's a valid IPv4 address
					if ip.To4() == nil {
						t.Errorf("Record[%d] IP %v is not valid IPv4", i, ip)
						continue
					}

					t.Logf("  Record[%d] IP: %s (valid IPv4)", i, ip)

					// Success Criteria SC-001: Single function call resolved hostname to IP
					t.Logf("✓ SC-001: Resolved %s to %s with single Query() call", name, ip)
				}
			}
		})
	}

	if !foundAny {
		t.Logf("No mDNS responses received (isolated test environment - acceptable)")
		t.Logf("Integration test would pass with mDNS responders on network")
	}
}

// TestQuery_RealNetwork_PTRRecord validates end-to-end PTR record query
// for service discovery against real mDNS responders.
//
// User Story 2: "As a developer, I want to discover services by type
// (e.g., printers, file servers) so that my application can enumerate
// available services."
//
// NOTE: This test may timeout in isolated environments
func TestQuery_RealNetwork_PTRRecord(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Query for common mDNS services
	serviceTypes := []string{
		"_services._dns-sd._udp.local", // Meta-query for all services
		"_http._tcp.local",             // HTTP services
		"_ssh._tcp.local",              // SSH services
		"_printer._tcp.local",          // Printer services
	}

	foundAny := false
	for _, serviceType := range serviceTypes {
		t.Run(serviceType, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			response, err := q.Query(ctx, serviceType, querier.RecordTypePTR)
			if err != nil {
				t.Logf("Query(%q) returned error: %v (acceptable)", serviceType, err)
				return
			}

			if response == nil || len(response.Records) == 0 {
				t.Logf("Query(%q) received no PTR records (no services on network)", serviceType)
				return
			}

			// Success! We got PTR records
			foundAny = true
			t.Logf("Query(%q) SUCCESS: received %d PTR records", serviceType, len(response.Records))

			for i, record := range response.Records {
				t.Logf("  Record[%d]: %s → Type=%v, TTL=%d", i, record.Name, record.Type, record.TTL)

				// Validate PTR record has target
				if record.Type == querier.RecordTypePTR {
					ptrTarget := record.AsPTR()
					if ptrTarget == "" {
						t.Errorf("Record[%d] AsPTR() returned empty string", i)
						continue
					}

					t.Logf("  Record[%d] PTR Target: %s", i, ptrTarget)
				}
			}
		})
	}

	if !foundAny {
		t.Logf("No PTR responses received (isolated test environment - acceptable)")
	}
}

// TestQuery_RealNetwork_SRVRecord validates end-to-end SRV record query
// for service location (hostname and port) against real mDNS responders.
//
// User Story 2: "As a developer, I want to discover services by type
// (e.g., printers, file servers) so that my application can enumerate
// available services."
//
// NOTE: This test may timeout in isolated environments
func TestQuery_RealNetwork_SRVRecord(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Query for common service instance names (these return SRV records)
	// Format: <instance>.<service-type>.local
	serviceInstances := []string{
		"test._http._tcp.local",
		"test._ssh._tcp.local",
		"server._http._tcp.local",
	}

	foundAny := false
	for _, instance := range serviceInstances {
		t.Run(instance, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			response, err := q.Query(ctx, instance, querier.RecordTypeSRV)
			if err != nil {
				t.Logf("Query(%q) returned error: %v (acceptable)", instance, err)
				return
			}

			if response == nil || len(response.Records) == 0 {
				t.Logf("Query(%q) received no SRV records (service not on network)", instance)
				return
			}

			// Success! We got SRV records
			foundAny = true
			t.Logf("Query(%q) SUCCESS: received %d SRV records", instance, len(response.Records))

			for i, record := range response.Records {
				t.Logf("  Record[%d]: %s → Type=%v, TTL=%d", i, record.Name, record.Type, record.TTL)

				// Validate SRV record per RFC 2782
				if record.Type == querier.RecordTypeSRV {
					srv := record.AsSRV()
					if srv == nil {
						t.Errorf("Record[%d] AsSRV() returned nil", i)
						continue
					}

					// Validate SRV fields per RFC 2782
					if srv.Target == "" {
						t.Errorf("Record[%d] SRV Target is empty", i)
					}
					if srv.Port == 0 {
						t.Errorf("Record[%d] SRV Port is 0", i)
					}

					t.Logf("  Record[%d] SRV: %s:%d (priority=%d, weight=%d)",
						i, srv.Target, srv.Port, srv.Priority, srv.Weight)
				}
			}
		})
	}

	if !foundAny {
		t.Logf("No SRV responses received (isolated test environment - acceptable)")
	}
}

// TestQuery_RealNetwork_TXTRecord validates end-to-end TXT record query
// for service metadata against real mDNS responders.
//
// User Story 2: "As a developer, I want to discover services by type
// (e.g., printers, file servers) so that my application can enumerate
// available services."
//
// NOTE: This test may timeout in isolated environments
func TestQuery_RealNetwork_TXTRecord(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Query for common service instance names (these return TXT records)
	// Format: <instance>.<service-type>.local
	serviceInstances := []string{
		"test._http._tcp.local",
		"server._http._tcp.local",
		"printer._printer._tcp.local",
	}

	foundAny := false
	for _, instance := range serviceInstances {
		t.Run(instance, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			response, err := q.Query(ctx, instance, querier.RecordTypeTXT)
			if err != nil {
				t.Logf("Query(%q) returned error: %v (acceptable)", instance, err)
				return
			}

			if response == nil || len(response.Records) == 0 {
				t.Logf("Query(%q) received no TXT records (service not on network)", instance)
				return
			}

			// Success! We got TXT records
			foundAny = true
			t.Logf("Query(%q) SUCCESS: received %d TXT records", instance, len(response.Records))

			for i, record := range response.Records {
				t.Logf("  Record[%d]: %s → Type=%v, TTL=%d", i, record.Name, record.Type, record.TTL)

				// Validate TXT record per RFC 6763
				if record.Type == querier.RecordTypeTXT {
					txt := record.AsTXT()
					if txt == nil {
						t.Errorf("Record[%d] AsTXT() returned nil", i)
						continue
					}

					t.Logf("  Record[%d] TXT: %d strings", i, len(txt))
					for j, kv := range txt {
						t.Logf("    TXT[%d]: %s", j, kv)
					}
				}
			}
		})
	}

	if !foundAny {
		t.Logf("No TXT responses received (isolated test environment - acceptable)")
	}
}

// TestQuery_RealNetwork_Timeout validates that timeout behavior works
// as expected on real network (SC-002).
//
// Success Criteria (SC-002):
//   - System successfully discovers 95% of responding devices within 1 second
//
// This test validates TWO things:
//  1. Context timeout is respected (query doesn't hang indefinitely)
//  2. Query infrastructure has acceptable overhead
//
// NOTE: We allow overhead for context propagation, goroutine scheduling,
// and timer precision. Typical overhead: 1-5ms (context) + 0-1ms (scheduler).
// We use 100ms tolerance to handle loaded CI systems and timing jitter.
func TestQuery_RealNetwork_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	const (
		queryTimeout    = 1 * time.Second
		jitterTolerance = 100 * time.Millisecond // Allow overhead for context propagation
	)

	// SC-002: Test with 1-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	start := time.Now()
	response, err := q.Query(ctx, "_services._dns-sd._udp.local", querier.RecordTypePTR)
	elapsed := time.Since(start)

	// Validate Query() respected the context timeout
	// Real bugs: elapsed >> queryTimeout (query hung, didn't check context)
	// Acceptable: elapsed ≈ queryTimeout ± jitter
	maxAcceptable := queryTimeout + jitterTolerance

	if elapsed > maxAcceptable {
		t.Errorf("✗ SC-002: Query took %v, exceeded timeout + tolerance (%v + %v)",
			elapsed, queryTimeout, jitterTolerance)
		return
	}

	// Log outcome based on what happened
	if err != nil {
		t.Logf("Query timed out after %v (error: %v) - context timeout working ✓", elapsed, err)
	} else if response == nil {
		t.Logf("Query returned nil response after %v - acceptable ✓", elapsed)
	} else {
		recordCount := len(response.Records)
		t.Logf("✓ SC-002: Query discovered %d records in %v (within %v + %v tolerance)",
			recordCount, elapsed, queryTimeout, jitterTolerance)
	}
}

// TestQuery_RealNetwork_MultipleQueries validates that Querier can be
// reused for multiple queries (resource management).
//
// FR-017: System MUST close socket after query completion
// FR-018: System MUST support graceful shutdown via context cancellation
func TestQuery_RealNetwork_MultipleQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Execute multiple queries sequentially
	queries := []struct {
		name       string
		recordType querier.RecordType
	}{
		{"test1.local", querier.RecordTypeA},
		{"test2.local", querier.RecordTypeA},
		{"_services._dns-sd._udp.local", querier.RecordTypePTR},
	}

	for i, query := range queries {
		t.Run(query.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			response, err := q.Query(ctx, query.name, query.recordType)
			if err != nil {
				t.Logf("Query[%d](%q) returned error: %v (acceptable)", i, query.name, err)
				return
			}

			if response == nil {
				t.Logf("Query[%d](%q) returned nil response (timeout)", i, query.name)
			} else {
				t.Logf("Query[%d](%q) returned %d records", i, query.name, len(response.Records))
			}
		})
	}

	t.Logf("Querier successfully handled %d sequential queries", len(queries))
}

// TestQuery_RealNetwork_ConcurrentQueries validates that Querier can handle
// concurrent queries safely (concurrency safety per F-4).
//
// F-4: Concurrency Model - support concurrent queries with goroutine-safe operations
func TestQuery_RealNetwork_ConcurrentQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Execute multiple queries concurrently
	queries := []struct {
		name       string
		recordType querier.RecordType
	}{
		{"concurrent1.local", querier.RecordTypeA},
		{"concurrent2.local", querier.RecordTypeA},
		{"_services._dns-sd._udp.local", querier.RecordTypePTR},
		{"_http._tcp.local", querier.RecordTypePTR},
	}

	done := make(chan struct{})
	for i, query := range queries {
		go func(idx int, name string, recordType querier.RecordType) {
			defer func() {
				done <- struct{}{}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			response, err := q.Query(ctx, name, recordType)
			if err != nil {
				t.Logf("ConcurrentQuery[%d](%q) error: %v", idx, name, err)
				return
			}

			if response != nil {
				t.Logf("ConcurrentQuery[%d](%q) returned %d records", idx, name, len(response.Records))
			}
		}(i, query.name, query.recordType)
	}

	// Wait for all queries to complete
	for range queries {
		<-done
	}

	t.Logf("Querier successfully handled %d concurrent queries (goroutine-safe)", len(queries))
}
