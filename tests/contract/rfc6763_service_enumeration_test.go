package contract

import (
	"context"
	"testing"

	"github.com/joshuafuller/beacon/responder"
)

// TestRFC6763_ServiceEnumeration_MetaQuery tests RFC 6763 §9 service type enumeration.
//
// TDD Phase: RED - This test will FAIL until _services._dns-sd._udp.local support is implemented
//
// RFC 6763 §9: "A DNS query for PTR records with the name '_services._dns-sd._udp.<Domain>'
// yields a set of PTR records, where the rdata of each PTR record is the two-label <Service>
// name, plus the same domain, e.g., '_http._tcp.<Domain>'."
//
// FR-027: System MUST respond to "_services._dns-sd._udp.local" PTR queries with a list
// of all registered service types
//
// T103: Contract test for service enumeration
func TestRFC6763_ServiceEnumeration_MetaQuery(t *testing.T) {
	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register 3 services with DIFFERENT service types
	services := []*responder.Service{
		{
			InstanceName: "Web Server",
			ServiceType:  "_http._tcp.local",
			Port:         8080,
		},
		{
			InstanceName: "SSH Server",
			ServiceType:  "_ssh._tcp.local",
			Port:         22,
		},
		{
			InstanceName: "FTP Server",
			ServiceType:  "_ftp._tcp.local",
			Port:         21,
		},
	}

	for _, svc := range services {
		err = r.Register(svc)
		if err != nil {
			t.Fatalf("Register(%q) error = %v, want nil", svc.InstanceName, err)
		}
	}

	// RFC 6763 §9: Test that responder can enumerate service types
	// This requires implementing:
	// 1. Recognizing "_services._dns-sd._udp.local" as a meta-query
	// 2. Collecting unique service types from registry
	// 3. Returning PTR records pointing to each service type
	//
	// For now, this is a placeholder test that documents the requirement.
	// Full implementation requires:
	// - T107: Implement _services._dns-sd._udp.local response in ResponseBuilder
	// - DNS message serialization (to send actual query/response over wire)

	// TODO: Once query/response mechanism is fully wired:
	// 1. Send PTR query for "_services._dns-sd._udp.local"
	// 2. Verify response contains 3 PTR records
	// 3. Verify each PTR record points to a registered service type

	t.Skip("Deferred until _services._dns-sd._udp.local response implementation (T107) and DNS message serialization")
}

// TestRFC6763_ServiceEnumeration_DuplicateTypes tests that duplicate service types
// only appear once in enumeration response.
//
// TDD Phase: RED
//
// RFC 6763 §9: Service type enumeration lists unique service types, not instances.
// If 3 services all use "_http._tcp.local", enumeration should list "_http._tcp.local" once.
//
// T103: Edge case - duplicate service types
func TestRFC6763_ServiceEnumeration_DuplicateTypes(t *testing.T) {
	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// Register 3 services with SAME service type
	services := []*responder.Service{
		{
			InstanceName: "Web Server 1",
			ServiceType:  "_http._tcp.local",
			Port:         8080,
		},
		{
			InstanceName: "Web Server 2",
			ServiceType:  "_http._tcp.local",
			Port:         8081,
		},
		{
			InstanceName: "Web Server 3",
			ServiceType:  "_http._tcp.local",
			Port:         8082,
		},
	}

	for _, svc := range services {
		err = r.Register(svc)
		if err != nil {
			t.Fatalf("Register(%q) error = %v, want nil", svc.InstanceName, err)
		}
	}

	// RFC 6763 §9: Should return exactly 1 PTR record for "_http._tcp.local"
	// NOT 3 PTR records (one per instance)
	// Expected behavior: _services._dns-sd._udp.local query → 1 PTR → "_http._tcp.local"

	t.Skip("Deferred until _services._dns-sd._udp.local response implementation (T107) and DNS message serialization")
}

// TestRFC6763_ServiceEnumeration_EmptyRegistry tests enumeration when no services registered.
//
// TDD Phase: RED
//
// RFC 6763 §9: If no services are registered, enumeration query should return empty response.
//
// T103: Edge case - empty registry
func TestRFC6763_ServiceEnumeration_EmptyRegistry(t *testing.T) {
	ctx := context.Background()
	r, err := responder.New(ctx)
	if err != nil {
		t.Fatalf("responder.New() error = %v, want nil", err)
	}
	defer func() { _ = r.Close() }()

	// No services registered - empty registry
	// RFC 6763 §9: Query for "_services._dns-sd._udp.local" should return empty response

	t.Skip("Deferred until _services._dns-sd._udp.local response implementation (T107) and DNS message serialization")
}
