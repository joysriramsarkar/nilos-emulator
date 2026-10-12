// Package sandbox provides the emulator's path guard and per-platform
// resource limits for app worker processes.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
		resolvedRoot, rerr := filepath.EvalSymlinks(root)
		if rerr != nil {
			resolvedRoot = root
		}
		if withinRoot(resolvedRoot, abs) {
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

func withinRoot(root, target string) bool {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	if target == root {
		return true
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			return caseInsensitiveWithin(root, target)
		}
		return false
	}
	return !filepath.IsAbs(rel)
}

// caseInsensitiveWithin reports whether target is inside root, comparing
// path components case-insensitively. It is used on Windows and macOS where
// the filesystem is case-insensitive and filepath.Rel can produce false
// positives when the case differs.
func caseInsensitiveWithin(root, target string) bool {
	rootSlash := filepath.ToSlash(root)
	targetSlash := filepath.ToSlash(target)
	if len(targetSlash) < len(rootSlash) {
		return false
	}
	if !strings.EqualFold(targetSlash[:len(rootSlash)], rootSlash) {
		return false
	}
	if len(targetSlash) == len(rootSlash) {
		return true
	}
	return targetSlash[len(rootSlash)] == '/'
}
