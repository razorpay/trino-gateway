# Infrastructure: Dependency Management

Validates dependency upgrades in `go.mod` to catch breaking changes. Only flags packages **explicitly imported in changed code** - ignores transitive infrastructure upgrades.

## When to Apply

- `go.mod` or `go.sum` modified
- Changed `.go` files import upgraded packages

---

## Check 1: Major Version Bump of Used Package

**Severity:** Critical

**Flag if:**
- Major version upgrade (v1.x → v2.x or v0.x → v1.x)
- Package imported in changed `.go` files

**Fix:** Review CHANGELOG for breaking changes, update calling code, test with edge cases.

---

## Check 2: Minor Version Bump of Used Package

**Severity:** High

**Flag if:**
- Minor version upgrade (v1.0.x → v1.1.x)
- Package imported in changed files
- No test changes covering the upgraded package

**Example:** currency v1.0.5 → v1.1.1 imported in `terminals/service.go` but no new tests.

**Fix:** Check release notes for behavior changes (stricter validation, new errors). Test against production-like data.

---

## Check 3: Validation Added to Update/Edit Flow

**Severity:** Critical

**Flag if:**
- Validation added to update/edit endpoint
- Validates ALL fields, not just changed ones
- No legacy data handling

**Example from INC-2026-02-19:**
```go
func (s *Service) DisableTerminal(id string) error {
    terminal, _ := s.repo.GetByID(id)
    if err := terminal.Validate(); err != nil {  // ❌ Validates unchanged fields
        return err  // Fails for legacy currency codes
    }
    terminal.Status = "disabled"
    return s.repo.Update(terminal)
}
```

**Fix:** Validate only changed fields OR add legacy value exemption.

---

## Check 4: Stricter Validation Without Data Migration

**Severity:** High

**Flag if:**
- Validation rules tightened (new checks, narrower ranges)
- No data migration mentioned
- No test with production-like legacy data

**Fix:** Query production for legacy values, migrate or grandfather them.

---

## Check 5: Package Used in Validation But Not Tested

**Severity:** Medium

**Flag if:**
- Upgraded package used in validation logic
- No test changes covering validation

**Fix:** Add tests for validation with upgraded package, including legacy data scenarios.

---

## Check 6: Transitive Dependency Bump

**Severity:** Low

**Flag if:**
- Package upgraded but NOT imported in changed files
- Informational only

**Fix:** No action needed unless CI fails.

---

## Check 7: go.mod Changes Without Code Usage

**Severity:** Medium

**Flag if:**
- Package added/upgraded in go.mod
- No `.go` files import it

**Fix:** Add import, remove dependency, or document as preparatory.

---

## Check 8: Missing go.sum Update

**Severity:** High

**Flag if:**
- go.mod modified
- go.sum not modified

**Fix:** Run `go mod tidy` and commit both files.

---

## Check 9: Indirect Dependency Became Direct

**Severity:** Medium

**Flag if:**
- `// indirect` comment removed
- Package now imported in changed files

**Fix:** Verify this is intentional architecture (usually fine).

---

## Check 10: Version Downgrade

**Severity:** High

**Flag if:**
- Version downgraded (newer → older)

**Fix:** Document WHY in PR description and add comment in go.mod.

---

## Check 11: Replace Directive Added

**Severity:** Medium

**Flag if:**
- `replace` directive added in go.mod

**Fix:** Document reason, link tracking issue for removal.

---

## Check 12: Vendor Directory Modified

**Severity:** Medium

**Flag if:**
- `vendor/` exists and tracked
- `vendor/` not updated after go.mod changes

**Fix:** Run `go mod vendor` after `go mod tidy`.

---

## Summary

| Check | Severity | Trigger | Key Risk |
|-------|----------|---------|----------|
| Major version bump (used) | Critical | Imported in PR | Breaking API changes |
| Minor version bump (used) | High | Imported + no tests | Behavior changes (stricter validation) |
| Validation on all fields | Critical | Full entity validation in update | Legacy data rejected (INC-2026-02-19) |
| Stricter validation | High | Tightened rules | Production data fails |
| Used in validation untested | Medium | Validation + no tests | Regressions |
| Transitive bump | Low | Not imported | Info only |
| Unused dependency | Medium | Added but unused | Incomplete refactor |
| Missing go.sum | High | go.mod changed | Build issues |
| Indirect → direct | Medium | // indirect removed | Architecture change |
| Version downgrade | High | Newer → older | Needs documentation |
| Replace directive | Medium | replace added | Temporary workaround |
| Stale vendor | Medium | vendor/ not updated | Build mismatch |

**Incident Prevention:** Checks 2-4 would have caught INC-2026-02-19 (currency upgrade + over-validation).
