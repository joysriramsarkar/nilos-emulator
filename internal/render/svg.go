// Package render turns a serialized Alap UI tree into a deterministic SVG
// image. It is the reference renderer used for headless screenshots and
// golden tests; the interactive Studio frontend renders the same tree.
package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Node mirrors the wire format produced by the Alap runtime.
type Node struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Props    map[string]any  `json:"props"`
	Children []Node          `json:"children"`
	Raw      json.RawMessage `json:"-"`
}

// Parse decodes a UI tree.
func Parse(raw json.RawMessage) (Node, error) {
	var n Node
	if err := json.Unmarshal(raw, &n); err != nil {
		return Node{}, err
	}
	return n, nil
}

// Options controls rendering.
type Options struct {
	Width     int
	Height    int
	Theme     string
	Currency  string
	SafeTop   int
	SafeBottom int
}

const (
	pad     = 12
	lineH   = 30
	btnH    = 44
	rowGap  = 8
)

// SVG renders the tree to an SVG document of the requested size.
func SVG(root Node, opt Options) string {
	if opt.Width <= 0 {
		opt.Width = 390
	}
	if opt.Height <= 0 {
		opt.Height = 844
	}
	bg, fg, accent := themeColors(opt.Theme)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		opt.Width, opt.Height, opt.Width, opt.Height)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, opt.Width, opt.Height, bg)

	y := opt.SafeTop + pad
	cursor := &cursor{y: y, x: pad, width: opt.Width - 2*pad, fg: fg, accent: accent}
	cursor.renderNode(&b, root)

	b.WriteString(`</svg>`)
	return b.String()
}

type cursor struct {
	x, y  int
	width int
	fg    string
	accent string
}

func (c *cursor) renderNode(b *strings.Builder, n Node) {
	switch strings.ToLower(n.Type) {
	case "column", "list", "":
		c.y += pad / 2
		for _, ch := range n.Children {
			c.renderNode(b, ch)
		}
		c.y += pad / 2
	case "row":
		startY := c.y
		startX := c.x
		colW := c.width
		if len(n.Children) > 0 {
			colW = (c.width - rowGap*(len(n.Children)-1)) / len(n.Children)
		}
		maxY := c.y
		for _, ch := range n.Children {
			c.x = startX
			c.y = startY
			c.width = colW
			c.renderNode(b, ch)
			if c.y > maxY {
				maxY = c.y
			}
			startX += colW + rowGap
		}
		c.y = maxY
		c.x = startX - colW - rowGap
		c.width = colW
	case "header":
		c.rect(b, c.x, c.y, c.width, 56, c.accent)
		title := text(n.Props, "title")
		sub := text(n.Props, "subtitle")
		c.label(b, c.x+pad, c.y+24, 18, "#ffffff", "", title)
		if sub != "" {
			c.label(b, c.x+pad, c.y+44, 12, "#e6e6e6", "", sub)
		}
		c.y += 56 + rowGap
	case "text":
		style := text(n.Props, "style")
		size := 16
		weight := "normal"
		if style == "title" {
			size = 20
			weight = "bold"
		} else if style == "display" {
			size = 32
			weight = "bold"
		}
		c.label(b, c.x, c.y+size, size, c.fg, weight, text(n.Props, "text"))
		c.y += size + lineH/2
	case "textfield":
		c.rectOutline(b, c.x, c.y, c.width, btnH, "#888888")
		t := text(n.Props, "text")
		if t == "" {
			t = text(n.Props, "placeholder")
		}
		c.label(b, c.x+8, c.y+btnH-14, 14, c.fg, "", t)
		c.y += btnH + rowGap
	case "button":
		c.rect(b, c.x, c.y, c.width, btnH, c.accent)
		c.label(b, c.x+8, c.y+btnH-14, 14, "#ffffff", "bold", text(n.Props, "label"))
		c.y += btnH + rowGap
	case "statusbar", "status":
		c.y = c.y + 4
		c.label(b, c.x, c.y+14, 12, c.fg, "", fmt.Sprintf("battery %s  ·  net %s", text(n.Props, "battery"), text(n.Props, "network")))
		c.y += 24
	default:
		c.renderChildren(b, n)
	}
}

func (c *cursor) renderChildren(b *strings.Builder, n Node) {
	for _, ch := range n.Children {
		c.renderNode(b, ch)
	}
}

func (c *cursor) rect(b *strings.Builder, x, y, w, h int, fill string) {
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="%s"/>`, x, y, w, h, fill)
}

func (c *cursor) rectOutline(b *strings.Builder, x, y, w, h int, stroke string) {
	fmt.Fprintf(b, `<rect x="%d" y="%d" width="%d" height="%d" rx="8" fill="none" stroke="%s" stroke-width="1.5"/>`, x, y, w, h, stroke)
}

func (c *cursor) label(b *strings.Builder, x, y, size int, fill, weight, s string) {
	if s == "" {
		return
	}
	fmt.Fprintf(b, `<text x="%d" y="%d" font-family="'Noto Sans Bengali','Noto Sans',sans-serif" font-size="%d" font-weight="%s" fill="%s">%s</text>`,
		x, y, size, weight, fill, escape(s))
}

func text(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		default:
			return fmt.Sprintf("%v", t)
		}
	}
	return ""
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func themeColors(theme string) (bg, fg, accent string) {
	switch theme {
	case "light":
		return "#f5f5f7", "#1a1a1a", "#2563eb"
	case "cyberpunk":
		return "#0d0221", "#00ffd5", "#ff2a6d"
	case "emerald":
		return "#04160f", "#d1fae5", "#10b981"
	default: // dark
		return "#0f1117", "#f2f4f8", "#4f7cff"
	}
}
