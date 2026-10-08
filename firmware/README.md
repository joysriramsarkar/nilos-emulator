# Sample firmware

`sample-notes.nilax` is an Ed25519-signed bundle built from
`sample-notes/`. Its public key is already trusted in `nilemu.toml`:

```toml
[keys]
dev-sample = "01aa2e9e5b466319306066d7fe23b9636450d723d93a1080340ac3cc74142ce5"
```

Rebuild after editing the app:

```powershell
# once: create your own key and add the printed public key to nilemu.toml
go run ./cmd/nilsign keygen --out dev.key

# sign the source directory
go run ./cmd/nilsign sign --dir firmware/sample-notes `
    --out firmware/sample-notes.nilax --key dev.key --key-id dev-sample
```

Install it into a running emulator:

```powershell
nilctl install firmware/sample-notes.nilax --device <device-id>
nilctl launch org.nilang.notes --device <device-id>
```

> The private key (`dev.key`) is git-ignored. Never commit it.
