package render

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func goldenTree() Node {
	return Node{ID: "root", Type: "Column", Children: []Node{
		{ID: "header", Type: "Header", Props: map[string]any{"title": "নীলাং নোটবুক (sample)", "subtitle": "1.0.0"}},
		{ID: "note.input", Type: "TextField", Props: map[string]any{"text": "আমার প্রথম নোট", "placeholder": "নতুন নোট লিখুন..."}},
		{ID: "btn.add", Type: "Button", Props: map[string]any{"label": "নোট যোগ করুন", "onTap": "h:1"}},
		{ID: "note.list", Type: "Column", Children: []Node{
			{ID: "note.0", Type: "Text", Props: map[string]any{"text": "নোট ১: ক্ষমা ও দ্বন্দ্ব", "style": "body"}},
		}},
		{ID: "status", Type: "StatusBar", Props: map[string]any{"battery": "98", "network": "wifi"}},
	}}
}

// TestGoldenNotesScreenshot is the golden-screenshot test the plan calls for:
// it renders a fixed tree under a frozen theme/size and compares byte-for-byte.
// Regenerate with: go test ./internal/render -run Golden -update
func TestGoldenNotesScreenshot(t *testing.T) {
	svg := SVG(goldenTree(), Options{Width: 390, Height: 844, Theme: "dark"})

	golden := filepath.Join("testdata", "notes.svg")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(svg), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden updated:", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden file (run with -update): %v", err)
	}
	if svg != string(want) {
		t.Fatalf("rendered SVG does not match golden %s\n--- got ---\n%s", golden, svg)
	}
}

// TestRenderIsDeterministic renders the same tree twice and requires identical
// output — the basis for deterministic golden tests.
func TestRenderIsDeterministic(t *testing.T) {
	a := SVG(goldenTree(), Options{Width: 390, Height: 844, Theme: "dark"})
	b := SVG(goldenTree(), Options{Width: 390, Height: 844, Theme: "dark"})
	if a != b {
		t.Fatal("renderer is not deterministic")
	}
}
