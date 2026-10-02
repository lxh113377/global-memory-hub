---
name: verification-before-completion
description: Verify claims before marking work complete. Use when about to say "done", "fixed", "passing", or "complete" — requires running evidence-gathering commands and confirming output before making any success claims. Evidence before assertions always. 触发词：完工验证、交付前核验、确认真的绿了、别虚报完成、声称已修需复验、证据先于结论、跑一遍验证再报、verify before done、evidence before claim、no unverified completion。
version: 1.1.0
triggers:
  - 完工验证
  - 交付前核验
  - 确认真的绿了
  - 别虚报完成
  - 声称已修需复验
  - 证据先于结论
  - 跑一遍验证再报
  - 自检未过
  - verify before done
  - evidence before claim
  - verification before completion
---

> **Provenance**: adapted from [obra/superpowers](https://github.com/obra/superpowers) `verification-before-completion`
> (292,433 stars, measured 2026-09-29 via `gh api repos/obra/superpowers`).

# Skill: Verification Before Completion

> **Core principle**: Evidence before assertions. Never claim work is complete without running the verification commands and reading their actual output.

## When to use
- About to tell the user "done", "fixed", "it works", "all passing"
- About to mark a task as completed
- About to commit or push changes
- Any time you're about to make a success claim

## 何时不用本 skill（边界）

| 场景 | 改走 |
|------|------|
| 写测试 / 跑测试 / 覆盖率 | `testing`、`test-driven-development` |
| 执行前检查链路是否可跑（文件/工具/路径可达） | `workflow-preflight-check` |
| Bug 根因定位与修复流程 | `debugging-fixing`、`superpowers:systematic-debugging` |
| 任务结束后的经验反哺落盘 | `A-get-memory` |
| 交付物质量评分与独立复核 | `openclaw-dual-gate-quality-audit` |

**与插件同名技能并存**：QD 端另有 `superpowers:verification-before-completion` 插件，内容同源。插件在役时以插件为准，本目录是其余六端（HM/WB/CX/TC/ZC/OC）的镜像承载 —— 与本地树已镜像 `brainstorming` / `test-driven-development` / `writing-plans` 的做法一致。两边口径若分叉，改插件侧优先，本侧只跟随不回写。

## The Rule

**Before claiming completion, you MUST:**

1. **Identify the claim** — What exactly are you asserting? ("tests pass", "build succeeds", "feature works", "bug is fixed")
2. **Select evidence command** — What command proves or disproves this claim? (see Claim→Evidence table below)
3. **Run the command** — Actually execute it; don't simulate or predict the output
4. **Read the output** — Look at what the command actually printed
5. **Confirm match** — Does the output match what "success" looks like? If yes, proceed. If no, go fix.

## Claim → Evidence Mapping

| Claim Type | Required Evidence | Command Examples |
|-----------|-------------------|-----------------|
| "Tests pass" | Test runner output showing 0 failures | `pytest -v`, `go test ./...`, `jest --verbose`, `cargo test` |
| "Build succeeds" | Build tool exit code 0 + artifact exists | `npm run build`, `go build ./...`, `cargo build` |
| "Type check passes" | Type checker output with no errors | `mypy .`, `tsc --noEmit`, `go vet ./...` |
| "Lint is clean" | Linter shows 0 errors (warnings OK if pre-existing) | `eslint .`, `ruff check .`, `golangci-lint run` |
| "Bug is fixed" | Reproduction case now produces expected output | Run the exact reproduction steps; diff output against expected |
| "Feature works" | End-to-end demonstration of the feature | Manual walkthrough or E2E test covering the golden path |
| "Performance improved" | Before/after benchmark with same inputs | `time command`, `hyperfine`, language-specific bench tools |
| "No regressions" | Full test suite passes, not just the new test | Run the complete suite, not just `test_new_feature.py` |

## Anti-Patterns (Excuse → Reality)

| Excuse | Reality |
|--------|---------|
| "I'm confident the tests pass" | You haven't run them this turn |
| "The build should work" | You haven't run the build command |
| "Based on the changes, it should be fixed" | You haven't reproduced the bug post-fix |
| "I'll verify later" | You won't. Verify now. |
| "The test output looked correct" | You skimmed it; read it fully |
| "I ran the tests earlier" | Earlier was before your latest change |
| "It's a simple change, it can't break anything" | Simple changes cause the most surprising bugs |

## Verification Depth by Risk

| Change Risk | Minimum Verification |
|------------|---------------------|
| **Low** (comment, rename internal var) | Parse/compile check |
| **Medium** (new function, logic change) | Compile + relevant tests pass |
| **High** (API change, data flow change) | Full test suite + type check + lint |
| **Critical** (production deploy, migration) | Full suite + staging smoke test + rollback plan |

## Post-Verification Checklist

Before telling the user you're done, confirm:

- [ ] I ran the verification command(s) this turn (not "earlier")
- [ ] I read the actual output (not just the exit code)
- [ ] The output matches what success looks like (not just "no obvious errors")
- [ ] I verified the specific claim I'm about to make (not a related but different claim)
- [ ] If I changed test code, I also ran the tests it's testing (not just the test file itself)

## Special Cases

### "It works on my machine"
- State the environment: OS, runtime version, relevant config
- If the verification is platform-specific, say so explicitly

### Tests that are slow
- Run the relevant subset first, then queue the full suite
- Report partial results: "Unit tests pass (47/47); integration suite still running"

### Flaky tests
- If a test fails, run it again before blaming the test
- If it fails twice in a row, treat as real failure
- Document known flaky tests; don't use "it's flaky" as an excuse to ignore failures

### No tests exist
- "No tests to run" is not verification
- Write at least one test that covers the change, then run it
- If writing tests is out of scope, say "untested — no test suite exists for this module"

## When Verification Fails

1. **Don't claim completion** — obvious but critical
2. **Read the failure output carefully** — the error message usually tells you what's wrong
3. **Fix the root cause** — not just the symptom
4. **Re-verify** — run the same command again after fixing
5. **Report honestly** — "Fixed X, but Y is now failing because Z" is better than hiding Y

## Version history

- **1.1.0** (2026-09-29) — Added 何时不用本 skill boundary table (vs `testing` / `workflow-preflight-check` /
  `debugging-fixing` / `A-get-memory` / `openclaw-dual-gate-quality-audit`) and the plugin-coexistence rule:
  `superpowers:verification-before-completion` wins on QD; this copy is the mirror for the other six endpoints.
- **1.0.0** (2026-09-28) — Created from obra/superpowers: claim-to-evidence mapping, anti-rationalization table,
  risk-tiered verification depth, post-verification checklist.
