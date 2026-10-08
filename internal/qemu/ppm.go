package qemu

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strconv"
)

// DecodePPM parses a binary P6 (or ASCII P3) PPM image.
func DecodePPM(data []byte) (image.Image, error) {
	r := bufio.NewReader(bytes.NewReader(data))
	token := func() (string, error) {
		for {
			b, err := r.ReadByte()
			if err != nil {
				return "", err
			}
			if b == '#' {
				for {
					c, err := r.ReadByte()
					if err != nil {
						return "", err
					}
					if c == '\n' {
						break
					}
				}
				continue
			}
			if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
				continue
			}
			buf := []byte{b}
			for {
				c, err := r.ReadByte()
				if err != nil {
					return string(buf), nil
				}
				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					return string(buf), nil
				}
				buf = append(buf, c)
			}
		}
	}

	magic, err := token()
	if err != nil {
		return nil, fmt.Errorf("ppm: %w", err)
	}
	if magic != "P6" && magic != "P3" {
		return nil, fmt.Errorf("ppm: unsupported format %q", magic)
	}
	wStr, err := token()
	if err != nil {
		return nil, err
	}
	hStr, err := token()
	if err != nil {
		return nil, err
	}
	maxStr, err := token()
	if err != nil {
		return nil, err
	}
	w, err := strconv.Atoi(wStr)
	if err != nil {
		return nil, fmt.Errorf("ppm: bad width %q", wStr)
	}
	h, err := strconv.Atoi(hStr)
	if err != nil {
		return nil, fmt.Errorf("ppm: bad height %q", hStr)
	}
	maxVal, err := strconv.Atoi(maxStr)
	if err != nil || maxVal <= 0 || maxVal > 255 {
		return nil, fmt.Errorf("ppm: unsupported maxval %q", maxStr)
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if magic == "P6" {
		// The last token already consumed the single whitespace separator.
		raw := make([]byte, w*h*3)
		if _, err := readFull(r, raw); err != nil {
			return nil, fmt.Errorf("ppm: truncated pixel data: %w", err)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := (y*w + x) * 3
				img.SetNRGBA(x, y, color.NRGBA{R: raw[i], G: raw[i+1], B: raw[i+2], A: 255})
			}
		}
		return img, nil
	}

	// P3 ASCII
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var comps [3]int
			for c := 0; c < 3; c++ {
				s, err := token()
				if err != nil {
					return nil, err
				}
				v, err := strconv.Atoi(s)
				if err != nil {
					return nil, err
				}
				comps[c] = v * 255 / maxVal
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(comps[0]), G: uint8(comps[1]), B: uint8(comps[2]), A: 255})
		}
	}
	return img, nil
}

// PPMToPNG decodes a PPM and re-encodes it as PNG bytes.
func PPMToPNG(data []byte) ([]byte, error) {
	img, err := DecodePPM(data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
