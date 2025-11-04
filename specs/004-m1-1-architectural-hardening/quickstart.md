# Quickstart: M1.1 Architectural Hardening

**Phase 1 Design** | **Date**: 2025-11-01
**Branch**: `004-m1-1-architectural-hardening`

## Overview

This quickstart guide demonstrates M1.1 features: Avahi/Bonjour coexistence, VPN privacy protection, rate limiting, and custom interface selection. Choose the scenario that matches your use case.

---

## Scenario 1: Default (Smart Interface Selection + Rate Limiting)

**Use Case**: Most applications - "just works" with secure defaults

**Features**:
- ✅ Excludes VPN interfaces (utun*, tun*, ppp*, wg*, tailscale*)
- ✅ Excludes Docker interfaces (docker0, veth*, br-*)
- ✅ Rate limiting enabled (100 qps per source IP, 60s cooldown)
- ✅ Coexists with Avahi/Bonjour on port 5353

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier with smart defaults
    q, err := querier.New()
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query for a .local hostname
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil {
        fmt.Println("No responses received (timeout or no devices)")
        return
    }

    // Print results
    fmt.Printf("Found %d A records for printer.local:\n", len(response.Records))
    for _, record := range response.Records {
        if record.Type == querier.RecordTypeA {
            ip := record.AsA()
            fmt.Printf("  %s → %s (TTL: %ds)\n", record.Name, ip, record.TTL)
        }
    }
}
```

### Expected Output

```text
Found 1 A records for printer.local:
  printer.local → 192.168.1.100 (TTL: 120s)
```

### What Happens Internally

1. **Interface Selection**: Automatically selects physical network interfaces (eth0, en0, wlan0)
2. **VPN Exclusion**: Skips utun0, tun0, tailscale0 (privacy protection)
3. **Docker Exclusion**: Skips docker0, veth* (performance optimization)
4. **Socket Options**: Sets SO_REUSEADDR + SO_REUSEPORT (Linux/macOS) for Avahi/Bonjour coexistence
5. **Rate Limiting**: Tracks query rate per source IP, drops packets from sources exceeding 100 qps
6. **Source Validation**: Drops packets from non-link-local sources (RFC 6762 §2 compliance)

---

## Scenario 2: Explicit Interface Selection (eth0 only)

**Use Case**: Server deployment — query only on specific interface

**Why**: Precise control over which interface sends mDNS queries (e.g., bind to LAN interface, skip WAN)

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Find eth0 interface
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatalf("Failed to list interfaces: %v", err)
    }

    var eth0 net.Interface
    found := false
    for _, iface := range ifaces {
        if iface.Name == "eth0" {
            eth0 = iface
            found = true
            break
        }
    }

    if !found {
        log.Fatal("eth0 interface not found")
    }

    // Create querier with explicit interface selection
    q, err := querier.New(
        querier.WithInterfaces([]net.Interface{eth0}),
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query for service discovery
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "_http._tcp.local", querier.RecordTypePTR)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No HTTP services found on eth0")
        return
    }

    // Print discovered services
    fmt.Printf("Found %d HTTP services on eth0:\n", len(response.Records))
    for _, record := range response.Records {
        if record.Type == querier.RecordTypePTR {
            target := record.AsPTR()
            fmt.Printf("  %s\n", target)
        }
    }
}
```

### Expected Output

```text
Found 2 HTTP services on eth0:
  webserver._http._tcp.local
  api._http._tcp.local
```

---

## Scenario 3: Custom Interface Filter (Ethernet only)

**Use Case**: Filter interfaces by pattern (e.g., allow only wired Ethernet, skip WiFi)

**Why**: Application-specific requirements (e.g., prefer wired over wireless)

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "strings"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier with custom filter (Ethernet only)
    q, err := querier.New(
        querier.WithInterfaceFilter(func(iface net.Interface) bool {
            // Allow only Ethernet interfaces
            // Linux: eth*, en* (systemd predictable names: enp*)
            // macOS: en* (en0 = Ethernet, en1 = WiFi)
            name := iface.Name
            return strings.HasPrefix(name, "eth") ||
                   strings.HasPrefix(name, "en") ||
                   strings.HasPrefix(name, "enp")
        }),
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query for devices
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "router.local", querier.RecordTypeA)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No response from router.local on Ethernet interfaces")
        return
    }

    // Print results
    for _, record := range response.Records {
        if record.Type == querier.RecordTypeA {
            ip := record.AsA()
            fmt.Printf("Router IP (Ethernet): %s\n", ip)
        }
    }
}
```

### Expected Output

```text
Router IP (Ethernet): 192.168.1.1
```

---

## Scenario 4: VPN Override (Allow VPN Interface)

**Use Case**: Query on VPN interface (override default VPN exclusion)

**Why**: Intentionally query devices on VPN network (e.g., corporate VPN with mDNS services)

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "strings"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Find tailscale0 interface
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatalf("Failed to list interfaces: %v", err)
    }

    var vpnIface net.Interface
    found := false
    for _, iface := range ifaces {
        if strings.HasPrefix(iface.Name, "tailscale") {
            vpnIface = iface
            found = true
            break
        }
    }

    if !found {
        log.Fatal("Tailscale interface not found")
    }

    // Override default VPN exclusion - explicitly allow VPN interface
    q, err := querier.New(
        querier.WithInterfaces([]net.Interface{vpnIface}),
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query on VPN network
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "corp-server.local", querier.RecordTypeA)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No response from corp-server.local on VPN")
        return
    }

    // Print results
    for _, record := range response.Records {
        if record.Type == querier.RecordTypeA {
            ip := record.AsA()
            fmt.Printf("Corporate server IP (VPN): %s\n", ip)
        }
    }
}
```

### Expected Output

```text
Corporate server IP (VPN): 100.64.1.50
```

**⚠️ Privacy Note**: Querying on VPN interfaces may leak mDNS queries to VPN provider. Only use when intentional.

---

## Scenario 5: Rate Limiting Configuration

**Use Case**: Tune rate limiting for specific network characteristics

**Why**: Stricter protection (untrusted networks) or relaxed limits (high-volume applications)

### Code (Stricter Limits for Untrusted Network)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier with stricter rate limiting
    q, err := querier.New(
        querier.WithRateLimit(true),                        // Enable rate limiting
        querier.WithRateLimitThreshold(50),                 // 50 qps (stricter)
        querier.WithRateLimitCooldown(90 * time.Second),    // 90s cooldown
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query for devices
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "_services._dns-sd._udp.local", querier.RecordTypePTR)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No services discovered")
        return
    }

    fmt.Printf("Discovered %d services (with strict rate limiting):\n", len(response.Records))
    for _, record := range response.Records {
        if record.Type == querier.RecordTypePTR {
            service := record.AsPTR()
            fmt.Printf("  %s\n", service)
        }
    }
}
```

### Code (Disable Rate Limiting for Testing)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier with rate limiting DISABLED (for testing only)
    q, err := querier.New(
        querier.WithRateLimit(false),
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query without rate limiting
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "test.local", querier.RecordTypeA)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No response from test.local")
        return
    }

    // Print results
    for _, record := range response.Records {
        if record.Type == querier.RecordTypeA {
            ip := record.AsA()
            fmt.Printf("Test device IP: %s (no rate limiting)\n", ip)
        }
    }
}
```

**⚠️ Security Note**: Disabling rate limiting removes protection against multicast storms. Only use in controlled test environments.

---

## Scenario 6: Full Configuration (Production Deployment)

**Use Case**: Production server deployment with comprehensive hardening

**Features**:
- ✅ Explicit interface selection (eth0 only)
- ✅ Stricter rate limiting (50 qps, 90s cooldown)
- ✅ All security features enabled

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Find eth0 interface
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatalf("Failed to list interfaces: %v", err)
    }

    var eth0 net.Interface
    found := false
    for _, iface := range ifaces {
        if iface.Name == "eth0" {
            eth0 = iface
            found = true
            break
        }
    }

    if !found {
        log.Fatal("eth0 interface not found")
    }

    // Create querier with full production configuration
    q, err := querier.New(
        // Interface: eth0 only (skip WiFi, VPN, Docker)
        querier.WithInterfaces([]net.Interface{eth0}),

        // Rate limiting: Strict protection
        querier.WithRateLimit(true),
        querier.WithRateLimitThreshold(50),              // 50 qps
        querier.WithRateLimitCooldown(90 * time.Second), // 90s cooldown
    )
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    // Query for devices on production network
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    response, err := q.Query(ctx, "printer.local", querier.RecordTypeA)
    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }

    if response == nil || len(response.Records) == 0 {
        fmt.Println("No response from printer.local on eth0")
        return
    }

    // Print results
    fmt.Println("Production query results:")
    for _, record := range response.Records {
        if record.Type == querier.RecordTypeA {
            ip := record.AsA()
            fmt.Printf("  Printer IP: %s (TTL: %ds)\n", ip, record.TTL)
        }
    }
}
```

### Expected Output

```text
Production query results:
  Printer IP: 192.168.1.100 (TTL: 120s)
```

---

## Scenario 7: Service Discovery (PTR + SRV + TXT)

**Use Case**: Discover services with full metadata (service name, host, port, TXT attributes)

**Why**: Complete service discovery workflow (DNS-SD per RFC 6763)

### Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/joshuafuller/beacon/querier"
)

func main() {
    // Create querier with defaults
    q, err := querier.New()
    if err != nil {
        log.Fatalf("Failed to create querier: %v", err)
    }
    defer q.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Step 1: Discover HTTP services (PTR query)
    fmt.Println("Step 1: Discovering HTTP services...")
    ptrResponse, err := q.Query(ctx, "_http._tcp.local", querier.RecordTypePTR)
    if err != nil {
        log.Fatalf("PTR query failed: %v", err)
    }

    if ptrResponse == nil || len(ptrResponse.Records) == 0 {
        fmt.Println("No HTTP services found")
        return
    }

    for _, record := range ptrResponse.Records {
        if record.Type == querier.RecordTypePTR {
            serviceName := record.AsPTR()
            fmt.Printf("  Found service: %s\n", serviceName)

            // Step 2: Query SRV record for host and port
            srvCtx, srvCancel := context.WithTimeout(context.Background(), 1*time.Second)
            srvResponse, err := q.Query(srvCtx, serviceName, querier.RecordTypeSRV)
            srvCancel()

            if err == nil && srvResponse != nil && len(srvResponse.Records) > 0 {
                for _, srvRecord := range srvResponse.Records {
                    if srvRecord.Type == querier.RecordTypeSRV {
                        srv := srvRecord.AsSRV()
                        if srv != nil {
                            fmt.Printf("    Host: %s, Port: %d\n", srv.Target, srv.Port)
                        }
                    }
                }
            }

            // Step 3: Query TXT record for metadata
            txtCtx, txtCancel := context.WithTimeout(context.Background(), 1*time.Second)
            txtResponse, err := q.Query(txtCtx, serviceName, querier.RecordTypeTXT)
            txtCancel()

            if err == nil && txtResponse != nil && len(txtResponse.Records) > 0 {
                for _, txtRecord := range txtResponse.Records {
                    if txtRecord.Type == querier.RecordTypeTXT {
                        txt := txtRecord.AsTXT()
                        if txt != nil && len(txt) > 0 {
                            fmt.Printf("    Metadata:\n")
                            for _, kv := range txt {
                                fmt.Printf("      %s\n", kv)
                            }
                        }
                    }
                }
            }
        }
    }
}
```

### Expected Output

```text
Step 1: Discovering HTTP services...
  Found service: webserver._http._tcp.local
    Host: webserver.local, Port: 8080
    Metadata:
      path=/
      version=1.0
  Found service: api._http._tcp.local
    Host: api.local, Port: 3000
    Metadata:
      path=/api/v1
      auth=bearer
```

---

## Common Patterns

### Pattern 1: List Available Interfaces

```go
package main

import (
    "fmt"
    "log"
    "net"
)

func main() {
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatalf("Failed to list interfaces: %v", err)
    }

    fmt.Println("Available network interfaces:")
    for _, iface := range ifaces {
        // Check flags
        up := iface.Flags&net.FlagUp != 0
        multicast := iface.Flags&net.FlagMulticast != 0
        loopback := iface.Flags&net.FlagLoopback != 0

        flags := ""
        if up { flags += "UP " }
        if multicast { flags += "MULTICAST " }
        if loopback { flags += "LOOPBACK " }

        fmt.Printf("  %s: %s [%s]\n", iface.Name, iface.HardwareAddr, flags)
    }
}
```

**Output**:
```text
Available network interfaces:
  lo: [LOOPBACK UP MULTICAST]
  eth0: 00:1a:2b:3c:4d:5e [UP MULTICAST]
  wlan0: 00:1a:2b:3c:4d:5f [UP MULTICAST]
  utun0: [UP]
  docker0: 02:42:ac:11:00:01 [UP MULTICAST]
```

### Pattern 2: Check Interface Addresses

```go
package main

import (
    "fmt"
    "log"
    "net"
)

func main() {
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatalf("Failed to list interfaces: %v", err)
    }

    for _, iface := range ifaces {
        addrs, err := iface.Addrs()
        if err != nil {
            continue
        }

        if len(addrs) > 0 {
            fmt.Printf("%s:\n", iface.Name)
            for _, addr := range addrs {
                fmt.Printf("  %s\n", addr.String())
            }
        }
    }
}
```

**Output**:
```text
eth0:
  192.168.1.50/24
  fe80::1a:2bff:fe3c:4d5e/64
wlan0:
  192.168.1.51/24
  fe80::1a:2bff:fe3c:4d5f/64
```

---

## Troubleshooting

### Issue: "address already in use" on Linux

**Cause**: System daemon (Avahi, systemd-resolved) already listening on port 5353

**Solution**: M1.1 automatically sets SO_REUSEPORT on Linux 3.9+ to enable coexistence

**Verify**:
```bash
# Check if Avahi is running
systemctl status avahi-daemon

# Check kernel version (must be >= 3.9)
uname -r
```

**If kernel <3.9**: Upgrade kernel or stop system daemon before using Beacon

---

### Issue: No responses from devices on network

**Cause 1**: VPN interface excluded by default
**Solution**: Use `WithInterfaces()` to explicitly allow VPN interface (see Scenario 4)

**Cause 2**: Firewall blocking multicast (224.0.0.251:5353)
**Solution**: Allow UDP port 5353 inbound/outbound

**Cause 3**: Devices not responding to mDNS queries
**Solution**: Verify devices are mDNS-capable (ping device.local works)

---

### Issue: Rate limiting blocking legitimate traffic

**Cause**: Application sends >100 queries/second
**Solution**: Increase threshold via `WithRateLimitThreshold()` (see Scenario 5)

**Verify**:
```go
// Check if rate limiting is the issue
q, _ := querier.New(querier.WithRateLimit(false))
// If queries succeed now, adjust threshold
```

---

## Next Steps

1. **Review** `spec.md` for functional requirements and success criteria
2. **Review** `data-model.md` for internal implementation details
3. **Review** `contracts/querier-options.md` for complete API documentation
4. **Run** integration tests to validate Avahi coexistence (SC-001)
5. **Generate** tasks via `/speckit.tasks` to begin implementation
