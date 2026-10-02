---
name: refactoring
description: Improve code structure without changing behavior. Use when user asks to "refactor", "clean up", "optimize", "reorganize", "重构", "优化", "整理代码", "拆分模块". Preserves external behavior; improves readability, maintainability, or performance.
version: 2.0.0
---

# Skill: Refactoring

## When to use
- User asks to refactor, reorganize, or clean up existing code
- Code smells: duplication, long functions, deep nesting, god classes
- Performance optimization requests (algorithmic, not micro-benchmarks)
- Module/dependency restructuring

## Workflow
1. **Understand current behavior** — Read the file(s) first; identify inputs, outputs, side effects
2. **Identify smells** — list specific issues (duplication, naming, coupling, etc.)
3. **Risk assessment** — classify each change by risk level (see Risk Matrix below)
4. **Plan transformation** — describe the change in 1-3 bullet points BEFORE editing
5. **Establish safety net** — run existing tests first; if none exist, write characterization tests for the behavior being preserved
6. **Refactor in small steps** — one logical change at a time; preserve behavior
7. **Verify after each step** — run tests / type checks / linters; see Verification Protocol
8. **Report** — summarize what changed and why, with before/after metrics

## Risk Assessment Matrix

Classify each planned refactoring before starting:

| Risk Level | Criteria | Safety Net Required |
|-----------|----------|-------------------|
| **Low** | Internal rename, extract local variable, reorder imports | Run linter + type check |
| **Medium** | Extract function/method, move method within same module | Run existing tests; add characterization test if coverage < 80% |
| **High** | Change public API surface, split module, alter data flow | Full test suite + manual smoke test of 3 critical paths |
| **Critical** | Database schema, wire format, cross-service contract | Full test suite + backward-compat test + rollback plan documented |

## Verification Protocol

After EACH refactoring step, verify in this order:

1. **Compile/Parse** — Does the code still compile/parse without errors?
2. **Type check** — Do static types still hold? (`mypy`, `tsc --noEmit`, `go vet`)
3. **Lint** — No new lint violations introduced?
4. **Test** — All existing tests pass? (`pytest`, `go test`, `jest`, etc.)
5. **Behavior spot-check** — Pick 2-3 representative inputs; confirm outputs match pre-refactor behavior
6. **Metrics delta** — Run relevant metrics (file count, function length, cyclomatic complexity) and confirm improvement direction

### When tests don't exist

Before refactoring untested code, write **characterization tests**:

```python
# Characterization test: capture CURRENT behavior, not ideal behavior
def test_current_deposit_behavior():
    """Locks in existing behavior before refactor. 
    If this breaks after refactor, either the refactor is wrong 
    or the original behavior was a bug (document which)."""
    account = Account()
    account.deposit(100)
    assert account.balance == 100  # This is what it does TODAY
    assert account.get_status() == "active"  # Lock all observable outputs
```

Key rules:
- Test observable outputs, not implementation details
- Cover the happy path + 2-3 edge cases the refactor might touch
- Mark these as `@pytest.mark.characterization` (or equivalent) so they can be converted to proper tests later

## Common refactorings
- **Extract function** — split long function into named subroutines
- **Replace conditional with polymorphism** — switch/if-else chains → strategy pattern
- **Introduce parameter object** — group related parameters into a struct/dict
- **Move method** — relocate to class/module that owns the data
- **Rename for clarity** — use descriptive names; update all call sites
- **Remove dead code** — unused exports, commented-out blocks, unreachable branches
- **Collapse nesting** — early returns / guard clauses to flatten deep indentation
- **Extract interface** — define abstraction before implementation to reduce coupling

## Safety rules
- NEVER change public API signatures without flagging it
- NEVER refactor and add features in the same step
- If behavior might change, ask user before proceeding
- Keep diffs small and reviewable
- NEVER delete tests to make a refactor pass — fix the refactor or update the test with documented reason

## Anti-patterns
- Don't refactor code that the user didn't ask to touch
- Don't introduce new dependencies for a refactor
- Don't "modernize" syntax (var→let, etc.) without explicit request
- Don't refactor without a safety net (tests or characterization tests)
- Don't do "big bang" refactors — if the diff exceeds ~300 lines, split into smaller steps
