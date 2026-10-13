# 📱 NilOS Studio — Nilang App-Runtime Emulator

A lightweight virtual-device studio for **NilLang / Onuron** apps. It runs the
real app runtime model — one worker process per app speaking a versioned JSON
protocol, rendering the app's own UI tree, and talking to virtual hardware —
instead of an Android-style full-system QEMU machine.

> **Goal:** an Android-Studio-like developer experience (device manager, install,
> run, logs, inspector, CLI, snapshots, fault injection) **without** the weight
> of hardware virtualization. QEMU remains an optional *System Mode*.

---

## ✨ What actually runs

| Layer | Implementation |
|---|---|
| Device manager | `internal/emulator` — profiles, lifecycle, persistence |
| App workers | `internal/runtime` — NilHost Protocol v1 over stdio, one session per app |
| Virtual hardware | `internal/hal` — battery, network, storage, printer (ESC/POS), scanner, sensors, clock, clipboard, notifications + **fault injection** |
| Inter-app bus | `internal/bus` — NilBus pub/sub + request/response with a DevTools log |
| Packages | `internal/packages` — `.nilax` zip bundles with Ed25519 signatures and zip-slip-safe extraction |
| Sandbox | `internal/vfs` + `internal/sandbox` — per-app virtual filesystem and resource limits (Job Object / cgroup) |
| Bangla text | `internal/textutil` — grapheme-cluster (UAX #29 subset) cursor & backspace |
| Record/replay | `internal/record` — `.nilrec` capture of input/IME/HAL events |
| Debugging | `internal/lsp` (`nilsp`) + `internal/dap` (`nildap`) — real LSP & DAP over stdio |
| Rendering | `internal/render` — Alap tree → SVG, with a golden screenshot test |
| System mode | `internal/qemu` — QMP client + real framebuffer via `screendump` (PPM→PNG) |
| Control API | `internal/api` — REST + Server-Sent Events (same API used by the UI and CLI) |
| UI | `public/studio.html` — device frame, live UI tree, HAL, Bangla keyboard, record/replay, debugger, system mode |
| CLI | `cmd/nilctl` (control), `cmd/nilsign` (signing), `cmd/nilsp` (LSP), `cmd/nildap` (DAP), `cmd/nilstudio` (native shell) |

Apps run as **separate processes** (`nilosd`). A crashing app cannot take the
emulator down, and the emulator detects exits even without a clean message.
When the real `nil serve` runtime is available, point the emulator at it; the
worker interface is identical.

---

## 🚀 Quick start

```powershell
# build the worker, then run the studio
go build -o bin/nilosd.exe ./cmd/nilosd
go run . --open
```

Open **http://127.0.0.1:8085/studio.html**.

1. **Create** a device (NilPhone Ultra / NilPad Pro / NilPOS / NilOS Desktop).
2. **Start** it.
3. **Install** `firmware/sample-notes.nilax` (already signed & trusted).
4. **Launch** `org.nilang.notes` — its UI tree renders live; taps go back to
   the app.

Or do it all from the CLI:

```powershell
go build -o bin/nilctl.exe ./cmd/nilctl
$d = (.\bin\nilctl.exe create --profile nilphone-ultra | ConvertFrom-Json).id
.\bin\nilctl.exe start $d
.\bin\nilctl.exe install firmware/sample-notes.nilax --device $d
.\bin\nilctl.exe launch org.nilang.notes --device $d
.\bin\nilctl.exe screenshot screen.svg org.nilang.notes --device $d
.\bin\nilctl.exe battery 15 --device $d
.\bin\nilctl.exe fault printer print --code paper_out --device $d
```

---

## 🔌 NilHost Protocol (v1)

Newline-delimited JSON over stdio. The VM sends `hello`, the host replies with
`init`, then frames/patches flow and `hal.call` requests are answered with
`hal.result`.

```
host → vm : {"t":"input","kind":"touch","phase":"up","x":120,"y":300}
host → vm : {"t":"hal.event","dev":"battery","level":15}
vm  → host : {"t":"frame","tree":{...}}
vm  → host : {"t":"patch","ops":[{"op":"set","id":"title","prop":"text","value":"..."}]}
vm  → host : {"t":"hal.call","id":42,"dev":"printer","op":"print","args":{...}}
```

Full specification: [`docs/protocol.md`](docs/protocol.md).
Virtual hardware contract: [`docs/hal.md`](docs/hal.md).

---

## 🧩 Packages (`.nilax`)

```
app.nilax
├── manifest.json      # id, name, version, entry, permissions, devices
├── SIGNATURE.json     # key_id, manifest/payload sha256, Ed25519 signature
└── payload/           # app files
```

```powershell
go run ./cmd/nilsign keygen --out dev.key          # prints the public key
go run ./cmd/nilsign sign --dir ./my-app --out my-app.nilax --key dev.key --key-id my-key
```

Trusted keys live in `nilemu.toml` under `[keys]`. Unsigned bundles are rejected
unless `allow_unsigned = true` (developer mode).

---

## 🧑‍💻 Developer workflow

- **Native shell:** `nilstudio` starts the server and opens the Studio in the
  OS WebView in app mode (frameless). This is the system-WebView approach
  (Wails-style) without bundling Chromium or needing a C toolchain.
- **Bangla input:** the Studio ships a virtual Bangla keyboard and accepts the
  physical keyboard; backspace is **grapheme-cluster aware** (“কি” deletes as
  one unit, “ক্ষ” stays whole).
- **Debugger:** the NilLang VM now has real debug hooks (source-position
  table, breakpoints, single-step, call stack, variables) exposed as a standard
  DAP server via `nil dap <file>`. `nildap` fronts it
  (`nildap --nil-bin <nil> --program main.nil`, or `--exec "nil dap main.nil"`)
  and can also attach to a running emulator app. `nilsp` (LSP) provides
  diagnostics, completion, hover and symbols. See [`docs/debug.md`](docs/debug.md).
- **Record/replay:** capture input/IME/HAL events (`.nilrec`) and reproduce a
  bug exactly, in real time or instantly.
- **Deterministic mode:** freeze the clock, seed the RNG and disable animations
  for byte-for-byte reproducible frames and golden tests.
- **Fault injection:** make any HAL call fail on demand to exercise error paths
  (`nilctl fault printer print --code paper_out`).

---

## ⚙️ Configuration

`nilemu.toml` (all keys overridable by env vars):

```toml
listen_addr = "127.0.0.1"
[server]
port = 8085
# data_dir = "~/.nilemu"
allow_unsigned = false
[keys]
dev-sample = "<hex ed25519 public key>"
```

Environment: `NILEMU_ADDR`, `NILEMU_PORT`, `NILEMU_DATA`, `NIL_BIN`,
`NILOSD_BIN`, `NILEMU_TOKEN`, `NILEMU_FIRMWARE`, `NILEMU_ALLOW_UNSIGNED`,
`NILEMU_QEMU`, `NILEMU_OS_IMAGE`.

### Security

- Binds **127.0.0.1** only.
- Cross-origin requests are rejected; mutating calls can require
  `X-Nilemu-Token` (`NILEMU_TOKEN`).
- App file access goes through the **virtual filesystem** (`..`, symlinks,
  drive/UNC paths rejected).
- The legacy terminal endpoint is a **virtual shell**; arbitrary host command
  execution has been removed.

---

## 🖥️ System Mode (optional, QEMU)

```powershell
go build -o bin/nilstudio.exe ./cmd/nilstudio
$env:NILEMU_QEMU = "C:\Program Files\qemu\qemu-system-x86_64.exe"
$env:NILEMU_OS_IMAGE = "onuron.img"
```

QEMU is launched headless with a QMP control socket; the **real** framebuffer is
captured with `screendump` and streamed as PNG (`/api/qemu/framebuffer`).
When QEMU or the image is missing the endpoint reports `available:false` — no
fabricated registers or kernel logs.

---

## 🧪 Testing

```powershell
go test ./...          # unit + integration
go vet ./...
go test ./internal/render -run Golden -update   # regenerate the golden screenshot
```

Covers path traversal/symlink escapes, bundle tamper detection, zip-slip,
protocol handshake/timeout/mismatch, HAL fault injection, ESC/POS round-trip,
PPM/QMP parsing, grapheme-cluster segmentation, record/replay, LSP and DAP
flows, device lifecycle, and a full HTTP end-to-end device→install→launch→render
flow.

---

## 📁 Layout

```
cmd/            nilemu (server), nilosd (worker), nilctl (control CLI),
                nilsign (signing), nilsp (LSP), nildap (DAP), nilstudio (shell)
internal/       config, vfs, sandbox, packages, emulator, runtime, hal, bus,
                supervisor, api, render, textutil, record, lsp, dap, qemu, jsonwire
devices/profiles/*.json    device templates (data, not code)
firmware/                  sample signed .nilax app
docs/                      protocol.md, hal.md, debug.md
public/                    studio.html + legacy device.html
os/ (qemu_bridge.go)       optional system mode
```

---

## 📜 License

AGPL-3.0 © 2026 Joy Sriram Sarkar & Nilang Core Team. See [`LICENSE`](./LICENSE).
