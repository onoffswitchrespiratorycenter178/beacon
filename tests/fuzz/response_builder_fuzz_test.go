// Package fuzz provides fuzz testing for the response builder.
//
// Fuzz testing validates that response building handles malformed queries
// without crashes or panics per NFR-003.
package fuzz

import (
	"testing"

	"github.com/joshuafuller/beacon/internal/message"
	"github.com/joshuafuller/beacon/internal/protocol"
	"github.com/joshuafuller/beacon/internal/responder"
)

// FuzzResponseBuilder tests response building with malformed query messages.
//
// This fuzzer validates that ResponseBuilder handles malformed DNS query
// messages without crashes or panics per NFR-003 (Safety & Robustness).
//
// The fuzzer tests:
//   - Valid DNS query messages (should build response successfully)
//   - Truncated DNS messages (too short, missing sections)
//   - Invalid compression pointers
//   - Malformed question names
//   - Out-of-range QDCOUNT values
//   - Random byte sequences
//
// Expected behavior:
//   - Valid queries: Successful response building
//   - Invalid queries: Return error (not panic)
//
// NFR-003: System MUST handle malformed packets without crashes or panics
// T119: Fuzz test for response builder with malformed queries
//
// Run with: go test -fuzz=FuzzResponseBuilder -fuzztime=10s ./tests/fuzz/
func FuzzResponseBuilder(f *testing.F) {
	// Seed corpus: Valid PTR query for "_http._tcp.local"
	validQuery := []byte{
		// Header: ID=0x1234, Flags=0x0000 (query)
		0x12, 0x34, // ID
		0x00, 0x00, // Flags (standard query)
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "_http._tcp.local" PTR IN
		0x05, '_', 'h', 't', 't', 'p',
		0x04, '_', 't', 'c', 'p',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x0C, // QTYPE = PTR
		0x00, 0x01, // QCLASS = IN
	}
	f.Add(validQuery)

	// Seed corpus: Query with compression pointer
	compressedQuery := []byte{
		0x12, 0x34, // ID
		0x00, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question with compression
		0x05, '_', 'h', 't', 't', 'p',
		0xC0, 0x0C, // Compression pointer to offset 12
		0x00, 0x0C, // QTYPE = PTR
		0x00, 0x01, // QCLASS = IN
	}
	f.Add(compressedQuery)

	// Seed corpus: Truncated message (too short)
	f.Add([]byte{0x12, 0x34, 0x00, 0x00})

	// Seed corpus: Invalid compression pointer (points beyond message)
	invalidPointer := []byte{
		0x12, 0x34, // ID
		0x00, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question with invalid pointer
		0xC0, 0xFF, // Compression pointer to offset 255 (beyond message)
		0x00, 0x0C, // QTYPE = PTR
		0x00, 0x01, // QCLASS = IN
	}
	f.Add(invalidPointer)

	f.Fuzz(func(t *testing.T, queryData []byte) {
		// Parse query message - should handle malformed input gracefully
		query, err := message.ParseMessage(queryData)

		// If parsing failed, that's OK - just ensure no panic
		if err != nil {
			// Expected for malformed messages - ensure error is returned, not panic
			return
		}

		// If parsing succeeded, try building a response
		rb := responder.NewResponseBuilder()

		// Create a test service for response building
		service := &responder.ServiceWithIP{
			InstanceName: "Test Service",
			ServiceType:  "_http._tcp.local",
			Domain:       "local",
			Port:         8080,
			IPv4Address:  []byte{192, 168, 1, 100},
			TXTRecords:   map[string]string{"version": "1.0"},
			Hostname:     "test.local",
		}

		// Attempt response building - should NEVER panic
		// Valid queries should succeed, malformed queries should return error
		response, err := rb.BuildResponse(service, query)

		// Don't assert on error - goal is to ensure NO PANIC
		// Invalid queries are expected to return errors, not crash
		_ = response
		_ = err
	})
}

// FuzzMessageBuilding tests message.BuildResponse with random resource records.
//
// This fuzzer validates that DNS message serialization handles malformed
// resource records without crashes or panics per NFR-003.
//
// The fuzzer tests:
//   - Valid resource records
//   - Records with very long names (> 255 bytes)
//   - Records with invalid types
//   - Records with empty/nil data
//   - Random RDATA content
//
// Expected behavior:
//   - Valid records: Successful serialization
//   - Invalid records: Return error (not panic)
//
// NFR-003: System MUST handle invalid input without crashes or panics
// T119: Fuzz test for message building with malformed records
//
// Run with: go test -fuzz=FuzzMessageBuilding -fuzztime=10s ./tests/fuzz/
func FuzzMessageBuilding(f *testing.F) {
	// Seed corpus: Valid A record
	f.Add("test.local", uint16(protocol.RecordTypeA), []byte{192, 168, 1, 100})

	// Seed corpus: Valid PTR record
	f.Add("_http._tcp.local", uint16(protocol.RecordTypePTR), []byte{
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
	})

	// Seed corpus: Empty data
	f.Add("test.local", uint16(protocol.RecordTypeA), []byte{})

	// Seed corpus: Very long name
	longName := string(make([]byte, 300))
	f.Add(longName, uint16(protocol.RecordTypeA), []byte{192, 168, 1, 1})

	// Seed corpus: Invalid type
	f.Add("test.local", uint16(0xFFFF), []byte{0x01, 0x02})

	f.Fuzz(func(t *testing.T, name string, recordType uint16, data []byte) {
		// Construct a resource record from fuzz inputs
		rr := &message.ResourceRecord{
			Name:       name,
			Type:       protocol.RecordType(recordType), // Cast uint16 to RecordType
			Class:      protocol.ClassIN,
			TTL:        120,
			Data:       data,
			CacheFlush: false,
		}

		// Attempt to build a DNS response message - should NEVER panic
		records := []*message.ResourceRecord{rr}
		responseBytes, err := message.BuildResponse(records)

		// Don't assert on error - goal is to ensure NO PANIC
		// Invalid records are expected to return errors, not crash
		_ = responseBytes
		_ = err

		// If building succeeded, try parsing the result to ensure round-trip validity
		if err == nil && len(responseBytes) > 0 {
			_, parseErr := message.ParseMessage(responseBytes)
			// Parsing may fail for malformed records - that's OK, just ensure no panic
			_ = parseErr
		}
	})
}

// FuzzQueryParsing tests DNS query message parsing with random inputs.
//
// This fuzzer validates that message parsing handles all possible byte
// sequences without crashes or panics per NFR-003.
//
// The fuzzer tests:
//   - Valid DNS query messages
//   - Messages with QDCOUNT > actual questions
//   - Messages with invalid flags
//   - Truncated headers
//   - Random byte sequences
//
// Expected behavior:
//   - Valid messages: Successful parsing
//   - Invalid messages: Return WireFormatError (not panic)
//
// NFR-003: System MUST handle malformed packets without crashes or panics
// T119: Fuzz test for query parsing
//
// Run with: go test -fuzz=FuzzQueryParsing -fuzztime=10s ./tests/fuzz/
func FuzzQueryParsing(f *testing.F) {
	// Seed corpus: Minimal valid query (header only)
	minimalQuery := []byte{
		0x00, 0x00, // ID
		0x00, 0x00, // Flags
		0x00, 0x00, // QDCOUNT = 0
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0
	}
	f.Add(minimalQuery)

	// Seed corpus: Query with QDCOUNT=1 but no question section (truncated)
	truncatedQuery := []byte{
		0x00, 0x00, // ID
		0x00, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1 (but no questions follow)
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0
	}
	f.Add(truncatedQuery)

	// Seed corpus: Header with all counts maxed out
	maxCountsQuery := []byte{
		0xFF, 0xFF, // ID
		0xFF, 0xFF, // Flags
		0xFF, 0xFF, // QDCOUNT = 65535
		0xFF, 0xFF, // ANCOUNT = 65535
		0xFF, 0xFF, // NSCOUNT = 65535
		0xFF, 0xFF, // ARCOUNT = 65535
	}
	f.Add(maxCountsQuery)

	// Seed corpus: Empty message
	f.Add([]byte{})

	// Seed corpus: Single byte
	f.Add([]byte{0x00})

	f.Fuzz(func(t *testing.T, queryData []byte) {
		// Attempt to parse query - should handle ALL inputs gracefully
		msg, err := message.ParseMessage(queryData)

		// Don't assert on error - goal is to ensure NO PANIC
		// Invalid messages are expected to return errors, not crash
		_ = msg
		_ = err

		// If parsing succeeded, ensure message fields are reasonable
		if err == nil && msg != nil {
			// Access message fields to ensure they're valid
			// This should never panic even for fuzzed inputs that parsed
			_ = msg.Header.ID
			_ = msg.Header.Flags
			_ = msg.Header.QDCount
			_ = len(msg.Questions)
			_ = len(msg.Answers)
		}
	})
}
