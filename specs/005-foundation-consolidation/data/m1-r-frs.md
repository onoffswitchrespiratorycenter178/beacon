# M1-Refactoring Functional Requirements (FR-M1R-001 through FR-M1R-004)

**Source**: Inferred from `archive/m1-refactoring/reports/REFACTORING_COMPLETE.md` and ADRs
**Milestone**: M1-Refactoring (Architectural refactoring, no new user-facing features)
**Task**: T006 (R003) - Extract and convert to milestone-prefixed IDs

---

## Architectural Requirements (4 FRs)

| FR-ID | Description | Status | Implementation | ADR Reference | Test Evidence |
|-------|-------------|--------|----------------|---------------|---------------|
| FR-M1R-001 | System MUST abstract network transport behind Transport interface to enable IPv6 and testability | ✅ Implemented | internal/transport/transport.go (interface), internal/transport/udp.go (UDPv4Transport) | ADR-001: Transport Interface Abstraction | tests/transport/TestUDPv4Transport |
| FR-M1R-002 | System MUST use buffer pooling to reduce allocations by ≥80% in receive path | ✅ Implemented | internal/transport/buffer_pool.go (sync.Pool), udp.go (Receive uses GetBuffer/PutBuffer) | ADR-002: Buffer Pooling Pattern | BenchmarkUDPv4Transport_Receive (99% reduction achieved) |
| FR-M1R-003 | System MUST propagate errors from all Close() methods per F-3 (no error swallowing) | ✅ Implemented | internal/transport/udp.go (Close returns error), querier/querier.go (propagates Close errors) | ADR-001 (error handling section) | tests/integration/TestQuerier_Close_PropagatesTransportErrors |
| FR-M1R-004 | Querier package MUST NOT directly import internal/network (strict layer boundaries per F-2) | ✅ Implemented | querier/querier.go (uses Transport interface instead of net.PacketConn) | ADR-001 (motivation section) | Validation: `grep -rn "internal/network" querier/` returns 0 matches |

---

## Summary

- **Total FRs**: 4
- **Status**: All ✅ Implemented (M1-Refactoring complete)
- **Functional Areas**:
  - Architectural: 4 FRs (interface abstraction, buffer pooling, error propagation, layer boundaries)
- **Performance Impact**: 99% allocation reduction (exceeds ≥80% target)
- **Quality Impact**: 84.8% coverage, zero regressions, 9/9 packages PASS

## Rationale

M1-Refactoring was an architectural milestone focused on code quality, performance, and future-proofing rather than new user-facing features. FRs were inferred from:
1. **ADR-001**: Transport interface abstraction (FR-M1R-001)
2. **ADR-002**: Buffer pooling pattern (FR-M1R-002)
3. **F-3 (Error Handling)**: Error propagation requirement (FR-M1R-003)
4. **F-2 (Layer Boundaries)**: Clean architecture constraint (FR-M1R-004)

These architectural improvements enable M2 (IPv6 support, responder) and improve maintainability.

---

**Generated**: 2025-11-02
**Next**: Use this data in T032 (aggregate M1-R FRs into FR matrix)
