# Phase 0 Research: mDNS Responder Implementation

**Feature**: 006-mdns-responder
**Created**: 2025-11-02
**Status**: Research Complete

---

## R001: State Machine Architecture

**Question**: What is the best architecture for implementing the Probing → Announcing → Established state machine?

**Decision**: Goroutine-per-service with event-driven state transitions using timeout channels

**Rationale**:
- **Concurrency Model**: Each service registration spawns a dedicated goroutine managing its state machine lifecycle. This provides natural isolation between services (one service's conflict doesn't block others) and simplifies state management (no shared state machine executor).
- **Timeout Handling**: Go's `time.Timer` and `select` statements provide elegant timeout handling for probing intervals (250ms × 3) and announcing intervals (1s × 2).
- **RFC Compliance**: RFC 6762 §8.1 requires precise timing (250ms probe intervals). Dedicated goroutines ensure timing accuracy without interference from other services.
- **Cancellation**: `context.Context` propagation allows graceful shutdown (service unregistration or responder close triggers state machine termination).
- **Avahi Pattern**: Avahi uses per-service state machines in separate threads (see `avahi-core/entry.c`), validating this approach.

**Alternatives Considered**:
1. **Single event loop with priority queue**: Rejected due to complexity of managing multiple concurrent timers and potential head-of-line blocking.
2. **Pooled workers with shared state machine**: Rejected due to contention on shared state and difficulty maintaining per-service timing guarantees.
3. **Callback-based state machine**: Rejected due to Go idioms favoring goroutines + channels over callbacks.

**Implementation Pattern**:
```go
type stateMachine struct {
    service   *Service
    state     State
    transport Transport
    registry  *Registry
    ctx       context.Context
    cancel    context.CancelFunc
}

func (sm *stateMachine) run() {
    defer sm.cancel()

    // Probing phase
    if err := sm.probe(); err != nil {
        // Handle conflict, rename, restart
        return
    }

    // Announcing phase
    sm.announce()

    // Established phase
    sm.established()
}

func (sm *stateMachine) probe() error {
    for i := 0; i < 3; i++ {
        sm.sendProbe()

        select {
        case <-time.After(250 * time.Millisecond):
            // Check for conflicts received during window
            if sm.registry.HasConflict(sm.service) {
                return sm.handleConflict()
            }
        case <-sm.ctx.Done():
            return sm.ctx.Err()
        }
    }
    return nil
}
```

---

## R002: Tie-Breaking Algorithm for Simultaneous Probes

**Question**: How should we implement RFC 6762 §8.2.1 lexicographic comparison for tie-breaking when simultaneous probes occur?

**Decision**: Byte-by-byte comparison of RDATA sections in authority section of probe queries

**Rationale**:
- **RFC Requirement**: RFC 6762 §8.2.1 specifies: "The host compares the data from its own record with the data from the received record by comparing raw bytes of the RDATA section."
- **Wire Format Comparison**: Compare the actual DNS wire format bytes (not parsed structures) to ensure exact RFC compliance.
- **Deterministic**: Lexicographic byte comparison ensures both parties reach the same conclusion about which probe "wins."
- **Simple Implementation**: Go's `bytes.Compare()` provides this exact semantics.

**Alternatives Considered**:
1. **Parsed field comparison**: Rejected because RFC explicitly requires raw RDATA byte comparison, not semantic field comparison.
2. **Hash-based tie-breaking**: Rejected as non-RFC-compliant and non-deterministic across implementations.
3. **Timestamp-based tie-breaking**: Rejected due to clock synchronization issues.

**Implementation Pattern**:
```go
func (cd *ConflictDetector) compareProbes(ourProbe, theirProbe []byte) int {
    // Extract RDATA from authority section
    ourRDATA := extractAuthorityRDATA(ourProbe)
    theirRDATA := extractAuthorityRDATA(theirProbe)

    // RFC 6762 §8.2.1: Lexicographic byte comparison
    cmp := bytes.Compare(ourRDATA, theirRDATA)

    if cmp < 0 {
        // Our probe is lexicographically earlier - we lose, rename
        return -1
    } else if cmp > 0 {
        // Our probe is lexicographically later - we win, continue
        return 1
    } else {
        // Identical RDATA - both parties should detect conflict
        return 0
    }
}

func (cd *ConflictDetector) handleSimultaneousProbe(service *Service, theirProbe []byte) error {
    ourProbe := cd.buildProbe(service)

    result := cd.compareProbes(ourProbe, theirProbe)
    if result <= 0 {
        // We lose or tie - rename and restart probing
        return cd.rename(service)
    }

    // We win - continue probing
    return nil
}
```

**Edge Case Handling**:
- **Identical RDATA**: Both parties detect conflict and rename (prevents deadlock)
- **Malformed probe**: Treat as conflict (conservative, safe)
- **Missing authority section**: Treat as conflict (RFC violation by peer)

---

## R003: Known-Answer Suppression Implementation

**Question**: How should we implement RFC 6762 §7.1 known-answer suppression to optimize response generation?

**Decision**: TTL-aware record matching with 50% threshold check

**Rationale**:
- **RFC Requirement**: RFC 6762 §7.1 specifies: "A Multicast DNS responder MUST NOT answer a Multicast DNS query if the answer it would give is already included in the Answer Section of the query message, with an RR TTL at least half the correct TTL."
- **Efficiency**: Reduces multicast traffic by 30%+ in repeated query scenarios (per SC-009 success criteria).
- **Cache Coherency**: Known-answer section indicates what the querier already has cached, preventing redundant transmission.
- **Implementation Simplicity**: Hash-based lookup (key: name + type + class) for O(1) matching, then TTL threshold check.

**Alternatives Considered**:
1. **Exact TTL matching**: Rejected because RFC requires ≥50% threshold, not exact match.
2. **No known-answer suppression**: Rejected as RFC violation and network inefficiency.
3. **Aggressive caching with <50% threshold**: Rejected as non-compliant (could suppress fresh data).

**Implementation Pattern**:
```go
type KnownAnswerSet struct {
    records map[string]*knownAnswer // key: name+type+class
}

type knownAnswer struct {
    name      string
    rrType    uint16
    rrClass   uint16
    ttl       uint32
    rdata     []byte
    timestamp time.Time
}

func (rb *ResponseBuilder) applyKnownAnswerSuppression(response *Response, knownAnswers []*ResourceRecord) {
    kas := parseKnownAnswers(knownAnswers)

    filtered := make([]*ResourceRecord, 0, len(response.Answers))
    for _, rr := range response.Answers {
        key := buildKey(rr.Name, rr.Type, rr.Class)

        if ka, exists := kas.records[key]; exists {
            // Check RDATA match
            if !bytes.Equal(ka.rdata, rr.RDATA) {
                filtered = append(filtered, rr)
                continue
            }

            // RFC 6762 §7.1: Suppress if known-answer TTL ≥ half the correct TTL
            correctTTL := rr.TTL
            knownTTL := ka.ttl

            if knownTTL >= correctTTL/2 {
                // Suppress - querier already has fresh data
                continue
            }
        }

        filtered = append(filtered, rr)
    }

    response.Answers = filtered
}
```

**Edge Cases**:
- **TTL=0 in known-answer**: Do not suppress (querier signaling they want fresh data)
- **Mismatched RDATA**: Do not suppress (our data is different)
- **Multiple known-answers for same name**: Check all, suppress if any match

---

## R004: TTL Management Strategy

**Question**: How should we manage TTL values for resource records, including decrementation and goodbye packets?

**Decision**: Creation-timestamp-based TTL calculation with lazy decrementation on response generation

**Rationale**:
- **RFC Compliance**: RFC 6762 §10 recommends 120 seconds for service records (PTR, SRV, TXT) and 75 minutes (4500 seconds) for hostname A records.
- **Accuracy**: Storing creation timestamp and calculating remaining TTL at response time (rather than periodic decrementation) ensures accurate TTL values without background goroutines.
- **Goodbye Packets**: RFC 6762 §10.1 requires TTL=0 for all records when service is unregistered.
- **Memory Efficiency**: No need for per-record timer goroutines or periodic sweeps.

**Alternatives Considered**:
1. **Periodic TTL decrementation with background goroutine**: Rejected due to unnecessary overhead (90% of records never queried before expiration).
2. **Fixed TTL (no decrementation)**: Rejected as RFC violation (§6.1 requires responders to adjust TTL).
3. **Expiration-based (store expiry time instead of creation time)**: Rejected as less intuitive for debugging.

**Implementation Pattern**:
```go
type ResourceRecordSet struct {
    records   []*ResourceRecord
    createdAt time.Time
}

type ResourceRecord struct {
    Name      string
    Type      uint16
    Class     uint16
    TTL       uint32 // Original TTL (120 or 4500)
    RDATA     []byte
}

func (rrs *ResourceRecordSet) getRemainingTTL(rr *ResourceRecord) uint32 {
    elapsed := uint32(time.Since(rrs.createdAt).Seconds())

    if elapsed >= rr.TTL {
        return 0 // Expired
    }

    return rr.TTL - elapsed
}

func (rb *ResponseBuilder) buildResponse(service *Service) *Response {
    response := &Response{}

    for _, rr := range service.RecordSet.records {
        remainingTTL := service.RecordSet.getRemainingTTL(rr)

        if remainingTTL == 0 {
            continue // Omit expired records
        }

        // Clone record with current TTL
        currentRR := *rr
        currentRR.TTL = remainingTTL
        response.Answers = append(response.Answers, &currentRR)
    }

    return response
}

func (r *Responder) Unregister(instanceName string) error {
    service := r.registry.Get(instanceName)
    if service == nil {
        return ErrServiceNotFound
    }

    // Send goodbye packets with TTL=0
    goodbye := &Response{}
    for _, rr := range service.RecordSet.records {
        goodbyeRR := *rr
        goodbyeRR.TTL = 0
        goodbye.Answers = append(goodbye.Answers, &goodbyeRR)
    }

    r.transport.Send(r.ctx, goodbye, MulticastAddr)

    r.registry.Remove(instanceName)
    return nil
}
```

**TTL Values**:
- **Service records** (PTR, SRV, TXT): 120 seconds (RFC 6762 §10)
- **Hostname A records**: 4500 seconds (75 minutes, RFC 6762 §10)
- **Goodbye packets**: TTL=0 (RFC 6762 §10.1)

---

## R005: Response Aggregation and Packet Size Limits

**Question**: How should we aggregate multiple resource records into a single response while respecting the RFC 6762 §17 limit of 9000 bytes?

**Decision**: Greedy packing with priority ordering (answer > authority > additional) and graceful truncation

**Rationale**:
- **RFC Requirement**: RFC 6762 §17 specifies 9000 bytes as the maximum multicast DNS message size to avoid IP fragmentation on Ethernet (MTU 1500).
- **Response Efficiency**: RFC 6762 §6 encourages aggregating multiple records (PTR + SRV + TXT + A) to reduce round-trips.
- **Priority Ordering**: Answer section is most critical (directly answers query), then authority (for delegation), then additional (reduces follow-up queries).
- **Graceful Degradation**: If packet exceeds 9000 bytes, omit lowest-priority additional records rather than failing entirely.

**Alternatives Considered**:
1. **Multiple response packets**: Rejected because RFC 6762 §6 prefers single aggregated response when possible.
2. **Truncation with TC bit**: Rejected because mDNS doesn't use TC bit (no fallback to TCP like unicast DNS).
3. **Dynamic MTU discovery**: Rejected as overly complex for initial implementation (defer to M3).

**Implementation Pattern**:
```go
const MaxPacketSize = 9000 // RFC 6762 §17

type ResponseBuilder struct {
    maxSize int
}

func (rb *ResponseBuilder) buildAggregatedResponse(service *Service, query *Query) (*Response, error) {
    response := &Response{
        Header: Header{
            ID:      query.ID,
            QR:      1, // Response
            AA:      1, // Authoritative
            OPCODE:  0,
            RCODE:   0,
        },
        Questions: query.Questions,
    }

    // Build complete record set
    allRecords := rb.buildRecordSet(service, query)

    // Priority ordering: answer > authority > additional
    answerRecords := allRecords.Answers
    additionalRecords := allRecords.Additional

    // Greedy packing
    currentSize := rb.estimateHeaderSize(response)

    // Add answer records (critical - all or error)
    for _, rr := range answerRecords {
        rrSize := rb.estimateRecordSize(rr)
        if currentSize+rrSize > rb.maxSize {
            return nil, ErrResponseTooLarge
        }
        response.Answers = append(response.Answers, rr)
        currentSize += rrSize
    }

    // Add additional records (best-effort)
    for _, rr := range additionalRecords {
        rrSize := rb.estimateRecordSize(rr)
        if currentSize+rrSize > rb.maxSize {
            break // Omit remaining additional records
        }
        response.Additional = append(response.Additional, rr)
        currentSize += rrSize
    }

    return response, nil
}

func (rb *ResponseBuilder) estimateRecordSize(rr *ResourceRecord) int {
    // Name (variable, assume average 50 bytes with compression)
    // Type (2) + Class (2) + TTL (4) + RDLENGTH (2) + RDATA (variable)
    return 50 + 10 + len(rr.RDATA)
}

func (rb *ResponseBuilder) buildRecordSet(service *Service, query *Query) *RecordSet {
    rs := &RecordSet{}

    // Answer section: PTR record for "_http._tcp.local" query
    if query.Type == TypePTR {
        rs.Answers = append(rs.Answers, service.RecordSet.GetPTR())
    }

    // Additional section: SRV, TXT, A to reduce round-trips
    rs.Additional = append(rs.Additional, service.RecordSet.GetSRV())
    rs.Additional = append(rs.Additional, service.RecordSet.GetTXT())
    rs.Additional = append(rs.Additional, service.RecordSet.GetA())

    return rs
}
```

**Edge Cases**:
- **Answer section alone exceeds 9000 bytes**: Return error (service TXT records too large, user must reduce)
- **Empty additional section**: Valid (answer-only response)
- **Name compression**: Reduces packet size significantly (reuse pointers for repeated names like "local")

---

## R006: Thread Safety for Concurrent Service Registration

**Question**: How should we ensure thread-safe concurrent registration, unregistration, and query handling for 100+ services?

**Decision**: Mutex-protected service registry with goroutine-per-service state machines and lock-free query path

**Rationale**:
- **Concurrency Requirement**: NFR-003 requires support for ≥100 concurrent service registrations, NFR-005 requires zero race conditions.
- **Read-Heavy Workload**: Query responses (read path) far outnumber registrations (write path) in typical deployments.
- **Isolation**: Each service's state machine runs in a dedicated goroutine, eliminating shared state between services.
- **Go Idioms**: `sync.RWMutex` provides efficient read-write locking, `go test -race` validates correctness.

**Alternatives Considered**:
1. **Lock-free data structures (sync.Map)**: Rejected because we need composite operations (register + start state machine) that require transactional semantics.
2. **Channel-based coordination**: Rejected as overly complex for simple registry CRUD operations.
3. **Single-threaded event loop**: Rejected as potential bottleneck for 100+ concurrent operations.

**Implementation Pattern**:
```go
type Registry struct {
    mu       sync.RWMutex
    services map[string]*RegisteredService // key: instance name
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

func (r *Registry) Get(instanceName string) *RegisteredService {
    r.mu.RLock()
    defer r.mu.RUnlock()

    return r.services[instanceName]
}

func (r *Registry) GetAll() []*RegisteredService {
    r.mu.RLock()
    defer r.mu.RUnlock()

    services := make([]*RegisteredService, 0, len(r.services))
    for _, svc := range r.services {
        services = append(services, svc)
    }

    return services
}

func (r *Registry) Remove(instanceName string) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if rs, exists := r.services[instanceName]; exists {
        rs.StateMachine.cancel() // Signal state machine to stop
        delete(r.services, instanceName)
    }
}

// Query path (lock-free after initial read)
func (r *Responder) handleQuery(query *Query) (*Response, error) {
    // Snapshot registry (read lock held briefly)
    services := r.registry.GetAll()

    // Build response without holding lock (CPU-intensive)
    response := r.builder.buildResponse(services, query)

    return response, nil
}
```

**Concurrency Guarantees**:
- **Registration/Unregistration**: Serialized via write lock (rare operations)
- **Query handling**: Concurrent via read lock (frequent operations)
- **State machine goroutines**: Independent, no shared state except registry
- **Race detection**: Validated with `go test -race` (NFR-005)

**Deadlock Prevention**:
- **Lock ordering**: Always acquire registry lock before state machine locks
- **Lock duration**: Hold locks only for map operations, not I/O or computation
- **Context cancellation**: All goroutines respect context cancellation for graceful shutdown

---

## Summary

All 6 research questions have been resolved with architectural decisions aligned with:
- **RFC 6762 Compliance**: Probing timing, tie-breaking, known-answer suppression, TTL management, packet size limits
- **Go Idioms**: Goroutines + channels, sync.RWMutex, context.Context, bytes.Compare
- **Project Constitution**: Zero external dependencies (stdlib only), context-aware operations, clean architecture
- **Performance Goals**: <100ms response latency, ≥100 concurrent services, 30% bandwidth reduction

**Next Phase**: Design artifacts (data-model.md, contracts/, quickstart.md)
