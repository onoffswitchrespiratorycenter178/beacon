# Git Hooks

This directory contains git hooks for the Beacon project.

## Available Hooks

### pre-commit

**Purpose**: Runs Semgrep static analysis on staged Go files before commit.

**What it does**:
- ✅ Scans only staged `.go` files (not entire codebase)
- ✅ Reports all Semgrep findings (ERROR, WARNING, INFO)
- ❌ **Blocks commit** if findings are detected
- ⏭️ Can be bypassed with `git commit --no-verify` (not recommended)

**Why it matters**:
- Catches bugs BEFORE they're committed
- Enforces Constitution principles automatically
- Prevents F-Spec violations from entering codebase
- Saves time in code review

## Installation

### Option 1: Configure Git to Use .githooks (Recommended)

This applies the hooks to your local repository:

```bash
git config core.hooksPath .githooks
```

**Advantages**:
- Hooks are automatically updated when you pull changes
- Works for all future hooks added to this directory
- Easy to enable/disable

**To verify**:
```bash
git config core.hooksPath
# Should output: .githooks
```

### Option 2: Symlink Individual Hooks

This is the traditional approach:

```bash
ln -sf ../../.githooks/pre-commit .git/hooks/pre-commit
```

**Advantages**:
- More control over which hooks are active
- Standard git approach

**Disadvantages**:
- Must manually update when hooks change
- Must repeat for each hook

## Testing the Hook

### Test with a Known Violation

1. Create a file with a Semgrep violation:

```bash
cat > test_violation.go <<'EOF'
package main

import "sync"

func bad() {
    var mu sync.Mutex
    mu.Lock()  // Missing defer mu.Unlock()
    // work here
}
EOF
```

2. Try to commit it:

```bash
git add test_violation.go
git commit -m "test"
```

3. You should see:

```
🔍 Running Semgrep static analysis...
   Scanning 1 Go file(s)...

❌ Semgrep found 1 issue(s)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[findings displayed]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

4. Fix the code and commit should succeed:

```bash
# Fix the violation
cat > test_violation.go <<'EOF'
package main

import "sync"

func good() {
    var mu sync.Mutex
    mu.Lock()
    defer mu.Unlock()  // ✅ Now has defer
    // work here
}
EOF

git add test_violation.go
git commit -m "test"  # Should succeed
```

## Bypassing the Hook

**⚠️ NOT RECOMMENDED** - Only use in exceptional circumstances:

```bash
git commit --no-verify
```

**When might you need this?**:
- Emergency hotfix (fix violations in follow-up commit)
- False positive that can't be easily suppressed
- Hook is malfunctioning

**Better alternatives**:
- Add `// nosemgrep:` suppression comment with explanation
- Fix the violation
- Submit a PR to improve the rule

## Troubleshooting

### Hook not running?

Check if it's installed:
```bash
git config core.hooksPath
# OR
ls -la .git/hooks/pre-commit
```

### Semgrep not found?

Install Semgrep:
```bash
pip install semgrep
```

### Too many false positives?

1. Check if you can suppress specific findings with `// nosemgrep:`
2. Open an issue to improve the rule
3. See `.semgrep-tests/MUTEX_RULE_TDD.md` for examples of accurate rules

### Hook fails with error?

Debug with:
```bash
bash -x .githooks/pre-commit
```

## For Claude AI Instances

**When you see a commit fail due to Semgrep findings**:

1. **READ THE FINDINGS** - They usually contain fix recommendations
2. **FIX THE CODE** - Don't bypass the check
3. **VERIFY** - Run `semgrep --config=.semgrep.yml <file>` to confirm fix
4. **COMMIT** - Try commit again

**If you need to suppress a finding**:
```go
// Manual unlock required: [explain WHY this is necessary]
mu.Lock() // nosemgrep: beacon-mutex-defer-unlock
```

**Never**:
- Don't use `--no-verify` without explicit user permission
- Don't suppress findings without explaining why
- Don't ignore findings just to make commit succeed

## Customization

### Change which files are scanned

Edit `.githooks/pre-commit` line 37:
```bash
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
```

### Change severity level

Edit `.githooks/pre-commit` line 55 to add `--severity ERROR`:
```bash
echo "$STAGED_GO_FILES" | xargs semgrep --config=.semgrep.yml --severity ERROR
```

### Make hook advisory (warn but don't block)

Change the final `exit 1` to `exit 0` in `.githooks/pre-commit` (line 87).

**Not recommended** - defeats the purpose of the hook.

## Maintenance

### Updating hooks

If you're using Option 1 (core.hooksPath), hooks are automatically updated when you pull changes.

If you're using Option 2 (symlink), the symlink will point to the latest version.

### Disabling hooks

**Temporarily**:
```bash
git commit --no-verify
```

**Permanently**:
```bash
# Option 1 users:
git config --unset core.hooksPath

# Option 2 users:
rm .git/hooks/pre-commit
```

## See Also

- `.semgrep.yml` - Semgrep rule configuration
- `.semgrep-tests/README.md` - Semgrep testing guide
- `SEMGREP_RULES_SUMMARY.md` - Complete rule documentation
- `CLAUDE.md` - Guidance for Claude AI on using Semgrep

---

**Last Updated**: 2025-11-02
**Status**: Active
