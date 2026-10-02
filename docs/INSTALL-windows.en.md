# Install & Run (Windows)

This guide gets `fenjue-agent` running on Windows and mounts your first platform to the unified library.

## 1. Download

Grab `fenjue-agent-windows-amd64.exe` from the
[latest release](https://github.com/lxh113377/global-memory-hub/releases/latest)
and put it in a folder you like, e.g. `C:\Tools\fenjue\`.

## 2. SmartScreen notice (first run)

Windows may show *"Windows protected your PC"* because the binary is not signed yet.

- Click **More info** → **Run anyway**.

Nothing else is needed; the agent runs as a normal user process and **does not require Administrator** (junctions created via `mklink /J` need no elevation).

## 3. Start the agent

Open PowerShell or CMD in the folder:

```powershell
.\fenjue-agent-windows-amd64.exe serve
```

Expected output:

```
fenjue-agent 0.2.0 (windows/amd64)
token file: C:\Users\<you>\.fenjue\token
Handshake: http://127.0.0.1:7799/console#token=<64 hex chars>
Web console: https://lxh113377.github.io/global-memory-hub/console/#token=<same token>
listening on 127.0.0.1:7799
```

On first start it also creates the unified library at `%USERPROFILE%\.fenjue\`
(`memory\` + `skills\`) and seeds it with a memory skeleton and a base skill pack.
An existing library is **never overwritten**.

## 4. Open the console

- Click/replace the printed **Handshake** link (local console), or
- open the **Web console** link (hosted page that talks to your machine only).

The token in the link stays in your browser session storage; it is never sent anywhere.

## 5. Enable a platform

In the console, flip the switch on a platform card (e.g. ZCode) — or use the CLI:

```powershell
.\fenjue-agent-windows-amd64.exe enable zc
```

What happens, in order (all reversible):

1. Anything already at the platform's mount points is **backed up** to `~\.fenjue\trash\`.
2. Links are created pointing the platform's skills/memory directories at your unified library.
3. A marker block is injected into the platform's boot file (e.g. `~\.zcode\AGENTS.md`).

Restart the platform tool; it now reads the shared library.

## 6. Disable / restore

```powershell
.\fenjue-agent-windows-amd64.exe disable zc        # soft: keeps a disabled marker shell
.\fenjue-agent-windows-amd64.exe disable zc --soft=false   # hard: removes the marker block
```

Backups can be restored via the console or the API (`POST /api/platforms/{id}/restore` with `{"backupId": "..."}`).

## 7. Verify

```powershell
.\fenjue-agent-windows-amd64.exe verify
```

Status meanings: `OK` healthy · `BROKEN` link points nowhere · `MISMATCH` link points somewhere else · `MISSING` nothing mounted · `SKIP` not applicable (e.g. wrong OS).

## Troubleshooting

| Symptom | Fix |
|---|---|
| Port 7799 already in use | `.\fenjue-agent... serve --port 7800` (console link prints with the new port) |
| Console shows 401 | Reopen the full handshake link printed at startup |
| Console shows 403 | The page origin is not whitelisted; start the agent with `--origin <url>` |
| `verify` shows BROKEN | The unified library folder was moved/deleted; re-run `serve` (re-creates and re-seeds if empty) |
| Antivirus flags the exe | Unsigned binary; verify the download against the Release page hash and whitelist it |

## Data & safety

Everything lives under `%USERPROFILE%\.fenjue\` on your machine. The agent listens on `127.0.0.1` only, and every mutating call requires a per-launch random token. Nothing is uploaded.
