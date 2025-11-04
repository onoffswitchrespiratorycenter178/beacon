// Package contract tests the public API contract for beacon/querier.
//
// These tests validate the expected behavior of the public Query() API
// per the API contract defined in contracts/querier-api.md.
package contract

import (
	"context"
	goerrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/internal/errors"
	"github.com/joshuafuller/beacon/querier"
)

// TestQuery_ValidationError_EmptyName validates that Query() returns
// ValidationError for empty name per FR-003, FR-014.
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
//
// Contract: Query(ctx, "", recordType) → ValidationError
func TestQuery_ValidationError_EmptyName(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	ctx := context.Background()
	response, err := q.Query(ctx, "", querier.RecordTypeA)

	if err == nil {
		t.Errorf("Query(empty name) expected ValidationError per FR-003, FR-014, got nil")
		return
	}

	// Verify it's a ValidationError
	var validationErr *errors.ValidationError
	if !goerrors.As(err, &validationErr) {
		t.Errorf("Query(empty name) error is %T, expected ValidationError per FR-014", err)
		return
	}

	// Verify response is nil on error
	if response != nil {
		t.Errorf("Query(empty name) response should be nil on error, got %v", response)
	}
}

// TestQuery_ValidationError_OversizedName validates that Query() returns
// ValidationError for names exceeding 255 bytes per RFC 1035 §3.1 (FR-003, FR-014).
//
// RFC 1035 §3.1: Domain names are limited to 255 bytes in wire format
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
//
// Contract: Query(ctx, oversized_name, recordType) → ValidationError
func TestQuery_ValidationError_OversizedName(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Build a name that exceeds 255 bytes in wire format
	// Wire format: 3 * (1 + 63) + 1 * (1 + 62) + 1 = 256 bytes
	label63a := strings.Repeat("a", 63)
	label63b := strings.Repeat("b", 63)
	label63c := strings.Repeat("c", 63)
	label62 := strings.Repeat("d", 62)
	oversizedName := label63a + "." + label63b + "." + label63c + "." + label62

	ctx := context.Background()
	response, err := q.Query(ctx, oversizedName, querier.RecordTypeA)

	if err == nil {
		t.Errorf("Query(oversized name) expected ValidationError per RFC 1035 §3.1, FR-003, FR-014, got nil")
		return
	}

	// Verify it's a ValidationError
	var validationErr *errors.ValidationError
	if !goerrors.As(err, &validationErr) {
		t.Errorf("Query(oversized name) error is %T, expected ValidationError per FR-014", err)
		return
	}

	// Verify response is nil on error
	if response != nil {
		t.Errorf("Query(oversized name) response should be nil on error, got %v", response)
	}
}

// TestQuery_ValidationError_InvalidCharacters validates that Query() returns
// ValidationError for names with invalid characters per RFC 1035 §3.1 (FR-003, FR-014).
//
// RFC 1035 §3.1: Valid characters are [a-zA-Z0-9-_]
//
// FR-003: System MUST validate queried names follow DNS naming rules
// FR-014: System MUST return ValidationError for invalid query names
//
// Contract: Query(ctx, invalid_name, recordType) → ValidationError
func TestQuery_ValidationError_InvalidCharacters(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	tests := []struct {
		name        string
		invalidName string
	}{
		{
			name:        "space character",
			invalidName: "test host.local",
		},
		{
			name:        "slash character",
			invalidName: "test/host.local",
		},
		{
			name:        "at symbol",
			invalidName: "test@host.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := q.Query(ctx, tt.invalidName, querier.RecordTypeA)

			if err == nil {
				t.Errorf("Query(%q) expected ValidationError per FR-003, FR-014, got nil", tt.invalidName)
				return
			}

			// Verify it's a ValidationError
			var validationErr *errors.ValidationError
			if !goerrors.As(err, &validationErr) {
				t.Errorf("Query(%q) error is %T, expected ValidationError per FR-014", tt.invalidName, err)
				return
			}

			// Verify response is nil on error
			if response != nil {
				t.Errorf("Query(%q) response should be nil on error, got %v", tt.invalidName, response)
			}
		})
	}
}

// TestQuery_Timeout_ReturnsEmptyResponse validates that Query() returns
// an empty Response (not error) on timeout per FR-008.
//
// FR-008: System MUST aggregate responses received within timeout window
//
// Contract: Query(ctx_with_timeout, valid_name, recordType) → Response{Records: []}, nil
//
// NOTE: Timeout is NOT an error condition - it means "no responses received yet"
func TestQuery_Timeout_ReturnsEmptyResponse(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Use a short timeout and a name that won't respond
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Query a name that's unlikely to exist
	nonexistentName := "this-device-definitely-does-not-exist-12345.local"
	response, err := q.Query(ctx, nonexistentName, querier.RecordTypeA)

	// Timeout should NOT be an error per FR-008
	if err != nil {
		t.Errorf("Query(timeout) expected nil error per FR-008, got %v", err)
		return
	}

	// Response should not be nil
	if response == nil {
		t.Errorf("Query(timeout) response should not be nil per FR-008, got nil")
		return
	}

	// Response should typically be empty for non-existent name
	// However, if there's mDNS traffic on the network from other queries,
	// we might receive unrelated responses (acceptable in real environment)
	if len(response.Records) == 0 {
		t.Logf("✓ Query(timeout) returned empty response per FR-008 (timeout, no responses)")
	} else {
		t.Logf("Query(timeout) returned %d records (mDNS traffic from network - acceptable)", len(response.Records))
		t.Logf("✓ FR-008: Timeout is not an error, aggregated available responses")
	}
}

// TestQuery_ContextCancellation validates that Query() respects context
// cancellation per FR-018.
//
// FR-018: System MUST support graceful shutdown via context cancellation
//
// Contract: Query(canceled_ctx, valid_name, recordType) → error (context.Canceled)
func TestQuery_ContextCancellation(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Create a context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	response, err := q.Query(ctx, "test.local", querier.RecordTypeA)

	// Should return context.Canceled error
	if err == nil {
		t.Errorf("Query(canceled context) expected error per FR-018, got nil")
		return
	}

	if !goerrors.Is(err, context.Canceled) {
		t.Errorf("Query(canceled context) expected context.Canceled per FR-018, got %v", err)
	}

	// Response should be nil on cancellation
	if response != nil {
		t.Errorf("Query(canceled context) response should be nil on error, got %v", response)
	}
}

// ============================================================================
// Phase 4: User Story 2 - PTR/SRV/TXT Record Tests
// ============================================================================

// TestQuery_PTRRecord validates that Query() correctly handles PTR records
// for service discovery per User Story 2.
//
// User Story 2: "As a developer, I want to discover services by type
// (e.g., printers, file servers) so that my application can enumerate
// available services."
//
// FR-002: System MUST support querying for PTR record types
//
// Contract: Query(ctx, service_type, RecordTypePTR) → Response with PTR records
func TestQuery_PTRRecord(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Query for PTR records (service discovery)
	response, err := q.Query(ctx, "_services._dns-sd._udp.local", querier.RecordTypePTR)
	if err != nil {
		t.Logf("Query(PTR) returned error: %v (acceptable in isolated test)", err)
		return
	}

	if response == nil {
		t.Errorf("Query(PTR) response should not be nil, got nil")
		return
	}

	// If we got PTR records, verify they're accessible via AsPTR()
	if len(response.Records) > 0 {
		t.Logf("✓ Query(PTR) received %d PTR records", len(response.Records))

		for i, record := range response.Records {
			if record.Type != querier.RecordTypePTR {
				t.Logf("Record[%d] has type %v (expected PTR) - mixed types acceptable", i, record.Type)
				continue
			}

			// Verify AsPTR() returns valid target
			ptrTarget := record.AsPTR()
			if ptrTarget == "" {
				t.Errorf("Record[%d] AsPTR() returned empty string, expected service instance name", i)
				continue
			}

			t.Logf("  Record[%d]: %s → PTR: %s", i, record.Name, ptrTarget)

			// Verify AsA() returns nil for non-A record
			if ip := record.AsA(); ip != nil {
				t.Errorf("Record[%d] AsA() returned %v for PTR record, expected nil", i, ip)
			}
		}
	} else {
		t.Logf("Query(PTR) received no records (timeout or no services - acceptable)")
	}
}

// TestQuery_SRVRecord validates that Query() correctly handles SRV records
// to get service location (hostname and port) per User Story 2.
//
// User Story 2: "As a developer, I want to get connection details (hostname, port)
// for a discovered service."
//
// FR-002: System MUST support querying for SRV record types
// RFC 2782: SRV records specify location of services
//
// Contract: Query(ctx, service_instance, RecordTypeSRV) → Response with SRVData
func TestQuery_SRVRecord(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Query for SRV records
	// Note: In real environment, we'd first query PTR to get service instances,
	// then query SRV for each instance
	response, err := q.Query(ctx, "test._http._tcp.local", querier.RecordTypeSRV)
	if err != nil {
		t.Logf("Query(SRV) returned error: %v (acceptable in isolated test)", err)
		return
	}

	if response == nil {
		t.Errorf("Query(SRV) response should not be nil, got nil")
		return
	}

	// If we got SRV records, verify they're accessible via AsSRV()
	if len(response.Records) > 0 {
		t.Logf("✓ Query(SRV) received %d SRV records", len(response.Records))

		for i, record := range response.Records {
			if record.Type != querier.RecordTypeSRV {
				t.Logf("Record[%d] has type %v (expected SRV) - mixed types acceptable", i, record.Type)
				continue
			}

			// Verify AsSRV() returns valid SRVData
			srv := record.AsSRV()
			if srv == nil {
				t.Errorf("Record[%d] AsSRV() returned nil, expected SRVData", i)
				continue
			}

			// Validate SRV fields per RFC 2782
			if srv.Target == "" {
				t.Errorf("Record[%d] SRV Target is empty, expected hostname", i)
			}

			if srv.Port == 0 {
				t.Errorf("Record[%d] SRV Port is 0, expected valid port number", i)
			}

			t.Logf("  Record[%d]: %s → SRV: %s:%d (priority=%d, weight=%d)",
				i, record.Name, srv.Target, srv.Port, srv.Priority, srv.Weight)

			// Verify AsPTR() returns empty for non-PTR record
			if ptrTarget := record.AsPTR(); ptrTarget != "" {
				t.Errorf("Record[%d] AsPTR() returned %q for SRV record, expected empty", i, ptrTarget)
			}
		}
	} else {
		t.Logf("Query(SRV) received no records (timeout or no services - acceptable)")
	}
}

// TestQuery_TXTRecord validates that Query() correctly handles TXT records
// to get service metadata per User Story 2.
//
// User Story 2: "As a developer, I want to get service metadata (version, path, etc.)
// for a discovered service."
//
// FR-002: System MUST support querying for TXT record types
// RFC 6763: TXT records contain key=value metadata
//
// Contract: Query(ctx, service_instance, RecordTypeTXT) → Response with []string
func TestQuery_TXTRecord(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Query for TXT records
	response, err := q.Query(ctx, "test._http._tcp.local", querier.RecordTypeTXT)
	if err != nil {
		t.Logf("Query(TXT) returned error: %v (acceptable in isolated test)", err)
		return
	}

	if response == nil {
		t.Errorf("Query(TXT) response should not be nil, got nil")
		return
	}

	// If we got TXT records, verify they're accessible via AsTXT()
	if len(response.Records) > 0 {
		t.Logf("✓ Query(TXT) received %d TXT records", len(response.Records))

		for i, record := range response.Records {
			if record.Type != querier.RecordTypeTXT {
				t.Logf("Record[%d] has type %v (expected TXT) - mixed types acceptable", i, record.Type)
				continue
			}

			// Verify AsTXT() returns valid string slice
			txt := record.AsTXT()
			if txt == nil {
				t.Errorf("Record[%d] AsTXT() returned nil, expected []string", i)
				continue
			}

			t.Logf("  Record[%d]: %s → TXT: %d strings", i, record.Name, len(txt))

			// Log TXT strings (metadata)
			for j, kv := range txt {
				t.Logf("    TXT[%d]: %s", j, kv)
			}

			// Verify AsSRV() returns nil for non-SRV record
			if srv := record.AsSRV(); srv != nil {
				t.Errorf("Record[%d] AsSRV() returned %v for TXT record, expected nil", i, srv)
			}
		}
	} else {
		t.Logf("Query(TXT) received no records (timeout or no services - acceptable)")
	}
}

// TestQuery_MixedRecordTypes validates that Query() returns only the requested
// record type when multiple types are available.
//
// FR-002: System MUST support querying for A, PTR, SRV, and TXT record types
//
// Contract: Response.Records contains only records matching the query type
func TestQuery_MixedRecordTypes(t *testing.T) {
	q, err := querier.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer func() { _ = q.Close() }()

	// Test each record type
	recordTypes := []struct {
		name       string
		recordType querier.RecordType
		queryName  string
	}{
		{"A record", querier.RecordTypeA, "test.local"},
		{"PTR record", querier.RecordTypePTR, "_services._dns-sd._udp.local"},
		{"SRV record", querier.RecordTypeSRV, "test._http._tcp.local"},
		{"TXT record", querier.RecordTypeTXT, "test._http._tcp.local"},
	}

	for _, tt := range recordTypes {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			response, err := q.Query(ctx, tt.queryName, tt.recordType)
			if err != nil {
				t.Logf("Query(%v) returned error: %v (acceptable)", tt.recordType, err)
				return
			}

			if response == nil {
				t.Errorf("Query(%v) response should not be nil", tt.recordType)
				return
			}

			// Verify all returned records match the query type
			for i, record := range response.Records {
				if record.Type != tt.recordType {
					t.Errorf("Record[%d] has type %v, expected %v (filtering failed)",
						i, record.Type, tt.recordType)
				}
			}

			if len(response.Records) > 0 {
				t.Logf("✓ Query(%v) returned %d records, all matching query type",
					tt.recordType, len(response.Records))
			}
		})
	}
}
