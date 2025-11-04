<!--
Sync Impact Report:
- Version Change: 1.0.0 → 1.1.0
- Type: MINOR (added new Principle V: Dependencies and Supply Chain)
- Amendment Date: 2025-11-01
- Modified Principles:
  - NEW: Principle V - Dependencies and Supply Chain (allows golang.org/x/* semi-standard libs)
  - RENUMBERED: Former V→VI (Open Source), VI→VII (Maintained), VII→VIII (Excellence)
- Added Sections: Dependency Policy with justification requirements
- Removed Sections: None
- Rationale: Platform-specific networking (socket options, multicast) requires golang.org/x/sys
  and golang.org/x/net. Standard library has unfixable bugs (Go Issues #73484, #34728) that
  prevent production-grade mDNS implementation. Semi-standard libraries maintained by Go team
  provide necessary platform-specific functionality while minimizing supply chain risk.
- Templates Requiring Updates:
  ⚠️ F-series architecture specs - add dependency justifications for golang.org/x/* imports
  ✅ spec-template.md - already includes constitution compliance
  ✅ plan-template.md - already includes Constitution Check section
  ✅ tasks-template.md - review pending (verify task categorization aligns)
  ✅ checklist-template.md - review pending (verify checklist aligns with principles)
- Follow-up TODOs:
  - Update F-2 (Package Structure) or create F-9 (Transport Layer) with socket configuration justification
  - Document golang.org/x/sys/unix usage for SO_REUSEPORT in architecture specs
  - Document golang.org/x/net/ipv4 usage for multicast group management in architecture specs
-->

# Beacon Constitution

## Mission

Build the best enterprise-grade mDNS & DNS-SD implementation in Go.

## Core Principles

### I. RFC Compliant (NON-NEGOTIABLE)

Strict adherence to RFC 6762 (mDNS) and RFC 6763 (DNS-SD). Every implementation decision MUST be validated against the authoritative RFCs. No deviations from MUST requirements are permitted.

**Rationale**: Protocol compliance is the foundation of interoperability. Non-compliant implementations cause network issues, fail to interoperate with other mDNS/DNS-SD implementations (Avahi, Bonjour), and violate the core mission of enterprise-grade quality.

**Enforcement**:
- All specifications MUST reference specific RFC sections for requirements
- All architecture decisions MUST be validated against RFC mandates
- RFC MUST requirements cannot be made configurable
- Validation reports required before implementation phases begin

### II. Spec-Driven Development (NON-NEGOTIABLE)

All features MUST be designed through GitHub Spec Kit specifications before implementation. No code is written without a complete, reviewed specification.

**Rationale**: Specifications force deliberate design, enable parallel development, provide documentation, and ensure alignment with principles before committing resources.

**Enforcement**:
- No feature branch without a specification in `/specs/<feature>/`
- Specifications MUST include user scenarios, technical design, and test strategy
- Architecture specifications (F-series) govern cross-cutting concerns
- Constitution Check required in every plan

### III. Test-Driven Development (NON-NEGOTIABLE)

All code MUST follow the RED → GREEN → REFACTOR cycle. Tests are written first, validated to fail, then implementation makes them pass.

**Rationale**: TDD catches bugs early, enables confident refactoring, serves as executable documentation, and enforces testable design. Coverage ≥80% is mandatory.

**Enforcement**:
- Specifications MUST define acceptance tests before implementation
- All tests MUST pass with `-race` flag (no data races)
- Coverage reports required for all deliverables
- No merge without passing tests

### IV. Phased Approach

Deliberate, well-planned incremental delivery through milestone-based implementation. Each milestone delivers working, testable functionality.

**Rationale**: Phased delivery provides early feedback, validates approach incrementally, enables course correction, and delivers value continuously rather than in one large release.

**Implementation**:
- Phase 0: Foundation (specifications, architecture, shared context)
- Milestone-based development (M1-M6: Basic Querier → Production Ready)
- Each milestone: 2-4 weeks, delivers working code
- Validation gates between phases

### V. Dependencies and Supply Chain

Minimize external dependencies while enabling necessary platform-specific operations through carefully vetted semi-standard libraries.

**Rationale**: External dependencies introduce supply chain risk, maintenance burden, and version conflicts. However, platform-specific networking operations (socket options, multicast group management) require access to system calls unavailable in the standard library. The Go team maintains `golang.org/x/*` semi-standard libraries specifically for this purpose.

**Dependency Policy**:

**Standard Library (PREFERRED)**:
- Use standard library (`stdlib`) for all functionality when possible
- No justification required for stdlib usage

**Semi-Standard Libraries (ALLOWED WITH JUSTIFICATION)**:
- `golang.org/x/sys/*`: Platform-specific system calls (socket options, syscalls)
- `golang.org/x/net/*`: Advanced networking (multicast group management, IPv4/IPv6 control)
- Must meet ALL criteria:
  1. Required for platform-specific operations (syscalls, low-level networking)
  2. No standard library alternative exists
  3. Maintained by Go team under `golang.org/x/*`
  4. Justification documented in architecture specifications

**Third-Party Libraries (PROHIBITED WITHOUT AMENDMENT)**:
- External libraries (e.g., `github.com/*`, `gopkg.in/*`) require constitutional amendment
- Amendment must demonstrate:
  1. Critical functionality unavailable in stdlib or golang.org/x/*
  2. Library is actively maintained and widely adopted
  3. Security audit and supply chain verification
  4. No suitable alternative exists

**Enforcement**:
- All golang.org/x/* imports MUST be justified in architecture specifications
- Dependency review required in pre-implementation validation
- No third-party dependencies without explicit constitutional amendment
- Justification template: "Required for [specific operation], no stdlib alternative, Go team maintained"

**Examples of Justified golang.org/x/* Usage**:
- `golang.org/x/sys/unix`: SO_REUSEPORT socket option (Linux/macOS) - required for mDNS port sharing per RFC 6762 §5
- `golang.org/x/net/ipv4`: Multicast group membership control - required for joining/leaving mDNS multicast groups
- Standard library bugs: Go Issues #73484, #34728 make `net.ListenMulticastUDP()` unsuitable for production mDNS

### VI. Open Source

Transparent development, welcoming contributions, public specifications, and community engagement.

**Rationale**: Open source ensures accountability, enables community contributions, demonstrates quality through transparency, and serves the broader Go ecosystem.

**Implementation**:
- MIT License
- Public GitHub repository
- Open specifications and design documents
- Contributing guidelines for external contributors
- Responsive to issues and pull requests

### VII. Maintained

Long-term commitment to support, evolution, security updates, and backward compatibility.

**Rationale**: Enterprise users require stability and ongoing support. Abandoned projects create technical debt and security risks.

**Implementation**:
- Semantic versioning (MAJOR.MINOR.PATCH)
- Security vulnerability response process
- Deprecation policy (minimum 2 minor versions notice)
- Regular releases and updates
- Long-term support commitments

### VIII. Excellence

Continuous improvement toward best-in-class implementation through code review, benchmarking, optimization, and industry best practices.

**Rationale**: "Best enterprise-grade implementation" is the mission. Excellence requires ongoing refinement, measurement, and commitment to quality.

**Implementation**:
- Code review required for all changes
- Benchmark tracking for performance-critical paths
- Interoperability testing against Avahi and Bonjour
- Go best practices and idioms enforced
- Regular retrospectives and process improvements

## Architecture Validation

All architecture specifications (F-series) MUST undergo RFC validation before implementation begins.

**Requirements**:
- Cross-reference against RFC 6762 and RFC 6763
- Validation by automated agents or manual review
- Issues categorized by severity (Critical, Major, Minor)
- All P0 (blocking) issues resolved before implementation
- Validation reports published in `/docs/`

**Rationale**: Architecture defines patterns used throughout the codebase. Errors in architecture propagate to all implementations. Validating architecture against RFCs prevents systemic compliance issues.

## Non-Negotiables

The following rules are absolute and have no exceptions:

- **No feature without a specification** - Spec Kit specifications required
- **No code without tests** - TDD cycle mandatory, ≥80% coverage
- **No compromise on RFC compliance** - MUST requirements are non-negotiable
- **No release without documentation** - User-facing docs, API docs, examples

**Violation Handling**: Any violation discovered must be addressed immediately. No merges, no releases, no exceptions.

## Governance

### Amendment Process

1. Propose amendment via pull request to `.specify/memory/constitution.md`
2. Document rationale and impact analysis
3. Update version following semantic versioning:
   - **MAJOR**: Removes/changes core principles (requires migration plan)
   - **MINOR**: Adds new principles or sections
   - **PATCH**: Clarifications, wording improvements
4. Update dependent templates and documentation
5. Requires approval before merge

### Compliance Review

**Pre-Implementation**: Constitution Check in every plan.md (see F-series specs for architecture alignment)

**During Development**: All pull requests MUST verify:
- Specification exists and is current
- Tests written and passing (TDD cycle)
- RFC compliance maintained
- Documentation updated

**Post-Release**: Retrospectives review constitutional compliance and identify improvements

### Conflict Resolution

Constitution supersedes all other documentation. In case of conflict:
1. Constitution principles take precedence
2. Architecture specifications (F-series) interpret principles for implementation
3. RFC requirements override all other concerns (Principle I)
4. Consult project maintainers for interpretation

### Living Document

This constitution is a living document. As Beacon evolves, principles may be refined, but core commitments (RFC compliance, spec-driven, TDD, maintenance) are permanent.

**Version**: 1.1.0 | **Ratified**: 2025-11-01 | **Last Amended**: 2025-11-01
