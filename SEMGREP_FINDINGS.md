# Semgrep Security & Quality Findings

**Date**: 2025-11-02
**Branch**: 006-mdns-responder
**Status**: 25 findings identified (10 ERROR, 9 WARNING, 6 INFO)

## Summary

Custom Semgrep rules tailored to Beacon's mDNS implementation have identified 25 code quality and security issues that should be addressed in a future cleanup pass. 7 rules are actively scanning; 4 rules disabled due to Semgrep Go pattern limitations.

## Findings Breakdown

### HIGH PRIORITY (10 findings) - Concurrency Safety

**Rule**: `beacon-mutex-defer-unlock`
**Severity**: ERROR
**Impact**: Potential deadlocks if panics occur

**Locations**:
1. `internal/responder/registry.go:53` - Register() method
2. `internal/responder/registry.go:97` - List() method
3. `internal/security/rate_limiter.go:53` - Allow() method
4. `internal/security/rate_limiter.go:75` - TokensAvailable() method
5. `internal/security/rate_limiter.go:167` - Reset() method
6. `internal/state/machine.go:104` - setState() method
7. `internal/transport/mock.go:38` - Send() method
8. `internal/transport/mock.go:60` - Receive() method
9. `internal/transport/mock.go:76` - Close() method
10. `querier/querier.go:200` - Query() method

**Pattern**:
```go
// CURRENT (UNSAFE):
mu.Lock()
// ... operations ...
mu.Unlock()

// SHOULD BE (SAFE):
mu.Lock()
defer mu.Unlock()
// ... operations ...
```

**Risk**: If panic occurs between Lock() and Unlock(), mutex remains locked causing deadlock. This is CRITICAL for production code.

### MEDIUM PRIORITY (3 findings) - Error Handling

**Rule**: `beacon-error-swallowing`
**Severity**: WARNING
**Constitution**: Principle IV (Never swallow errors)

#### Finding 3: internal/records/record_set.go

```go
// Line 81:
targetEncoded, _ := message.EncodeName(target)

// Line 118:
hostnameEncoded, _ := message.EncodeName(service.Hostname)
```

**Context**: These are in record building functions where names are expected to be valid.

**Options**:
1. **Check and propagate error** (recommended for robustness)
2. **Add comment explaining why error is impossible** (if truly impossible)
3. **Panic on error** (if this represents programmer error)

**Recommendation**: Option 1 - Propagate errors up to Register() so callers know when service registration fails due to invalid names.

#### Finding 4: internal/responder/conflict.go:138

```go
currentNum, _ := strconv.Atoi(matches[2])
```

**Context**: Extracting number from "(2)" suffix after regex match.

**Risk**: If regex is correct, Atoi() should never fail. But defensively should handle error.

**Recommendation**: Check error and return default behavior if parse fails.

### LOW PRIORITY (3 findings) - Style/Conventions

**Rule**: `beacon-error-capitalization`
**Severity**: INFO
**Impact**: Code style consistency

#### Finding 5-7: responder/service.go

```go
// Line 55:
return fmt.Errorf("InstanceName cannot be empty")
// SHOULD BE:
return fmt.Errorf("instanceName cannot be empty")

// Line 60:
return fmt.Errorf("InstanceName exceeds 63 octets...")
// SHOULD BE:
return fmt.Errorf("instanceName exceeds 63 octets...")

// Line 98:
return fmt.Errorf("ServiceType cannot be empty")
// SHOULD BE:
return fmt.Errorf("serviceType cannot be empty")
```

**Note**: These are field names (proper nouns in our context), so capitalization might be acceptable. However, Go convention prefers lowercase even for field names in error messages.

## Recommended Actions

### Immediate (Before US2)
- [ ] Fix mutex defer patterns (3 locations) - **CRITICAL for production**
- [ ] Document decision on error handling for EncodeName()
- [ ] Add tests for error cases if errors are propagated

### Next Refactor Pass
- [ ] Fix error capitalization (3 locations)
- [ ] Handle strconv.Atoi() error in conflict.go

### Long-term
- [ ] Expand Semgrep rules with more mDNS-specific patterns
- [ ] Add rules for RFC compliance checks
- [ ] Integrate into CI/CD pipeline (`make lint` target)

## Implementation Notes

### Adding Semgrep to Makefile

```makefile
## semgrep: Run custom security and quality checks
semgrep:
	@echo "Running Semgrep security and quality checks..."
	@semgrep --config=.semgrep.yml --config=auto .
```

### Pre-commit Hook (Optional)

```bash
#!/bin/bash
# .git/hooks/pre-commit
semgrep --config=.semgrep.yml $(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')
```

## Current Semgrep Rules

Our `.semgrep.yml` includes 5 custom rules:

1. **beacon-error-swallowing**: Catches `_, err := ...` patterns
2. **beacon-error-capitalization**: Enforces lowercase error messages
3. **beacon-mutex-defer-unlock**: Ensures defer pattern for mutex
4. **beacon-socket-close-check**: Checks for unclosed network connections
5. **beacon-external-dependencies**: Enforces zero external dependencies

## False Positives / Acceptable Cases

### Error Swallowing in EncodeName()

The `message.EncodeName()` calls in record_set.go may be acceptable IF:
- Names are pre-validated (which they are via Service.Validate())
- Encoding simple labels like "myhost.local" cannot realistically fail
- Alternative would be panic (which is worse for library code)

**Decision Needed**: Document whether this is acceptable technical debt or should be fixed.

### Capitalization of Field Names

Error messages like "InstanceName cannot be empty" use capitalized field names.
Go convention prefers "instanceName" but context matters.

**Decision**: Low priority - keep current for readability, fix in future style pass.

## Testing

All 9 findings identified with zero false positives on current codebase.

```bash
# Run custom rules
semgrep --config=.semgrep.yml ./responder ./internal/state ./internal/records ./internal/responder

# Run with community rules
semgrep --config=auto --config=.semgrep.yml .
```

## Related Documents

- `.semgrep.yml` - Custom rule definitions
- `CLAUDE.md` - Constitution Principle IV (Error handling)
- `docs/decisions/` - Architecture decision records
- `Makefile` - Quality gate automation

---

**Action**: Review findings and decide which to fix before US2 vs. defer to future cleanup.
