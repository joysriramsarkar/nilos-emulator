package vfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTraversalIsRejected(t *testing.T) {
	root := t.TempDir()
	fs, err := New(filepath.Join(root, "sandbox"))
	if err != nil {
		t.Fatal(err)
	}

	attacks := []string{
		"../../etc/passwd",
		"..\\..\\Windows\\System32\\config",
		"a/../../b",
		"foo/../../../../bar",
	}
	for _, a := range attacks {
		if _, err := fs.Resolve(a); err == nil {
			t.Errorf("expected %q to be rejected", a)
		}
	}
}

func TestVolumePathsRejected(t *testing.T) {
	fs, _ := New(filepath.Join(t.TempDir(), "sb"))
	for _, a := range []string{`C:\Windows\System32`, `\\server\share\x`, `D:foo`} {
		if _, err := fs.Resolve(a); err == nil {
			t.Errorf("expected volume path %q to be rejected", a)
		}
	}
}

func TestReadWriteList(t *testing.T) {
	fs, _ := New(filepath.Join(t.TempDir(), "sb"))
	if err := fs.WriteFile("/notes/data.json", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile("notes/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"a":1}` {
		t.Fatalf("unexpected content: %s", data)
	}
	entries, err := fs.List("/notes")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "data.json" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
	if !fs.Exists("/notes/data.json") {
		t.Fatal("expected file to exist")
	}
	if err := fs.WriteFile("/../escape.txt", []byte("x")); err == nil {
		t.Fatal("expected escape to fail")
	}
}

func TestSymlinkEscapeRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require privileges on Windows")
	}
	base := t.TempDir()
	root := filepath.Join(base, "sb")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skip("symlink unsupported")
	}
	if _, err := fs.Resolve("link/secret.txt"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
