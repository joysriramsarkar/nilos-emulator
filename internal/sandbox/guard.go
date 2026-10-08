// Package sandbox provides the emulator's path guard and per-platform
// resource limits for app worker processes.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Guard restricts filesystem access to a set of allowed roots.
type Guard struct {
	roots []string
}

// NewGuard creates a guard allowing access beneath the given roots. Each
// root is converted to an absolute path.
func NewGuard(roots ...string) *Guard {
	g := &Guard{}
	for _, r := range roots {
		if r == "" {
			continue
		}
		abs, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		g.roots = append(g.roots, filepath.Clean(abs))
	}
	return g
}

// Roots returns the allowed roots.
func (g *Guard) Roots() []string {
	out := make([]string, len(g.roots))
	copy(out, g.roots)
	return out
}

// Check resolves p and verifies it lies within one of the allowed roots.
// It rejects traversal and symlink escapes.
func (g *Guard) Check(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("sandbox: empty path")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	// Resolve symlinks for the existing prefix.
	if resolved, err := resolveExisting(abs); err == nil {
		abs = resolved
	}

	for _, root := range g.roots {
		if root == abs {
			return abs, nil
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			continue
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("sandbox: path %q is outside the allowed roots", p)
}

// Within reports whether p is inside any allowed root.
func (g *Guard) Within(p string) bool {
	_, err := g.Check(p)
	return err == nil
}

func resolveExisting(p string) (string, error) {
	cur := p
	var tail []string
	for {
		if _, err := os.Lstat(cur); err == nil {
			resolved, rerr := filepath.EvalSymlinks(cur)
			if rerr != nil {
				return p, nil
			}
			parts := append([]string{resolved}, tail...)
			return filepath.Join(parts...), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p, nil
		}
		tail = append([]string{filepath.Base(cur)}, tail...)
		cur = parent
	}
}
