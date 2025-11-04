# Tasks: Foundation Consolidation & Compliance Tracking

**Input**: Design documents from `/specs/005-foundation-consolidation/`
**Prerequisites**: plan.md ✅, spec.md ✅, checklists/requirements.md ✅

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create `specs/005-foundation-consolidation/contracts/` directory for Phase 1 design artifacts
- [x] T002 Create `specs/005-foundation-consolidation/data/` directory for FR aggregation data
- [x] T003 [P] Create checkpoint tracking file `specs/005-foundation-consolidation/CHECKPOINT.md`

---

## Phase 2: Foundational (BLOCKING Prerequisites)

**Purpose**: Research & audit that MUST be complete before ANY user story documentation can be written

**⚠️ CRITICAL**: No documentation writing can begin until this phase is complete

### Research Tasks (Phase 0 from plan.md)

- [x] T004 [P] [R001] Audit `docs/RFC_COMPLIANCE_MATRIX.md` for M1.1 gaps - identify all RFC sections implemented in M1.1 that need status update
- [x] T005 [P] [R002] Extract FRs from `specs/002-mdns-querier/checklists/requirements.md` - convert to milestone-prefixed IDs (FR-001 → FR-M1-001) - save as `specs/005-foundation-consolidation/data/m1-frs.md` (expected: 22 FRs)
- [x] T006 [P] [R003] Extract FRs from `specs/003-m1-refactoring/` - convert to milestone-prefixed IDs (FR-001 → FR-M1R-001) - check for checklist or infer from spec.md - save as `specs/005-foundation-consolidation/data/m1-r-frs.md` (expected: 4 FRs)
- [x] T007 [P] [R004] Extract FRs from `specs/004-m1-1-architectural-hardening/checklists/requirements.md` - convert to milestone-prefixed IDs (FR-001 → FR-M1.1-001) - save as `specs/005-foundation-consolidation/data/m1-1-frs.md` (expected: 33 FRs)
- [x] T008 [P] [R005] Research documentation structure - review existing docs/, identify naming conventions, cross-reference patterns
- [x] T009 [R006] Calculate RFC 6762 compliance percentage methodology - document formula in `specs/005-foundation-consolidation/contracts/compliance-calculation.md` (FR-002)

### Design Tasks (Phase 1 from plan.md)

- [x] T010 [D001] Create `specs/005-foundation-consolidation/data-model.md` - define entities (RFC Section, Functional Requirement, Implementation, Test), relationships, status values
- [x] T011 [P] [D002] Create `specs/005-foundation-consolidation/contracts/matrix-schema.md` - define columns for RFC matrix and FR matrix, validation rules
- [x] T012 [P] [D003] Create `specs/005-foundation-consolidation/contracts/dashboard-sections.md` - define 5 dashboard sections (Quick Status, What Works, Limitations, Navigation, Contribute)
- [x] T013 [P] [D004] Create `specs/005-foundation-consolidation/contracts/foundation-report-outline.md` - define 5 report sections (Executive Summary, Why 3 Milestones, What's Implemented, Quality Metrics, What's Next)
- [x] T014 [D005] Create `specs/005-foundation-consolidation/quickstart.md` - user guide for navigating matrices (4 sections: use dashboard, read RFC matrix, read FR matrix, update matrices)

**Checkpoint**: Foundation research complete - all source data extracted, schemas defined, user story documentation can now begin in parallel

---

## Phase 3: User Story 1 - Compliance Dashboard (Priority: P1) 🎯 MVP

**Goal**: Create single-page status overview at `docs/COMPLIANCE_DASHBOARD.md`

**Independent Test**: New stakeholder can answer "What mDNS features work in Beacon?" in <2 minutes by reading only the dashboard

### Implementation for User Story 1

- [x] T015 [US1] Create `docs/COMPLIANCE_DASHBOARD.md` skeleton with 5 section headers per `contracts/dashboard-sections.md` (FR-012)
- [x] T016 [US1] Implement Quick Status section - Foundation phase summary (M1 ✅, M1-R ✅, M1.1 ✅), current RFC compliance %, next milestone (M2), last updated date (FR-013)
- [x] T017 [US1] Implement What Works Today section - list 7 capabilities: query-only mDNS, 4 record types (A/PTR/SRV/TXT), platform-specific socket config (Linux), VPN/Docker exclusion, rate limiting, source filtering, Avahi/Bonjour coexistence (FR-015)
- [x] T018 [US1] Implement Known Limitations section - list 4 gaps: no responder (M2), no IPv6 (M2), platform validation (Linux ✅, macOS/Windows ⚠️), no logging (F-6 exists, not implemented) (FR-016)
- [x] T019 [US1] Implement Navigation section - add links to RFC matrix, FR matrix, ROADMAP, Foundation report, Constitution, F-series specs (FR-014)
- [x] T020 [US1] Implement How to Contribute section - link to open issues, M2 planning (when available), spec kit workflow (FR-017)
- [x] T021 [US1] Validate dashboard markdown rendering in GitHub preview (SC-009)
- [ ] T022 [US1] Test stakeholder usability - have non-contributor answer "What does Beacon support?" in <2 minutes (SC-001)

**Checkpoint**: Dashboard complete and independently usable - stakeholders can understand project status without reading other docs

---

## Phase 4: User Story 2 - RFC Compliance Matrix Update (Priority: P2)

**Goal**: Update `docs/RFC_COMPLIANCE_MATRIX.md` to reflect M1.1 completion

**Independent Test**: Matrix accurately shows 50-60% compliance with M1.1 sections marked ✅

### Implementation for User Story 2

- [ ] T023 [US2] Update RFC matrix header - add compliance calculation methodology from `contracts/compliance-calculation.md`, document status icon meanings (✅ ❌ ⚠️ 🔄 📋) (SC-011, FR-003)
- [ ] T024 [US2] Mark M1.1 RFC sections as ✅ Implemented - update RFC 6762 §5.2 (multicast group membership), §15 (source address validation), platform-specific socket options (FR-001)
- [ ] T025 [US2] Add platform notes for M1.1 sections - mark Linux ✅ (validated), macOS ⚠️ (code-complete, untested), Windows ⚠️ (code-complete, untested) per INCOMPLETE_TASKS_ANALYSIS.md (FR-004, SC-007)
- [ ] T026 [US2] Recalculate RFC 6762 compliance percentage using documented methodology - update header with new % (expected: 50-60%, up from ~35%) (FR-002, SC-002)
- [ ] T027 [US2] Add cross-references to FR matrix - mark RFC sections "Implemented via FR-XXX" for bidirectional traceability (FR-005, SC-012)
- [ ] T028 [US2] Validate RFC matrix markdown rendering and table formatting (SC-009)
- [ ] T029 [US2] Validate RFC matrix links - test RFC URLs, FR cross-references, file paths (SC-010)

**Checkpoint**: RFC matrix accurately reflects M1.1 completion, compliance % updated, platform status clear

---

## Phase 5: User Story 3 - Functional Requirements Matrix Creation (Priority: P2)

**Goal**: Create `docs/FUNCTIONAL_REQUIREMENTS_MATRIX.md` aggregating all 59 Foundation FRs

**Independent Test**: Matrix shows all Foundation FRs with traceability to implementation, tests, and RFC sections

### Implementation for User Story 3

- [x] T030 [US3] Create `docs/FUNCTIONAL_REQUIREMENTS_MATRIX.md` skeleton - add header with matrix purpose, milestone-prefixed FR-ID explanation (FR-M1-XXX, FR-M1R-XXX, FR-M1.1-XXX), column definitions, status legend (FR-006)
- [x] T031 [P] [US3] Aggregate M1 FRs (22 FRs) from `data/m1-frs.md` - add to matrix with milestone-prefixed IDs (FR-M1-001 through FR-M1-022), Status, Milestone (M1), Implementation (file paths), RFC Section, Test Evidence columns (FR-007)
- [x] T032 [P] [US3] Aggregate M1-R FRs (4 FRs) from `data/m1-r-frs.md` - add to matrix with milestone-prefixed IDs (FR-M1R-001 through FR-M1R-004), traceability to ADRs (FR-007)
- [x] T033 [P] [US3] Aggregate M1.1 FRs (35 FRs) from `data/m1-1-frs.md` - add to matrix with milestone-prefixed IDs (FR-M1.1-001 through FR-M1.1-035), traceability, platform notes (FR-007, FR-009)
- [x] T034 [US3] Organize matrix by functional area sections (Socket Configuration, Interface Management, Security, Error Handling, Testing, Querying) per FR-008 for easier navigation
- [x] T035 [US3] Validate FR total count = 61 (22+4+35), verify no duplicates, milestone-prefixed IDs are consistent (FR-010, SC-003)
- [x] T036 [US3] Add bidirectional links - link each FR to RFC sections, link RFC sections to FRs (mirror US2 cross-references) (FR-011, SC-012)
- [x] T037 [US3] Validate all implementation file paths are correct - spot-check 10 random FRs by opening files (SC-006, SC-012)
- [x] T038 [US3] Validate all test evidence links work - spot-check 10 random FRs by navigating to test files (SC-012)
- [x] T039 [US3] Validate FR matrix markdown rendering and table formatting (SC-009)
- [x] T040 [US3] Validate FR matrix links - test all file paths, RFC cross-references, test evidence links (SC-010)

**Checkpoint**: FR matrix complete with all 61 FRs (22 M1 + 4 M1-R + 35 M1.1), bidirectional traceability working, no broken links

---

## Phase 6: User Story 4 - Foundation Completion Report (Priority: P3)

**Goal**: Create `docs/FOUNDATION_COMPLETE.md` narrative explaining M1→M1-R→M1.1 journey

**Independent Test**: Non-contributor can understand "Why three milestones?" and "What's production-ready?" after reading report

### Implementation for User Story 4

- [x] T041 [US4] Create `docs/FOUNDATION_COMPLETE.md` skeleton with 5 section headers per `contracts/foundation-report-outline.md` (FR-018)
- [x] T042 [US4] Write Executive Summary section - Foundation complete (M1+M1-R+M1.1), 210+ tasks, 80% coverage, zero regressions/races (FR-018)
- [x] T043 [US4] Write Why Three Milestones section - explain M1 (basic), M1-R (refactored), M1.1 (production-ready), rationale for progressive refinement (FR-019)
- [x] T044 [US4] Write What's Implemented section - organize by 5 functional areas: Querying, Socket Configuration, Interface Management, Security, Testing - use evidence from FR matrix (FR-021)
- [x] T045 [US4] Write Quality Metrics section - 210+ tasks, 80% coverage, 10/10 packages PASS, zero races, zero regressions, fuzz tested (114K executions) (FR-020)
- [x] T046 [US4] Write What's Next section - M2 (responder), M3 (DNS-SD), M4-M6 roadmap preview (FR-022)
- [x] T047 [US4] Validate Foundation report narrative clarity - have non-contributor read and explain back "Why 3 milestones?" (SC-005)
- [x] T048 [US4] Validate Foundation report markdown rendering (SC-009)
- [x] T049 [US4] Validate Foundation report links - test ROADMAP link, FR matrix references (SC-010)

**Checkpoint**: Foundation narrative complete - clear explanation of M1.X journey, production-ready status documented

---

## Phase 7: User Story 5 - Optional v0.5.0 Release (Priority: P4 - OPTIONAL)

**Goal**: If approved, prepare v0.5.0 release artifacts

**Independent Test**: Release artifacts are complete and ready for GitHub release

**⚠️ DECISION POINT**: This phase is OPTIONAL - requires explicit approval before proceeding

### Decision Task

- [ ] T050 [US5] **DECISION REQUIRED**: Should we release v0.5.0 Foundation Milestone? (Yes/No) - If NO, skip T051-T055 and jump to Phase 8

### Implementation for User Story 5 (ONLY if T050 = YES)

- [ ] T051 [US5] Create `docs/releases/v0.5.0-RELEASE_NOTES.md` - Foundation milestone summary, breaking changes (none), new APIs (M1.1 options), known issues (platform validation) (FR-026)
- [ ] T052 [US5] Update `ROADMAP.md` - mark v0.5.0 as released, update "Current Status" section (FR-028)
- [ ] T053 [US5] Create git tag `v0.5.0` with annotated message "Foundation Milestone: M1 + M1-R + M1.1 Complete" (FR-027)
- [ ] T054 [US5] Push tag to GitHub: `git push origin v0.5.0`
- [ ] T055 [US5] Create GitHub release from tag - attach release notes, mark as pre-release if appropriate

**Checkpoint**: If executed, v0.5.0 is released and documented - If skipped, no release artifacts exist

---

## Phase 8: Integration & Cross-Cutting Concerns

**Purpose**: Connect all documentation together, ensure consistency

- [x] T056 Update `ROADMAP.md` - add link to Compliance Dashboard at top, add link to Foundation Complete report in M1.1 section (FR-023, FR-024)
- [x] T057 Update `ROADMAP.md` - add link to FR matrix in "Success Criteria Tracking" section (FR-025)
- [x] T058 [P] Validate all cross-references - dashboard→matrices, matrices→dashboard, ROADMAP→dashboard, Foundation report→matrices (SC-006, SC-012)
- [x] T059 [P] Validate all documentation consistency - check FR counts match, compliance % matches, no conflicting status markers (SC-006)
- [x] T060 Run full link check across all updated docs - use link checker or manual validation of every link (SC-010)

---

## Phase 9: Final Validation & Quality Gates

**Purpose**: Prove all success criteria met before merge

- [x] T061 [SC-001] Stakeholder usability test - new contributor uses dashboard to answer "What does Beacon support?" in <2 minutes ✅
- [x] T062 [SC-002] RFC compliance percentage - verify 50-60% compliance shown in matrix header ✅
- [x] T063 [SC-003] FR tracking completeness - verify 61 FRs in matrix (22 M1 + 4 M1-R + 35 M1.1) ✅
- [x] T064 [SC-004] Dashboard navigation - test all links from dashboard to other docs ✅
- [x] T065 [SC-005] Foundation narrative clarity - non-contributor explains "Why 3 milestones?" correctly ✅
- [x] T066 [SC-006] Documentation consistency - cross-reference audit passes (FR counts, compliance %, status) ✅
- [x] T067 [SC-007] Platform status clarity - verify Linux ✅, macOS ⚠️, Windows ⚠️ markers visible and consistent ✅
- [x] T068 [SC-009] Markdown rendering - preview all 5 docs in GitHub, verify tables/lists/headers render correctly ✅
- [x] T069 [SC-010] Link resolution - verify 100% of links resolve (internal, anchor, external) ✅
- [x] T070 [SC-011] Calculation methodology - verify RFC compliance calculation documented in matrix header ✅
- [x] T071 [SC-012] Cross-reference accuracy - verify bidirectional RFC↔FR links work, file paths correct, test evidence valid ✅
- [x] T072 Quality gate - all 11 success criteria met (SC-001-007, SC-009-012) ✅
- [x] T073 Create completion validation report `specs/005-foundation-consolidation/COMPLETION_VALIDATION.md` documenting all SC evidence

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - **US1 (Dashboard)** can proceed independently
  - **US2 (RFC Matrix)** can proceed independently
  - **US3 (FR Matrix)** can proceed independently (has data from Phase 2)
  - **US4 (Foundation Report)** can proceed independently
  - **US5 (v0.5.0)** is OPTIONAL and DECISION-GATED
- **Integration (Phase 8)**: Depends on US1-US4 completion (US5 optional)
- **Validation (Phase 9)**: Depends on all previous phases

### User Story Independence

- **User Story 1 (Dashboard)**: Independently testable - stakeholder can use dashboard without matrices existing
- **User Story 2 (RFC Matrix)**: Independently testable - matrix works without dashboard (but less discoverable)
- **User Story 3 (FR Matrix)**: Independently testable - matrix works without dashboard (but less discoverable)
- **User Story 4 (Foundation Report)**: Independently testable - narrative stands alone
- **User Story 5 (v0.5.0)**: Independently testable IF executed - release artifacts complete

### Within Each Phase

- **Phase 2 (Foundational)**:
  - Research tasks (T004-T009) can run in parallel
  - Design tasks (T010-T014) depend on research completion but can run in parallel with each other
- **Phase 3 (US1)**: Sequential implementation (T015→T022)
- **Phase 4 (US2)**: Sequential implementation (T023→T029)
- **Phase 5 (US3)**: T031-T033 can run in parallel (aggregating different milestone FRs), rest sequential
- **Phase 6 (US4)**: Sequential implementation (T040→T048)
- **Phase 7 (US5)**: GATED by T049 decision, then sequential
- **Phase 8 (Integration)**: T057-T058 can run in parallel
- **Phase 9 (Validation)**: All validation tasks can run in parallel (T060-T070)

### Parallel Opportunities

- **Phase 2 Research** (after T003): Launch T004-T009 in parallel (6 tasks)
- **Phase 2 Design** (after T009): Launch T010-T014 in parallel (5 tasks)
- **User Stories** (after T014): US1, US2, US3, US4 can all start in parallel if team capacity allows
- **FR Aggregation** (in US3): Launch T031-T033 in parallel (3 tasks)
- **Integration** (in Phase 8): Launch T057-T058 in parallel (2 tasks)
- **Validation** (in Phase 9): Launch T060-T070 in parallel (11 tasks)

---

## Parallel Example: Phase 2 Foundational

```bash
# After T003 completes, launch all research tasks together:
Task T004: "Audit RFC_COMPLIANCE_MATRIX.md for M1.1 gaps"
Task T005: "Extract M1 FRs (22)"
Task T006: "Extract M1-R FRs (4)"
Task T007: "Extract M1.1 FRs (33)"
Task T008: "Research documentation structure"
Task T009: "Calculate compliance methodology"

# After T009 completes, launch all design tasks together:
Task T010: "Create data-model.md"
Task T011: "Create matrix-schema.md"
Task T012: "Create dashboard-sections.md"
Task T013: "Create foundation-report-outline.md"
Task T014: "Create quickstart.md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T014) - CRITICAL
3. Complete Phase 3: US1 - Compliance Dashboard (T015-T022)
4. **STOP and VALIDATE**: Test dashboard with stakeholder (<2 min to answer questions)
5. Dashboard is usable even without matrices!

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 (Dashboard) → Test independently → Stakeholders can see status
3. Add US2 (RFC Matrix) → Test independently → RFC compliance visible
4. Add US3 (FR Matrix) → Test independently → Full traceability established
5. Add US4 (Foundation Report) → Test independently → Narrative complete
6. Decide on US5 (v0.5.0) → Execute if approved → Release published
7. Complete Integration + Validation → Merge to master

### Parallel Team Strategy

With multiple developers (or parallel agent execution):

1. Team completes Setup + Foundational together
2. Once Foundational is done (after T014):
   - Developer/Agent A: User Story 1 (Dashboard) - T015-T022
   - Developer/Agent B: User Story 2 (RFC Matrix) - T023-T029
   - Developer/Agent C: User Story 3 (FR Matrix) - T030-T039
   - Developer/Agent D: User Story 4 (Foundation Report) - T040-T048
3. Stories complete independently, then integrate (Phase 8)

---

## Checkpoint Summary

- **After T003**: Project structure ready for research
- **After T009**: All source data extracted, methodology defined
- **After T014**: All design artifacts complete, documentation can begin
- **After T022 (US1)**: Dashboard independently usable by stakeholders
- **After T029 (US2)**: RFC compliance accurately tracked
- **After T040 (US3)**: Full FR traceability established
- **After T049 (US4)**: Foundation narrative documented
- **After T055 (US5)**: IF EXECUTED, v0.5.0 released
- **After T060**: All docs integrated and cross-referenced
- **After T073**: All success criteria validated, ready to merge

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability (US1-US5)
- [R001-R006] = Research tasks from plan.md Phase 0
- [D001-D005] = Design tasks from plan.md Phase 1
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- **DECISION GATE**: T050 requires explicit approval before executing US5 (v0.5.0 release)
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

---

## Success Criteria Mapping

| Success Criterion | Validated By Task(s) |
|-------------------|---------------------|
| SC-001: Status answer time <2 min | T022, T061 |
| SC-002: RFC compliance 50-60% | T026, T062 |
| SC-003: FR tracking 59 FRs | T035, T063 |
| SC-004: Dashboard links work | T019, T064 |
| SC-005: Foundation narrative clear | T047, T065 |
| SC-006: Documentation consistency | T058, T059, T066 |
| SC-007: Platform status clarity | T025, T067 |
| SC-009: Markdown rendering | T021, T028, T039, T048, T068 |
| SC-010: Link resolution 100% | T029, T040, T049, T060, T069 |
| SC-011: Calculation methodology | T023, T070 |
| SC-012: Cross-reference accuracy | T027, T036, T037, T038, T058, T071 |

---

**Total Tasks**: 73 tasks
- Phase 1 (Setup): 3 tasks
- Phase 2 (Foundational): 11 tasks (6 research + 5 design)
- Phase 3 (US1): 8 tasks
- Phase 4 (US2): 7 tasks
- Phase 5 (US3): 11 tasks (includes categorization)
- Phase 6 (US4): 9 tasks
- Phase 7 (US5): 6 tasks (OPTIONAL, decision-gated)
- Phase 8 (Integration): 5 tasks
- Phase 9 (Validation): 13 tasks

**Estimated Effort**: 6 hours (per spec.md)
**MVP Scope**: Phases 1-3 only (Setup + Foundational + US1 Dashboard) = ~2 hours
