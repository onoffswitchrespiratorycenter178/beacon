# Foundation Consolidation Data Model (T010 - D001)

**Task**: Define entities, relationships, and status values for compliance tracking
**Date**: 2025-11-02

---

## Entities

### RFC Section
Represents a single requirement from RFC 6762 or RFC 6763.

**Attributes**:
- `section_number` (string): RFC section number (e.g., "§5.3", "§11")
- `rfc_number` (string): RFC number (e.g., "RFC 6762", "RFC 6763")
- `requirement_text` (string): Brief description of requirement
- `status` (enum): ✅ Implemented, ❌ Not Implemented, ⚠️ Partial, 🔄 In Progress, 📋 Planned
- `implementation_evidence` (string): Link to code/spec (e.g., "querier/querier.go:45")
- `platform_notes` (string): Platform-specific status (e.g., "Linux ✅, macOS ⚠️, Windows ⚠️")
- `fr_references` (array): List of FR-IDs that implement this requirement (for bidirectional traceability)

**Relationships**:
- Many-to-Many with **Functional Requirement** (implements)

---

### Functional Requirement (FR)
Represents a single testable requirement.

**Attributes**:
- `fr_id` (string): Milestone-prefixed ID (e.g., "FR-M1-001", "FR-M1R-003", "FR-M1.1-015")
- `description` (string): Requirement text (from spec checklists)
- `status` (enum): Implemented, Deferred, Platform-Specific
- `milestone` (string): M1, M1-R, M1.1
- `functional_area` (string): Category (e.g., "Socket Configuration", "Interface Management", "Security", "Querying", "Error Handling", "Testing")
- `implementation_files` (array): Relative file paths (e.g., ["querier/querier.go:45", "internal/message/builder.go:23"])
- `rfc_references` (array): RFC sections (e.g., ["RFC 6762 §5.1", "RFC 1035 §4.1.4"])
- `test_evidence` (array): Test file paths (e.g., ["tests/integration/TestQuery", "tests/unit/TestQueryValidation"])

**Relationships**:
- Many-to-One with **Milestone** (belongs to)
- Many-to-Many with **RFC Section** (implements)

**Rationale for milestone-prefixed IDs**: Preserves traceability to source code comments, git commit messages, and original spec checklists without renumbering risk (see analysis report).

---

### Success Criterion (SC)
Represents a measurable outcome from spec.md.

**Attributes**:
- `sc_id` (string): Success criterion ID (e.g., "SC-001", "SC-M1.1-001")
- `description` (string): Criterion text
- `status` (enum): Met, Partially Met, Not Met
- `validation_method` (string): How to measure (e.g., "Stakeholder usability test")
- `evidence` (string): Proof of completion (e.g., "tests/integration/TestAvahiCoexistence")

**Relationships**:
- Many-to-Many with **Functional Requirement** (validates)

---

### Milestone
Represents a development phase.

**Attributes**:
- `milestone_id` (string): M1, M1-R, M1.1
- `name` (string): "Basic Querier", "Architectural Refactoring", "Architectural Hardening"
- `status` (enum): Complete, In Progress, Planned
- `completion_date` (date): YYYY-MM-DD
- `task_count` (integer): Total tasks completed
- `fr_count` (integer): Total FRs in this milestone

**Relationships**:
- One-to-Many with **Functional Requirement** (contains)

---

### Compliance Status
Aggregate project health metric.

**Attributes**:
- `rfc_6762_percentage` (float): Compliance % (calculated)
- `rfc_6763_percentage` (float): Compliance % (calculated)
- `total_frs` (integer): Total Foundation FRs (59)
- `implemented_frs` (integer): Implemented count
- `last_updated` (date): YYYY-MM-DD

**Relationships**:
- Calculated from **RFC Section** and **Functional Requirement** entities

---

## Relationships

```
RFC Section ←→ Functional Requirement (many-to-many)
  - RFC Section has "Implemented via FR-XXX" column
  - Functional Requirement has "RFC Reference(s)" column

Functional Requirement → Milestone (many-to-one)
  - Each FR belongs to exactly one milestone
  - Milestone contains multiple FRs

Success Criterion → Functional Requirement (many-to-many)
  - Each SC validates one or more FRs
  - Each FR may be validated by multiple SCs

Compliance Status ← RFC Section (calculated)
  - Count ✅ and ⚠️ statuses
  - Apply compliance formula
```

---

## Status Value Definitions

### RFC Section Status
- **✅ Implemented**: Feature complete and tested
- **❌ Not Implemented**: Not yet started
- **⚠️ Partial**: Some functionality implemented, some deferred
- **🔄 In Progress**: Actively being implemented
- **📋 Planned**: Specified and scheduled

### FR Status
- **Implemented**: Fully implemented and tested
- **Deferred**: Implementation deferred to future milestone (e.g., "Deferred to M2")
- **Platform-Specific**: Implemented with platform variations (e.g., "Linux ✅, macOS ⚠️, Windows ⚠️")

### Milestone Status
- **Complete**: All tasks and FRs finished
- **In Progress**: Active development
- **Planned**: Specified but not started

---

**Generated**: 2025-11-02
**Next**: Use this data model when creating matrices and dashboard
