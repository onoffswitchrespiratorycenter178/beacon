package contract

import (
	"context"
	"testing"

	"github.com/joshuafuller/beacon/internal/protocol"
	"github.com/joshuafuller/beacon/responder"
)

// TestRFC6762_TTL_ServiceRecords tests RFC 6762 §10 TTL values for service records.
//
// RFC 6762 §10: "The recommended TTL value for Multicast DNS resource records
// with a host name as the resource record's name (e.g., A, AAAA, HINFO) or a
// resource record containing data relating to a host name (e.g., SRV, reverse
// mapping PTR record) is 120 seconds."
//
// This test validates that:
//   - PTR records use TTL = 120 seconds
//   - SRV records use TTL = 120 seconds
//   - TXT records use TTL = 120 seconds
//
// FR-019: System MUST use RFC 6762 §10 TTL values (120s for service records)
// T115: Contract test for RFC 6762 §10 TTL handling
func TestRFC6762_TTL_ServiceRecords(t *testing.T) {
	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register a service
	svc := &responder.Service{
		InstanceName: "Test Service",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
		TXTRecords:   map[string]string{"version": "1.0"},
	}

	err = r.Register(svc)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Get announced records
	records := r.GetLastAnnouncedRecords()
	if len(records) == 0 {
		t.Fatalf("GetLastAnnouncedRecords() = 0 records, want >0")
	}

	// Validate TTL values per RFC 6762 §10
	foundPTR := false
	foundSRV := false
	foundTXT := false

	for _, rr := range records {
		switch rr.Type {
		case protocol.RecordTypePTR:
			foundPTR = true
			if rr.TTL != protocol.TTLService {
				t.Errorf("PTR record TTL = %d, want %d (RFC 6762 §10: service record TTL = 120s)",
					rr.TTL, protocol.TTLService)
			}

		case protocol.RecordTypeSRV:
			foundSRV = true
			if rr.TTL != protocol.TTLService {
				t.Errorf("SRV record TTL = %d, want %d (RFC 6762 §10: service record TTL = 120s)",
					rr.TTL, protocol.TTLService)
			}

		case protocol.RecordTypeTXT:
			foundTXT = true
			if rr.TTL != protocol.TTLService {
				t.Errorf("TXT record TTL = %d, want %d (RFC 6762 §10: service record TTL = 120s)",
					rr.TTL, protocol.TTLService)
			}
		}
	}

	// Verify we found all expected record types
	if !foundPTR {
		t.Error("No PTR record found in announced records")
	}
	if !foundSRV {
		t.Error("No SRV record found in announced records")
	}
	if !foundTXT {
		t.Error("No TXT record found in announced records")
	}
}

// TestRFC6762_TTL_HostnameRecords tests RFC 6762 §10 TTL values for hostname records.
//
// RFC 6762 §10: "The recommended TTL value for other Multicast DNS resource
// records is 75 minutes (4500 seconds)."
//
// This test validates that:
//   - A records use TTL = 4500 seconds (75 minutes)
//
// FR-019: System MUST use RFC 6762 §10 TTL values (4500s for hostname records)
// T115: Contract test for RFC 6762 §10 TTL handling
func TestRFC6762_TTL_HostnameRecords(t *testing.T) {
	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register a service
	svc := &responder.Service{
		InstanceName: "Test Service",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	err = r.Register(svc)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Get announced records
	records := r.GetLastAnnouncedRecords()
	if len(records) == 0 {
		t.Fatalf("GetLastAnnouncedRecords() = 0 records, want >0")
	}

	// Validate TTL values per RFC 6762 §10
	foundA := false

	for _, rr := range records {
		if rr.Type == protocol.RecordTypeA {
			foundA = true
			if rr.TTL != protocol.TTLHostname {
				t.Errorf("A record TTL = %d, want %d (RFC 6762 §10: hostname record TTL = 4500s)",
					rr.TTL, protocol.TTLHostname)
			}
		}
	}

	// Verify we found an A record
	if !foundA {
		t.Error("No A record found in announced records")
	}
}

// TestRFC6762_TTL_GoodbyePackets tests RFC 6762 §10.1 goodbye packet TTL=0.
//
// RFC 6762 §10.1: "In the case of shared records (e.g., PTR records), or any
// record where there may legitimately be more than one responder on the network,
// where the data in the responses is beneficially aggregated, a host SHOULD send
// a 'goodbye' announcement with TTL zero when it knows that the record is no longer
// valid (e.g., when the service is being shut down or the host is going to sleep)."
//
// This test validates that unregistering a service sends records with TTL=0.
//
// FR-033: System MUST send goodbye announcements (TTL=0) on service removal
// T115: Contract test for RFC 6762 §10.1 goodbye packets
func TestRFC6762_TTL_GoodbyePackets(t *testing.T) {
	t.Skip("Deferred: Goodbye packet functionality not yet implemented (TODO in Responder.Unregister)")

	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register a service
	svc := &responder.Service{
		InstanceName: "Test Service",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	err = r.Register(svc)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	// Unregister the service
	err = r.Unregister("Test Service")
	if err != nil {
		t.Fatalf("Unregister() error = %v, want nil", err)
	}

	// TODO: Capture goodbye packets
	// When goodbye packet functionality is implemented:
	// 1. Capture the last multicast message
	// 2. Parse the message
	// 3. Verify all records have TTL = 0
	// 4. Verify record types match original announcement (PTR, SRV, TXT, A)
}
