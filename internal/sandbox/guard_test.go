package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuardAllowsInsideRejectsOutside(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	g := NewGuard(root)

	if _, err := g.Check(filepath.Join(sub, "file.txt")); err != nil {
		t.Fatalf("inside path rejected: %v", err)
	}
	if _, err := g.Check(filepath.Join(root, "..", "outside.txt")); err == nil {
		t.Fatal("expected outside path rejected")
	}
	if !g.Within(sub) {
		t.Fatal("expected Within(sub)=true")
	}
	if g.Within(filepath.Dir(root)) {
		t.Fatal("expected parent outside guard")
	}
}

func TestGuardMultipleRoots(t *testing.T) {
	r1 := t.TempDir()
	r2 := t.TempDir()
	g := NewGuard(r1, r2)
	if _, err := g.Check(filepath.Join(r2, "x")); err != nil {
		t.Fatalf("second root rejected: %v", err)
	}
}
