# Data Model: mDNS Responder

**Feature**: 006-mdns-responder
**Created**: 2025-11-02
**Status**: Design Complete

---

## Overview

This document defines the core entities, relationships, and validation rules for the mDNS Responder feature. The data model supports RFC 6762-compliant service registration, probing, announcing, and query response.

**Design Principles**:
- **Immutability**: Service attributes (name, type, port) are immutable after registration
- **Lifecycle Management**: State machines own service lifecycle (Probing → Announcing → Established)
- **Concurrent Safety**: Registry provides thread-safe access to services
- **TTL Accuracy**: Creation timestamps enable precise TTL calculation

---

## Entity Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          Responder                              │
│  ┌───────────────┐         ┌──────────────┐                    │
│  │   Registry    │◄────────│  Transport   │                    │
│  │               │         │  (M1.1)      │                    │
│  └───────┬───────┘         └──────────────┘                    │
│          │                                                       │
│          │ 1:N                                                  │
│          ▼                                                       │
│  ┌───────────────────┐                                         │
│  │ RegisteredService │                                         │
│  │  - Service        │◄─────┐                                  │
│  │  - RecordSet      │      │                                  │
│  │  - StateMachine   │      │ 1:1                              │
│  │  - CreatedAt      │      │                                  │
│  └───────────────────┘      │                                  │
│          │                  │                                  │
│          │ 1:1              │                                  │
│          ▼                  │                                  │
│  ┌──────────────┐    ┌──────────────┐                         │
│  │   Service    │    │StateMachine  │                         │
│  │  - Instance  │    │  - State     │                         │
│  │  - Type      │    │  - Timers    │                         │
│  │  - Port      │    │  - Context   │                         │
│  │  - TXT       │    └──────────────┘                         │
│  │  - Hostname  │                                              │
│  └──────────────┘                                              │
│          │                                                       │
│          │ 1:1                                                  │
│          ▼                                                       │
│  ┌──────────────────┐                                          │
│  │ ResourceRecordSet│                                          │
│  │  - PTR           │                                          │
│  │  - SRV           │                                          │
│  │  - TXT           │                                          │
│  │  - A/AAAA        │                                          │
│  │  - CreatedAt     │                                          │
│  └──────────────────┘                                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Entities

### 1. Service

Represents a single mDNS service instance to be registered and advertised on the network.

**Attributes**:

| Name          | Type       | Required | Validation                                    | Description                                      |
|---------------|------------|----------|-----------------------------------------------|--------------------------------------------------|
| `InstanceName`| `string`   | Yes      | 1-63 chars, UTF-8, no leading/trailing spaces | Service instance name (e.g., "MyApp")            |
| `ServiceType` | `string`   | Yes      | RFC 6763 §7 format `_<service>._<proto>.local`| Service type (e.g., "_http._tcp.local")          |
| `Port`        | `uint16`   | Yes      | 1-65535                                       | Port number where service is listening           |
| `TXTRecords`  | `[]string` | No       | Each ≤255 bytes, total <9000 bytes            | Key-value metadata (e.g., "version=1.0")         |
| `Hostname`    | `string`   | No       | Valid hostname, defaults to system hostname   | Hostname for SRV/A records (e.g., "myhost.local")|

**Validation Rules**:

```go
func (s *Service) Validate() error {
    // FR-002: Service type format validation
    if !isValidServiceType(s.ServiceType) {
        return ErrInvalidServiceType
    }

    // Instance name validation
    if len(s.InstanceName) == 0 || len(s.InstanceName) > 63 {
        return ErrInvalidInstanceName
    }

    // Port validation
    if s.Port == 0 {
        return ErrInvalidPort
    }

    // TXT records validation
    totalSize := 0
    for _, txt := range s.TXTRecords {
        if len(txt) > 255 {
            return ErrTXTRecordTooLarge
        }
        totalSize += len(txt)
    }

    if totalSize > 8000 {
        // Conservative limit (leave room for other records)
        return ErrTXTRecordsTooLarge
    }

    return nil
}

func isValidServiceType(serviceType string) bool {
    // RFC 6763 §7: _<service>._<proto>.local
    // Example: "_http._tcp.local", "_ssh._tcp.local"
    pattern := `^_[a-zA-Z0-9-]+\._(?:tcp|udp)\.local$`
    matched, _ := regexp.MatchString(pattern, serviceType)
    return matched
}
```

**Immutability**:
- After registration, `InstanceName`, `ServiceType`, and `Port` are immutable
- `TXTRecords` can be updated via `UpdateService()` without re-probing (FR-004)

**Example**:
```go
service := &Service{
    InstanceName: "MyApp",
    ServiceType:  "_http._tcp.local",
    Port:         8080,
    TXTRecords:   []string{"version=1.0", "path=/api"},
    Hostname:     "myhost.local",
}
```

---

### 2. ResourceRecordSet

A collection of DNS resource records (PTR, SRV, TXT, A/AAAA) associated with a service instance. Each record has a TTL and creation timestamp for accurate TTL calculation.

**Attributes**:

| Name        | Type                | Description                                          |
|-------------|---------------------|------------------------------------------------------|
| `PTR`       | `*ResourceRecord`   | PTR record: "_http._tcp.local" → "MyApp._http._tcp.local" |
| `SRV`       | `*ResourceRecord`   | SRV record: "MyApp._http._tcp.local" → "myhost.local:8080" |
| `TXT`       | `*ResourceRecord`   | TXT record: "MyApp._http._tcp.local" → ["version=1.0", ...] |
| `A`         | `*ResourceRecord`   | A record: "myhost.local" → 192.168.1.100 (IPv4)       |
| `AAAA`      | `*ResourceRecord`   | AAAA record: "myhost.local" → fe80::1 (IPv6, M3)     |
| `CreatedAt` | `time.Time`         | Timestamp when record set was created (for TTL calc) |

**ResourceRecord Structure**:

```go
type ResourceRecord struct {
    Name   string  // Fully qualified name (e.g., "MyApp._http._tcp.local")
    Type   uint16  // DNS type (PTR=12, SRV=33, TXT=16, A=1, AAAA=28)
    Class  uint16  // DNS class (IN=1, with cache-flush bit=0x8001)
    TTL    uint32  // Time-to-live in seconds (120 or 4500)
    RDATA  []byte  // Wire-format record data
}
```

**TTL Values** (RFC 6762 §10):
- **PTR, SRV, TXT**: 120 seconds (2 minutes)
- **A/AAAA**: 4500 seconds (75 minutes)
- **Goodbye packets**: TTL=0 (FR-021)

**TTL Calculation** (R004 decision):

```go
func (rrs *ResourceRecordSet) GetRemainingTTL(rr *ResourceRecord) uint32 {
    elapsed := uint32(time.Since(rrs.CreatedAt).Seconds())

    if elapsed >= rr.TTL {
        return 0 // Expired
    }

    return rr.TTL - elapsed
}

func (rrs *ResourceRecordSet) IsExpired(rr *ResourceRecord) bool {
    return rrs.GetRemainingTTL(rr) == 0
}
```

**Cache-Flush Bit** (RFC 6762 §10.2):
- Set high bit of class field (0x8001) for unique records (SRV, TXT, A)
- Do NOT set for shared records (PTR)

**Example**:
```go
recordSet := &ResourceRecordSet{
    PTR: &ResourceRecord{
        Name:  "_http._tcp.local",
        Type:  12, // PTR
        Class: 1,  // IN (no cache-flush for PTR)
        TTL:   120,
        RDATA: encodeName("MyApp._http._tcp.local"),
    },
    SRV: &ResourceRecord{
        Name:  "MyApp._http._tcp.local",
        Type:  33, // SRV
        Class: 0x8001, // IN with cache-flush bit
        TTL:   120,
        RDATA: encodeSRV(0, 0, 8080, "myhost.local"),
    },
    TXT: &ResourceRecord{
        Name:  "MyApp._http._tcp.local",
        Type:  16, // TXT
        Class: 0x8001,
        TTL:   120,
        RDATA: encodeTXT([]string{"version=1.0", "path=/api"}),
    },
    A: &ResourceRecord{
        Name:  "myhost.local",
        Type:  1, // A
        Class: 0x8001,
        TTL:   4500,
        RDATA: []byte{192, 168, 1, 100},
    },
    CreatedAt: time.Now(),
}
```

---

### 3. StateMachine

Manages the lifecycle of a service registration through the Probing → Announcing → Established state transitions.

**States**:

| State         | Description                                                         | Duration                  |
|---------------|---------------------------------------------------------------------|---------------------------|
| `Probing`     | Sending 3 probe queries to detect name conflicts                   | 750ms (3 × 250ms)         |
| `Announcing`  | Sending 2 unsolicited announcements after successful probing        | 1 second (2 × 1s)         |
| `Established` | Ready to respond to queries, service is discoverable                | Until unregistered        |
| `Conflict`    | Name conflict detected, renaming and restarting probing             | Transient (restart)       |

**State Transitions**:

```
                ┌─────────────┐
                │   Probing   │
                └──────┬──────┘
                       │
         ┌─────────────┼─────────────┐
         │ Success     │             │ Conflict detected
         │ (no conflict)             │ (response received)
         │                           │
         ▼                           ▼
┌─────────────┐             ┌─────────────┐
│ Announcing  │             │  Conflict   │
└──────┬──────┘             └──────┬──────┘
       │                           │
       │ 2 announcements sent      │ Rename (MyApp → MyApp (2))
       │                           │
       ▼                           ▼
┌─────────────┐             ┌─────────────┐
│ Established │             │   Probing   │◄──┐
└──────┬──────┘             └─────────────┘   │
       │                                       │
       │ Service unregistered                  │ Loop until
       │ (Goodbye packets sent)                │ name available
       ▼                                       │
     (End)                                     └──
```

**Attributes**:

| Name          | Type                  | Description                                          |
|---------------|-----------------------|------------------------------------------------------|
| `service`     | `*Service`            | Service instance being registered                    |
| `state`       | `State`               | Current state (enum)                                 |
| `probeCount`  | `int`                 | Number of probes sent (0-3)                          |
| `announceCount`| `int`                | Number of announcements sent (0-2)                   |
| `transport`   | `Transport`           | Transport layer for sending/receiving packets        |
| `registry`    | `*Registry`           | Reference to service registry                        |
| `ctx`         | `context.Context`     | Context for cancellation                             |
| `cancel`      | `context.CancelFunc`  | Cancel function to stop state machine                |

**Event Handling**:

```go
type stateMachine struct {
    service       *Service
    state         State
    probeCount    int
    announceCount int
    transport     Transport
    registry      *Registry
    ctx           context.Context
    cancel        context.CancelFunc
}

func (sm *stateMachine) run() {
    defer sm.cancel()

    for {
        switch sm.state {
        case StateProbing:
            if err := sm.handleProbing(); err != nil {
                return
            }

        case StateAnnouncing:
            if err := sm.handleAnnouncing(); err != nil {
                return
            }

        case StateEstablished:
            return sm.handleEstablished()

        case StateConflict:
            if err := sm.handleConflict(); err != nil {
                return
            }
        }
    }
}

func (sm *stateMachine) handleProbing() error {
    for sm.probeCount < 3 {
        sm.sendProbe()
        sm.probeCount++

        select {
        case <-time.After(250 * time.Millisecond):
            // Check for conflicts received during window
            if sm.registry.HasConflict(sm.service) {
                sm.state = StateConflict
                return nil
            }

        case <-sm.ctx.Done():
            return sm.ctx.Err()
        }
    }

    // Success - transition to Announcing
    sm.state = StateAnnouncing
    sm.probeCount = 0
    return nil
}
```

---

### 4. Registry

Thread-safe storage for all registered services. Provides concurrent access for registration, unregistration, and query handling.

**Attributes**:

| Name       | Type                           | Description                                      |
|------------|--------------------------------|--------------------------------------------------|
| `mu`       | `sync.RWMutex`                 | Read-write mutex for concurrent access           |
| `services` | `map[string]*RegisteredService`| Map of instance name → RegisteredService         |

**Operations**:

| Method                     | Lock Type | Description                                      |
|----------------------------|-----------|--------------------------------------------------|
| `Register(service, sm)`    | Write     | Add new service to registry                      |
| `Get(instanceName)`        | Read      | Retrieve single service by name                  |
| `GetAll()`                 | Read      | Retrieve all services (for query response)       |
| `Remove(instanceName)`     | Write     | Remove service and cancel state machine          |
| `HasConflict(service)`     | Read      | Check if service name conflicts with existing    |
| `GetByType(serviceType)`   | Read      | Retrieve all services of a given type            |

**Implementation** (R006 decision):

```go
type Registry struct {
    mu       sync.RWMutex
    services map[string]*RegisteredService
}

type RegisteredService struct {
    Service      *Service
    RecordSet    *ResourceRecordSet
    StateMachine *stateMachine
    CreatedAt    time.Time
}

func (r *Registry) Register(service *Service, sm *stateMachine) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.services[service.InstanceName]; exists {
        return ErrServiceAlreadyRegistered
    }

    r.services[service.InstanceName] = &RegisteredService{
        Service:      service,
        RecordSet:    buildRecordSet(service),
        StateMachine: sm,
        CreatedAt:    time.Now(),
    }

    return nil
}

func (r *Registry) GetByType(serviceType string) []*RegisteredService {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []*RegisteredService
    for _, rs := range r.services {
        if rs.Service.ServiceType == serviceType {
            result = append(result, rs)
        }
    }

    return result
}
```

**Concurrency Guarantees** (NFR-005):
- **Read lock**: Multiple concurrent readers (query handling)
- **Write lock**: Exclusive writer (registration/unregistration)
- **Race-free**: Validated with `go test -race`

---

### 5. ConflictDetector

Detects name conflicts during probing and implements tie-breaking logic per RFC 6762 §8.2.1.

**Attributes**:

| Name             | Type                  | Description                                      |
|------------------|-----------------------|--------------------------------------------------|
| `registry`       | `*Registry`           | Reference to service registry                    |
| `transport`      | `Transport`           | Transport layer for receiving probe responses    |
| `conflictCache`  | `map[string]time.Time`| Recently detected conflicts (to avoid loops)     |

**Operations**:

| Method                              | Description                                          |
|-------------------------------------|------------------------------------------------------|
| `DetectConflict(service, response)` | Check if received response conflicts with service    |
| `CompareProbes(ourProbe, theirProbe)`| Tie-breaking logic (RFC 6762 §8.2.1)                |
| `Rename(service)`                   | Generate alternative name (MyApp → MyApp (2))        |

**Tie-Breaking Logic** (R002 decision):

```go
func (cd *ConflictDetector) CompareProbes(ourProbe, theirProbe []byte) int {
    // Extract RDATA from authority section
    ourRDATA := extractAuthorityRDATA(ourProbe)
    theirRDATA := extractAuthorityRDATA(theirProbe)

    // RFC 6762 §8.2.1: Lexicographic byte comparison
    return bytes.Compare(ourRDATA, theirRDATA)
}

func (cd *ConflictDetector) Rename(service *Service) *Service {
    // FR-009: Automatic renaming (MyApp → MyApp (2) → MyApp (3))
    baseName := service.InstanceName

    // Extract existing number suffix
    re := regexp.MustCompile(`^(.*) \((\d+)\)$`)
    if matches := re.FindStringSubmatch(baseName); matches != nil {
        baseName = matches[1]
        num, _ := strconv.Atoi(matches[2])
        return &Service{
            InstanceName: fmt.Sprintf("%s (%d)", baseName, num+1),
            ServiceType:  service.ServiceType,
            Port:         service.Port,
            TXTRecords:   service.TXTRecords,
            Hostname:     service.Hostname,
        }
    }

    // First conflict - add (2)
    return &Service{
        InstanceName: fmt.Sprintf("%s (2)", baseName),
        ServiceType:  service.ServiceType,
        Port:         service.Port,
        TXTRecords:   service.TXTRecords,
        Hostname:     service.Hostname,
    }
}
```

---

### 6. ResponseBuilder

Constructs mDNS response packets with appropriate records (PTR, SRV, TXT, A), respecting known-answer suppression and packet size limits.

**Attributes**:

| Name         | Type                  | Description                                      |
|--------------|-----------------------|--------------------------------------------------|
| `registry`   | `*Registry`           | Reference to service registry                    |
| `maxSize`    | `int`                 | Maximum packet size (9000 bytes, RFC 6762 §17)   |

**Operations**:

| Method                                      | Description                                          |
|---------------------------------------------|------------------------------------------------------|
| `BuildResponse(services, query)`            | Build aggregated response for PTR/SRV/TXT/A queries  |
| `ApplyKnownAnswerSuppression(response, ka)` | Omit records matching known-answer section (RFC 7.1) |
| `EstimatePacketSize(response)`              | Calculate wire-format size of response               |

**Known-Answer Suppression** (R003 decision):

```go
func (rb *ResponseBuilder) ApplyKnownAnswerSuppression(response *Response, knownAnswers []*ResourceRecord) {
    kas := parseKnownAnswers(knownAnswers)

    filtered := make([]*ResourceRecord, 0, len(response.Answers))
    for _, rr := range response.Answers {
        key := buildKey(rr.Name, rr.Type, rr.Class)

        if ka, exists := kas[key]; exists {
            // Check RDATA match
            if bytes.Equal(ka.RDATA, rr.RDATA) && ka.TTL >= rr.TTL/2 {
                // RFC 6762 §7.1: Suppress if TTL ≥ half
                continue
            }
        }

        filtered = append(filtered, rr)
    }

    response.Answers = filtered
}
```

**Response Aggregation** (R005 decision):

```go
func (rb *ResponseBuilder) BuildResponse(service *Service, query *Query) (*Response, error) {
    response := &Response{
        Header:    buildHeader(query),
        Questions: query.Questions,
    }

    // Answer section (critical - must fit)
    if query.Type == TypePTR {
        response.Answers = append(response.Answers, service.RecordSet.PTR)
    }

    // Additional section (best-effort)
    currentSize := rb.EstimatePacketSize(response)

    for _, rr := range []*ResourceRecord{service.RecordSet.SRV, service.RecordSet.TXT, service.RecordSet.A} {
        rrSize := rb.EstimateRecordSize(rr)
        if currentSize+rrSize > rb.maxSize {
            break // Omit remaining additional records
        }
        response.Additional = append(response.Additional, rr)
        currentSize += rrSize
    }

    return response, nil
}
```

---

## Entity Relationships

### 1. Responder → Registry (1:1)
- **Ownership**: Responder owns a single Registry instance
- **Lifecycle**: Registry created with Responder, destroyed on Close()

### 2. Registry → RegisteredService (1:N)
- **Ownership**: Registry owns all RegisteredService instances
- **Lifecycle**: RegisteredService created on Register(), destroyed on Unregister()
- **Access**: Thread-safe via RWMutex

### 3. RegisteredService → Service (1:1)
- **Ownership**: RegisteredService owns Service
- **Immutability**: Service attributes immutable after registration (except TXT)

### 4. RegisteredService → ResourceRecordSet (1:1)
- **Ownership**: RegisteredService owns ResourceRecordSet
- **Lifecycle**: Created on registration, updated on TXT change

### 5. RegisteredService → StateMachine (1:1)
- **Ownership**: RegisteredService owns StateMachine
- **Lifecycle**: StateMachine runs in dedicated goroutine until Established or cancelled

### 6. StateMachine → Transport (N:1)
- **Shared**: All state machines share single Transport instance
- **Thread-safe**: Transport supports concurrent Send/Receive

---

## Validation Rules Summary

| Entity              | Validation Rule                                                  | Error Code                    |
|---------------------|------------------------------------------------------------------|-------------------------------|
| Service             | ServiceType must match `_<service>._<proto>.local`              | `ErrInvalidServiceType`       |
| Service             | InstanceName must be 1-63 characters                             | `ErrInvalidInstanceName`      |
| Service             | Port must be 1-65535                                             | `ErrInvalidPort`              |
| Service             | Each TXT record ≤255 bytes                                       | `ErrTXTRecordTooLarge`        |
| Service             | Total TXT records <8000 bytes                                    | `ErrTXTRecordsTooLarge`       |
| ResourceRecordSet   | TTL must be 120 (service) or 4500 (hostname)                     | N/A (set by constructor)      |
| ResourceRecordSet   | Total packet size <9000 bytes                                    | `ErrResponseTooLarge`         |
| Registry            | Service instance name must be unique                             | `ErrServiceAlreadyRegistered` |
| StateMachine        | Probe count ≤3                                                   | N/A (state machine enforces)  |
| StateMachine        | Announce count ≤2                                                | N/A (state machine enforces)  |

---

## Wire Format Examples

### PTR Record
```
Name:  "_http._tcp.local"
Type:  12 (PTR)
Class: 1 (IN, no cache-flush)
TTL:   120
RDATA: "MyApp._http._tcp.local"

Wire format (hex):
  04 5f 68 74 74 70 04 5f 74 63 70 05 6c 6f 63 61 6c 00  ; Name
  00 0c                                                    ; Type (PTR)
  00 01                                                    ; Class (IN)
  00 00 00 78                                              ; TTL (120)
  00 18                                                    ; RDLENGTH (24)
  05 4d 79 41 70 70 c0 00                                  ; RDATA (compressed name)
```

### SRV Record
```
Name:  "MyApp._http._tcp.local"
Type:  33 (SRV)
Class: 0x8001 (IN with cache-flush bit)
TTL:   120
RDATA: Priority=0, Weight=0, Port=8080, Target="myhost.local"

Wire format (hex):
  05 4d 79 41 70 70 c0 00                                  ; Name (compressed)
  00 21                                                    ; Type (SRV)
  80 01                                                    ; Class (IN + cache-flush)
  00 00 00 78                                              ; TTL (120)
  00 14                                                    ; RDLENGTH (20)
  00 00 00 00                                              ; Priority=0, Weight=0
  1f 90                                                    ; Port=8080
  06 6d 79 68 6f 73 74 05 6c 6f 63 61 6c 00                ; Target="myhost.local"
```

---

## State Machine Timing Diagram

```
Registration Start
       │
       ├─ Probing State (750ms total)
       │   ├─ Send Probe #1
       │   ├─ Wait 250ms ────► Check for conflicts
       │   ├─ Send Probe #2
       │   ├─ Wait 250ms ────► Check for conflicts
       │   ├─ Send Probe #3
       │   └─ Wait 250ms ────► Check for conflicts
       │
       ├─ Announcing State (1000ms total)
       │   ├─ Send Announcement #1
       │   ├─ Wait 1000ms
       │   └─ Send Announcement #2
       │
       └─ Established State
           └─ Respond to queries indefinitely
```

**Total Registration Time**: 1.75 seconds (750ms probing + 1000ms announcing)

---

## Next Steps

With the data model defined, proceed to:
1. **API Contracts** (`contracts/responder-api.md`) - Public Responder API specification
2. **State Machine Contract** (`contracts/state-machine.md`) - State machine behavior specification
3. **Quickstart Guide** (`quickstart.md`) - Usage examples for developers

**Status**: Design artifacts ready for task breakdown (`/speckit.tasks`)
