// Package querier provides a high-level API for mDNS (.local) service discovery.
//
// # Overview
//
// The querier package implements Multicast DNS (mDNS) per RFC 6762, enabling
// discovery of services and devices on the local network using .local hostnames.
//
// # Quick Start
//
// Discover a device by name:
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "log"
//	    "time"
//
//	    "github.com/joshuafuller/beacon/querier"
//	)
//
//	func main() {
//	    // Create querier
//	    q, err := querier.New()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer q.Close()
//
//	    // Query for A record with 1-second timeout
//	    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
//	    defer cancel()
//
//	    response, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // Print discovered IPs
//	    for _, record := range response.Records {
//	        if ip := record.AsA(); ip != nil {
//	            fmt.Printf("Found printer at %s\n", ip)
//	        }
//	    }
//	}
//
// # Service Discovery
//
// Discover services by type using PTR records:
//
//	// Discover all HTTP services
//	response, err := q.Query(ctx, "_http._tcp.local", querier.RecordTypePTR)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, record := range response.Records {
//	    if target := record.AsPTR(); target != "" {
//	        fmt.Printf("Found HTTP service: %s\n", target)
//	    }
//	}
//
// # Service Details
//
// Get service location (hostname and port) using SRV records:
//
//	response, err := q.Query(ctx, "webserver._http._tcp.local", querier.RecordTypeSRV)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, record := range response.Records {
//	    if srv := record.AsSRV(); srv != nil {
//	        fmt.Printf("Service at %s:%d (priority=%d, weight=%d)\n",
//	            srv.Target, srv.Port, srv.Priority, srv.Weight)
//	    }
//	}
//
// # Service Metadata
//
// Get service metadata using TXT records:
//
//	response, err := q.Query(ctx, "webserver._http._tcp.local", querier.RecordTypeTXT)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, record := range response.Records {
//	    if txt := record.AsTXT(); txt != nil {
//	        for _, kv := range txt {
//	            fmt.Printf("Metadata: %s\n", kv)
//	        }
//	    }
//	}
//
// # Configuration
//
// Customize query timeout using functional options:
//
//	q, err := querier.New(querier.WithTimeout(2 * time.Second))
//
// # Supported Record Types
//
// The querier supports four DNS record types per RFC 1035/RFC 6762:
//
//   - RecordTypeA (1): IPv4 address records
//   - RecordTypePTR (12): Pointer records (service discovery)
//   - RecordTypeSRV (33): Service records (hostname and port)
//   - RecordTypeTXT (16): Text records (service metadata)
//
// # Error Handling
//
// The querier returns typed errors for specific failure modes:
//
//   - ValidationError: Invalid input (empty name, oversized name, invalid characters)
//   - NetworkError: Network failures (socket creation, send/receive errors)
//   - context.Canceled: Context was canceled
//   - context.DeadlineExceeded: Timeout occurred (this is NOT an error - returns empty response)
//
// Example error handling:
//
//	response, err := q.Query(ctx, name, querier.RecordTypeA)
//	if err != nil {
//	    var validationErr *errors.ValidationError
//	    if errors.As(err, &validationErr) {
//	        // Handle validation error (bad input)
//	        fmt.Printf("Invalid input: %v\n", err)
//	        return
//	    }
//	    // Handle other errors
//	    return err
//	}
//
// # Concurrency
//
// Querier is safe for concurrent use. Multiple goroutines can call Query()
// simultaneously on the same Querier instance.
//
// # Resource Management
//
// Always call Close() to release resources:
//
//	q, err := querier.New()
//	if err != nil {
//	    return err
//	}
//	defer q.Close() // Critical: releases UDP socket and stops background goroutines
//
// # Timeout Behavior
//
// Timeout is NOT an error. Query() returns all responses collected within the
// timeout window. An empty response means no devices responded within the timeout.
//
//	response, err := q.Query(ctx, "device.local", querier.RecordTypeA)
//	if err != nil {
//	    // Real error (validation, network, cancellation)
//	    return err
//	}
//
//	if len(response.Records) == 0 {
//	    // No responses received (timeout - not an error)
//	    fmt.Println("No devices found")
//	} else {
//	    // Process responses
//	    for _, record := range response.Records {
//	        // ...
//	    }
//	}
//
// # RFC Compliance
//
// This implementation follows:
//   - RFC 6762: Multicast DNS
//   - RFC 1035: Domain Names - Implementation and Specification
//   - RFC 2782: A DNS RR for specifying the location of services (DNS SRV)
//
// # Limitations (M1 - Basic mDNS Querier)
//
//   - IPv4 only (no IPv6/AAAA records)
//   - Query-only (no mDNS responder functionality)
//   - No Known Answer suppression (RFC 6762 §7.1)
//   - No continuous monitoring (one-shot queries only)
//   - Authority and Additional sections ignored
//
// # Performance
//
// Success Criteria (SC-002): The querier discovers 95% of responding devices
// within 1 second on typical local networks.
//
// # Thread Safety
//
// All public methods are goroutine-safe and can be called concurrently.
package querier
