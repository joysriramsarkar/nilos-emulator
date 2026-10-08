# NilHost Protocol v1

The NilHost Protocol is the versioned, newline-delimited JSON contract between
the emulator host (`nilemu`) and a NilLang runtime worker (a "VM"). It is the
system's backbone: the emulator never contains app logic, it only sends events
and renders frames.

- **Transport:** stdin/stdout pipes by default. `stdout` carries protocol
  messages only; all `println`/debug output goes to `stderr` or to `log`
  messages. (Loopback WebSocket is a future transport; the `Codec` is
  transport-agnostic.)
- **Framing:** one JSON object per line (`\n`).
- **Version:** `proto` is an integer. The host speaks `proto: 1`. On mismatch
  the host aborts with a clear error.
- **Capabilities:** the VM advertises `caps`; hosts ignore unknown features and
  fall back gracefully.

## Handshake

The VM sends `hello` first:

```json
{"t":"hello","proto":1,"runtime":"nil 0.9.2","app":"com.nilang.notes","caps":["ui.tree.patch","hal.v1","debug.dap"]}
```

The host answers with `init`, which carries **all** device facts so the VM never
assumes a screen size:

```json
{"t":"init","proto":1,
 "device":{"profile":"nilphone-ultra","w":390,"h":844,"dpr":3,
           "safe":{"t":47,"b":34,"l":0,"r":0},"refresh_hz":120,
           "orientation":"portrait","inputs":["touch","keyboard"]},
 "locale":"bn-BD","theme":"dark","font_scale":1,
 "hal":{"battery":{"level":80,"charging":false},"net":{"type":"wifi","online":true}},
 "permissions":["filesystem","printer"],
 "app_data":"/data/apps/com.nilang.notes/data",
 "saved_state":null}
```

Changing device profile means sending a new `init` (or a `config` message).

## Host → VM messages

| `t` | Purpose |
|---|---|
| `input` | touch/key/text/scroll/back/home. Example: `{"t":"input","kind":"touch","phase":"down","id":0,"x":120.5,"y":300,"ts":183422}` |
| `ime` | input-method editor: `show`,`hide`,`commit`,`compose`,`delete` |
| `hal.event` | device change, e.g. `{"t":"hal.event","dev":"battery","level":15}` |
| `lifecycle` | `create`,`start`,`resume`,`pause`,`stop`,`destroy` |
| `config` | rotation, theme, locale, font scale, device size |
| `vsync` | frame credit / tick |
| `reload` | `restart` or `reload` |
| `debug` | debugger command (breakpoint, step, continue, ...) |

## VM → Host messages

| `t` | Purpose |
|---|---|
| `frame` | full UI tree (first paint) |
| `patch` | tree diff operations |
| `hal.call` | device request: `{"t":"hal.call","id":42,"dev":"printer","op":"print","args":{...}}` |
| `bus` | NilBus message |
| `log` | `{"t":"log","level":"info","tag":"app","msg":"..."}` |
| `trace` | DevTools timing: frame ms, GC, bytecode events |
| `state` | saved-state blob for restart/restore |
| `exit` | `{"t":"exit","code":0,"reason":"..."}` |
| `error` | `{"t":"error","code":"E_RUNTIME","msg":"...","stack":[...]}` |

Every `hal.call` must be answered by the host with a `hal.result`:

```json
{"t":"hal.result","id":42,"ok":true,"data":{"printed":6}}
```

## UI tree format

```json
{"id":"root","type":"Column","props":{"gap":8,"padding":16},
 "children":[
   {"id":"title","type":"Text","props":{"text":"নতুন নোট","style":"title"}},
   {"id":"save","type":"Button","props":{"label":"সংরক্ষণ","onTap":"h:42"}}
 ]}
```

- Node `id`s are **stable across renders**; without them diffing and animation
  are impossible.
- Event handlers are references (`"h:42"`), never functions. On tap the host
  sends `{"t":"input","kind":"tap","code":"h:42"}`.
- All props are serializable; anything VM-internal stays in the VM.

### Patch operations

```json
{"t":"patch","ops":[
  {"op":"set","id":"title","prop":"text","value":"সংরক্ষিত"},
  {"op":"insert","parent":"root","index":2,"node":{...}},
  {"op":"remove","id":"note.3"},
  {"op":"move","id":"note.1","parent":"root","index":0}
]}
```

Diffing is the runtime's responsibility; the emulator only applies patches.

## Backpressure & frame pacing

The host issues **frame credits** (`vsync`) at the display refresh rate. The VM
sends at most one `patch` per credit, coalescing intermediate changes. Input is
also coalesced: `touch` `move` events are merged, but `down`/`up` are never
dropped.

## Errors & crashes

A VM crash is detected by process exit even without an `exit` message; the
supervisor synthesises one and the frontend shows "App has stopped. Restart?".
