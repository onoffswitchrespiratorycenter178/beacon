package integration

import (
	"context"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/responder"
)

// TestQueryResponse_ResponseLatency tests end-to-end query response latency.
//
// RFC 6762 §6: "When a host... is able to answer every question in the query message,
// and for all of those answer records it has previously verified that the name, rrtype,
// and rrclass are unique on the link), it SHOULD NOT impose any random delay before
// responding, and SHOULD normally generate its response within at most 10 ms."
//
// SC-006: Response time MUST be <100ms for registered services
//
// TDD Phase: RED (test written first)
//
// T072 [US3]: Integration test query registered service, verify response <100ms
func TestQueryResponse_ResponseLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create responder
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder: %v", err)
	}
	defer func() { _ = r.Close() }()

	// Register service
	service := &responder.Service{
		InstanceName: "TestPrinter",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	err = r.Register(service)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Wait for probing + announcing to complete (~1.75s per M1 timing)
	time.Sleep(2 * time.Second)

	// Currently this test will SKIP because query handling (US3) is not yet implemented.
	// When US3 is complete, uncomment the following to test response latency:
	t.Skip("Query handling not yet implemented (US3 in progress)")

	// Future implementation when US3 complete:
	// 1. Create querier
	// 2. Send PTR query for "_http._tcp.local"
	// 3. Measure response time
	// 4. Verify response received within 100ms
	// 5. Verify response contains PTR + SRV + TXT + A records
}

// TestQueryResponse_PTRQueryWithAdditionalRecords tests PTR query response structure.
//
// RFC 6762 §6: When responding to a PTR query, responder should include:
//   - Answer section: PTR record
//   - Additional section: SRV, TXT, A records (reduces round-trips)
//
// TDD Phase: RED
//
// T072 [US3]: Verify PTR response includes additional records
func TestQueryResponse_PTRQueryWithAdditionalRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder: %v", err)
	}
	defer func() { _ = r.Close() }()

	service := &responder.Service{
		InstanceName: "TestService",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
		TXTRecords:   map[string]string{"txtvers": "1", "path": "/"},
	}

	err = r.Register(service)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Wait for registration
	time.Sleep(2 * time.Second)

	// Currently skipped - US3 implementation pending
	t.Skip("Query handling not yet implemented (US3 in progress)")

	// Future implementation:
	// 1. Send PTR query for "_http._tcp.local"
	// 2. Parse response
	// 3. Verify answer section contains PTR record
	// 4. Verify additional section contains SRV, TXT, A records
	// 5. Verify TXT record contains "txtvers=1" and "path=/"
}

// TestQueryResponse_QUBitHandling tests unicast response per RFC 6762 §5.4.
//
// RFC 6762 §5.4: "When receiving a question with the unicast-response bit set, a
// responder SHOULD usually respond with a unicast packet directed back to the querier."
//
// TDD Phase: RED
//
// T072 [US3]: Verify QU bit triggers unicast response
func TestQueryResponse_QUBitHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder: %v", err)
	}
	defer func() { _ = r.Close() }()

	service := &responder.Service{
		InstanceName: "TestService",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	err = r.Register(service)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Currently skipped - US3 implementation pending
	t.Skip("Query handling not yet implemented (US3 in progress)")

	// Future implementation:
	// 1. Send PTR query with QU bit set (class = 0x8001)
	// 2. Verify response is sent via unicast (not multicast)
	// 3. Send PTR query without QU bit
	// 4. Verify response is sent via multicast
}
