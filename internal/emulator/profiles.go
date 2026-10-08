package emulator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// DefaultProfiles is the built-in set of device templates. Additional
// profiles can be dropped as JSON files into <dataDir>/profiles/.
var DefaultProfiles = map[string]DeviceProfile{
	"nilphone-ultra": {
		ID:          "nilphone-ultra",
		DisplayName: `NilPhone Ultra (6.7")`,
		Category:    "phone",
		Screen: ScreenSpec{
			WidthPx:        1080,
			HeightPx:       2400,
			Density:        3.0,
			RefreshHz:      120,
			DynamicIsland:  true,
			RoundedCorners: 48,
			Orientation:    "portrait",
			SafeArea:       Insets{Top: 48, Right: 24, Bottom: 34, Left: 24},
		},
		Input: InputSpec{Touch: true, Keyboard: true, Mouse: false, VirtualKeys: true},
		Sensors: SensorSpec{
			Accelerometer: true, Gyroscope: true, Proximity: true,
			AmbientLight: true, GPS: true, Fingerprint: true,
		},
		Battery: BatterySpec{CapacityMah: 5000, MaxCharge: 100, Health: 100},
		Features: []string{
			"dynamic-island", "bangla-virtual-keyboard", "multi-window-phone",
		},
	},
	"nilpad-pro": {
		ID:          "nilpad-pro",
		DisplayName: `NilPad Pro (11.0")`,
		Category:    "tablet",
		Screen: ScreenSpec{
			WidthPx:        1600,
			HeightPx:       2560,
			Density:        2.0,
			RefreshHz:      120,
			RoundedCorners: 32,
			Orientation:    "portrait",
			SafeArea:       Insets{Top: 24, Right: 24, Bottom: 24, Left: 24},
		},
		Input:   InputSpec{Touch: true, Keyboard: true, Mouse: true},
		Sensors: SensorSpec{Accelerometer: true, Gyroscope: true, GPS: true},
		Battery: BatterySpec{CapacityMah: 8000, MaxCharge: 100, Health: 100},
		Features: []string{
			"split-mode", "tablet-productivity", "external-keyboard",
		},
	},
	"nilpos-terminal": {
		ID:          "nilpos-terminal",
		DisplayName: "NilPOS Smart Terminal",
		Category:    "pos",
		Screen: ScreenSpec{
			WidthPx:        800,
			HeightPx:       1280,
			Density:        1.5,
			RefreshHz:      60,
			RoundedCorners: 12,
			Orientation:    "portrait",
		},
		Input:   InputSpec{Touch: true, Keyboard: true, BarcodeScanner: true, ThermalPrinter: true},
		Sensors: SensorSpec{},
		Battery: BatterySpec{CapacityMah: 3000, MaxCharge: 100, Health: 100},
		Features: []string{
			"live-laser-barcode-scanner", "animated-thermal-receipt-printer", "retail-mode",
		},
	},
	"nilos-desktop": {
		ID:          "nilos-desktop",
		DisplayName: "NilOS Desktop",
		Category:    "desktop",
		Screen: ScreenSpec{
			WidthPx:        1920,
			HeightPx:       1080,
			Density:        1.0,
			RefreshHz:      60,
			RoundedCorners: 0,
			Orientation:    "landscape",
		},
		Input:   InputSpec{Touch: false, Keyboard: true, Mouse: true},
		Sensors: SensorSpec{},
		Battery: BatterySpec{CapacityMah: 0, MaxCharge: 100, Health: 100},
		Features: []string{
			"floating-multi-window", "desktop-workspace", "taskbar",
		},
	},
}

// LoadProfiles merges built-in profiles with any *.json files found in dir.
func LoadProfiles(dir string) map[string]DeviceProfile {
	out := make(map[string]DeviceProfile, len(DefaultProfiles))
	for k, v := range DefaultProfiles {
		out[k] = v
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var p DeviceProfile
		if err := json.Unmarshal(data, &p); err != nil || p.ID == "" {
			continue
		}
		out[p.ID] = p
	}
	return out
}

// SortedProfileIDs returns profile ids in a stable order.
func SortedProfileIDs(profiles map[string]DeviceProfile) []string {
	ids := make([]string, 0, len(profiles))
	for id := range profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
