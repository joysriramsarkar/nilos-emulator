// Package vfs implements the emulator's virtual filesystem. Every app gets a
// private subtree under a device root; absolute host paths and traversal
// escapes ("..", symlinks, Windows drive/UNC paths) are rejected.
package vfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ErrEscape is returned when a requested path would leave the sandbox root.
var ErrEscape = errors.New("vfs: path escapes virtual filesystem")

// FS is a sandboxed filesystem rooted at Root.
type FS struct {
	root         string
	resolvedRoot string
}

// New creates the root directory if needed and returns an FS.
func New(root string) (*FS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = abs
	}
	return &FS{root: abs, resolvedRoot: resolved}, nil
}

// Root returns the absolute host path of the sandbox root.
func (fs *FS) Root() string { return fs.root }

// Resolve maps a virtual path (e.g. "/data/apps/notes/data.json") to an
// absolute host path guaranteed to be inside the root.
func (fs *FS) Resolve(virtualPath string) (string, error) {
	if strings.ContainsRune(virtualPath, 0) {
		return "", fmt.Errorf("vfs: invalid NUL in path")
	}

	// Normalise separators first, then reject Windows volume / UNC style
	// absolute paths before trimming so that "//server/share" is caught.
	v := strings.ReplaceAll(virtualPath, "\\", "/")
	if hasVolumePrefix(v) {
		return "", ErrEscape
	}
	v = strings.TrimLeft(v, "/")

	cleaned := filepath.Clean(filepath.FromSlash(v))
	if cleaned == "." {
		cleaned = ""
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", ErrEscape
	}

	target := filepath.Join(fs.root, cleaned)
	if !withinRoot(fs.root, target) {
		return "", ErrEscape
	}

	// If the path (or an ancestor) already exists, make sure no symlink
	// redirects us outside the root.
	if resolved, err := evalExistingAncestors(target); err == nil {
		if !withinRoot(fs.resolvedRoot, resolved) {
			return "", ErrEscape
		}
		target = resolved
	}

	return target, nil
}

// ResolveVirtual is a stricter helper used by the HTTP API: it accepts only
// paths relative to the sandbox and returns the cleaned virtual form.
func (fs *FS) ResolveVirtual(virtualPath string) (string, error) {
	abs, err := fs.Resolve(virtualPath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(fs.root, abs)
	if err != nil {
		return "", err
	}
	return "/" + filepath.ToSlash(rel), nil
}

// ReadFile reads a file inside the sandbox.
func (fs *FS) ReadFile(path string) ([]byte, error) {
	real, err := fs.Resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(real)
}

// WriteFile writes a file inside the sandbox, creating parents.
func (fs *FS) WriteFile(path string, data []byte) error {
	real, err := fs.Resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		return err
	}
	return os.WriteFile(real, data, 0o644)
}

// AppendFile appends to a file inside the sandbox, creating it if needed.
func (fs *FS) AppendFile(path string, data []byte) error {
	real, err := fs.Resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(real, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// MkdirAll creates a directory tree inside the sandbox.
func (fs *FS) MkdirAll(path string) error {
	real, err := fs.Resolve(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(real, 0o755)
}

// Remove deletes a file or directory (recursively) inside the sandbox.
func (fs *FS) Remove(path string) error {
	real, err := fs.Resolve(path)
	if err != nil {
		return err
	}
	if real == fs.root {
		return fmt.Errorf("vfs: refusing to remove sandbox root")
	}
	return os.RemoveAll(real)
}

// Exists reports whether the virtual path exists inside the sandbox.
func (fs *FS) Exists(path string) bool {
	real, err := fs.Resolve(path)
	if err != nil {
		return false
	}
	_, err = os.Stat(real)
	return err == nil
}

// Entry describes one directory entry.
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
}

// List returns the entries of a directory inside the sandbox.
func (fs *FS) List(dir string) ([]Entry, error) {
	real, err := fs.Resolve(dir)
	if err != nil {
		return nil, err
	}
	des, err := os.ReadDir(real)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		info, ierr := de.Info()
		var size, mod int64
		if ierr == nil {
			size = info.Size()
			mod = info.ModTime().UnixMilli()
		}
		vpath, _ := fs.ResolveVirtual(filepath.Join(dir, de.Name()))
		out = append(out, Entry{
			Name:    de.Name(),
			Path:    vpath,
			IsDir:   de.IsDir(),
			Size:    size,
			ModTime: mod,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
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

// evalExistingAncestors walks up until it finds an existing path, resolving
// symlinks for that prefix.
func evalExistingAncestors(target string) (string, error) {
	cur := target
	var tail []string
	for {
		if _, err := os.Lstat(cur); err == nil {
			resolved, rerr := filepath.EvalSymlinks(cur)
			if rerr != nil {
				return target, nil
			}
			parts := append([]string{resolved}, tail...)
			return filepath.Join(parts...), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return target, nil
		}
		tail = append([]string{filepath.Base(cur)}, tail...)
		cur = parent
	}
}

func hasVolumePrefix(v string) bool {
	if runtime.GOOS != "windows" {
		// Volume prefixes (C:) and UNC paths (\\server\share) are Windows-only
		// concepts. On Unix, a path like "a:b" is a valid filename.
		return false
	}
	if len(v) >= 2 && v[1] == ':' {
		c := v[0]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			return true
		}
	}
	// UNC: //server/share
	if strings.HasPrefix(v, "//") {
		return true
	}
	return false
}
