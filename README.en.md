# Fenjue Global (global-memory-hub)

[![CI](https://github.com/lxh113377/global-memory-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/lxh113377/global-memory-hub/actions/workflows/ci.yml)
[![Release](https://github.com/lxh113377/global-memory-hub/actions/workflows/release.yml/badge.svg)](https://github.com/lxh113377/global-memory-hub/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-2ea44f.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-9-8A2BE2)](platforms.json)
[![Go](https://img.shields.io/badge/go-1.x-00ADD8?logo=go&logoColor=white)](agent/go.mod)

> 中文版: [README.md](README.md)

> Chinese version: [README.md](README.md)

Give multiple AI coding agents **one shared local memory store and one shared skill store**. Change it once, and every connected agent sees the same thing.

No network calls, no uploads, no background service. Everything stays on your machine, and the program listens only on the loopback interface.

---

## The problem it solves

Anyone who runs more than one AI coding assistant has hit the same wall:

- **Fragmented memory** - what you taught agent A, agent B has no idea about. The same knowledge gets written down over and over.
- **Scattered skills** - a skill installed under A's directory and again under B's. You fix one and forget the other.
- **No shared ground** - each tool keeps its memory and skills in its own private location, invisible to the rest.

`Fenjue Global` takes a different route: build **one** authoritative memory store and skill store on your machine, then point each agent at it through directory links. After that, no matter which agent makes a change, every other agent sees the same content immediately.

Disconnecting an agent is **reversible**. The default is a "soft close" - the link is kept, only the injection is stopped - so you can restore it at any time with one click.

---

## Three steps

### 1. Get the program

Prebuilt single-file binaries for all three platforms are published on this repository's Releases page. If they are not published yet, or you would rather build it yourself, see [Build from source](#build-from-source).

### 2. Run it

```bash
./fenjue-agent serve
```

The program starts a console on the loopback address `127.0.0.1:7799` and prints a **handshake link** in the terminal:

```
Handshake: http://127.0.0.1:7799/console#token=<random hex>
```

Open that link to reach the console. The link carries an access token generated for this run only. The token is created locally and regenerated every time you start the program.

### 3. Flip the switches

The console lists every supported agent platform as a card. Each card shows whether that platform is **connected**, **not connected**, or **broken**, together with the actual mount paths. Turn on whatever you need.

Turn it off the same way. The default close mode deletes nothing - original files are backed up before any change.

---

## Supported platforms

| ID | Platform | Memory mount | Skill mount |
|---|---|---|---|
| `wb` | WorkBuddy | directory link | directory link |
| `tr` | TRAE SOLO CN | directory link | directory link |
| `cx` | Codex CLI | directory link | physical mirror (a real copy, not a link) |
| `hm` | Hermes Agent | directory link | per-skill links (one link per skill) |
| `zc` | ZCode | directory link | directory link |
| `oc` | OpenCode | directory link | directory link |
| `qd` | Qoder CN | directory link | directory link |
| `mm` | MiniMax Code | directory link | directory link |
| `ds` | DeepSeek Harness Desktop | directory link | directory link |

**Three mount shapes, three behaviours.** In the table above, `cx` and `hm` do **not** behave like the rest. The console labels them accordingly:

- **Directory link** - one directory points at another. The simplest on/off semantics.
- **Physical mirror** (`cx`) - a real copy rather than a link. "Off" therefore means **stop syncing**, not "cut a link". Files already synced are not deleted automatically.
- **Per-skill links** (`hm`) - roughly one hundred and sixty individual links. Enabling and disabling are batch operations, with a concurrency cap and rollback on failure so no half-finished state is left behind.

Platform definitions live in `platforms.json` at the repository root. Adding a platform is a data change, not a code change.

---

## Command line

```
fenjue-agent serve    [--port 7799] [--platforms <path>] [--origin <url>]...
fenjue-agent verify   [--platforms <path>]
fenjue-agent enable   <platform-id>  [--platforms <path>]
fenjue-agent disable  <platform-id>  [--soft=true] [--platforms <path>]
fenjue-agent version
```

| Flag | Meaning |
|---|---|
| `--port` | Listening port, default `7799`. **Binds to `127.0.0.1` only and accepts no external connections.** |
| `--platforms` | Path to the platform definition file. Auto-discovered from the repository root by default. |
| `--origin` | Extra allowed page origin, repeatable. Point it at the domain where you host the console. |
| `--soft` | Close mode, default `true` (soft close: keep the link, stop the injection only). |

`verify` also works without `--platforms`, which makes it convenient for scheduled health checks.

---

## Security design

Once a local server is running, in principle **any web page** can send requests to your loopback port. The program therefore enforces four gates; failing any one is an immediate rejection:

| Gate | Purpose |
|---|---|
| **Host header check** | Accepts only `127.0.0.1:<port>` and `localhost:<port>`, shutting down DNS-rebinding style attacks |
| **Origin allowlist** | Only your site's domain plus origins you add explicitly with `--origin`; everything else is rejected |
| **CORS preflight handling** | Answers cross-origin preflight correctly for allowlisted origins, so a public page can talk to the local service in browsers |
| **Access token** | Every endpoint except the health check requires it; comparison uses a constant-time implementation to avoid timing side channels |

Additionally:

- The process **binds to loopback only**. It never listens on `0.0.0.0`, so other machines on your network cannot reach it directly.
- The token is regenerated on every start, stored in a restricted local file, and handed to the page via a URL fragment in the printed handshake link (fragments are never sent to servers and never leak into Referer).
- Logs never print the token in plaintext.

---

## Data and backups

All program state lives under `~/.fenjue/`:

| Path | Contents |
|---|---|
| `~/.fenjue/token` | Access token for the current run |
| `~/.fenjue/trash/<backup-id>/` | Backups of original files taken before changes (with a `manifest.json` change list) |
| `~/.fenjue/logs/agent.log` | Runtime log |
| `~/.fenjue/memory` | Default memory root |
| `~/.fenjue/skills` | Default skill root |

**Every write is preceded by a backup.** The response returns a backup ID, and the console uses it to offer one-click restore. Backups are append-only, and restore reads the backup rather than any older live state.

Both roots are configurable, either from the settings page in the console or by editing `roots` in `platforms.json`.

---

## Build from source

You need Go and Node.js. **The build order cannot be reversed**: the Go side embeds the frontend build output, so the frontend must be built first.

### Windows

```powershell
git clone <repo-url> global-memory-hub
cd global-memory-hub\web
npm ci
npm run build
cd ..\agent
pwsh .\build.ps1
```

`build.ps1` syncs `web\dist` into the embed directory before compiling. If `go` is not on `PATH`, the script falls back to the default installation path; if your toolchain lives elsewhere, either put its absolute path in the script or add it to `PATH`.

### macOS / Linux

```bash
git clone <repo-url> global-memory-hub
cd global-memory-hub/web
npm ci
npm run build
cd ../agent
mkdir -p cmd/fenjue-agent/dist
cp -r ../web/dist/. cmd/fenjue-agent/dist/
go build -o fenjue-agent ./cmd/fenjue-agent
```

**Why the embed directory must contain at least one file.** Go's embed directive requires a non-empty target directory, and fails at **compile time** otherwise. The repository keeps a placeholder file there for exactly this reason; the `cp` step above overwrites it with the real build output. If you skip the frontend build the program still compiles, but the console will report that the frontend assets are missing.

---

## Platform compatibility

| Platform | Link mechanism | Elevation required |
|---|---|---|
| Windows | directory junction | **No** administrator rights needed |
| macOS | symbolic link | No |
| Linux | symbolic link | No |

On Windows the project deliberately avoids the symbolic-link form, which would require developer mode or elevation.

---

## How this differs from other approaches

Same problem space (several agents sharing one config or skill set), different choices:

| Approach | Endpoints | What is shared | Shape | Where it differs |
|---|---|---|---|---|
| **Fenjue Global (this project)** | 9 | **memory + skills** | Single binary + browser console | The only approach here that puts the **memory store** in scope as well; reversible soft-close with full restore |
| [vercel-labs/skills](https://github.com/vercel-labs/skills) | 79 | skills | Node CLI | Far larger ecosystem; skills only, no memory; aimed at distribution rather than local sharing |
| [xingkongliang/skills-manager](https://github.com/xingkongliang/skills-manager) | 50+ | skills | Desktop app | Has a marketplace, presets and multi-device sync; no memory layer |
| [dyoshikawa/rulesync](https://github.com/dyoshikawa/rulesync) | — | rule files | CLI | Manages each vendor's rule files, not a memory/skill store |
| Each agent's own config | 1 | only its own | vendor apps | Splits by design: changing one changes one |

In one line: **others mostly solve how skills get installed; this project solves keeping memory and skills as one copy on your own machine.**
That is also why there is deliberately no skill marketplace and no cloud multi-device sync here: both need network access, which would break the no-network promise.

Full eight-dimension comparison, with quantified gaps and the improvement backlog, is in the [benchmark report](docs/BENCHMARK-2026-10-02.md).

---

## Further reading

- [Install guide - Windows](docs/INSTALL-windows.md)
- [Install guide - macOS](docs/INSTALL-macos.md)
- [Install guide - Linux](docs/INSTALL-linux.md)
- [Architecture](docs/ARCHITECTURE.md) - global view, mount forms, where the token sits in the request path
- [FAQ](docs/FAQ.md)
- [Security policy](docs/SECURITY.md) - the four gates, known limitations, how to report a vulnerability
- [Contributing](docs/CONTRIBUTING.md) - build order, gate list, seed pack rules
- [Changelog](docs/CHANGELOG.md)
- [Skill tiers](docs/SKILLS-TIERS.md) - what may be distributed publicly, and why
- [Roadmap: open work and next steps](docs/ROADMAP.md)
- [Benchmark report - 2026-10-02](docs/BENCHMARK-2026-10-02.md) - eight-dimension comparison

> **Language note.** The three platform install guides are currently written in Chinese only. Full English localisation of the documentation is tracked as G9 in the [roadmap](docs/ROADMAP.md#2-优先级排序); this README pair is the first step of it.

---

## License

MIT. See [LICENSE](LICENSE).
