package packages

import (
	"archive/zip"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func buildSignedApp(t *testing.T, dir string) (bundlePath, pubHex string) {
	t.Helper()
	src := filepath.Join(dir, "app")
	if err := os.MkdirAll(filepath.Join(src, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := Manifest{
		ID: "com.example.demo", Name: "Demo", Version: "1.0.0",
		Entry: "payload/index.html", Permissions: []string{"filesystem", "network"},
	}
	mb, _ := json.MarshalIndent(man, "", "  ")
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), mb, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "payload", "index.html"), []byte("<h1>hello</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "demo.nilax")
	if err := SignDir(src, out, priv, "test-key"); err != nil {
		t.Fatal(err)
	}
	return out, pub
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	bundlePath, pub := buildSignedApp(t, dir)

	opts := DefaultOptions()
	opts.TrustKeys = TrustKeysFromHex(map[string]string{"test-key": pub})
	dest := filepath.Join(dir, "extract")
	bundle, err := LoadAndVerify(bundlePath, dest, opts)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if bundle.Manifest.ID != "com.example.demo" {
		t.Fatalf("unexpected id: %s", bundle.Manifest.ID)
	}
	if bundle.Signature == nil || bundle.Signature.KeyID != "test-key" {
		t.Fatal("signature not recorded")
	}
	if _, err := os.Stat(filepath.Join(dest, "payload", "index.html")); err != nil {
		t.Fatal("payload not extracted")
	}
}

func TestTamperedPayloadRejected(t *testing.T) {
	dir := t.TempDir()
	bundlePath, pub := buildSignedApp(t, dir)
	rewriteEntry(t, bundlePath, "payload/index.html", []byte("<h1>tampered</h1>"))

	opts := DefaultOptions()
	opts.TrustKeys = TrustKeysFromHex(map[string]string{"test-key": pub})
	if _, err := LoadAndVerify(bundlePath, filepath.Join(dir, "out"), opts); err == nil {
		t.Fatal("expected tampered bundle to be rejected")
	}
}

func TestUntrustedKeyRejected(t *testing.T) {
	dir := t.TempDir()
	bundlePath, _ := buildSignedApp(t, dir)
	opts := DefaultOptions()
	opts.TrustKeys = map[string]ed25519.PublicKey{}
	if _, err := LoadAndVerify(bundlePath, filepath.Join(dir, "out"), opts); err == nil {
		t.Fatal("expected untrusted key rejection")
	}
}

func TestUnsignedPolicy(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "u")
	_ = os.MkdirAll(filepath.Join(src, "payload"), 0o755)
	man := Manifest{ID: "com.example.u", Name: "U", Version: "1.0.0", Entry: "payload/i.html"}
	mb, _ := json.Marshal(man)
	_ = os.WriteFile(filepath.Join(src, "manifest.json"), mb, 0o644)
	_ = os.WriteFile(filepath.Join(src, "payload", "i.html"), []byte("x"), 0o644)
	bundlePath := filepath.Join(dir, "u.nilax")
	if err := ZipDir(src, bundlePath); err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	if _, err := LoadAndVerify(bundlePath, filepath.Join(dir, "o1"), opts); err == nil {
		t.Fatal("expected unsigned bundle to be rejected")
	}
	opts.AllowUnsigned = true
	if _, err := LoadAndVerify(bundlePath, filepath.Join(dir, "o2"), opts); err != nil {
		t.Fatalf("expected unsigned bundle accepted: %v", err)
	}
}

func TestZipSlipRejected(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "evil.nilax")
	f, err := os.Create(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	man := Manifest{ID: "com.evil.app", Name: "Evil", Version: "1.0.0", Entry: "payload/index.html"}
	mb, _ := json.Marshal(man)
	w, _ := zw.Create("manifest.json")
	w.Write(mb)
	w2, _ := zw.Create("payload/../../evil.txt")
	w2.Write([]byte("pwned"))
	zw.Close()
	f.Close()

	opts := DefaultOptions()
	opts.AllowUnsigned = true
	if _, err := LoadAndVerify(bundlePath, filepath.Join(dir, "out"), opts); err == nil {
		t.Fatal("expected zip-slip to be rejected")
	}
}

func TestManifestValidation(t *testing.T) {
	bad := [][]byte{
		[]byte(`{"id":"bad id","name":"x","version":"1","entry":"payload/a"}`),
		[]byte(`{"id":"com.a.b","name":"x","version":"1","entry":"outside/a"}`),
		[]byte(`{"id":"com.a.b","name":"","version":"1","entry":"payload/a"}`),
	}
	for _, b := range bad {
		if _, err := ParseManifest(b); err == nil {
			t.Errorf("expected invalid manifest: %s", b)
		}
	}
	good := []byte(`{"id":"com.a.b","name":"x","version":"1","entry":"payload/a"}`)
	if _, err := ParseManifest(good); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
}

func rewriteEntry(t *testing.T, path, name string, newData []byte) {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	tmp := path + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		zr.Close()
		t.Fatal(err)
	}
	zw := zip.NewWriter(out)
	for _, file := range zr.File {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: file.Name, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == name {
			w.Write(newData)
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(w, rc)
		rc.Close()
	}
	zw.Close()
	out.Close()
	zr.Close()
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}
