package hal

import (
	"strings"
)

// ReceiptLine is a decoded line of a thermal receipt.
type ReceiptLine struct {
	Text  string `json:"text"`
	Align string `json:"align"` // left | center | right
	Bold  bool   `json:"bold"`
	Size  int    `json:"size"` // 1 or 2 (double width/height)
}

// ReceiptDoc is the decoded form of ESC/POS bytes for on-screen rendering.
type ReceiptDoc struct {
	Lines []ReceiptLine `json:"lines"`
	Bytes int           `json:"bytes"`
	Cut   bool          `json:"cut"`
}

// ESC/POS command bytes.
var (
	escInit     = []byte{0x1B, 0x40}       // ESC @
	escAlign    = []byte{0x1B, 0x61}       // ESC a n
	escBold     = []byte{0x1B, 0x45}       // ESC E n
	escSize     = []byte{0x1D, 0x21}       // GS ! n
	escCut      = []byte{0x1D, 0x56, 0x00} // GS V 0
)

// EncodeReceipt builds a small ESC/POS byte stream from plain text lines.
// The first line is printed centred and bold, mimicking a receipt header.
func EncodeReceipt(lines []string) []byte {
	var b []byte
	b = append(b, escInit...)
	if len(lines) > 0 {
		b = append(b, escAlign...)
		b = append(b, 1) // center
		b = append(b, escBold...)
		b = append(b, 1)
		b = append(b, []byte(lines[0])...)
		b = append(b, '\n')
		b = append(b, escBold...)
		b = append(b, 0)
		b = append(b, escAlign...)
		b = append(b, 0) // left
		for _, l := range lines[1:] {
			b = append(b, []byte(l)...)
			b = append(b, '\n')
		}
	}
	b = append(b, '\n', '\n', '\n')
	b = append(b, escCut...)
	return b
}

// DecodeESCPS parses a subset of ESC/POS into a renderable document.
func DecodeESCPS(data []byte) ReceiptDoc {
	doc := ReceiptDoc{Bytes: len(data)}
	align := "left"
	bold := false
	size := 1
	var cur strings.Builder

	flush := func() {
		text := cur.String()
		if text != "" || len(doc.Lines) == 0 {
			doc.Lines = append(doc.Lines, ReceiptLine{Text: text, Align: align, Bold: bold, Size: size})
		}
		cur.Reset()
	}

	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == 0x1B && i+1 < len(data) {
			switch data[i+1] {
			case 0x40: // init
				i++
				continue
			case 0x61: // align
				if i+2 < len(data) {
					align = alignName(data[i+2])
					i += 2
					continue
				}
			case 0x45: // bold
				if i+2 < len(data) {
					bold = data[i+2] != 0
					i += 2
					continue
				}
			}
		}
		if c == 0x1D && i+1 < len(data) {
			switch data[i+1] {
			case 0x21: // size
				if i+2 < len(data) {
					n := data[i+2]
					w := (n >> 4) & 0x0F
					h := n & 0x0F
					if w > 0 || h > 0 {
						size = 2
					} else {
						size = 1
					}
					i += 2
					continue
				}
			case 0x56: // cut
				doc.Cut = true
				i += 2
				continue
			}
		}
		if c == '\n' {
			flush()
			continue
		}
		if c == '\r' {
			continue
		}
		cur.WriteByte(c)
	}
	flush()
	return doc
}

func alignName(n byte) string {
	switch n {
	case 1:
		return "center"
	case 2:
		return "right"
	default:
		return "left"
	}
}
