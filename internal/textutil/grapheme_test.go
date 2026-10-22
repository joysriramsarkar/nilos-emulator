package textutil

import (
	"reflect"
	"testing"
)

func TestClustersBangla(t *testing.T) {
	cases := map[string][]string{
		"ক্ষমা":    {"ক্ষ", "মা"},    // conjunct then ma+aa
		"কি":       {"কি"},           // ka + i-matra
		"দ্বন্দ্ব": {"দ্ব", "ন্দ্ব"}, // stacked conjuncts
		"র্যাব":    {"র্যা", "ব"},    // ra+virama+ya, then ba
		"abc":      {"a", "b", "c"},
		"":         nil,
	}
	for in, want := range cases {
		got := Clusters(in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Clusters(%q) = %#v, want %#v", in, got, want)
		}
	}
}

func TestCountMatchesClusters(t *testing.T) {
	for _, s := range []string{"ক্ষমা", "কি", "hello", "১২৩", "নীলাং"} {
		if got, want := Count(s), len(Clusters(s)); got != want {
			t.Errorf("Count(%q)=%d, len(Clusters)=%d", s, got, want)
		}
	}
}

func TestDeleteLastClusterVsRune(t *testing.T) {
	// Backspace by cluster removes the whole "কি"; by rune removes only "ি".
	if got := DeleteLastCluster("কি"); got != "" {
		t.Errorf("DeleteLastCluster(কি) = %q, want empty", got)
	}
	if got := DeleteLastRune("কি"); got != "ক" {
		t.Errorf("DeleteLastRune(কি) = %q, want ক", got)
	}
	if got := DeleteLastCluster("ক্ষমা"); got != "ক্ষ" {
		t.Errorf("DeleteLastCluster(ক্ষমা) = %q, want ক্ষ", got)
	}
}

func TestBoundariesAndNavigation(t *testing.T) {
	s := "ক্ষমা" // clusters: [ক্ষ][মা], byte offsets 0,6,12
	b := Boundaries(s)
	if b[0] != 0 || b[len(b)-1] != len(s) {
		t.Fatalf("unexpected boundaries: %#v", b)
	}
	if got := NextBoundary(s, 0); got != len("ক্ষ") {
		t.Errorf("NextBoundary(0) = %d, want %d", got, len("ক্ষ"))
	}
	if got := PrevBoundary(s, len(s)); got != len("ক্ষ") {
		t.Errorf("PrevBoundary(end) = %d, want %d", got, len("ক্ষ"))
	}
}
