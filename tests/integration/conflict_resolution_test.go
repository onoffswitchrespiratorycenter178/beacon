package integration

import (
	"context"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/responder"
)

// TestConflictResolution_TwoServicesRename tests RFC 6762 §9 conflict resolution.
//
// Scenario: Two responders attempt to register services with same name
// Expected: Second service automatically renames to "MyService-2"
//
// RFC 6762 §9: "If a host receives a response containing a record that conflicts
// with one of its unique records, the host MUST immediately rename the record."
//
// T066: Integration test for User Story 2 (Name Conflict Resolution)
func TestConflictResolution_TwoServicesRename(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create two responders
	r1, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder 1: %v", err)
	}
	defer r1.Close()

	r2, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder 2: %v", err)
	}
	defer r2.Close()

	// Register first service
	service1 := &responder.Service{
		InstanceName: "MyService",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	err = r1.Register(service1)
	if err != nil {
		t.Fatalf("Failed to register service1: %v", err)
	}

	// Currently this test will SKIP because full conflict detection
	// requires US3 (Response to Queries) to be implemented.
	//
	// For now, we validate the rename mechanism works via unit tests.
	// Full integration test will work once US3 is complete.
	t.Skip("Full conflict resolution requires US3 (Response to Queries)")

	// Attempt to register second service with SAME name
	// This should trigger conflict detection and automatic rename
	// service2 := &responder.Service{
	// 	InstanceName: "MyService", // Same name - will conflict
	// 	ServiceType:  "_http._tcp.local",
	// 	Port:         8081,
	// }

	// Future implementation (when US3 complete):
	// err = r2.Register(service2)
	// if err != nil {
	// 	t.Fatalf("Failed to register service2: %v", err)
	// }
	//
	// // Verify service2 was renamed to "MyService-2"
	// if service2.InstanceName != "MyService-2" {
	// 	t.Errorf("service2.InstanceName = %q, want %q", service2.InstanceName, "MyService-2")
	// }
}

// TestConflictResolution_MaxRenameAttempts tests max rename limit per RFC 6762 §9.
//
// Scenario: Conflict loop hits maximum rename attempts (10)
// Expected: Registration fails with clear error message
//
// FR-032: System MUST handle registration failures gracefully
//
// T066: Integration test for max rename attempts (T062)
func TestConflictResolution_MaxRenameAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("Failed to create responder: %v", err)
	}
	defer r.Close()

	// Inject conflict to force rename loop
	r.InjectConflictDuringProbing(true)

	service := &responder.Service{
		InstanceName: "MyService",
		ServiceType:  "_http._tcp.local",
		Port:         8080,
	}

	// This should fail after 10 rename attempts
	err = r.Register(service)
	if err == nil {
		t.Fatal("Register() succeeded, want error after max rename attempts")
	}

	// Verify error message mentions max attempts
	wantError := "max rename attempts"
	if !contains(err.Error(), wantError) {
		t.Errorf("Register() error = %q, want error containing %q", err.Error(), wantError)
	}
}

// contains checks if string s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
