---
name: debugging-fixing
description: 系统化诊断并修复 bug/报错/异常行为。覆盖报错栈、测试失败、崩溃、进程卡住不返回、rc=124 恒定超时、daemon 线程或子进程未清理、竞态与共享可变状态、环境变量与路径类失败、页面白屏/本地 HTML 双击无反应/构建产物无内容。Use when 用户说「报错了」「跑不通」「卡住了」「不返回」「白屏」「为什么超时」「crash/闪退/hang」。流程=先复现取证据→列 2-3 个根因假设→最小改动→修完走验证 Gate→记录根因。不适用：代码规范与性能体检（走 code-review）、编写测试（走 testing）、完工前证据核验（走 verification-before-completion）。
version: 2.2.0
---

# Skill: Debugging & Fixing

## When to use
- User reports an error message, stack trace, or unexpected behavior
- Test failures
- Runtime exceptions, crashes, hangs
- Wrong output (logic bug)
- **页面白屏 / 空白 / 构建产物打开无内容** — 排查思路：先 F12 看 Console+Network，常见 Vite 产物的 file:// 兼容问题：资源路径(需 base:'./') → 脚本加载(module→defer) → DOM 就绪时序
- **文件双击打开没反应 / 本地 HTML 无法运行** — 检查 script type/crossorigin/路径，按 Workflow 逐层诊断

## Workflow
1. **Reproduce** — confirm the issue from the user's report; get exact error message/stack trace
2. **Gather evidence** — read the failing file(s), recent logs, related tests. DO NOT guess identifiers
3. **Diagnose** — list 2-3 most likely root causes with reasoning
4. **Propose fix** — show the minimal change; explain why it works
5. **Verify** — 修完走「修完验证」独立节（5 步 Gate；跳步 = 虚报）
6. **Document** — what was the root cause, what was fixed, how to prevent recurrence

## Diagnosis checklist
- Recent changes? (git log, git diff)
- Input validation? (null, empty, type mismatch)
- Environment? (env vars, paths, OS, dependencies)
- State? (race conditions, shared mutable state, cache)
- External dependencies? (API rate limits, network, DB connection)
- Permissions? (file access, admin rights)
- **生命周期清理（R199 硬性检查项）** — 涉及线程 / 子进程 / 后台任务 / watchdog 时，必须验证：
  ①能否正常取消（stop_event / join / terminate / kill）；②返回前是否显式释放资源（try/finally 或 context manager）；③daemon 线程是否会在宿主进程存活时触发 `os._exit`/强制终止。
  **触发示例**（命中任一即检查）：`os._exit(124)`、`threading.Thread(daemon=True)` 未 stop、`subprocess` 未 terminate、watchdog 未取消、后台任务无法退出、进程悬挂、假 FAIL 误报（rc=124 恒定且与超时参数无关）。
  **诊断口诀**：rc=124 / 恒定超时 → 先查 os._exit / watchdog / daemon 线程，别怀疑网络或超时参数；单文件跑通过 ≠ 整目录跑通过——必须复现「长于 deadline 的宿主进程」场景。

## When to read more files vs ask the user
- **Read more** if error message points to a file/line
- **Ask user** if you need credentials, private keys, or physical access
- **Ask user** if multiple files could be the cause and you can't tell from context

## Anti-patterns
- Don't add try/except around errors without understanding root cause
- Don't disable tests to make them pass
- Don't change unrelated code while fixing
- Don't claim "fixed" without verifying

## Reporting format
When done, report:
- **Root cause:** [one sentence]
- **Fix:** [file:line + change description]
- **Verification:** [how you confirmed it works]
## 修完验证（独立节）
修完≠证完，先过 5 步 Gate（细则 references/verification.md）：定命令→全量跑→读输出→对证据→再断言。连败 3 次（次次新位置冒头）→ 停手质疑架构，讨论后再试。

## Version history
- V2.0.0 (2026-08-16): Diagnosis checklist 新增「生命周期清理」硬性检查项（R199）——线程/子进程/后台任务/watchdog 必须可取消、返回前显式释放（try/finally 或 context manager）；附触发示例（os._exit(124)/daemon 未 stop/subprocess 未 terminate/watchdog 未取消/进程悬挂/假 FAIL）与诊断口诀（rc=124 恒定 → 先查 os._exit/watchdog，别怀疑超时参数；单文件过 ≠ 整目录过）。源自两次真实踩坑：unified_router watchdog 残留强杀 pytest（R198.11）+ pre-commit 派生件 stash 假 FAIL（R198.6）

- V2.1.0 (2026-09-23): 新增「修完验证」节（5 步 Gate + 3 次熔断转架构评审），细则入 references/verification.md
