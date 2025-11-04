package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joshuafuller/beacon/internal/security"
)

// TestMulticastStormSimulation validates rate limiter protects against multicast storms.
// Per SC-005: CPU <20% under 1000 qps storm.
// Per SC-006: Cooldown applied within 1 second of threshold breach.
func TestMulticastStormSimulation(t *testing.T) {
	// Skip if short tests requested
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create rate limiter with default settings (100 qps threshold, 60s cooldown)
	rl := security.NewRateLimiter(100, 60*time.Second, 10000)

	floodingIP := "192.168.1.200"
	legitimateIP := "192.168.1.50"

	// Phase 1: Simulate multicast storm (1000 queries/second from flooding source)
	stormDuration := 2 * time.Second
	stormStart := time.Now()

	floodingBlocked := 0
	floodingAllowed := 0

	// Send queries at ~1000 qps for 2 seconds
	ctx, cancel := context.WithTimeout(context.Background(), stormDuration)
	defer cancel()

	ticker := time.NewTicker(1 * time.Millisecond) // ~1000 qps
	defer ticker.Stop()

	cooldownDetectedAt := time.Time{}

	for {
		select {
		case <-ctx.Done():
			goto stormComplete
		case <-ticker.C:
			allowed := rl.Allow(floodingIP)
			if allowed {
				floodingAllowed++
			} else {
				floodingBlocked++
				// Record when cooldown first triggered
				if cooldownDetectedAt.IsZero() {
					cooldownDetectedAt = time.Now()
				}
			}
		}
	}

stormComplete:

	// SC-006: Verify cooldown applied within 1 second of storm start
	if cooldownDetectedAt.IsZero() {
		t.Error("Expected cooldown to be triggered during storm, but no queries were blocked")
	} else {
		cooldownDelay := cooldownDetectedAt.Sub(stormStart)
		if cooldownDelay > 1*time.Second {
			t.Errorf("SC-006 FAIL: Cooldown took %v to apply (expected <1s)", cooldownDelay)
		} else {
			t.Logf("SC-006 PASS: Cooldown applied in %v", cooldownDelay)
		}
	}

	// Verify most storm queries were blocked
	if floodingBlocked == 0 {
		t.Error("Expected flooding source to be rate limited, but no queries were blocked")
	}

	t.Logf("Storm results: %d allowed, %d blocked from flooding source", floodingAllowed, floodingBlocked)

	// Phase 2: Verify legitimate traffic from different source continues to be processed
	legitimateAllowed := 0
	for i := 0; i < 50; i++ {
		if rl.Allow(legitimateIP) {
			legitimateAllowed++
		}
	}

	if legitimateAllowed < 45 { // Allow some margin
		t.Errorf("Expected legitimate source to be allowed, but only %d/50 queries succeeded", legitimateAllowed)
	} else {
		t.Logf("Legitimate traffic: %d/50 queries allowed", legitimateAllowed)
	}

	// Phase 3: Verify flooding source remains blocked during cooldown
	floodingStillBlocked := 0
	for i := 0; i < 10; i++ {
		if !rl.Allow(floodingIP) {
			floodingStillBlocked++
		}
	}

	if floodingStillBlocked < 9 { // Should block nearly all
		t.Errorf("Expected flooding source to remain blocked during cooldown, but only %d/10 were blocked", floodingStillBlocked)
	}

	// SC-005: CPU usage verification (manual observation or profiling)
	// Note: Automated CPU measurement requires runtime profiling, typically done in benchmark tests
	t.Log("SC-005: CPU usage should be monitored via go test -cpuprofile")
}

// TestRateLimiterConcurrentAccess validates rate limiter handles concurrent queries safely.
// Per data-model.md: RWMutex allows concurrent reads for hot path performance.
func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := security.NewRateLimiter(100, 60*time.Second, 10000)

	// Simulate concurrent queries from multiple goroutines
	numGoroutines := 10
	queriesPerGoroutine := 100

	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			sourceIP := fmt.Sprintf("192.168.1.%d", goroutineID)
			for i := 0; i < queriesPerGoroutine; i++ {
				rl.Allow(sourceIP)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify no panics or race conditions occurred
	// (This test primarily validates via -race flag)
	t.Log("Concurrent access test completed without panics")
}
