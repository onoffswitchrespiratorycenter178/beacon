# Contract: State Machine Behavior

**Feature**: 006-mdns-responder
**Created**: 2025-11-02
**Status**: Contract Complete

---

## Overview

This document defines the behavioral contract for the mDNS Responder state machine. The state machine manages the lifecycle of service registration from initial probing through conflict detection, announcing, and established operation.

**RFC Foundation**: RFC 6762 §8 (Probing and Announcing on Startup)

---

## State Diagram

```
                          ┌─────────────────────┐
                          │   Registration      │
                          │   Requested         │
                          └──────────┬──────────┘
                                     │
                                     │ Validate service
                                     │
                          ┌──────────▼──────────┐
                          │     PROBING         │◄────────┐
                          │                     │         │
                          │  - Send 3 probes    │         │
                          │  - 250ms intervals  │         │ Conflict detected
                          │  - Listen for       │         │ → Rename service
                          │    conflicts        │         │ → Retry probing
                          └──────────┬──────────┘         │
                                     │                    │
                 ┌───────────────────┼────────────────────┘
                 │ Success            │ Conflict
                 │ (no response)      │ (response received)
                 │                    │
      ┌──────────▼──────────┐  ┌──────▼───────┐
      │    ANNOUNCING        │  │   CONFLICT   │
      │                      │  │              │
      │  - Send 2 announcements  │  - Rename   │
      │  - 1 second apart    │  │  - Restart   │
      └──────────┬───────────┘  └──────────────┘
                 │
                 │ Announcements sent
                 │
      ┌──────────▼──────────┐
      │    ESTABLISHED       │
      │                      │
      │  - Respond to queries│
      │  - Monitor conflicts │
      │  - Handle updates    │
      └──────────┬───────────┘
                 │
                 │ Unregister requested
                 │
      ┌──────────▼──────────┐
      │   GOODBYE            │
      │                      │
      │  - Send TTL=0        │
      │  - Remove from       │
      │    registry          │
      └──────────────────────┘
```

---

## States

### PROBING

**Purpose**: Detect name conflicts before announcing the service.

**Entry Conditions**:
- Service registration requested
- Conflict detected (after rename)

**Behavior**:
1. Send 3 simultaneous query + probe packets (RFC 6762 §8.1)
   - Query section: Question for the proposed name (ANY type)
   - Authority section: Proposed resource records (SRV, TXT)
2. Wait 250ms between each probe
3. Monitor for conflicting responses during each 250ms window
4. Total probing duration: 750ms (3 × 250ms)

**Exit Conditions**:
- **Success**: No conflicting responses received after 3 probes → Transition to ANNOUNCING
- **Conflict**: Conflicting response received → Transition to CONFLICT
- **Tie-break Loss**: Simultaneous probe with lexicographically earlier RDATA → Transition to CONFLICT
- **Cancellation**: Context cancelled → Terminate state machine

**Invariants**:
- Exactly 3 probes are sent (FR-006)
- Probes are spaced exactly 250ms apart (RFC 6762 §8.1)
- Conflict detection checks run after each 250ms window

**RFC Compliance**:
- RFC 6762 §8.1: "Before claiming a name, a host must probe to verify that no other host is using that name."
- RFC 6762 §8.1: "The host sends three probe packets, each 250 milliseconds apart."

---

### ANNOUNCING

**Purpose**: Broadcast the service to the network after successful probing.

**Entry Conditions**:
- Probing completed successfully (no conflicts)

**Behavior**:
1. Send first unsolicited multicast announcement
   - Answer section: PTR, SRV, TXT, A records
   - Set cache-flush bit for unique records (SRV, TXT, A)
2. Wait 1 second
3. Send second unsolicited multicast announcement
4. Total announcing duration: 1 second

**Exit Conditions**:
- **Success**: 2 announcements sent → Transition to ESTABLISHED
- **Cancellation**: Context cancelled → Terminate state machine

**Invariants**:
- Exactly 2 announcements are sent (FR-011)
- Announcements are spaced exactly 1 second apart (RFC 6762 §8.3)

**RFC Compliance**:
- RFC 6762 §8.3: "The Multicast DNS responder MUST send at least two unsolicited responses, one second apart."

---

### ESTABLISHED

**Purpose**: Service is fully registered and responding to queries.

**Entry Conditions**:
- Announcing completed successfully

**Behavior**:
1. Respond to multicast queries for the service type or instance name
2. Send response packets containing PTR, SRV, TXT, A records
3. Apply known-answer suppression (RFC 6762 §7.1)
4. Respect TTL values (120s for service records, 4500s for A records)
5. Monitor for late-arriving conflicts (rare, handle with re-probing)

**Exit Conditions**:
- **Unregister**: Service unregistration requested → Transition to GOODBYE
- **Conflict**: Late conflict detected → Transition to CONFLICT (rare edge case)
- **Shutdown**: Responder.Close() called → Transition to GOODBYE

**Invariants**:
- Query responses sent within 100ms of query receipt (NFR-002)
- TTL decremented based on elapsed time since record creation (FR-020)

**RFC Compliance**:
- RFC 6762 §6: "Multicast DNS responses MUST NOT be delayed."
- RFC 6762 §7.1: "A Multicast DNS responder MUST NOT answer a query if the answer is already in the Known-Answer section."

---

### CONFLICT

**Purpose**: Handle name conflicts by renaming the service and restarting probing.

**Entry Conditions**:
- Conflicting response received during PROBING
- Simultaneous probe with tie-break loss
- Late conflict detected in ESTABLISHED (rare)

**Behavior**:
1. Determine conflict type:
   - **Direct conflict**: Another service with same name exists
   - **Simultaneous probe**: Both parties probing same name, lexicographic tie-break
2. Rename service (FR-009):
   - First conflict: "MyApp" → "MyApp (2)"
   - Subsequent conflicts: "MyApp (2)" → "MyApp (3)", etc.
3. Limit: Max 10 rename attempts (FR-032)
4. Restart state machine from PROBING with new name

**Exit Conditions**:
- **Retry**: Service renamed → Transition to PROBING
- **Max retries**: 10 conflicts → Return error to caller
- **Cancellation**: Context cancelled → Terminate state machine

**Invariants**:
- Rename attempts are sequential (no infinite loops)
- Max 10 rename attempts (configurable constant)

**RFC Compliance**:
- RFC 6762 §8.2.1: "If the host finds that the name is already in use, it MUST choose a different name."
- RFC 6762 §8.2.1: Tie-breaking via lexicographic comparison of RDATA

---

### GOODBYE

**Purpose**: Gracefully remove service from network caches.

**Entry Conditions**:
- Service unregistration requested
- Responder shutdown (Close() called)

**Behavior**:
1. Send multicast goodbye packet (RFC 6762 §10.1):
   - Answer section: All service records (PTR, SRV, TXT, A)
   - TTL: 0 for all records
2. Remove service from registry
3. Cancel state machine goroutine

**Exit Conditions**:
- **Complete**: Goodbye packet sent → Terminate state machine

**Invariants**:
- Goodbye packet sent at least 1 second before final shutdown if possible (FR-021)
- All records have TTL=0 (FR-014)

**RFC Compliance**:
- RFC 6762 §10.1: "To allow for this, a host may send an unsolicited Multicast DNS response packet with TTL zero."

---

## State Transitions

### Valid Transitions

| From State    | Event                         | To State     | RFC Reference |
|---------------|-------------------------------|--------------|---------------|
| -             | Register() called             | PROBING      | §8.1          |
| PROBING       | 3 probes sent, no conflict    | ANNOUNCING   | §8.1          |
| PROBING       | Conflict detected             | CONFLICT     | §8.2          |
| PROBING       | Tie-break loss                | CONFLICT     | §8.2.1        |
| PROBING       | Context cancelled             | (Terminate)  | -             |
| ANNOUNCING    | 2 announcements sent          | ESTABLISHED  | §8.3          |
| ANNOUNCING    | Context cancelled             | (Terminate)  | -             |
| ESTABLISHED   | Unregister() called           | GOODBYE      | §10.1         |
| ESTABLISHED   | Close() called                | GOODBYE      | §10.1         |
| ESTABLISHED   | Late conflict (rare)          | CONFLICT     | §9            |
| CONFLICT      | Service renamed (<10 attempts)| PROBING      | §8.2.1        |
| CONFLICT      | Max retries (10)              | (Error)      | -             |
| GOODBYE       | Goodbye packet sent           | (Terminate)  | §10.1         |

---

## Timing Requirements

### Probing Timing (RFC 6762 §8.1)

```
T=0ms     Send Probe #1
T=250ms   Send Probe #2
T=500ms   Send Probe #3
T=750ms   Probing complete (if no conflicts)
```

**Tolerance**: ±10ms (system scheduling jitter)

**Implementation**:
```go
for i := 0; i < 3; i++ {
    sm.sendProbe()

    select {
    case <-time.After(250 * time.Millisecond):
        if sm.hasConflict() {
            return sm.transitionTo(StateConflict)
        }
    case <-sm.ctx.Done():
        return sm.ctx.Err()
    }
}

return sm.transitionTo(StateAnnouncing)
```

---

### Announcing Timing (RFC 6762 §8.3)

```
T=0ms     Send Announcement #1
T=1000ms  Send Announcement #2
T=1000ms  Announcing complete
```

**Tolerance**: ±10ms

**Implementation**:
```go
for i := 0; i < 2; i++ {
    sm.sendAnnouncement()

    if i < 1 { // Don't wait after 2nd announcement
        select {
        case <-time.After(1 * time.Second):
            // Continue
        case <-sm.ctx.Done():
            return sm.ctx.Err()
        }
    }
}

return sm.transitionTo(StateEstablished)
```

---

### Total Registration Time

```
Probing:     750ms  (3 probes × 250ms)
Announcing: 1000ms  (2 announcements × 1s)
─────────────────
Total:      1750ms  (1.75 seconds)
```

**Success Criteria**: SC-001 requires services discoverable within 2 seconds of registration (allows 250ms margin).

---

## Conflict Detection

### Direct Conflict

**Scenario**: Another service with same name already exists.

**Detection**:
- During PROBING: Receive response with matching name in answer section
- Record type: ANY (PTR, SRV, TXT, A all indicate conflict)

**Action**:
- Transition to CONFLICT state
- Rename service (add " (2)" suffix)
- Restart probing

**Example**:
```
Network has: "MyApp._http._tcp.local"
We probe:    "MyApp._http._tcp.local"
Response:    PTR "MyApp._http._tcp.local" (conflict!)
Action:      Rename to "MyApp (2)._http._tcp.local", re-probe
```

---

### Simultaneous Probe Conflict

**Scenario**: Two hosts probe for the same name concurrently.

**Detection**:
- During PROBING: Receive probe query (not response) with matching name in authority section

**Action** (RFC 6762 §8.2.1):
1. Extract RDATA from our authority section
2. Extract RDATA from their authority section
3. Lexicographic comparison: `bytes.Compare(ourRDATA, theirRDATA)`
4. If `ourRDATA < theirRDATA`: We lose, rename and re-probe
5. If `ourRDATA > theirRDATA`: We win, continue probing
6. If `ourRDATA == theirRDATA`: Both lose, both rename (prevents deadlock)

**Example**:
```
Our probe RDATA:    0x00 0x00 0x1f 0x90 ... (port 8080)
Their probe RDATA:  0x00 0x00 0x50 0x00 ... (port 20480)

Comparison: 0x1f < 0x50 → We lose
Action: Rename to "MyApp (2)", re-probe
```

**Implementation**:
```go
func (cd *ConflictDetector) handleSimultaneousProbe(ourProbe, theirProbe []byte) error {
    ourRDATA := extractAuthorityRDATA(ourProbe)
    theirRDATA := extractAuthorityRDATA(theirProbe)

    cmp := bytes.Compare(ourRDATA, theirRDATA)

    if cmp <= 0 {
        // We lose or tie - rename
        return cd.rename()
    }

    // We win - continue probing
    return nil
}
```

---

### Late Conflict (ESTABLISHED State)

**Scenario**: Conflict detected after service is already established (rare - typically due to network partition healing).

**Detection**:
- Receive announcement or response for our service name with different RDATA

**Action**:
- Transition to CONFLICT state
- Rename service
- Restart probing
- Send goodbye packets for old name (TTL=0)

**RFC Compliance**: RFC 6762 §9 "Conflict Resolution" (ongoing monitoring)

---

## Query Response Behavior

### PTR Query Response

**Trigger**: Receive PTR query for service type (e.g., "_http._tcp.local")

**Response**:
- **Answer Section**: PTR record ("_http._tcp.local" → "MyApp._http._tcp.local")
- **Additional Section** (reduce round-trips):
  - SRV record ("MyApp._http._tcp.local" → "myhost.local:8080")
  - TXT record ("MyApp._http._tcp.local" → ["version=1.0"])
  - A record ("myhost.local" → 192.168.1.100)

**Known-Answer Suppression**:
- If query includes PTR in known-answer section with TTL ≥ 60s (half of 120s), suppress PTR
- Still send additional records if not in known-answers

**Example**:
```
Query:
  Questions: PTR "_http._tcp.local"
  Known-Answers: (empty)

Response:
  Answers:
    PTR "_http._tcp.local" → "MyApp._http._tcp.local" TTL=120
  Additional:
    SRV "MyApp._http._tcp.local" → "myhost.local:8080" TTL=120
    TXT "MyApp._http._tcp.local" → "version=1.0" TTL=120
    A   "myhost.local" → 192.168.1.100 TTL=4500
```

---

### SRV/TXT Query Response

**Trigger**: Receive SRV or TXT query for specific instance (e.g., "MyApp._http._tcp.local")

**Response**:
- **Answer Section**: Requested record (SRV or TXT)
- **Additional Section**: A record for hostname

**Example**:
```
Query:
  Questions: SRV "MyApp._http._tcp.local"

Response:
  Answers:
    SRV "MyApp._http._tcp.local" → "myhost.local:8080" TTL=120
  Additional:
    A "myhost.local" → 192.168.1.100 TTL=4500
```

---

### Unicast Response (QU Bit)

**Trigger**: Query has QU (unicast response) bit set in question class field (RFC 6762 §5.4)

**Response**:
- Send response via unicast to querier's IP:port (not multicast)
- Same record content as multicast response

**Implementation**:
```go
if query.Question.Class & 0x8000 != 0 {
    // QU bit set - send unicast response
    sm.transport.Send(sm.ctx, response, query.SrcAddr)
} else {
    // Send multicast response
    sm.transport.Send(sm.ctx, response, MulticastAddr)
}
```

---

## TTL Management

### TTL Assignment (RFC 6762 §10)

| Record Type | TTL (seconds) | Rationale                                    |
|-------------|---------------|----------------------------------------------|
| PTR         | 120           | Service instance names may change frequently|
| SRV         | 120           | Port/hostname may change                     |
| TXT         | 120           | Metadata updates frequently                  |
| A/AAAA      | 4500 (75 min) | Hostnames/IPs are relatively stable          |

---

### TTL Decrementation (FR-020)

**Strategy**: Creation-timestamp-based calculation

**Implementation**:
```go
type ResourceRecordSet struct {
    records   []*ResourceRecord
    createdAt time.Time
}

func (rrs *ResourceRecordSet) getRemainingTTL(rr *ResourceRecord) uint32 {
    elapsed := uint32(time.Since(rrs.createdAt).Seconds())

    if elapsed >= rr.TTL {
        return 0 // Expired
    }

    return rr.TTL - elapsed
}
```

**Behavior**:
- TTL calculated at response generation time (not stored/updated)
- Records with TTL=0 are omitted from responses
- No background goroutines for TTL management

---

### Goodbye Packets (RFC 6762 §10.1)

**Purpose**: Notify browsers to immediately remove service from cache.

**Implementation**:
```go
func (sm *stateMachine) sendGoodbye() error {
    goodbye := &Response{
        Answers: []ResourceRecord{
            {Name: sm.service.PTR.Name,  TTL: 0, ...},
            {Name: sm.service.SRV.Name,  TTL: 0, ...},
            {Name: sm.service.TXT.Name,  TTL: 0, ...},
            {Name: sm.service.A.Name,    TTL: 0, ...},
        },
    }

    return sm.transport.Send(sm.ctx, goodbye, MulticastAddr)
}
```

**Timing**: Sent at least 1 second before final shutdown if possible (FR-021).

---

## Error Handling

### Transient Errors

| Error                     | State      | Action                                      |
|---------------------------|------------|---------------------------------------------|
| Network send failure      | PROBING    | Retry probe (up to 3 attempts)              |
| Network send failure      | ANNOUNCING | Log warning, continue (best-effort)         |
| Network send failure      | GOODBYE    | Log warning, complete shutdown              |

---

### Fatal Errors

| Error                     | State      | Action                                      |
|---------------------------|------------|---------------------------------------------|
| Context cancelled         | Any        | Terminate state machine immediately         |
| Max conflicts (10)        | CONFLICT   | Return error to caller, terminate           |
| Service validation failed | -          | Return error before entering PROBING        |

---

## Concurrency Guarantees

### Goroutine Lifecycle

- **1 goroutine per service**: Each state machine runs in dedicated goroutine (R001 decision)
- **Lifecycle**: Created on Register(), terminated on Unregister() or Close()
- **Context cancellation**: All goroutines respect `context.Context` for graceful shutdown

### Thread Safety

- **Registry access**: Protected by `sync.RWMutex` (R006 decision)
- **State transitions**: Internal to state machine goroutine (no shared state)
- **Transport**: Thread-safe, supports concurrent Send/Receive

### Race Detection

- **Validation**: All code validated with `go test -race` (NFR-005)
- **No data races**: Zero race conditions allowed

---

## Testing Contract

### Unit Tests (State Transitions)

```go
func TestStateMachine_Probing_NoConflict_TransitionsToAnnouncing(t *testing.T) {
    sm := newStateMachine(service, mockTransport, registry)

    // Simulate probing
    sm.run()

    assert.Equal(t, StateAnnouncing, sm.state)
    assert.Equal(t, 3, mockTransport.probesSent)
}

func TestStateMachine_Probing_ConflictDetected_TransitionsToConflict(t *testing.T) {
    sm := newStateMachine(service, mockTransport, registry)

    // Simulate conflict response
    mockTransport.injectResponse(conflictResponse)

    sm.run()

    assert.Equal(t, StateConflict, sm.state)
}
```

---

### Integration Tests (RFC Compliance)

```go
func TestStateMachine_FullLifecycle_RFC6762Compliance(t *testing.T) {
    // T000: Register service
    r := newRealResponder()
    service := &Service{InstanceName: "Test", ServiceType: "_http._tcp.local", Port: 8080}

    start := time.Now()
    r.Register(context.Background(), service)
    elapsed := time.Since(start)

    // Verify total time ~1.75s
    assert.InDelta(t, 1750, elapsed.Milliseconds(), 100)

    // T+2s: Query from another device
    time.Sleep(2 * time.Second)
    response := sendPTRQuery("_http._tcp.local")

    // Verify response contains PTR, SRV, TXT, A
    assert.Contains(t, response.Answers, "Test._http._tcp.local")
    assert.Contains(t, response.Additional, "8080")

    // T+3s: Unregister
    r.Unregister("Test")

    // Verify goodbye packets sent (TTL=0)
    goodbye := capturePackets(1 * time.Second)
    assert.Equal(t, 0, goodbye[0].TTL)
}
```

---

## Next Steps

With the state machine contract defined, proceed to:
1. **Quickstart Guide** (`quickstart.md`) - End-to-end usage examples
2. **Task Breakdown** (`/speckit.tasks`) - Granular implementation tasks

**Status**: State machine contract ready for implementation
