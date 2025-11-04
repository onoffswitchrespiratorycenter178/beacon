# API Contract: Responder Public API

**Feature**: 006-mdns-responder
**Created**: 2025-11-02
**Status**: Contract Complete

---

## Overview

This document defines the public API contract for the `responder` package. The Responder API allows applications to register mDNS services, update metadata, and handle graceful shutdown.

**Design Principles**:
- **Context-aware**: All blocking operations accept `context.Context` (Principle IV)
- **Options pattern**: Functional options for configuration flexibility
- **Error transparency**: Typed errors with clear semantics
- **Thread-safe**: All methods are safe for concurrent use

---

## Package Structure

```
responder/
├── responder.go      # Responder type and core methods
├── service.go        # Service type definition
├── options.go        # Functional options (WithHostname, etc.)
├── errors.go         # Typed errors
└── responder_test.go # API contract tests
```

---

## Types

### Responder

The main type for registering and managing mDNS services.

```go
// Responder manages mDNS service registration, probing, announcing, and query response.
// Responder is safe for concurrent use by multiple goroutines.
//
// A Responder must be created with New() and closed with Close() when done.
type Responder struct {
    // Private fields (not exposed)
}
```

---

### Service

Represents a single mDNS service instance to be registered.

```go
// Service defines an mDNS service instance with name, type, port, and metadata.
type Service struct {
    // InstanceName is the service instance name (e.g., "MyApp").
    // Must be 1-63 characters, UTF-8 encoded.
    // After registration, InstanceName is immutable.
    InstanceName string

    // ServiceType is the DNS-SD service type (e.g., "_http._tcp.local").
    // Must follow RFC 6763 §7 format: _<service>._<proto>.local
    // After registration, ServiceType is immutable.
    ServiceType string

    // Port is the port number where the service is listening (1-65535).
    // After registration, Port is immutable.
    Port uint16

    // TXTRecords contains optional key-value metadata (e.g., "version=1.0").
    // Each record must be ≤255 bytes.
    // Total size of all records must be <8000 bytes.
    // TXTRecords can be updated after registration via UpdateService().
    TXTRecords []string

    // Hostname is the hostname for SRV and A records (e.g., "myhost.local").
    // If empty, defaults to the system hostname + ".local".
    // After registration, Hostname is immutable.
    Hostname string
}
```

---

## Constructor

### New

Creates a new Responder instance.

```go
// New creates a new Responder with the given options.
//
// Options:
//   - WithHostname(hostname string): Set default hostname for all services
//   - WithInterfaces(interfaces []net.Interface): Bind to specific interfaces
//   - WithInterfaceFilter(filter func(net.Interface) bool): Custom interface filter
//   - WithRateLimit(enabled bool): Enable/disable rate limiting (default: true)
//
// Returns an error if:
//   - Transport initialization fails (network unavailable)
//   - Invalid options provided
//
// Example:
//   r, err := responder.New(
//       responder.WithHostname("myhost.local"),
//       responder.WithInterfaces([]net.Interface{eth0}),
//   )
func New(opts ...Option) (*Responder, error)
```

**Errors**:
- `ErrTransportFailed`: Network transport initialization failed
- `ErrInvalidOption`: Invalid option provided

**Thread Safety**: Safe to call concurrently.

**Example**:
```go
r, err := responder.New()
if err != nil {
    log.Fatalf("Failed to create responder: %v", err)
}
defer r.Close()
```

---

## Methods

### Register

Registers a new mDNS service. Starts the probing/announcing state machine in a background goroutine.

```go
// Register registers a new mDNS service.
//
// The service will undergo probing (3 queries × 250ms) to detect name conflicts,
// then announce itself (2 announcements × 1s). After ~1.75 seconds, the service
// will be in the Established state and discoverable via mDNS queries.
//
// If a name conflict is detected during probing, the service will be automatically
// renamed (e.g., "MyApp" → "MyApp (2)") and probing will restart.
//
// Parameters:
//   - ctx: Context for cancellation. If cancelled, registration is aborted.
//   - service: Service instance to register. Must pass validation.
//
// Returns an error if:
//   - Service validation fails (invalid type, port, TXT records)
//   - Service with same InstanceName already registered
//   - Context cancelled before registration completes
//   - Probing fails after 10 rename attempts (max conflicts)
//
// Thread Safety: Safe to call concurrently with different services.
//
// Example:
//   service := &Service{
//       InstanceName: "MyApp",
//       ServiceType:  "_http._tcp.local",
//       Port:         8080,
//       TXTRecords:   []string{"version=1.0"},
//   }
//   err := r.Register(ctx, service)
func Register(ctx context.Context, service *Service) error
```

**Errors**:
- `ErrInvalidServiceType`: ServiceType does not match RFC 6763 §7 format
- `ErrInvalidInstanceName`: InstanceName is empty or >63 characters
- `ErrInvalidPort`: Port is 0
- `ErrTXTRecordTooLarge`: Individual TXT record >255 bytes
- `ErrTXTRecordsTooLarge`: Total TXT records >8000 bytes
- `ErrServiceAlreadyRegistered`: Service with same InstanceName exists
- `ErrMaxConflicts`: Failed to find available name after 10 rename attempts
- `context.Canceled`: Context cancelled during registration

**Behavior**:
- **Blocking**: Returns after probing completes (~750ms) or conflict detected
- **Async state machine**: Announcing and Established phases run in background goroutine
- **Automatic renaming**: Handles conflicts transparently

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

service := &Service{
    InstanceName: "MyApp",
    ServiceType:  "_http._tcp.local",
    Port:         8080,
    TXTRecords:   []string{"version=1.0", "path=/api"},
}

err := r.Register(ctx, service)
if err != nil {
    log.Fatalf("Registration failed: %v", err)
}

fmt.Println("Service registered successfully")
```

---

### UpdateService

Updates the TXT records for an already-registered service without re-probing.

```go
// UpdateService updates the TXT records for an already-registered service.
//
// Per RFC 6762, TXT record updates do not require re-probing (only name/type/port changes do).
// Updated TXT records are immediately reflected in query responses.
//
// Parameters:
//   - instanceName: Name of the service to update
//   - txtRecords: New TXT records (replaces existing records)
//
// Returns an error if:
//   - Service with instanceName not found
//   - TXT records fail validation (too large)
//
// Thread Safety: Safe to call concurrently.
//
// Example:
//   err := r.UpdateService("MyApp", []string{"version=2.0", "path=/v2/api"})
func UpdateService(instanceName string, txtRecords []string) error
```

**Errors**:
- `ErrServiceNotFound`: No service with instanceName exists
- `ErrTXTRecordTooLarge`: Individual TXT record >255 bytes
- `ErrTXTRecordsTooLarge`: Total TXT records >8000 bytes

**Behavior**:
- **No re-probing**: TXT updates are immediate (FR-004)
- **Atomic update**: All TXT records replaced atomically

**Example**:
```go
err := r.UpdateService("MyApp", []string{"version=2.0"})
if err != nil {
    log.Printf("Failed to update TXT records: %v", err)
}
```

---

### Unregister

Unregisters a service and sends goodbye packets (TTL=0).

```go
// Unregister unregisters a service and sends goodbye packets.
//
// Per RFC 6762 §10.1, goodbye packets (TTL=0) are sent for all resource records
// to notify browsers to immediately remove the service from their cache.
//
// Parameters:
//   - instanceName: Name of the service to unregister
//
// Returns an error if:
//   - Service with instanceName not found
//   - Goodbye packet send fails (non-fatal - service still removed)
//
// Thread Safety: Safe to call concurrently.
//
// Example:
//   err := r.Unregister("MyApp")
func Unregister(instanceName string) error
```

**Errors**:
- `ErrServiceNotFound`: No service with instanceName exists

**Behavior**:
- **Goodbye packets**: Sent immediately with TTL=0 (FR-014, FR-021)
- **State machine cleanup**: Cancels background goroutine
- **Best-effort**: Goodbye packet failures are logged but don't return error

**Example**:
```go
err := r.Unregister("MyApp")
if err != nil {
    log.Printf("Failed to unregister service: %v", err)
}
```

---

### Close

Closes the Responder, unregistering all services and shutting down the transport layer.

```go
// Close closes the Responder, unregistering all services and shutting down.
//
// All registered services are unregistered with goodbye packets sent.
// The transport layer is closed, releasing network resources.
//
// After Close(), the Responder cannot be reused. Create a new Responder with New().
//
// Returns an error if:
//   - Transport close fails (logged, not critical)
//
// Thread Safety: Safe to call concurrently (idempotent - multiple calls are no-op).
//
// Example:
//   defer r.Close()
func Close() error
```

**Errors**:
- `ErrTransportFailed`: Transport close failed (rare, logged)

**Behavior**:
- **Graceful shutdown**: Sends goodbye packets for all services (FR-021)
- **Idempotent**: Multiple calls are safe (no-op after first)
- **Resource cleanup**: Closes transport, cancels all goroutines

**Example**:
```go
r, err := responder.New()
if err != nil {
    log.Fatal(err)
}
defer r.Close()

// Use responder...
```

---

## Options

Functional options for configuring Responder behavior.

### WithHostname

Sets the default hostname for all registered services.

```go
// WithHostname sets the default hostname for SRV and A records.
//
// If not specified, defaults to system hostname + ".local".
//
// Example:
//   r, err := responder.New(responder.WithHostname("myhost.local"))
func WithHostname(hostname string) Option
```

---

### WithInterfaces

Binds the Responder to specific network interfaces.

```go
// WithInterfaces binds the Responder to specific network interfaces.
//
// If not specified, defaults to all non-VPN, non-Docker interfaces (M1.1 DefaultInterfaces).
//
// Example:
//   eth0, _ := net.InterfaceByName("eth0")
//   r, err := responder.New(responder.WithInterfaces([]net.Interface{*eth0}))
func WithInterfaces(interfaces []net.Interface) Option
```

---

### WithInterfaceFilter

Provides a custom interface filter function.

```go
// WithInterfaceFilter provides a custom function to filter network interfaces.
//
// Example:
//   filter := func(iface net.Interface) bool {
//       return strings.HasPrefix(iface.Name, "en") // Only "en*" interfaces
//   }
//   r, err := responder.New(responder.WithInterfaceFilter(filter))
func WithInterfaceFilter(filter func(net.Interface) bool) Option
```

---

### WithRateLimit

Enables or disables rate limiting for query responses.

```go
// WithRateLimit enables or disables rate limiting (default: true).
//
// When enabled, uses M1.1 per-source-IP rate limiting (100 qps threshold, 60s cooldown).
//
// Example:
//   r, err := responder.New(responder.WithRateLimit(false)) // Disable for testing
func WithRateLimit(enabled bool) Option
```

---

## Errors

All errors are typed for programmatic handling.

```go
var (
    // Service validation errors
    ErrInvalidServiceType      = errors.New("responder: invalid service type format")
    ErrInvalidInstanceName     = errors.New("responder: invalid instance name")
    ErrInvalidPort             = errors.New("responder: invalid port number")
    ErrTXTRecordTooLarge       = errors.New("responder: TXT record exceeds 255 bytes")
    ErrTXTRecordsTooLarge      = errors.New("responder: total TXT records exceed 8000 bytes")

    // Registration errors
    ErrServiceAlreadyRegistered = errors.New("responder: service with this name already registered")
    ErrServiceNotFound          = errors.New("responder: service not found")
    ErrMaxConflicts             = errors.New("responder: max conflict resolution attempts exceeded")

    // Transport errors
    ErrTransportFailed         = errors.New("responder: transport operation failed")
    ErrResponseTooLarge        = errors.New("responder: response packet exceeds 9000 bytes")

    // Option errors
    ErrInvalidOption           = errors.New("responder: invalid option")
)
```

**Error Handling Pattern**:
```go
err := r.Register(ctx, service)
if err != nil {
    if errors.Is(err, responder.ErrServiceAlreadyRegistered) {
        // Handle duplicate registration
    } else if errors.Is(err, context.Canceled) {
        // Handle cancellation
    } else {
        // Handle other errors
    }
}
```

---

## Usage Examples

### Basic Registration

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    // Create responder
    r, err := responder.New()
    if err != nil {
        log.Fatal(err)
    }
    defer r.Close()

    // Register HTTP service
    service := &responder.Service{
        InstanceName: "MyWebServer",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
        TXTRecords:   []string{"path=/", "version=1.0"},
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err = r.Register(ctx, service)
    if err != nil {
        log.Fatalf("Registration failed: %v", err)
    }

    log.Println("Service registered successfully")

    // Keep running
    select {}
}
```

---

### Multiple Services

```go
func main() {
    r, _ := responder.New()
    defer r.Close()

    // Register HTTP service
    httpService := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }
    r.Register(context.Background(), httpService)

    // Register SSH service
    sshService := &responder.Service{
        InstanceName: "MyApp SSH",
        ServiceType:  "_ssh._tcp.local",
        Port:         22,
    }
    r.Register(context.Background(), sshService)

    select {} // Keep running
}
```

---

### Updating TXT Records

```go
func main() {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
        TXTRecords:   []string{"version=1.0"},
    }
    r.Register(context.Background(), service)

    // Update version after deployment
    time.Sleep(10 * time.Second)
    r.UpdateService("MyApp", []string{"version=2.0"})

    select {}
}
```

---

### Graceful Shutdown

```go
func main() {
    r, _ := responder.New()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }
    r.Register(context.Background(), service)

    // Wait for SIGINT
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    // Graceful shutdown - sends goodbye packets
    log.Println("Shutting down...")
    r.Close()
    log.Println("Goodbye packets sent")
}
```

---

### Custom Hostname and Interfaces

```go
func main() {
    eth0, _ := net.InterfaceByName("eth0")

    r, err := responder.New(
        responder.WithHostname("custom-host.local"),
        responder.WithInterfaces([]net.Interface{*eth0}),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }
    r.Register(context.Background(), service)

    select {}
}
```

---

### Error Handling

```go
func registerService(r *responder.Responder, service *responder.Service) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := r.Register(ctx, service)
    if err != nil {
        switch {
        case errors.Is(err, responder.ErrServiceAlreadyRegistered):
            // Update existing service instead
            return r.UpdateService(service.InstanceName, service.TXTRecords)

        case errors.Is(err, responder.ErrMaxConflicts):
            // Too many conflicts - suggest different name
            return fmt.Errorf("name '%s' is too popular, try a more unique name", service.InstanceName)

        case errors.Is(err, context.DeadlineExceeded):
            // Registration timeout
            return fmt.Errorf("registration timeout - network issues?")

        default:
            return err
        }
    }

    return nil
}
```

---

## Testing Contract

### Unit Tests

```go
func TestRegister_ValidService_Success(t *testing.T) {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "Test",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }

    err := r.Register(context.Background(), service)
    assert.NoError(t, err)
}

func TestRegister_DuplicateName_Error(t *testing.T) {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "Test",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }

    r.Register(context.Background(), service)

    err := r.Register(context.Background(), service)
    assert.ErrorIs(t, err, responder.ErrServiceAlreadyRegistered)
}
```

---

## Performance Requirements

- **NFR-001**: Registration completes within 1 second (excluding 750ms probing delay)
- **NFR-002**: Query response latency <100ms (90th percentile)
- **NFR-003**: Support ≥100 concurrent service registrations

---

## Next Steps

With the API contract defined, proceed to:
1. **State Machine Contract** (`contracts/state-machine.md`) - State machine behavior specification
2. **Quickstart Guide** (`quickstart.md`) - End-to-end usage examples

**Status**: API contract ready for implementation
