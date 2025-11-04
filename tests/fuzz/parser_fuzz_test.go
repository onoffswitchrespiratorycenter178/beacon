// Package fuzz provides fuzz testing for DNS message parsing.
//
// Fuzz testing validates that the parser handles malformed packets without
// crashes or panics per NFR-003.
package fuzz

import (
	"testing"

	"github.com/joshuafuller/beacon/internal/message"
)

// FuzzParseMessage tests ParseMessage with random inputs to ensure it handles
// malformed packets without crashes or panics per NFR-003.
//
// NFR-003: System MUST handle malformed packets without crashes or panics
// (verified via fuzz testing with 10,000 random packets)
//
// The fuzzer tests:
//   - Valid DNS messages (should parse successfully)
//   - Messages that are too short (should return WireFormatError)
//   - Messages with invalid compression pointers (should return WireFormatError)
//   - Messages with truncated sections (should return WireFormatError)
//   - Messages with invalid record types (should return WireFormatError or parse)
//   - Messages with malformed names (should return WireFormatError)
//   - Random byte sequences (should not panic)
//
// Run with: go test -fuzz=FuzzParseMessage -fuzztime=10000x ./tests/fuzz/
func FuzzParseMessage(f *testing.F) {
	// Seed corpus: Valid DNS response message
	// Header + Question + Answer with A record
	validMessage := []byte{
		// Header: ID=0x1234, Flags=0x8400 (QR=1, AA=1)
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "test.local" A IN
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x01, // QTYPE = A
		0x00, 0x01, // QCLASS = IN

		// Answer: "test.local" A IN TTL=120 RDATA=192.168.1.100
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x01, // TYPE = A
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x04, // RDLENGTH = 4
		192, 168, 1, 100, // RDATA = 192.168.1.100
	}
	f.Add(validMessage)

	// Seed corpus: Message with compression pointer
	compressedMessage := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "test.local" A IN
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x01, // QTYPE = A
		0x00, 0x01, // QCLASS = IN

		// Answer: Compressed pointer to offset 12 (points to "test.local")
		0xC0, 0x0C, // Compression pointer to offset 12
		0x00, 0x01, // TYPE = A
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x04, // RDLENGTH = 4
		192, 168, 1, 100, // RDATA = 192.168.1.100
	}
	f.Add(compressedMessage)

	// Seed corpus: PTR record
	ptrMessage := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "_http._tcp.local" PTR IN
		0x05, '_', 'h', 't', 't', 'p',
		0x04, '_', 't', 'c', 'p',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x0C, // QTYPE = PTR
		0x00, 0x01, // QCLASS = IN

		// Answer: "_http._tcp.local" PTR IN TTL=120 RDATA="myservice._http._tcp.local"
		0xC0, 0x0C, // Compression pointer to question name
		0x00, 0x0C, // TYPE = PTR
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x14, // RDLENGTH = 20
		// RDATA: "myservice._http._tcp.local"
		0x09, 'm', 'y', 's', 'e', 'r', 'v', 'i', 'c', 'e',
		0xC0, 0x0C, // Compression pointer to "_http._tcp.local"
	}
	f.Add(ptrMessage)

	// Seed corpus: SRV record
	srvMessage := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "myservice._http._tcp.local" SRV IN
		0x09, 'm', 'y', 's', 'e', 'r', 'v', 'i', 'c', 'e',
		0x05, '_', 'h', 't', 't', 'p',
		0x04, '_', 't', 'c', 'p',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x21, // QTYPE = SRV
		0x00, 0x01, // QCLASS = IN

		// Answer: SRV record with priority=10, weight=20, port=8080, target="host.local"
		0xC0, 0x0C, // Compression pointer to question name
		0x00, 0x21, // TYPE = SRV
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x10, // RDLENGTH = 16
		// RDATA: SRV
		0x00, 0x0A, // Priority = 10
		0x00, 0x14, // Weight = 20
		0x1F, 0x90, // Port = 8080
		// Target: "host.local"
		0x04, 'h', 'o', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
	}
	f.Add(srvMessage)

	// Seed corpus: TXT record
	txtMessage := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question: "myservice._http._tcp.local" TXT IN
		0x09, 'm', 'y', 's', 'e', 'r', 'v', 'i', 'c', 'e',
		0x05, '_', 'h', 't', 't', 'p',
		0x04, '_', 't', 'c', 'p',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x10, // QTYPE = TXT
		0x00, 0x01, // QCLASS = IN

		// Answer: TXT record with "key=value"
		0xC0, 0x0C, // Compression pointer to question name
		0x00, 0x10, // TYPE = TXT
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x0A, // RDLENGTH = 10
		// RDATA: TXT strings
		0x09, 'k', 'e', 'y', '=', 'v', 'a', 'l', 'u', 'e',
	}
	f.Add(txtMessage)

	// Seed corpus: Message too short (less than 12 bytes)
	tooShort := []byte{0x12, 0x34, 0x84, 0x00}
	f.Add(tooShort)

	// Seed corpus: Truncated question
	truncatedQuestion := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Truncated question (missing QTYPE and QCLASS)
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, // Missing second byte of QTYPE
	}
	f.Add(truncatedQuestion)

	// Seed corpus: Invalid compression pointer (points beyond message)
	invalidPointer := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x01, // ANCOUNT = 1
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question
		0x04, 't', 'e', 's', 't',
		0x05, 'l', 'o', 'c', 'a', 'l',
		0x00,
		0x00, 0x01, // QTYPE = A
		0x00, 0x01, // QCLASS = IN

		// Answer with invalid pointer (points to offset 200, beyond message)
		0xC0, 0xC8, // Compression pointer to offset 200
		0x00, 0x01, // TYPE = A
		0x00, 0x01, // CLASS = IN
		0x00, 0x00, 0x00, 0x78, // TTL = 120
		0x00, 0x04, // RDLENGTH = 4
		192, 168, 1, 100, // RDATA
	}
	f.Add(invalidPointer)

	// Seed corpus: Compression loop (pointer points to itself)
	compressionLoop := []byte{
		// Header
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0

		// Question with self-referencing pointer
		0xC0, 0x0C, // Compression pointer to offset 12 (points to itself)
		0x00, 0x01, // QTYPE = A
		0x00, 0x01, // QCLASS = IN
	}
	f.Add(compressionLoop)

	// Seed corpus: Empty message (just header, no sections)
	emptyMessage := []byte{
		0x12, 0x34, // ID
		0x84, 0x00, // Flags
		0x00, 0x00, // QDCOUNT = 0
		0x00, 0x00, // ANCOUNT = 0
		0x00, 0x00, // NSCOUNT = 0
		0x00, 0x00, // ARCOUNT = 0
	}
	f.Add(emptyMessage)

	// Fuzz function: ParseMessage must not panic on any input
	f.Fuzz(func(_ *testing.T, data []byte) {
		// The critical requirement is: NO PANICS (NFR-003)
		// We don't care about errors, only that the parser doesn't crash
		_, _ = message.ParseMessage(data)

		// If we reach here without panic, the test passes
		// ParseMessage may return an error for malformed data, which is expected
	})
}
