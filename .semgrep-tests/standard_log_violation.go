// Quick test for beacon-standard-log-usage rule
// This file is NOT a test file, so rule should trigger
package test

import "log"

func BadLogUsageForTesting() {
	// Should trigger: beacon-standard-log-usage
	log.Println("Starting service")
	log.Printf("Message: %s", "test")
}
