package contract

import (
	"testing"

	"github.com/joshuafuller/beacon/internal/records"
	"github.com/joshuafuller/beacon/internal/protocol"
)

// TestRFC6762_RateLimiting_PerRecordOneSecondMinimum tests RFC 6762 §6.2 rate limiting.
//
// RFC 6762 §6.2: "A Multicast DNS responder MUST NOT multicast a given resource record
// on a given interface until at least one second has elapsed since the last time that
// resource record was multicast on that particular interface."
//
// This is a contract test - it verifies our implementation strictly adheres to RFC requirement.
//
// TDD Phase: RED
//
// T071 [P] [US3]: Contract test RFC 6762 §6.2 per-record rate limiting compliance
func TestRFC6762_RateLimiting_PerRecordOneSecondMinimum(t *testing.T) {
	rr := &records.ResourceRecord{
		Name:  "test._http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   4500,
		Data:  []byte{0x04, 't', 'e', 's', 't'},
	}

	rs := records.NewRecordSet()

	// RFC 6762 §6.2 MUST: Cannot multicast same record within 1 second
	rs.RecordMulticast(rr, "eth0")

	// Immediate retry - MUST be denied
	if rs.CanMulticast(rr, "eth0") {
		t.Error("RFC 6762 §6.2 VIOLATION: CanMulticast() = true immediately after multicast, MUST be false (1 second minimum)")
	}
}

// TestRFC6762_RateLimiting_PerInterface tests per-interface rate limiting per RFC 6762 §6.2.
//
// RFC 6762 §6.2: Rate limiting is "on a given interface" - same record can be multicast
// on different interfaces simultaneously.
//
// TDD Phase: RED
//
// T071 [P] [US3]: Contract test per-interface independence
func TestRFC6762_RateLimiting_PerInterface(t *testing.T) {
	rr := &records.ResourceRecord{
		Name:  "test._http._tcp.local",
		Type:  protocol.RecordTypePTR,
		Class: protocol.ClassIN,
		TTL:   4500,
		Data:  []byte{0x04, 't', 'e', 's', 't'},
	}

	rs := records.NewRecordSet()

	// Multicast on eth0
	rs.RecordMulticast(rr, "eth0")

	// RFC 6762 §6.2: Rate limit is per-interface - wlan0 should be independent
	if !rs.CanMulticast(rr, "wlan0") {
		t.Error("RFC 6762 §6.2 VIOLATION: CanMulticast(wlan0) = false after eth0 multicast, should be independent")
	}
}

// TestRFC6762_RateLimiting_ProbeDefense250ms tests probe defense exception per RFC 6762 §6.2.
//
// RFC 6762 §6.2: "The one exception is that a Multicast DNS responder MUST respond
// quickly (at most 250 ms after detecting the conflict) when answering probe queries
// for the purpose of defending its name."
//
// TDD Phase: RED
//
// T071 [P] [US3]: Contract test probe defense 250ms exception
func TestRFC6762_RateLimiting_ProbeDefense250ms(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing test in short mode")
	}

	rr := &records.ResourceRecord{
		Name:  "myhost.local",
		Type:  protocol.RecordTypeA,
		Class: protocol.ClassIN,
		TTL:   120,
		Data:  []byte{192, 168, 1, 100},
	}

	rs := records.NewRecordSet()

	// Multicast record
	rs.RecordMulticast(rr, "eth0")

	// RFC 6762 §6.2: Probe defense MUST respond within 250ms
	// Implementation should allow probe defense multicast after 250ms
	// (Regular responses still require 1 second)

	// This test documents the requirement - implementation will need
	// CanMulticastProbeDefense() method with 250ms threshold
	_ = rs
}
