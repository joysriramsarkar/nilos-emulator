package qemu

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func makePPM(w, h int) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "P6\n%d %d\n255\n", w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			buf.Write([]byte{byte(x), byte(y), 128})
		}
	}
	return buf.Bytes()
}

func TestDecodePPM(t *testing.T) {
	img, err := DecodePPM(makePPM(4, 3))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 4 || img.Bounds().Dy() != 3 {
		t.Fatalf("unexpected bounds: %v", img.Bounds())
	}
	r, g, b, a := img.At(3, 2).RGBA()
	if r>>8 != 3 || g>>8 != 2 || b>>8 != 128 || a>>8 != 255 {
		t.Fatalf("unexpected pixel: %d %d %d %d", r>>8, g>>8, b>>8, a>>8)
	}
}

func TestPPMToPNG(t *testing.T) {
	pngBytes, err := PPMToPNG(makePPM(5, 5))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 5 {
		t.Fatalf("png wrong size: %v", img.Bounds())
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if a == 0 {
		t.Fatal("expected opaque pixel")
	}
	_ = color.NRGBA{}
	_ = r
	_ = g
	_ = b
}

func TestDecodePPMRejectsGarbage(t *testing.T) {
	if _, err := DecodePPM([]byte("not a ppm")); err == nil {
		t.Fatal("expected error for garbage input")
	}
}

// TestQMPClientWithFakeServer exercises the QMP handshake and screendump
// command against a scripted server (no QEMU installation needed).
func TestQMPClientWithFakeServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	dir := t.TempDir()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// greeting
		fmt.Fprintln(conn, `{"QMP":{"version":{"qemu":{"major":8,"minor":0,"micro":0}}}}`)
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			var req map[string]any
			if json.Unmarshal(sc.Bytes(), &req) != nil {
				continue
			}
			switch req["execute"] {
			case "qmp_capabilities":
				fmt.Fprintln(conn, `{"return":{}}`)
			case "screendump":
				args, _ := req["arguments"].(map[string]any)
				name, _ := args["filename"].(string)
				if err := os.WriteFile(name, makePPM(3, 2), 0o644); err != nil {
					fmt.Fprintln(conn, `{"error":{"class":"GenericError","desc":"write failed"}}`)
					continue
				}
				fmt.Fprintln(conn, `{"return":{}}`)
			default:
				fmt.Fprintln(conn, `{"return":{}}`)
			}
		}
	}()

	client, err := Dial(ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	out := filepath.Join(dir, "fb.ppm")
	if _, err := client.Execute("screendump", map[string]any{"filename": out}); err != nil {
		t.Fatalf("screendump: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodePPM(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() != 3 {
		t.Fatalf("unexpected image: %v", img.Bounds())
	}
}
