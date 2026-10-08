package render

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSVGContainsContent(t *testing.T) {
	tree := Node{ID: "root", Type: "Column", Children: []Node{
		{ID: "h", Type: "Header", Props: map[string]any{"title": "নোটস", "subtitle": "v1.0.0"}},
		{ID: "t", Type: "Text", Props: map[string]any{"text": "সংখ্যা: 5"}},
		{ID: "b", Type: "Button", Props: map[string]any{"label": "বাড়াও"}},
	}}
	svg := SVG(tree, Options{Width: 390, Height: 844, Theme: "dark"})
	for _, want := range []string{"<svg", "নোটস", "সংখ্যা:", "বাড়াও", "</svg>"} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG missing %q\n%s", want, svg)
		}
	}
}

func TestParseAndEscaping(t *testing.T) {
	raw := json.RawMessage(`{"id":"t","type":"Text","props":{"text":"<b> & co"}}`)
	n, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	svg := SVG(n, Options{})
	if strings.Contains(svg, "<b> & co") {
		t.Fatal("expected escaping of markup characters")
	}
	if !strings.Contains(svg, "&lt;b&gt; &amp; co") {
		t.Fatalf("expected escaped text, got:\n%s", svg)
	}
}
