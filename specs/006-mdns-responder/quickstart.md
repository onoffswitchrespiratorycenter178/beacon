# Quickstart Guide: mDNS Responder

**Feature**: 006-mdns-responder
**Created**: 2025-11-02
**Audience**: Developers integrating mDNS Responder into their applications

---

## Overview

This quickstart guide provides practical examples for using the Beacon mDNS Responder. The Responder enables your Go applications to advertise services on the local network, making them discoverable by browsers like Avahi (Linux), Bonjour (macOS), and DNS-SD Browser (iOS).

**What You'll Learn**:
- Basic service registration
- Handling name conflicts
- Updating service metadata
- Multi-service registration
- Graceful shutdown
- Production deployment patterns

---

## Installation

```bash
# Install Beacon (after M2 release)
go get github.com/joshuafuller/beacon/responder
```

---

## Quick Start (5 Minutes)

### Example 1: Register a Web Server

The simplest use case: register an HTTP service so other devices can discover your web server.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    // Step 1: Create responder
    r, err := responder.New()
    if err != nil {
        log.Fatalf("Failed to create responder: %v", err)
    }
    defer r.Close()

    // Step 2: Define service
    service := &responder.Service{
        InstanceName: "My Web Server",           // User-friendly name
        ServiceType:  "_http._tcp.local",        // Standard HTTP service type
        Port:         8080,                      // Port where HTTP server listens
        TXTRecords:   []string{"path=/"},        // Optional metadata
    }

    // Step 3: Register service
    ctx := context.Background()
    err = r.Register(ctx, service)
    if err != nil {
        log.Fatalf("Registration failed: %v", err)
    }

    log.Println("✓ Service registered - discoverable on the network")

    // Step 4: Start HTTP server
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from mDNS!")
    })

    go http.ListenAndServe(":8080", nil)

    // Step 5: Wait for interrupt signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)
    <-sigCh

    log.Println("Shutting down...")
}
```

**Run the example**:
```bash
go run main.go
```

**Verify discovery**:
- **Linux (Avahi)**: `avahi-browse -r _http._tcp`
- **macOS**: Open "Bonjour Browser" app
- **iOS**: Install "Discovery - DNS-SD Browser" app

You should see "My Web Server" appear within 2 seconds.

---

## Common Scenarios

### Scenario 1: SSH Server Registration

Register an SSH service for remote access discovery.

```go
package main

import (
    "context"
    "log"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    r, _ := responder.New()
    defer r.Close()

    sshService := &responder.Service{
        InstanceName: "MyHost SSH",
        ServiceType:  "_ssh._tcp.local",
        Port:         22,
        TXTRecords: []string{
            "u=john",         // Username hint
            "os=linux",       // Operating system
        },
    }

    err := r.Register(context.Background(), sshService)
    if err != nil {
        log.Fatalf("Failed to register SSH service: %v", err)
    }

    log.Println("SSH service registered - clients can auto-discover this host")

    // Keep running
    select {}
}
```

**Test**:
```bash
# Discover SSH services
avahi-browse -r _ssh._tcp

# Auto-connect (if client supports mDNS)
ssh john@MyHost-SSH.local
```

---

### Scenario 2: API Server with Version Metadata

Register a REST API with version information in TXT records.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"

    "github.com/joshuafuller/beacon/responder"
)

const apiVersion = "v2.1.0"

func main() {
    r, _ := responder.New()
    defer r.Close()

    apiService := &responder.Service{
        InstanceName: "MyApp API",
        ServiceType:  "_http._tcp.local",
        Port:         3000,
        TXTRecords: []string{
            "api_version=" + apiVersion,
            "path=/api/v2",
            "auth=bearer",
            "format=json",
        },
    }

    err := r.Register(context.Background(), apiService)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("API %s registered at http://myhost.local:3000/api/v2", apiVersion)

    http.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, apiVersion)
    })

    http.ListenAndServe(":3000", nil)
}
```

**Benefits**:
- Clients can discover API endpoints without hardcoded IPs
- TXT records provide API contract information (version, auth type, response format)
- Zero-config service discovery in microservices architectures

---

### Scenario 3: Multiple Services (Web + SSH)

Register multiple services from a single application.

```go
package main

import (
    "context"
    "log"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    r, _ := responder.New()
    defer r.Close()

    // Register HTTP service
    httpService := &responder.Service{
        InstanceName: "MyApp Web",
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

    // Register custom service
    customService := &responder.Service{
        InstanceName: "MyApp Control",
        ServiceType:  "_myapp-ctl._tcp.local", // Custom service type
        Port:         9999,
        TXTRecords:   []string{"protocol=v1"},
    }
    r.Register(context.Background(), customService)

    log.Println("3 services registered")

    select {} // Keep running
}
```

**Test**:
```bash
# List all services from this host
avahi-browse -a | grep MyApp

# Should show:
# + eth0 IPv4 MyApp Web          _http._tcp           local
# + eth0 IPv4 MyApp SSH          _ssh._tcp            local
# + eth0 IPv4 MyApp Control      _myapp-ctl._tcp      local
```

---

### Scenario 4: Dynamic TXT Record Updates

Update service metadata without re-registering (e.g., version bumps, status changes).

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
        TXTRecords:   []string{"version=1.0.0", "status=starting"},
    }

    r.Register(context.Background(), service)
    log.Println("Service registered with version=1.0.0")

    // Simulate application startup
    time.Sleep(5 * time.Second)

    // Update status to ready
    err := r.UpdateService("MyApp", []string{
        "version=1.0.0",
        "status=ready",
        "uptime=5s",
    })
    if err != nil {
        log.Fatalf("Failed to update TXT records: %v", err)
    }

    log.Println("TXT records updated - status=ready")

    // Simulate version upgrade
    time.Sleep(10 * time.Second)

    err = r.UpdateService("MyApp", []string{
        "version=2.0.0",
        "status=ready",
        "uptime=15s",
    })

    log.Println("TXT records updated - version=2.0.0")

    select {}
}
```

**Key Points**:
- `UpdateService()` does NOT re-probe (fast, no 1.75s delay)
- TXT updates are immediate
- Useful for health status, metrics, feature flags

---

### Scenario 5: Graceful Shutdown

Send goodbye packets before exit to immediately remove service from browsers.

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    r, _ := responder.New()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }

    r.Register(context.Background(), service)
    log.Println("Service registered")

    // Setup signal handler
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

    <-sigCh
    log.Println("Shutdown signal received")

    // Graceful shutdown sequence:
    // 1. Stop accepting new requests
    // 2. Wait for in-flight requests (up to 5s)
    // 3. Send goodbye packets
    // 4. Close responder

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Simulate draining connections
    select {
    case <-time.After(2 * time.Second):
        log.Println("All connections drained")
    case <-shutdownCtx.Done():
        log.Println("Shutdown timeout - forcing exit")
    }

    // Close responder (sends goodbye packets)
    err := r.Close()
    if err != nil {
        log.Printf("Warning: Close() failed: %v", err)
    }

    log.Println("Goodbye packets sent - service removed from browsers")
}
```

**Test**:
```bash
# Terminal 1: Run app
go run main.go

# Terminal 2: Monitor services
watch -n 1 'avahi-browse _http._tcp --resolve --terminate'

# Terminal 1: Press Ctrl+C
# Terminal 2: Service disappears within 1 second (goodbye packet received)
```

---

### Scenario 6: Custom Hostname

Use a custom hostname instead of system hostname.

```go
package main

import (
    "context"
    "log"
    "net"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    // Custom hostname for all services registered by this responder
    r, err := responder.New(
        responder.WithHostname("my-custom-host.local"),
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

    log.Println("Service registered with hostname: my-custom-host.local")
    log.Println("Clients will resolve: http://my-custom-host.local:8080")

    select {}
}
```

**Use Cases**:
- Multiple services on same host need different DNS names
- Branding (e.g., "printer-1234.local" for printer appliances)
- Testing (avoid conflicts with system hostname)

---

### Scenario 7: Interface Selection

Bind responder to specific network interfaces (e.g., exclude Wi-Fi, only use Ethernet).

```go
package main

import (
    "context"
    "log"
    "net"

    "github.com/joshuafuller/beacon/responder"
)

func main() {
    // Get Ethernet interface
    eth0, err := net.InterfaceByName("eth0")
    if err != nil {
        log.Fatalf("Failed to find eth0: %v", err)
    }

    // Create responder on specific interface
    r, err := responder.New(
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

    log.Println("Service registered on eth0 only (not Wi-Fi)")

    select {}
}
```

**Alternative: Custom Filter**:
```go
// Only use interfaces starting with "en" (macOS Ethernet)
r, err := responder.New(
    responder.WithInterfaceFilter(func(iface net.Interface) bool {
        return strings.HasPrefix(iface.Name, "en")
    }),
)
```

---

### Scenario 8: Error Handling

Robust error handling for production deployments.

```go
package main

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/joshuafuller/beacon/responder"
)

func registerServiceWithRetry(r *responder.Responder, service *responder.Service) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := r.Register(ctx, service)
    if err != nil {
        switch {
        case errors.Is(err, responder.ErrServiceAlreadyRegistered):
            // Already registered - update TXT records instead
            log.Println("Service already registered, updating TXT records")
            return r.UpdateService(service.InstanceName, service.TXTRecords)

        case errors.Is(err, responder.ErrMaxConflicts):
            // Too many conflicts - suggest different name
            log.Printf("Name '%s' conflicts with 10+ services, choose a more unique name", service.InstanceName)
            return err

        case errors.Is(err, context.DeadlineExceeded):
            // Timeout - retry once
            log.Println("Registration timeout, retrying...")
            retryCtx, retryCancel := context.WithTimeout(context.Background(), 10*time.Second)
            defer retryCancel()
            return r.Register(retryCtx, service)

        case errors.Is(err, responder.ErrInvalidServiceType):
            // Invalid service type - fix and retry
            log.Printf("Invalid service type '%s', must be '_<service>._tcp.local'", service.ServiceType)
            return err

        default:
            // Unknown error
            log.Printf("Unexpected error: %v", err)
            return err
        }
    }

    return nil
}

func main() {
    r, err := responder.New()
    if err != nil {
        log.Fatalf("Failed to create responder: %v", err)
    }
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }

    err = registerServiceWithRetry(r, service)
    if err != nil {
        log.Fatalf("Failed to register after retry: %v", err)
    }

    log.Println("Service registered successfully")
    select {}
}
```

---

## Production Patterns

### Pattern 1: Health Check via TXT Records

```go
type HealthStatus string

const (
    HealthStarting HealthStatus = "starting"
    HealthReady    HealthStatus = "ready"
    HealthDegraded HealthStatus = "degraded"
    HealthFailing  HealthStatus = "failing"
)

func updateHealthStatus(r *responder.Responder, instanceName string, status HealthStatus) error {
    txtRecords := []string{
        "status=" + string(status),
        "timestamp=" + time.Now().Format(time.RFC3339),
    }

    return r.UpdateService(instanceName, txtRecords)
}

func main() {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
    }

    r.Register(context.Background(), service)
    updateHealthStatus(r, "MyApp", HealthStarting)

    // Simulate startup
    time.Sleep(5 * time.Second)
    updateHealthStatus(r, "MyApp", HealthReady)

    // Monitor health in background
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            if isHealthy() {
                updateHealthStatus(r, "MyApp", HealthReady)
            } else {
                updateHealthStatus(r, "MyApp", HealthDegraded)
            }
        }
    }()

    select {}
}

func isHealthy() bool {
    // Check database connection, disk space, etc.
    return true
}
```

---

### Pattern 2: Multi-Instance Load Balancing

```go
// Register multiple instances of the same service on different ports
// Browsers will discover all instances (built-in load balancing)

func main() {
    r, _ := responder.New()
    defer r.Close()

    // Instance 1
    r.Register(context.Background(), &responder.Service{
        InstanceName: "MyApp Instance 1",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
        TXTRecords:   []string{"instance=1", "cpu=30%"},
    })

    // Instance 2
    r.Register(context.Background(), &responder.Service{
        InstanceName: "MyApp Instance 2",
        ServiceType:  "_http._tcp.local",
        Port:         8081,
        TXTRecords:   []string{"instance=2", "cpu=25%"},
    })

    log.Println("2 instances registered - clients can choose based on CPU")

    select {}
}
```

**Client Discovery**:
```bash
avahi-browse _http._tcp --resolve

# Output:
# + eth0 IPv4 MyApp Instance 1  _http._tcp  local  8080  cpu=30%
# + eth0 IPv4 MyApp Instance 2  _http._tcp  local  8081  cpu=25%

# Client can connect to least-loaded instance (8081)
```

---

### Pattern 3: Service Deprecation

```go
// Announce service deprecation via TXT records before removal

func main() {
    r, _ := responder.New()
    defer r.Close()

    service := &responder.Service{
        InstanceName: "MyApp Legacy API",
        ServiceType:  "_http._tcp.local",
        Port:         8080,
        TXTRecords:   []string{"version=1.0", "status=active"},
    }

    r.Register(context.Background(), service)

    // 30 days later: Mark as deprecated
    time.Sleep(30 * 24 * time.Hour) // Simulated
    r.UpdateService("MyApp Legacy API", []string{
        "version=1.0",
        "status=deprecated",
        "sunset_date=2025-12-31",
        "migration_url=https://docs.example.com/api-v2",
    })

    log.Println("Legacy API marked as deprecated - clients warned")

    // 60 days later: Remove service
    time.Sleep(30 * 24 * time.Hour)
    r.Unregister("MyApp Legacy API")

    log.Println("Legacy API unregistered - clients must use v2")
}
```

---

## Debugging

### Enable Debug Logging

```bash
# Set MDNS_DEBUG=1 for verbose logging (implementation-dependent)
export MDNS_DEBUG=1
go run main.go
```

**Expected Output**:
```
[DEBUG] Transport: Binding to 0.0.0.0:5353
[DEBUG] Transport: Joined multicast group 224.0.0.251
[DEBUG] StateMachine: MyApp → PROBING (probe 1/3)
[DEBUG] StateMachine: MyApp → PROBING (probe 2/3)
[DEBUG] StateMachine: MyApp → PROBING (probe 3/3)
[DEBUG] StateMachine: MyApp → ANNOUNCING (announcement 1/2)
[DEBUG] StateMachine: MyApp → ANNOUNCING (announcement 2/2)
[DEBUG] StateMachine: MyApp → ESTABLISHED
[INFO]  Service registered: MyApp._http._tcp.local
```

---

### Monitor Network Traffic

```bash
# Capture mDNS packets (port 5353)
sudo tcpdump -i any -n port 5353

# Filter for your service name
sudo tcpdump -i any -n port 5353 | grep "MyApp"
```

**Expected Packets**:
- **T+0ms**: Probe #1 (query for "MyApp._http._tcp.local")
- **T+250ms**: Probe #2
- **T+500ms**: Probe #3
- **T+750ms**: Announcement #1 (response with PTR, SRV, TXT, A)
- **T+1750ms**: Announcement #2

---

### Verify Service Discovery

**Linux (Avahi)**:
```bash
# Browse all HTTP services
avahi-browse -r _http._tcp

# Resolve specific service
avahi-resolve --name MyApp._http._tcp.local
```

**macOS**:
```bash
# List all services
dns-sd -B _http._tcp local

# Resolve specific service
dns-sd -L "MyApp" _http._tcp local
```

**Windows**:
```powershell
# Use Bonjour Browser (GUI) or dns-sd.exe (if Apple Bonjour SDK installed)
dns-sd.exe -B _http._tcp local
```

---

## Troubleshooting

### Problem 1: Service Not Discoverable

**Symptoms**: Service registered successfully, but browsers don't see it.

**Causes**:
1. **Firewall blocking multicast**: Check UDP port 5353 and multicast group 224.0.0.251
2. **VPN/Docker interfering**: VPN often blocks multicast traffic
3. **Network isolation**: Devices on different VLANs/subnets

**Solutions**:
```bash
# Check firewall (Linux)
sudo iptables -L | grep 5353

# Allow mDNS traffic
sudo iptables -A INPUT -p udp --dport 5353 -j ACCEPT
sudo iptables -A OUTPUT -p udp --dport 5353 -j ACCEPT

# Check if multicast is working
ping 224.0.0.251  # Should reach other mDNS devices

# Test from another device
avahi-browse -a  # Should show your service
```

---

### Problem 2: Name Conflicts

**Symptoms**: Service renamed to "MyApp (2)" automatically.

**Cause**: Another device already has a service named "MyApp".

**Solutions**:
1. **Use unique names**: Include hostname or UUID (e.g., "MyApp-hostname-12345")
2. **Accept renaming**: The " (2)" suffix is RFC-compliant behavior
3. **Check existing services**: `avahi-browse -a` before registration

---

### Problem 3: Goodbye Packets Not Sent

**Symptoms**: Service lingers in browsers for 120 seconds after app exit.

**Cause**: App crashed or killed without calling `Close()`.

**Solutions**:
```go
// ALWAYS use defer r.Close()
func main() {
    r, _ := responder.New()
    defer r.Close()  // ← Ensures goodbye packets sent on panic/exit

    // ... rest of code
}

// Handle SIGKILL gracefully
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
go func() {
    <-sigCh
    r.Close()
    os.Exit(0)
}()
```

---

## Next Steps

1. **Read API Contract**: See `contracts/responder-api.md` for full method documentation
2. **Read State Machine Contract**: See `contracts/state-machine.md` for lifecycle details
3. **Run Examples**: Check `examples/` directory for more code samples
4. **Integration Testing**: Use Apple Bonjour Conformance Test (BCT) to validate behavior

---

## FAQ

**Q: Can I register services on port 0 (dynamic port)?**
A: No, port must be 1-65535. If using dynamic ports, call `listener.Addr()` to get assigned port before registration.

**Q: Can I use IPv6?**
A: Not in M2 (006-mdns-responder). IPv6 support (FF02::FB) is planned for M3.

**Q: Do I need root/admin privileges?**
A: No, Beacon uses SO_REUSEPORT to share port 5353 with Avahi/Bonjour (no conflicts).

**Q: Can I register services without a hostname?**
A: Yes, defaults to system hostname + ".local". Use `WithHostname()` to customize.

**Q: What happens if network disconnects?**
A: Responder continues running. When network reconnects, services are re-announced automatically (M3 feature).

**Q: Can I register 100+ services?**
A: Yes, NFR-003 guarantees support for ≥100 concurrent services.

---

**Status**: Quickstart guide complete and ready for developers
