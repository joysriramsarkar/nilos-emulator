// Command nilsign creates Ed25519 keys and signs .nilax bundles.
//
//	nilsign keygen --out dev.key
//	nilsign sign --dir ./my-app --out my-app.nilax --key dev.key --key-id dev
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joysriramsarkar/nilos-emulator/internal/packages"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "sign":
		err = sign(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`nilsign — .nilax signing tool

  nilsign keygen --out dev.key                 generate a signing key
  nilsign sign --dir ./app --out app.nilax \
               --key dev.key --key-id dev      sign an unpacked app

The app directory must contain manifest.json and a payload/ directory.
`)
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	out := fs.String("out", "dev.key", "private key output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	priv, pubHex, err := packages.GenerateKeyPair()
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, []byte(hex.EncodeToString(priv)), 0o600); err != nil {
		return err
	}
	fmt.Printf("private key: %s\n", *out)
	fmt.Printf("public key : %s\n", pubHex)
	fmt.Println("Add to nilemu.toml:")
	fmt.Println("[keys]")
	fmt.Printf("my-key = \"%s\"\n", pubHex)
	return nil
}

func sign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	dir := fs.String("dir", "", "unpacked app directory")
	out := fs.String("out", "app.nilax", "output bundle")
	key := fs.String("key", "dev.key", "private key file")
	keyID := fs.String("key-id", "dev", "signing key id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return fmt.Errorf("--dir is required")
	}
	priv, err := loadKey(*key)
	if err != nil {
		return err
	}
	if err := packages.SignDir(*dir, *out, priv, *keyID); err != nil {
		return err
	}
	fmt.Println("signed:", *out)
	return nil
}

func loadKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size %d", len(raw))
	}
	return ed25519.PrivateKey(raw), nil
}
