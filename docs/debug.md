# Debugging: LSP, DAP, DevTools

## Language Server (`nilsp`)

`nilsp` is a standard Language Server (LSP over Content-Length framed JSON on
stdio). Configure your editor to launch it for `.nil` files.

```jsonc
// VS Code settings.json
{
  "nilang.server.path": "path/to/nilsp",
  "nilang.server.args": []
}
```

Capabilities:

| Feature | Supported |
|---|---|
| `textDocument/publishDiagnostics` | ✅ (unbalanced braces/parens, `চল` without value, `ফাংশন` without `()`) |
| `textDocument/completion` | ✅ (keywords, decorators, NilBus/print) |
| `textDocument/hover` | ✅ |
| `textDocument/documentSymbol` | ✅ (`@Component`, `ফাংশন`) |

The same logic backs the HTTP endpoints `/api/lsp/{analyze,complete,hover}` used
by the Studio.

## Debug Adapter (`nildap`)

`nildap` speaks the Debug Adapter Protocol, so VS Code/Neovim/etc. can attach.

```powershell
# standalone mock program (works anywhere)
nildap

# attach to a running emulator app
nildap --device <device-id> --app <app-id> --server http://127.0.0.1:8085
```

VS Code launch configuration:

```jsonc
{
  "type": "nilos",
  "request": "attach",
  "name": "Attach to NilOS app",
  "debugServer": 4711      // or launch nildap directly
}
```

Supported requests: `initialize`, `launch`/`attach`, `setBreakpoints`,
`configurationDone`, `threads`, `stackTrace`, `scopes`, `variables`,
`continue`, `next`, `stepIn`, `stepOut`, `pause`, `evaluate`, `disconnect`.

### Real NilLang VM stepping

Source-line stepping is implemented in the NilLang VM itself
(`compiler/vm/debug.go` and the position table added to `compiler/compiler`):

- the compiler records an **offset → file:line:column** table (per function,
  with local names and function names);
- the VM exposes **breakpoints, single-step, call stack, locals/globals/free
  variables and name lookup**;
- `nil dap <file.nil>` (`pkg/dap` in the nilLang repo) serves the Debug
  Adapter Protocol on top of those hooks.

`nildap` fronts it so one configuration works everywhere:

```powershell
nildap --nil-bin "C:\path\to\nil.exe" --program .\main.nil
# or, equivalently:
nildap --exec "nil dap .\main.nil"
```

Editors can also launch `nil dap <file>` directly.

Both paths are covered by committed, process-level tests:

```powershell
# builds `nil` itself and runs a full DAP session against `nil dap <file>`
go test ./pkg/dap/ -run Process          # in the nilLang repo

# drives nildap's proxy mode against a real `nil` binary
$env:NIL_BIN = "C:\path\to\nil.exe"
go test ./cmd/nildap/ -run EndToEnd      # in this repo
```

When attached to a *running emulator app* (`--device/--app`), the adapter
reports the concrete data the emulator has (session/device variables, saved
state, worker frame). For source-line stepping inside the emulator, run the app
on the real runtime so `nil`'s debug hooks drive it (set `NIL_BIN` and use the
NilHost `debug` channel); the built-in mock runtime reports no bytecode lines
and the adapter never invents them.

The emulator exposes the same data over HTTP so the DevTools panel and the DAP
adapter share one source of truth:

```
GET  /api/v1/devices/{d}/apps/{a}/debug/stack
GET  /api/v1/devices/{d}/apps/{a}/debug/scopes
GET  /api/v1/devices/{d}/apps/{a}/debug/variables?ref=1
POST /api/v1/devices/{d}/apps/{a}/debug/breakpoints   {"source":"main.nil","lines":[4]}
POST /api/v1/devices/{d}/apps/{a}/debug/command       {"command":"next"}
GET  /api/v1/devices/{d}/apps/{a}/debug/evaluate?expr=count
```

## Record / replay

Capture every input, IME and HAL event, then reproduce a bug exactly.

```powershell
nilctl record start --device <id> --app org.nilang.notes --name bug1
# ... interact ...
nilctl record stop --device <id>
nilctl recordings --device <id>
nilctl replay bug1 --device <id> --app org.nilang.notes          # real-time
nilctl replay bug1 --device <id> --app org.nilang.notes --instant # as fast as possible
```

Recordings are stored as `~/.nilemu/devices/<id>/recordings/<name>.nilrec` and
double as test fixtures.

## Deterministic mode

For reproducible screenshots and CI:

```powershell
$env:NILEMU_CLOCK_ISO = "2026-10-07T10:00:00+06:00"
$env:NILEMU_CLOCK_FROZEN = "1"
$env:NILEMU_SEED = "42"
$env:NILEMU_ANIMATIONS = "off"
```

The virtual clock stops, RNG is seeded, and animations are disabled, so the
same actions produce the same frames. `internal/render` has a byte-for-byte
golden test (`go test ./internal/render -run Golden -update` to regenerate).
