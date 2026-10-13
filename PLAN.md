# nilos-emulator — প্ল্যান (NilOS Studio)

> Go 1.21, MIT। প্রতি-অ্যাপ worker (`nilosd`) + NilHost v1 (NDJSON stdio) + 12-class HAL + fault-injection + record/replay + deterministic golden + LSP/DAP + REST/SSE + QMP (graceful-degrade)।

## বর্তমান অবস্থা
- Headless-ready: `nilctl` (create/start/install/launch/screenshot/battery/fault/record/replay/snapshot) UI ছাড়া; frozen clock+seed গোল্ডেন; CI 3-OS (`vet→test→7-bin build` + lightweight-budget <30MB)।
- প্রোটোকল: hello→init→input/touch/ime/hal/lifecycle/vsync/reload/debug → frame/patch/hal.result/bus/log/trace/state/exit/error; stable id + `h:ref` + coalesced patch।
- ফাঁক: real `nil serve` worker নয় (mockvm/stub পথ), WS loopback future, camera/GPS/audio fidelity + `hal.event` push, replay-golden CI gate, system-mode E2E (onuron.img নেই), nilstudio প্যাকেজিং।

## পরের 6
1. **Real worker integration** — `mockvm/nilosd` stub → actual NilLang runtime end-to-end + conformance test।
2. **WS loopback transport** — `Codec`-over-WebSocket (browser-hosted VM)।
3. **HAL hardening** — camera/GPS/audio, battery/net push, permission-matrix টেস্ট।
4. **Deterministic replay gate** — `record→replay --instant` + golden SVG diff + frozen-clock job।
5. **System-mode E2E** — minimal `onuron.img` + QEMU framebuffer streaming CI (এখন degrade-only)।
6. **Release polish** — nilstudio single-binary, token/CORS default, snapshot/record doc।

## DoD
- `go test ./...` + golden SVG + replay determinism সবুজ; প্রোটোকল বদলে `docs/protocol.md` + ফিক্সচার একসাথে।

## এই সেশনে
- [x] 13 Oct: `go test ./...` সবুজ + golden SVG।
- [x] 13 Oct: deterministic replay gate (`internal/runtime/replay_determinism_test.go`) — `nilctl replay --instant` path: live vs replay vs replay patch stream identical।
