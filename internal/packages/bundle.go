package packages

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SignatureFile is the contents of SIGNATURE.json inside a .nilax bundle.
type SignatureFile struct {
	KeyID          string `json:"key_id"`
	ManifestSHA256 string `json:"manifest_sha256"`
	PayloadSHA256  string `json:"payload_sha256"`
	Signature      string `json:"signature"`
}

// Bundle is a verified (or, when AllowUnsigned is set, accepted) .nilax.
type Bundle struct {
	Path      string
	Manifest  *Manifest
	Signature *SignatureFile
	Root      string
}

// Options controls bundle loading and verification.
type Options struct {
	TrustKeys     map[string]ed25519.PublicKey
	MaxBundleSize int64
	MaxFileCount  int
	MaxFileSize   int64
	MaxTotalSize  int64
	AllowUnsigned bool
}

// DefaultOptions returns sensible safety limits.
func DefaultOptions() Options {
	return Options{
		TrustKeys:     map[string]ed25519.PublicKey{},
		MaxBundleSize: 100 * 1024 * 1024,
		MaxFileCount:  5000,
		MaxFileSize:   50 * 1024 * 1024,
		MaxTotalSize:  200 * 1024 * 1024,
	}
}

// TrustKeysFromHex builds the trust map from key-id to hex public key.
func TrustKeysFromHex(m map[string]string) map[string]ed25519.PublicKey {
	out := make(map[string]ed25519.PublicKey, len(m))
	for id, hexKey := range m {
		raw, err := hex.DecodeString(strings.TrimSpace(hexKey))
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		out[id] = ed25519.PublicKey(raw)
	}
	return out
}

// LoadAndVerify opens a .nilax bundle, verifies its manifest/payload hashes
// and Ed25519 signature, and extracts the payload into destDir.
func LoadAndVerify(path string, destDir string, opts Options) (*Bundle, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if opts.MaxBundleSize > 0 && fi.Size() > opts.MaxBundleSize {
		return nil, fmt.Errorf("bundle too large: %d bytes", fi.Size())
	}

	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open nilax bundle: %w", err)
	}
	defer zr.Close()

	if len(zr.File) == 0 {
		return nil, errors.New("empty bundle")
	}
	if opts.MaxFileCount > 0 && len(zr.File) > opts.MaxFileCount {
		return nil, fmt.Errorf("too many files: %d", len(zr.File))
	}

	manifestFile, err := findFile(zr.File, "manifest.json")
	if err != nil {
		return nil, err
	}
	manifestBytes, err := readFileLimited(manifestFile, opts.MaxFileSize)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return nil, err
	}

	sigFile, err := findFile(zr.File, "SIGNATURE.json")
	if err != nil {
		if !opts.AllowUnsigned {
			return nil, err
		}
		if err := extractPayload(zr.File, destDir, opts); err != nil {
			return nil, err
		}
		return &Bundle{Path: path, Manifest: manifest, Root: destDir}, nil
	}

	sigBytes, err := readFileLimited(sigFile, opts.MaxFileSize)
	if err != nil {
		return nil, err
	}
	var sig SignatureFile
	if err := json.Unmarshal(sigBytes, &sig); err != nil {
		return nil, fmt.Errorf("invalid signature json: %w", err)
	}

	manifestDigest := sha256.Sum256(manifestBytes)
	if !strings.EqualFold(hex.EncodeToString(manifestDigest[:]), sig.ManifestSHA256) {
		return nil, errors.New("manifest hash mismatch")
	}

	payloadDigest, err := computePayloadDigest(zr.File, opts)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(hex.EncodeToString(payloadDigest[:]), sig.PayloadSHA256) {
		return nil, errors.New("payload hash mismatch")
	}

	pub, ok := opts.TrustKeys[sig.KeyID]
	if !ok {
		return nil, fmt.Errorf("untrusted signing key: %s", sig.KeyID)
	}
	rawSig, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}
	message := append(append([]byte{}, manifestDigest[:]...), payloadDigest[:]...)
	if !ed25519.Verify(pub, message, rawSig) {
		return nil, errors.New("ed25519 signature verification failed")
	}

	if err := extractPayload(zr.File, destDir, opts); err != nil {
		return nil, err
	}
	return &Bundle{Path: path, Manifest: manifest, Signature: &sig, Root: destDir}, nil
}

func findFile(files []*zip.File, name string) (*zip.File, error) {
	for _, f := range files {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("required file not found: %s", name)
}

func readFileLimited(f *zip.File, max int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	if max <= 0 {
		return io.ReadAll(rc)
	}
	buf := make([]byte, max+1)
	n, err := io.ReadFull(rc, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	if int64(n) > max {
		return nil, fmt.Errorf("file too large: %s", f.Name)
	}
	return buf[:n], nil
}

func computePayloadDigest(files []*zip.File, opts Options) ([32]byte, error) {
	var payloadFiles []*zip.File
	for _, f := range files {
		if strings.HasPrefix(f.Name, "payload/") && !f.FileInfo().IsDir() {
			payloadFiles = append(payloadFiles, f)
		}
	}
	sort.Slice(payloadFiles, func(i, j int) bool { return payloadFiles[i].Name < payloadFiles[j].Name })

	h := sha256.New()
	var total int64
	for _, f := range payloadFiles {
		data, err := readFileLimited(f, opts.MaxFileSize)
		if err != nil {
			return [32]byte{}, err
		}
		total += int64(len(data))
		if opts.MaxTotalSize > 0 && total > opts.MaxTotalSize {
			return [32]byte{}, errors.New("total payload size exceeds limit")
		}
		fileHash := sha256.Sum256(data)
		fmt.Fprintf(h, "%s\t%s\n", f.Name, hex.EncodeToString(fileHash[:]))
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func extractPayload(files []*zip.File, destDir string, opts Options) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	var total int64
	for _, f := range files {
		if !strings.HasPrefix(f.Name, "payload/") {
			continue
		}
		target, err := SafeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		// Guard against symlink entries in the archive.
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		limited := io.LimitReader(rc, opts.MaxFileSize+1)
		n, err := io.Copy(out, limited)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
		if opts.MaxFileSize > 0 && n > opts.MaxFileSize {
			return fmt.Errorf("file too large: %s", f.Name)
		}
		total += n
		if opts.MaxTotalSize > 0 && total > opts.MaxTotalSize {
			return errors.New("total extracted size exceeds limit")
		}
	}
	return nil
}

// SafeJoin joins name under root, rejecting path traversal and zip-slip.
func SafeJoin(root, name string) (string, error) {
	cleanRoot := filepath.Clean(root)
	target := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(name)))
	if target != cleanRoot && !strings.HasPrefix(target, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe path: %s", name)
	}
	return target, nil
}

// GenerateKeyPair creates a new Ed25519 key pair and returns the private key
// and the hex-encoded public key.
func GenerateKeyPair() (ed25519.PrivateKey, string, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, "", err
	}
	return priv, hex.EncodeToString(pub), nil
}

// SignDir signs sourceDir (containing manifest.json and payload/) with priv
// and writes a deterministic .nilax zip to outFile.
func SignDir(sourceDir, outFile string, priv ed25519.PrivateKey, keyID string) error {
	manifestBytes, err := os.ReadFile(filepath.Join(sourceDir, "manifest.json"))
	if err != nil {
		return err
	}
	manifest, err := ParseManifest(manifestBytes)
	if err != nil {
		return err
	}
	_ = manifest

	manifestDigest := sha256.Sum256(manifestBytes)
	payloadDigest, err := ComputeDirDigest(filepath.Join(sourceDir, "payload"))
	if err != nil {
		return err
	}
	message := append(append([]byte{}, manifestDigest[:]...), payloadDigest[:]...)
	sig := ed25519.Sign(priv, message)

	sigFile := SignatureFile{
		KeyID:          keyID,
		ManifestSHA256: hex.EncodeToString(manifestDigest[:]),
		PayloadSHA256:  hex.EncodeToString(payloadDigest[:]),
		Signature:      base64.StdEncoding.EncodeToString(sig),
	}
	sigBytes, err := json.MarshalIndent(sigFile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SIGNATURE.json"), sigBytes, 0o644); err != nil {
		return err
	}
	return ZipDir(sourceDir, outFile)
}

// ComputeDirDigest computes the payload digest for a directory tree using the
// same algorithm as bundle verification.
func ComputeDirDigest(root string) ([32]byte, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return [32]byte{}, err
	}
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return [32]byte{}, err
		}
		fileHash := sha256.Sum256(data)
		fmt.Fprintf(h, "payload/%s\t%s\n", rel, hex.EncodeToString(fileHash[:]))
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

// ZipDir writes a deterministic zip of src to dst: entries are sorted, use a
// fixed timestamp, and fixed compression so identical inputs hash equally.
func ZipDir(src, dst string) error {
	var files []string
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, path := range files {
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hdr := &zip.FileHeader{Name: name, Method: zip.Deflate}
		hdr.Modified = fixed
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}
