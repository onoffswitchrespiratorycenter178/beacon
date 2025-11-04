# API Contract: Querier Functional Options

**Phase 1 Design** | **Date**: 2025-11-01
**Branch**: `004-m1-1-architectural-hardening`

## Overview

This document defines the public API functional options for M1.1 interface selection and rate limiting configuration. All options follow the standard Go functional options pattern used in M1.

**Package**: `querier` (public API)

---

## Functional Options Pattern

### Pattern Definition

```go
// Option is a functional option for configuring a Querier.
type Option func(*Querier) error

// New creates a new Querier with the given options.
func New(opts ...Option) (*Querier, error) {
    q := &Querier{
        // Defaults
        rateLimitEnabled: true,
        rateLimitThreshold: 100,
        rateLimitCooldown: 60 * time.Second,
    }

    for _, opt := range opts {
        if err := opt(q); err != nil {
            return nil, err
        }
    }

    return q, nil
}
```

---

## Option 1: WithInterfaces

### Signature

```go
// WithInterfaces configures the Querier to use only the specified interfaces.
// This overrides the default interface selection logic (which excludes VPN/Docker).
//
// Use this when you need explicit control over which interfaces send mDNS queries.
//
// Example:
//   ifaces, _ := net.Interfaces()
//   eth0 := ifaces[0]
//   q, _ := querier.New(querier.WithInterfaces([]net.Interface{eth0}))
//
// If the provided list is empty, New() returns an error.
func WithInterfaces(ifaces []net.Interface) Option
```

### Behavior

**Input**: `[]net.Interface` — Explicit list of interfaces to use
**Effect**: Querier binds ONLY to these interfaces (skips default filtering logic)
**Validation**: Returns error if list is empty
**Priority**: Overrides `WithInterfaceFilter()` if both are specified

### Implementation Contract

```go
func WithInterfaces(ifaces []net.Interface) Option {
    return func(q *Querier) error {
        if len(ifaces) == 0 {
            return &errors.ValidationError{
                Field:   "interfaces",
                Value:   ifaces,
                Message: "interface list cannot be empty",
            }
        }

        // Set explicit interface list (overrides filter)
        q.interfaceFilter = &interfaceFilter{
            explicit: ifaces,
            filter:   nil, // Explicit list takes priority
        }

        return nil
    }
}
```

### Use Cases

1. **Testing**: Bind to specific interface for deterministic tests
2. **Server Deployment**: Bind to eth0 only (skip wlan0, virtual interfaces)
3. **Multi-Homed Systems**: Query on WAN interface only (skip LAN interfaces)

### Example

```go
// Use only eth0 for mDNS queries
ifaces, err := net.Interfaces()
if err != nil {
    log.Fatal(err)
}

var eth0 net.Interface
for _, iface := range ifaces {
    if iface.Name == "eth0" {
        eth0 = iface
        break
    }
}

q, err := querier.New(querier.WithInterfaces([]net.Interface{eth0}))
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Option 2: WithInterfaceFilter

### Signature

```go
// WithInterfaceFilter configures the Querier with a custom interface selection filter.
// The filter function is called for each available interface; return true to include.
//
// This option is ignored if WithInterfaces() is also specified (explicit list takes priority).
//
// Example (allow only Ethernet interfaces):
//   q, _ := querier.New(querier.WithInterfaceFilter(func(iface net.Interface) bool {
//       return strings.HasPrefix(iface.Name, "eth")
//   }))
//
// Default behavior (if neither WithInterfaces nor WithInterfaceFilter is specified):
//   Excludes VPN (utun*, tun*, ppp*, wg*, tailscale*, wireguard*)
//   Excludes Docker (docker0, veth*, br-*)
//   Excludes loopback, down, and non-multicast interfaces
func WithInterfaceFilter(filter func(net.Interface) bool) Option
```

### Behavior

**Input**: `func(net.Interface) bool` — Custom filter function
**Effect**: Querier uses this function to select interfaces
**Default**: `DefaultInterfaces()` logic (exclude VPN/Docker)
**Priority**: Overridden by `WithInterfaces()` if both are specified

### Implementation Contract

```go
func WithInterfaceFilter(filter func(net.Interface) bool) Option {
    return func(q *Querier) error {
        if filter == nil {
            return &errors.ValidationError{
                Field:   "interfaceFilter",
                Value:   nil,
                Message: "filter function cannot be nil",
            }
        }

        // Set custom filter (if explicit list not set)
        if q.interfaceFilter == nil || q.interfaceFilter.explicit == nil {
            q.interfaceFilter = &interfaceFilter{
                explicit: nil,
                filter:   filter,
            }
        }

        return nil
    }
}
```

### Use Cases

1. **Custom VPN Exclusion**: Exclude enterprise VPN with non-standard name (e.g., "corp-vpn0")
2. **Interface Type Filtering**: Allow only wired Ethernet (skip WiFi)
3. **Prefix Filtering**: Allow only interfaces starting with "en" (macOS Ethernet)

### Example

```go
// Allow only Ethernet interfaces (skip WiFi, VPN, Docker)
q, err := querier.New(querier.WithInterfaceFilter(func(iface net.Interface) bool {
    // Ethernet interfaces typically start with "eth" (Linux) or "en" (macOS)
    name := iface.Name
    return strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en")
}))
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Option 3: WithRateLimit

### Signature

```go
// WithRateLimit enables or disables rate limiting.
// Rate limiting protects against multicast storms by tracking per-source-IP query rates.
//
// Default: Enabled (true)
//
// When enabled, sources exceeding the threshold (default: 100 qps) are rate-limited
// for a cooldown period (default: 60 seconds).
//
// Example (disable rate limiting for testing):
//   q, _ := querier.New(querier.WithRateLimit(false))
func WithRateLimit(enabled bool) Option
```

### Behavior

**Input**: `bool` — Enable (true) or disable (false) rate limiting
**Default**: Enabled (`true`)
**Effect**:
- If `true`: Create `RateLimiter` with default threshold (100 qps) and cooldown (60s)
- If `false`: Skip rate limiter creation (accept all packets)

### Implementation Contract

```go
func WithRateLimit(enabled bool) Option {
    return func(q *Querier) error {
        q.rateLimitEnabled = enabled
        return nil
    }
}
```

### Use Cases

1. **Production**: Enable (default) to protect against multicast storms
2. **Testing**: Disable for deterministic tests without rate limiting
3. **Trusted Networks**: Disable on isolated test networks

### Example

```go
// Disable rate limiting for integration tests
q, err := querier.New(querier.WithRateLimit(false))
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Option 4: WithRateLimitThreshold

### Signature

```go
// WithRateLimitThreshold sets the query rate threshold (queries per second per source IP).
// Sources exceeding this threshold are rate-limited for the cooldown period.
//
// Default: 100 queries/second (balances storm protection with legitimate high-volume use)
//
// This option is ignored if rate limiting is disabled via WithRateLimit(false).
//
// Example (stricter threshold for untrusted networks):
//   q, _ := querier.New(querier.WithRateLimitThreshold(50))
func WithRateLimitThreshold(threshold int) Option
```

### Behavior

**Input**: `int` — Max queries per second per source IP
**Default**: 100 qps
**Validation**: Must be > 0
**Effect**: Sources exceeding threshold are rate-limited

### Implementation Contract

```go
func WithRateLimitThreshold(threshold int) Option {
    return func(q *Querier) error {
        if threshold <= 0 {
            return &errors.ValidationError{
                Field:   "rateLimitThreshold",
                Value:   threshold,
                Message: "threshold must be greater than 0",
            }
        }

        q.rateLimitThreshold = threshold
        return nil
    }
}
```

### Use Cases

1. **Stricter Protection**: Lower threshold (e.g., 50 qps) for untrusted networks
2. **Relaxed Protection**: Higher threshold (e.g., 200 qps) for high-volume applications
3. **Tuning**: Adjust based on observed network behavior

### Example

```go
// Stricter rate limiting (50 qps) for untrusted network
q, err := querier.New(
    querier.WithRateLimit(true),
    querier.WithRateLimitThreshold(50),
)
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Option 5: WithRateLimitCooldown

### Signature

```go
// WithRateLimitCooldown sets the duration to drop packets from a source after
// it exceeds the rate limit threshold.
//
// Default: 60 seconds (long enough to detect persistent storms, short enough to recover)
//
// This option is ignored if rate limiting is disabled via WithRateLimit(false).
//
// Example (shorter cooldown for transient storms):
//   q, _ := querier.New(querier.WithRateLimitCooldown(30 * time.Second))
func WithRateLimitCooldown(cooldown time.Duration) Option
```

### Behavior

**Input**: `time.Duration` — Duration to drop packets after threshold exceeded
**Default**: 60 seconds
**Validation**: Must be > 0
**Effect**: Rate-limited sources are blocked for this duration

### Implementation Contract

```go
func WithRateLimitCooldown(cooldown time.Duration) Option {
    return func(q *Querier) error {
        if cooldown <= 0 {
            return &errors.ValidationError{
                Field:   "rateLimitCooldown",
                Value:   cooldown,
                Message: "cooldown must be greater than 0",
            }
        }

        q.rateLimitCooldown = cooldown
        return nil
    }
}
```

### Use Cases

1. **Persistent Storms**: Longer cooldown (e.g., 120s) for persistent misbehavior
2. **Transient Storms**: Shorter cooldown (e.g., 30s) for transient issues
3. **Tuning**: Adjust based on network characteristics

### Example

```go
// Longer cooldown (2 minutes) for persistent storms
q, err := querier.New(
    querier.WithRateLimit(true),
    querier.WithRateLimitThreshold(100),
    querier.WithRateLimitCooldown(120 * time.Second),
)
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Option Combinations and Priority

### Priority Rules

1. **Interface Selection**:
   - `WithInterfaces()` > `WithInterfaceFilter()` > `DefaultInterfaces()`
   - If `WithInterfaces()` is specified, `WithInterfaceFilter()` is ignored

2. **Rate Limiting**:
   - If `WithRateLimit(false)`, threshold and cooldown options are ignored
   - If `WithRateLimit(true)`, use specified threshold/cooldown or defaults

### Example: Full Configuration

```go
// Comprehensive configuration for production deployment
q, err := querier.New(
    // Interface selection: Use only eth0
    querier.WithInterfaces([]net.Interface{eth0}),

    // Rate limiting: Enabled with stricter threshold
    querier.WithRateLimit(true),
    querier.WithRateLimitThreshold(50),           // 50 qps (stricter)
    querier.WithRateLimitCooldown(90 * time.Second), // 90s cooldown
)
if err != nil {
    log.Fatal(err)
}
defer q.Close()
```

---

## Error Handling

### Validation Errors

All functional options return `error` if validation fails. Common errors:

#### WithInterfaces

```go
// ERROR: Empty interface list
q, err := querier.New(querier.WithInterfaces([]net.Interface{}))
// err: ValidationError{Field: "interfaces", Message: "interface list cannot be empty"}
```

#### WithInterfaceFilter

```go
// ERROR: Nil filter function
q, err := querier.New(querier.WithInterfaceFilter(nil))
// err: ValidationError{Field: "interfaceFilter", Message: "filter function cannot be nil"}
```

#### WithRateLimitThreshold

```go
// ERROR: Zero threshold
q, err := querier.New(querier.WithRateLimitThreshold(0))
// err: ValidationError{Field: "rateLimitThreshold", Message: "threshold must be greater than 0"}
```

#### WithRateLimitCooldown

```go
// ERROR: Zero cooldown
q, err := querier.New(querier.WithRateLimitCooldown(0))
// err: ValidationError{Field: "rateLimitCooldown", Message: "cooldown must be greater than 0"}
```

### Error Types

All validation errors use the `errors.ValidationError` type defined in `internal/errors`:

```go
type ValidationError struct {
    Field   string      // Name of the invalid field
    Value   interface{} // Invalid value
    Message string      // Human-readable error message
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s: %s (value: %v)", e.Field, e.Message, e.Value)
}
```

---

## Backward Compatibility

### M1 Compatibility

All new options are **additive** (do not break existing M1 API):

**M1 Code** (still works):
```go
q, err := querier.New() // Uses defaults (VPN/Docker exclusion, rate limiting enabled)
```

**M1.1 Code** (new options available):
```go
q, err := querier.New(
    querier.WithInterfaces([]net.Interface{eth0}),
    querier.WithRateLimit(true),
)
```

### Default Behavior Changes

| Setting | M1 Behavior | M1.1 Behavior | Breaking Change? |
|---------|-------------|---------------|------------------|
| Interfaces | All (including VPN/Docker) | Exclude VPN/Docker | ⚠️ **YES** (intentional hardening) |
| Rate Limiting | N/A | Enabled (100 qps, 60s cooldown) | ✅ **NO** (new feature) |

**Migration Path** (if user needs old behavior):
```go
// M1: Bind to all interfaces (including VPN/Docker)
// M1.1: Use WithInterfaceFilter to allow all
q, err := querier.New(querier.WithInterfaceFilter(func(iface net.Interface) bool {
    return iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0
}))
```

---

## Testing Contracts

### Unit Tests (per option)

#### TestWithInterfaces_ValidList

```go
func TestWithInterfaces_ValidList(t *testing.T) {
    ifaces, _ := net.Interfaces()
    if len(ifaces) == 0 {
        t.Skip("No interfaces available")
    }

    q, err := querier.New(querier.WithInterfaces(ifaces[:1]))
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    defer q.Close()

    // Verify querier uses only the specified interface
    // (internal validation, not exposed in public API)
}
```

#### TestWithInterfaces_EmptyList

```go
func TestWithInterfaces_EmptyList(t *testing.T) {
    _, err := querier.New(querier.WithInterfaces([]net.Interface{}))
    if err == nil {
        t.Fatal("Expected validation error for empty interface list")
    }

    var validationErr *errors.ValidationError
    if !errors.As(err, &validationErr) {
        t.Fatalf("Expected ValidationError, got %T", err)
    }

    if validationErr.Field != "interfaces" {
        t.Errorf("Expected Field='interfaces', got %q", validationErr.Field)
    }
}
```

#### TestWithRateLimitThreshold_Invalid

```go
func TestWithRateLimitThreshold_Invalid(t *testing.T) {
    testCases := []int{0, -1, -100}
    for _, threshold := range testCases {
        t.Run(fmt.Sprintf("threshold=%d", threshold), func(t *testing.T) {
            _, err := querier.New(querier.WithRateLimitThreshold(threshold))
            if err == nil {
                t.Fatal("Expected validation error for invalid threshold")
            }

            var validationErr *errors.ValidationError
            if !errors.As(err, &validationErr) {
                t.Fatalf("Expected ValidationError, got %T", err)
            }

            if validationErr.Field != "rateLimitThreshold" {
                t.Errorf("Expected Field='rateLimitThreshold', got %q", validationErr.Field)
            }
        })
    }
}
```

### Integration Tests

#### TestQuerier_AvahiCoexistence (SC-001)

```go
// Validate WithInterfaces allows coexistence with Avahi on Linux
func TestQuerier_AvahiCoexistence(t *testing.T) {
    if runtime.GOOS != "linux" {
        t.Skip("Avahi coexistence test only runs on Linux")
    }

    // Check if Avahi is running (listening on port 5353)
    // If not, skip test (requires Avahi installed)

    q, err := querier.New() // Default options (VPN/Docker excluded)
    if err != nil {
        t.Fatalf("Querier initialization failed with Avahi running: %v", err)
    }
    defer q.Close()

    // Query should succeed without "address already in use" error
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    _, err = q.Query(ctx, "_services._dns-sd._udp.local", querier.RecordTypePTR)
    if err != nil && err != context.DeadlineExceeded {
        t.Errorf("Query failed with Avahi running: %v", err)
    }

    // SC-001: SUCCESS - Querier coexists with Avahi
    t.Log("✓ SC-001: Querier successfully coexists with Avahi on port 5353")
}
```

---

## Performance Considerations

### Initialization Overhead

| Option | Overhead | Notes |
|--------|----------|-------|
| WithInterfaces | O(1) | Simple assignment |
| WithInterfaceFilter | O(1) | Simple assignment |
| WithRateLimit | O(1) | Boolean flag |
| WithRateLimitThreshold | O(1) | Integer assignment |
| WithRateLimitCooldown | O(1) | Duration assignment |

**Total**: Negligible (<1μs per option)

### Runtime Overhead

| Feature | Per-Query Overhead | Per-Packet Overhead | Notes |
|---------|-------------------|---------------------|-------|
| Interface Filtering | 0 (resolved at init) | 0 | No runtime cost |
| Rate Limiting | 0 | O(1) map lookup | RWMutex allows concurrent reads |

**Hot Path** (per packet received):
```text
1. SourceFilter.IsValid()  — O(n) where n = # subnets (~2-4)
2. RateLimiter.Allow()     — O(1) map lookup
Total: ~100-200ns (negligible compared to parsing)
```

---

## Next Steps

Proceed to `quickstart.md` for developer-friendly usage examples.
