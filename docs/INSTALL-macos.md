# Install guide - macOS

> Applies to macOS 11 and newer, on both Apple silicon and Intel.
> No administrator rights are required.
> Convention: `<repo>` is the path where you cloned this repository; `~` is your home directory.

---

## 0. Prerequisites

- At least one agent application you intend to connect (a platform that is not installed simply shows as "not connected" in the console, which is expected).
- To build from source: Go and Node.js. Not needed if you use a prebuilt binary.

---

## 1. Get the program

Download the macOS binary for your architecture from this repository's Releases page, then make it executable and put it somewhere convenient:

```bash
mkdir -p ~/fenjue
# move the downloaded binary into ~/fenjue
chmod +x ~/fenjue/fenjue-agent
```

> Do not leave it in a temporary directory that the system cleans up. The program stores its token, logs and backups under `~/.fenjue/`; the binary's own location does not matter.

**Removing the quarantine flag.** Files downloaded through a browser get a quarantine attribute, and Gatekeeper will refuse to run an unsigned binary. Either clear the attribute explicitly:

```bash
xattr -d com.apple.quarantine ~/fenjue/fenjue-agent
```

Or run it once, let Gatekeeper block it, then open **System Settings - Privacy & Security** and choose **Open Anyway** next to the blocked item. The binary is unsigned because code signing requires a paid Apple Developer account; this is expected for free distribution, not corruption.

---

## 2. First run

```bash
~/fenjue/fenjue-agent serve
```

Expected output:

```
fenjue-agent 0.1.0 (darwin/arm64)
platforms: <repo>/platforms.json
token file: /Users/<you>/.fenjue/token
Handshake: http://127.0.0.1:7799/console#token=<random hex>
listening on 127.0.0.1:7799
```

Copy the **entire** `Handshake` line into your browser. Do **not** stop at the `#` - everything after it is the access token, and without it the console will refuse you.

The server binds to `127.0.0.1` only, so no other machine on your network can reach it. Keep the terminal window open; closing it stops the service.

---

## 3. Connect a platform

### Option A - through the console

The console lists every supported platform. Find the one you want and flip its switch. The card shows live status and the real mount paths.

### Option B - command line

```bash
~/fenjue/fenjue-agent enable wb     # connect
~/fenjue/fenjue-agent disable wb    # soft close (default)
```

Platform IDs are listed in the [README](../README.md#supported-platforms).

**A note specific to macOS.** Junction points do not exist here; links are real symbolic links. The program uses `os.Symlink`, which needs no elevation. What it does *not* do is silently replace an existing real directory - if a platform's mount point already holds real data, you will see `MISMATCH` rather than a destructive overwrite.

---

## 4. Verify

```bash
~/fenjue/fenjue-agent verify
```

This prints every platform, every mount and every injection point with a status:

| Status | Meaning | What to do |
|---|---|---|
| `OK` | Link exists, points at the expected target, and the target is reachable | Nothing |
| `MISSING` | The mount point does not exist | The platform is not connected yet; connect it from the console |
| `MISMATCH` | The path exists but is not a link, or points somewhere else | A real directory occupies that path. Check that it holds nothing you need, then rebuild |
| `BROKEN` | It is a link, but the target is unreachable | The target volume is unmounted or the directory was deleted; restore the target and re-run |

`verify` also works without `--platforms`, which makes it suitable for a scheduled health check.

---

## 5. Disconnecting and restoring

- **Soft close (default):** keeps the link, stops only the injection. Fully reversible, nothing is lost.
- **Restore to the pre-operation state:** every write is preceded by a backup under `~/.fenjue/trash/<backup-id>/`. The console offers one-click restore from its backup list; from the command line you need the backup ID returned by the operation.

---

## 6. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Port already in use; exits immediately | Use another port: `serve --port 7800`. Remember to also change the port in the handshake link |
| Browser says unauthorized | The part of the hash after `#` was lost. Copy the whole link again from the terminal |
| Console keeps saying "local program not detected" | The service is not running, or the port differs. Check the terminal window |
| Page served from a public domain cannot reach the local service | Add it to the origin allowlist: `serve --origin https://your-domain`. Repeatable |
| A platform shows `MISMATCH` | A real directory occupies the mount point. Confirm its contents, then remove and rebuild |
| A platform shows `BROKEN` | The link target is unreachable - typically an external volume or another partition that is not mounted |
| `verify` reports Hermes-related trouble | That platform depends on the `HERMES_HOME` environment variable. When it is missing the program warns explicitly instead of failing silently - set the variable first |

---

## 7. Build from source (optional)

```bash
cd <repo>/web
npm ci
npm run build
cd ../agent
mkdir -p cmd/fenjue-agent/dist
cp -r ../web/dist/. cmd/fenjue-agent/dist/
go build -o fenjue-agent ./cmd/fenjue-agent
```

**The order matters.** The Go side embeds the frontend output, so the frontend must be built first.

**Why the embed directory cannot be empty.** Go's embed directive requires a non-empty target directory and fails at compile time otherwise. The repository keeps a placeholder file there for that reason; the `cp` step overwrites it with the real build output.

The result is `agent/fenjue-agent`.
