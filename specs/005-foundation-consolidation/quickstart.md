# Compliance Documentation Quickstart (T014 - D005)

## How to Use the Compliance Dashboard
Single entry point to project status. Answers "Does Beacon support X?" in <2 minutes.

## How to Read the RFC Compliance Matrix
- Status icons: ✅ Implemented, ❌ Not Implemented, ⚠️ Partial, 🔄 In Progress, 📋 Planned
- Platform notes: Linux ✅ (validated), macOS/Windows ⚠️ (code-complete, untested)

## How to Read the FR Matrix
- FR-ID: Milestone-prefixed (FR-M1-XXX, FR-M1R-XXX, FR-M1.1-XXX) preserves traceability
- Implementation column: Links to actual code
- Test Evidence: Links to test files

## How to Update Matrices (Maintainers)
- Each milestone: Update RFC matrix status, add FRs to FR matrix, recalculate compliance %
- Validate cross-references before merging
