# NilHAL — Virtual Hardware Contract v1

NilHAL is a contract, not an implementation. Every device class is addressed by
name and operation over the NilHost Protocol. The emulator ships *virtual*
drivers; Onuron OS will eventually ship real ones behind the same interface, so
apps never change.

## Permission mapping

| HAL device | Manifest permission |
|---|---|
| `battery`, `power` | `power` |
| `network`, `net` | `network` |
| `printer` | `printer` |
| `scanner` | `scanner` |
| `storage`, `filesystem`, `fs` | `filesystem` |
| `sensors` | `sensors` |
| `clipboard` | `clipboard` |
| `notifications` | `notifications` |
| `camera` | `camera` |

A call to a device whose permission is not granted returns `E_PERMISSION`.
Development shells (empty `app_id`) are always granted.

## Devices

### battery / power
- `level` / `get` → `{level, charging, temperature_c}`
- `set` → `{level?, charging?}`

### network / net
- `status` / `get` → `{mode, online, latency_ms, down_kbps}`
- `set` → `{mode?, online?, latency_ms?}`
- Named profiles: `5g`, `4g`, `3g`, `edge`, `wifi`, `offline`.

### clock / time
- `now` / `get` → `{unix_ms, iso, tz, frozen}`
- `set` → `{unix_ms? | iso?, tz?, frozen?}`. A frozen clock makes screenshots
  deterministic.

### storage / filesystem / fs
Paths are **relative to the app's private directory**; `..` escapes are
rejected by the virtual filesystem.

- `read` → `{path}` → `{data, size}`
- `write` → `{path, data}`
- `list` → `{path}` → `{entries:[{name,path,is_dir,size,mod_time}]}`
- `delete` → `{path}`
- `exists` → `{path}` → `{exists}`

### printer (ESC/POS)
- `print` → `{lines?:[...], data?:bytes}` → `{bytes, receipt:{lines:[{text,align,bold,size}],cut}}`
- `list` → `[PrinterInfo]`
- `last` → last rendered receipt

The emulator encodes/decodes an ESC/POS subset (init, alignment, bold, size,
cut) so "printed on the emulator" and "printed on real hardware" agree.

### scanner
- `scan` → `{value}` → `{format, value, ts}`
- `wedge` → `{value}` → `{keys:[...]}` (keyboard-wedge HID mode)

### clipboard
- `get` → `{text}`
- `set` → `{text}`

### notifications
- `post` → `{title, body, app_id}`
- `list`, `clear`, `dismiss {id}`

### sensors
- `accel`, `gyro`, `gps`
- `set` → `{accel?, gyro?, lat?, lng?}`

### display
- `info` → `{width_px, height_px, dpr, locale}`

### audio / haptics
- `vibrate`
- `set` → `{volume?, brightness?}`

## Fault injection

Any operation can be made to fail deterministically:

```json
{"dev":"printer","op":"print","fail":"paper_out","times":1}
```

Set via the control panel or CLI (`nilctl fault printer print --code paper_out
--times 1 --device <id>`). This is the single most valuable developer feature:
real apps break on paper-out, disk-full and permission-denied paths.
