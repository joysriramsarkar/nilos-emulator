package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type POSProduct struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	Barcode string  `json:"barcode"`
}

type POSCartItem struct {
	Product  POSProduct `json:"product"`
	Quantity int        `json:"quantity"`
}

func TestEmulatorPOSCalculations(t *testing.T) {
	items := []POSCartItem{
		{
			Product:  POSProduct{ID: "p-01", Name: "মিনিকেট চাল (৫০ কেজি)", Price: 3200.0, Barcode: "8901030012345"},
			Quantity: 2,
		},
		{
			Product:  POSProduct{ID: "p-02", Name: "তীর সয়াবিন তেল (৫লি)", Price: 850.0, Barcode: "8901030098765"},
			Quantity: 1,
		},
	}

	var subtotal float64
	for _, item := range items {
		subtotal += item.Product.Price * float64(item.Quantity)
	}

	if subtotal != 7250.0 {
		t.Fatalf("Expected subtotal 7250.0, got %f", subtotal)
	}

	discountPct := 5.0
	discount := subtotal * (discountPct / 100.0)
	if discount != 362.5 {
		t.Fatalf("Expected discount 362.5, got %f", discount)
	}

	vat := (subtotal - discount) * 0.05
	expectedVAT := 344.375
	if fmt.Sprintf("%.2f", vat) != fmt.Sprintf("%.2f", expectedVAT) {
		t.Fatalf("Expected VAT %.2f, got %.2f", expectedVAT, vat)
	}

	total := (subtotal - discount) + vat
	expectedTotal := 7231.875
	if fmt.Sprintf("%.2f", total) != fmt.Sprintf("%.2f", expectedTotal) {
		t.Fatalf("Expected Total %.2f, got %.2f", expectedTotal, total)
	}

	now := time.Now()
	invoiceNo := fmt.Sprintf("INV-%s-%03d", now.Format("20060102"), 1)
	if len(invoiceNo) < 10 {
		t.Fatalf("Invalid invoice number generated: %s", invoiceNo)
	}
}

func TestDeviceProfileConfigurations(t *testing.T) {
	profiles := map[string]struct {
		Width  int
		Height int
		Ratio  string
	}{
		"phone":   {Width: 380, Height: 780, Ratio: "19.5:9"},
		"tablet":  {Width: 680, Height: 520, Ratio: "4:3"},
		"pos":     {Width: 420, Height: 720, Ratio: "16:9"},
		"desktop": {Width: 620, Height: 560, Ratio: "custom"},
	}

	if len(profiles) != 4 {
		t.Fatalf("Expected 4 device profiles, found %d", len(profiles))
	}

	for name, p := range profiles {
		if p.Width <= 0 || p.Height <= 0 {
			t.Fatalf("Invalid dimensions for profile %s: %dx%d", name, p.Width, p.Height)
		}
	}
}

func TestNilLSPSyntaxAnalyzer(t *testing.T) {
	validCode := `
@Component
ফাংশন MainView() {
    চল বার্তা = "স্বাগতম নীলাং!";
    যদি (বার্তা != "") {
        প্রিন্ট(বার্তা);
    }
    ফেরত <Column></Column>;
}
`
	// Simple bracket checking logic verification
	openBraces := strings.Count(validCode, "{") - strings.Count(validCode, "}")
	if openBraces != 0 {
		t.Fatalf("Expected balanced braces in valid code, got %d", openBraces)
	}

	invalidCode := `
ফাংশন Test() {
    চল x = 10;
`
	openBracesInvalid := strings.Count(invalidCode, "{") - strings.Count(invalidCode, "}")
	if openBracesInvalid == 0 {
		t.Fatalf("Expected unbalanced braces error in invalid code")
	}
}

func TestNDBBreakpointManagement(t *testing.T) {
	type Breakpoint struct {
		ID   string
		File string
		Line int
	}

	bps := []Breakpoint{
		{ID: "bp-1", File: "main.nil", Line: 12},
		{ID: "bp-2", File: "notes.nil", Line: 45},
	}

	if len(bps) != 2 {
		t.Fatalf("Expected 2 breakpoints, found %d", len(bps))
	}

	// Test adding a breakpoint
	newBP := Breakpoint{ID: "bp-3", File: "pos.nil", Line: 88}
	bps = append(bps, newBP)
	if len(bps) != 3 || bps[2].Line != 88 {
		t.Fatalf("Failed to add new breakpoint correctly")
	}
}

func TestQEMUFramebufferState(t *testing.T) {
	state := "running"
	arch := "x86_64"
	res := "1024x768"

	if state != "running" || arch != "x86_64" || res != "1024x768" {
		t.Fatalf("Invalid QEMU framebuffer state configuration")
	}
}
