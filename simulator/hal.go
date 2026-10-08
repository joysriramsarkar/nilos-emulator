package main

import (
	"fmt"
	"time"
)

var pBeep = kernel32.NewProc("Beep")

func PlayNativeSound(freq, durationMs uint32) {
	go func() {
		pBeep.Call(uintptr(freq), uintptr(durationMs))
	}()
}

func PlayClickTone() {
	PlayNativeSound(800, 20)
}

func PlayShutterTone() {
	PlayNativeSound(1400, 40)
}

func PlayLockTone() {
	PlayNativeSound(400, 50)
}

func PlayUnlockTone() {
	PlayNativeSound(950, 60)
}

func PlayScanTone() {
	PlayNativeSound(1800, 80)
}

// SimulateThermalReceipt creates a real receipt simulation
func (s *AppState) SimulateThermalReceipt(title string, items []string, total float64) string {
	now := time.Now()
	receipt := fmt.Sprintf(
		"========================================\n"+
			"       🏪 %s\n"+
			"       অনুরণ রিটেইল পিওএস টার্মিনাল\n"+
			"  ইনভয়েস: INV-%s-%03d\n"+
			"  তারিখ: %s   সময়: %s\n"+
			"----------------------------------------\n",
		title,
		now.Format("20060102"),
		now.Unix()%1000,
		now.Format("02/01/2006"),
		now.Format("03:04:05 PM"),
	)

	for _, itm := range items {
		receipt += fmt.Sprintf("  • %-26s\n", itm)
	}

	receipt += fmt.Sprintf(
		"----------------------------------------\n"+
			"  সর্বমোট:              ৳ %.2f\n"+
			"  ভ্যাট (৫%%):           ৳ %.2f\n"+
			"  পরিশোধ:               ৳ %.2f [নগদ]\n"+
			"========================================\n"+
			"        ধন্যবাদ! আবার আসবেন।\n"+
			"       ||| |||| || ||||| |||||\n",
		total, total*0.05, total*1.05,
	)

	s.HAL.PrintedReceipts = append(s.HAL.PrintedReceipts, receipt)
	s.SetStatus("🖨️ রিসিট প্রিন্ট সম্পন্ন")
	s.AddIPCLog("NilHAL.Printer", "ESC/POS print job success (284 bytes)")
	PlayScanTone()
	return receipt
}
