# Beacon Documentation

This directory contains **active** project documentation. For historical/archived documents, see [.archive/](./../.archive/).

---

## Quick Navigation

**Start Here**:
- **[Compliance Dashboard](./COMPLIANCE_DASHBOARD.md)** ⭐ - Project status overview (<2 min read)

**Compliance & Tracking**:
- [RFC Compliance Matrix](./RFC_COMPLIANCE_MATRIX.md) - Section-by-section RFC 6762 implementation (52.8% complete)
- [Functional Requirements Matrix](./FUNCTIONAL_REQUIREMENTS_MATRIX.md) - All 61 Foundation FRs with traceability
- [Foundation Completion Report](./FOUNDATION_COMPLETE.md) - M1→M1-R→M1.1 narrative

**Architecture & Security**:
- [Architectural Pitfalls & Mitigations](./ARCHITECTURAL_PITFALLS_AND_MITIGATIONS.md) - Security and resilience requirements
- [Architecture Decision Records](./decisions/) - ADRs documenting key technical decisions

---

## Document Purposes

### Compliance Dashboard (Primary Entry Point)
**Purpose**: Single-page status overview answering "Does Beacon support X?" in <2 minutes

**Use when**:
- Onboarding new contributors
- Checking current project status
- Understanding what's implemented vs. planned

**Sections**:
- Quick Status (milestones, compliance %, coverage)
- What Works Today (features, platform support)
- Known Limitations (out of scope, validation status)
- Navigation (links to all documentation)
- How to Contribute (testing needs, open issues)

---

### RFC Compliance Matrix
**Purpose**: Track RFC 6762/6763 compliance section-by-section

**Use when**:
- Planning new features (check what RFC sections are needed)
- Validating RFC compliance
- Understanding current vs. planned implementation

**Key Features**:
- Section-level status (✅ Implemented, ⚠️ Partial, ❌ Not Implemented, 🔄 In Progress, 📋 Planned)
- Platform status notation (Linux ✅, macOS/Windows ⚠️)
- Compliance calculation methodology
- Links to implementing code

**Last Updated**: 2025-11-02 (M1.1 sections marked complete, 52.8% compliance)

---

### Functional Requirements Matrix
**Purpose**: Track all Foundation FRs with traceability to code and tests

**Use when**:
- Understanding what's implemented
- Finding implementation code for a specific requirement
- Validating test coverage for FRs

**Key Features**:
- 61 FRs across 3 milestones (22 M1 + 4 M1-R + 35 M1.1)
- Milestone-prefixed IDs (FR-M1-XXX, FR-M1R-XXX, FR-M1.1-XXX)
- Bidirectional RFC↔FR links
- Implementation file paths
- Test evidence links

**Last Updated**: 2025-11-02 (Foundation complete)

---

### Foundation Completion Report
**Purpose**: Narrative explaining M1→M1-R→M1.1 progression

**Use when**:
- Understanding why we had 3 Foundation milestones
- Learning what's implemented and why
- Reviewing quality metrics
- Planning M2 (responder) work

**Key Sections**:
- Executive Summary
- Why Three Milestones? (Basic → Refactored → Production-Ready)
- What's Implemented (5 functional areas)
- Quality Metrics (210+ tasks, 80% coverage, zero regressions)
- What's Next (M2-M6 roadmap)

**Last Updated**: 2025-11-02 (Foundation complete, 602 lines)

---

### Architectural Pitfalls & Mitigations
**Purpose**: Document security and resilience requirements to avoid common mDNS pitfalls

**Use when**:
- Designing new features (check security requirements)
- Reviewing security implications
- Validating against known attack vectors

**Last Updated**: 2025-10-XX (M1.1 planning phase)

---

### Architecture Decision Records (ADRs)
**Purpose**: Document WHY we made key architectural decisions, not just WHAT

**Current ADRs**:
- [ADR-001: Transport Interface Abstraction](./decisions/001-transport-interface-abstraction.md)
- [ADR-002: Buffer Pooling Pattern](./decisions/002-buffer-pooling-pattern.md)
- [ADR-003: Integration Test Timing Tolerance](./decisions/003-integration-test-timing-tolerance.md)

**Use when**:
- Understanding rationale for architectural patterns
- Making new architectural decisions (check for precedent)
- Onboarding contributors (understand design philosophy)

---

## Document Lifecycle

**Active** (this directory):
- Compliance tracking (ongoing)
- Current milestone completion reports
- ADRs (permanent architectural record)
- Reference documents (pitfalls, security)

**Archived** ([.archive/](./../.archive/)):
- Historical planning artifacts
- Superseded validation matrices
- Early strategic analysis
- Research documents
- Milestone-specific reports (M1 analysis, M1-Refactoring metrics)

**Retention Policy**:
- Milestone completion reports: Keep active for current milestone + 1, then archive
- Planning artifacts: Archive immediately after milestone completion
- Compliance matrices: Keep latest, archive superseded versions
- ADRs: Never archive (permanent record)

---

## Related Documentation

**Project Governance**:
- [Constitution](../.specify/memory/constitution.md) - Project principles
- [ROADMAP](../ROADMAP.md) - Milestone plan (M1-M6)

**Specifications**:
- [.specify/specs/](../.specify/specs/) - F-series foundation specs (F-2 through F-11)
- [specs/](../specs/) - Feature specifications (M1, M1-R, M1.1, etc.)

**Protocol References**:
- [RFC 6762](../RFC%20Docs/rfc6762.txt) - Multicast DNS (PRIMARY AUTHORITY)
- [RFC 6763](../RFC%20Docs/rfc6763.txt) - DNS-SD

**Archived Documentation**:
- [.archive/](./../.archive/) - Historical documents (see .archive/README.md)

---

**Documentation Version**: 2.0 (Post Foundation Consolidation)
**Last Updated**: 2025-11-02
