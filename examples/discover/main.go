// discover is a simple example demonstrating the Beacon mDNS library.
//
// This program discovers devices and services on your local network using mDNS.
//
// Usage:
//
//	go run examples/discover/main.go
//
// Example output:
//
//	Discovering devices on local network...
//
//	Found device: printer.local → 192.168.1.100
//	Found device: macbook.local → 192.168.1.50
//
//	Discovering services...
//
//	HTTP Services:
//	- My Printer._http._tcp.local (192.168.1.100:80)
//	- Home Server._http._tcp.local (192.168.1.50:8080)
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joshuafuller/beacon/querier"
)

func main() {
	// Create a querier instance (starts the mDNS listener)
	q, err := querier.New()
	if err != nil {
		log.Fatalf("Failed to create querier: %v", err)
	}
	defer q.Close()

	fmt.Println("🔍 Discovering devices on local network...")

	// Discover devices using DNS-SD service discovery
	discoverServices(q)

	fmt.Println("\n---")

	// Look for specific devices (printers, file servers, etc.)
	findSpecificDevices(q)
}

// discoverServices finds all advertised services on the network
func discoverServices(q *querier.Querier) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Query for all services (_services._dns-sd._udp.local returns a list of service types)
	response, err := q.Query(ctx, "_services._dns-sd._udp.local", querier.RecordTypePTR)
	if err != nil {
		log.Printf("Service discovery failed: %v", err)
		return
	}

	if len(response.Records) == 0 {
		fmt.Println("No services found (you may need to enable mDNS on your devices)")
		return
	}

	fmt.Printf("📡 Found %d service type(s):\n", len(response.Records))
	for _, record := range response.Records {
		serviceType := record.AsPTR()
		if serviceType != "" {
			fmt.Printf("  • %s\n", serviceType)
		}
	}
}

// findSpecificDevices looks for common device types
func findSpecificDevices(q *querier.Querier) {
	// Common service types to look for
	services := []struct {
		name  string
		query string
		emoji string
	}{
		{"Printers", "_printer._tcp.local", "🖨️"},
		{"HTTP Services", "_http._tcp.local", "🌐"},
		{"SSH Servers", "_ssh._tcp.local", "🔒"},
		{"File Servers", "_smb._tcp.local", "📁"},
	}

	for _, svc := range services {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		response, err := q.Query(ctx, svc.query, querier.RecordTypePTR)
		cancel()

		if err != nil || len(response.Records) == 0 {
			continue // Skip if no results
		}

		fmt.Printf("\n%s %s:\n", svc.emoji, svc.name)
		for _, record := range response.Records {
			instanceName := record.AsPTR()
			if instanceName != "" {
				// Get more details (SRV record for hostname and port)
				details := getServiceDetails(q, instanceName)
				fmt.Printf("  • %s%s\n", instanceName, details)
			}
		}
	}
}

// getServiceDetails retrieves hostname and port for a service
func getServiceDetails(q *querier.Querier, instanceName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Query for SRV record (contains hostname and port)
	response, err := q.Query(ctx, instanceName, querier.RecordTypeSRV)
	if err != nil || len(response.Records) == 0 {
		return ""
	}

	for _, record := range response.Records {
		if srv := record.AsSRV(); srv != nil {
			return fmt.Sprintf(" → %s:%d", srv.Target, srv.Port)
		}
	}

	return ""
}
