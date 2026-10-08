package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/joysriramsarkar/nilos-emulator/internal/textutil"
)

// Node is a serializable UI tree node.
type Node map[string]any

// N builds a UI node.
func N(id, typ string, props map[string]any, children ...Node) Node {
	if props == nil {
		props = map[string]any{}
	}
	n := Node{"id": id, "type": typ, "props": props}
	if len(children) > 0 {
		n["children"] = children
	}
	return n
}

// MockVM is a self-contained NilLang app implementation used when the real
// `nil serve` runtime is unavailable. It speaks the full NilHost Protocol so
// the emulator, sessions, patching, lifecycle and HAL all work end to end.
type MockVM struct {
	app      AppInfo
	enc      *json.Encoder
	seq      uint64
	nextH    int
	handlers map[int]func() Node // returns patch-producing action
	patchFor map[int]func() []map[string]any

	counter int
	text    string
	theme   string
	items   []string
	display string
	calcAcc []string
	cart    []string
	playing string
	battery any
}

// RunMockVM runs the mock VM over the given streams until the app is
// destroyed or the input closes.
func RunMockVM(app AppInfo, in io.Reader, out io.Writer) error {
	vm := &MockVM{
		app:      app,
		enc:      json.NewEncoder(out),
		handlers: map[int]func() Node{},
		patchFor: map[int]func() []map[string]any{},
		theme:    "dark",
		display:  "0",
	}
	return vm.run(in)
}

func (vm *MockVM) send(v any) error { return vm.enc.Encode(v) }

func (vm *MockVM) run(in io.Reader) error {
	if err := vm.send(Hello{
		Type:    "hello",
		Proto:   ProtoVersion,
		Runtime: "nilemu-mockvm/1.0",
		App:     vm.app.ID,
		Caps:    []string{"ui.tree.patch", "hal.v1", "lifecycle"},
	}); err != nil {
		return err
	}

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if !validLine(line) {
			continue
		}
		var env Envelope
		if json.Unmarshal(line, &env) != nil {
			continue
		}
		switch env.Type {
		case "init":
			vm.sendFrame()
			vm.send(HalCallMsg{Type: "hal.call", ID: 1, Dev: "battery", Op: "level"})
		case "input":
			var msg InputMsg
			if json.Unmarshal(line, &msg) == nil {
				vm.handleInput(msg)
			}
		case "ime":
			var msg ImeMsg
			if json.Unmarshal(line, &msg) == nil {
				vm.handleIME(msg)
			}
		case "lifecycle":
			var msg LifecycleMsg
			if json.Unmarshal(line, &msg) == nil {
				if msg.State == LifecyclePaused || msg.State == LifecycleStopped {
					_ = vm.send(StateMsg{Type: "state", Blob: vm.stateBlob()})
				}
				if msg.State == LifecycleDestroyed || msg.State == LifecycleStopped {
					_ = vm.send(ExitMsg{Type: "exit", Code: 0, Reason: "lifecycle " + msg.State})
					return nil
				}
				if msg.State == LifecycleResumed || msg.State == LifecycleStarted {
					vm.sendFrame()
				}
			}
		case "reload":
			vm.counter = 0
			vm.text = ""
			vm.cart = nil
			vm.display = "0"
			vm.calcAcc = nil
			vm.sendFrame()
		case "config":
			var msg ConfigMsg
			if json.Unmarshal(line, &msg) == nil {
				if msg.Theme != "" {
					vm.theme = msg.Theme
				}
				vm.sendFrame()
			}
		case "hal.result":
			var res HalResultMsg
			if json.Unmarshal(line, &res) == nil && res.OK {
				var v any
				if json.Unmarshal(res.Data, &v) == nil {
					vm.battery = v
					vm.sendFrame()
				}
			}
		}
	}
	return nil
}

func (vm *MockVM) stateBlob() json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"counter": vm.counter,
		"text":    vm.text,
		"items":   vm.items,
		"cart":    vm.cart,
	})
	return b
}

func (vm *MockVM) handleInput(msg InputMsg) {
	switch msg.Kind {
	case "tap", "longpress":
		if msg.Code != "" {
			if id := atoiSafe(strings.TrimPrefix(msg.Code, "h:")); id > 0 {
				if fn, ok := vm.handlers[id]; ok {
					fn()
				}
			}
		}
	case "text":
		vm.text += msg.Text
		vm.sendFrame()
	case "back":
		vm.sendFrame()
	}
}

func (vm *MockVM) handleIME(msg ImeMsg) {
	switch msg.Op {
	case "commit":
		vm.text += msg.Text
	case "compose":
		// Draft/underlined composition; the field shows it as-is.
		vm.text += msg.Text
	case "delete":
		// Backspace is grapheme-cluster aware: "কি" deletes as one unit.
		n := msg.N
		if n <= 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			vm.text = textutil.DeleteLastCluster(vm.text)
		}
	case "cursorPrev":
		// the runtime owns the cursor; nothing to do in the mock
	case "show", "hide":
	}
	vm.sendFrame()
}

func (vm *MockVM) sendFrame() {
	vm.seq++
	tree := vm.buildTree()
	_ = vm.send(FrameMsg{Type: "frame", Seq: vm.seq, Tree: mustJSON(tree)})
}

func (vm *MockVM) sendPatch(ops []map[string]any) {
	vm.seq++
	if len(ops) == 0 {
		return
	}
	_ = vm.send(PatchMsg{Type: "patch", Seq: vm.seq, Ops: mustJSON(ops)})
}

func (vm *MockVM) h(fn func() Node) int {
	vm.nextH++
	vm.handlers[vm.nextH] = fn
	return vm.nextH
}

func (vm *MockVM) buildTree() Node {
	var children []Node
	children = append(children, vm.header())

	switch {
	case strings.Contains(vm.app.ID, "calc") || strings.Contains(vm.app.Name, "ক্যাল"):
		children = append(children, vm.calculator()...)
	case strings.Contains(vm.app.ID, "pos") || strings.Contains(vm.app.Name, "পিওএস"):
		children = append(children, vm.posApp()...)
	case strings.Contains(vm.app.ID, "music") || strings.Contains(vm.app.Name, "মিউজিক"):
		children = append(children, vm.musicApp()...)
	case strings.Contains(vm.app.ID, "notes") || strings.Contains(vm.app.Name, "নোট"):
		children = append(children, vm.notesApp()...)
	default:
		children = append(children, vm.counterApp()...)
	}
	children = append(children, vm.statusBar())
	return N("root", "Column", map[string]any{"gap": 12, "padding": 16, "theme": vm.theme}, children...)
}

func (vm *MockVM) header() Node {
	return N("header", "Header", map[string]any{
		"title":    vm.app.Name,
		"subtitle": vm.app.Version,
		"theme":    vm.theme,
	})
}

func (vm *MockVM) statusBar() Node {
	bat := "—"
	if vm.battery != nil {
		bat = fmt.Sprintf("%v%%", vm.battery)
	}
	return N("status", "StatusBar", map[string]any{
		"battery": bat,
		"network": "wifi",
		"theme":   vm.theme,
	})
}

func (vm *MockVM) counterApp() []Node {
	incID := vm.h(func() Node {
		vm.counter += 1
		vm.sendPatch([]map[string]any{{"op": "set", "id": "counter", "prop": "text", "value": fmt.Sprintf("সংখ্যা: %d", vm.counter)}})
		return vm.buildTree()
	})
	resetID := vm.h(func() Node {
		vm.counter = 0
		vm.sendPatch([]map[string]any{{"op": "set", "id": "counter", "prop": "text", "value": "সংখ্যা: 0"}})
		return vm.buildTree()
	})
	return []Node{
		N("counter", "Text", map[string]any{"text": fmt.Sprintf("সংখ্যা: %d", vm.counter), "style": "title"}),
		N("row1", "Row", map[string]any{"gap": 8},
			N("btn.inc", "Button", map[string]any{"label": "বাড়াও", "onTap": fmt.Sprintf("h:%d", incID)}),
			N("btn.reset", "Button", map[string]any{"label": "রিসেট", "onTap": fmt.Sprintf("h:%d", resetID)}),
		),
	}
}

func (vm *MockVM) calculator() []Node {
	press := func(label string) int {
		return vm.h(func() Node {
			vm.calcPress(label)
			return vm.buildTree()
		})
	}
	grid := []string{"7", "8", "9", "÷", "4", "5", "6", "×", "1", "2", "3", "-", "0", ".", "=", "+"}
	var rows []Node
	for i := 0; i < len(grid); i += 4 {
		var btns []Node
		for j := i; j < i+4 && j < len(grid); j++ {
			label := grid[j]
			btns = append(btns, N("calc."+label, "Button", map[string]any{
				"label": label, "onTap": fmt.Sprintf("h:%d", press(label)),
			}))
		}
		rows = append(rows, N(fmt.Sprintf("calc.row.%d", i/4), "Row", map[string]any{"gap": 8}, btns...))
	}
	top := []Node{
		N("calc.display", "Text", map[string]any{"text": vm.display, "style": "display"}),
		N("calc.clear", "Button", map[string]any{"label": "C", "onTap": fmt.Sprintf("h:%d", press("C"))}),
	}
	return append(top, rows...)
}

func (vm *MockVM) calcPress(label string) {
	switch label {
	case "C":
		vm.display = "0"
		vm.calcAcc = nil
	case "=":
		vm.display = vm.eval()
	default:
		if vm.display == "0" {
			vm.display = label
		} else {
			vm.display += label
		}
	}
}

func (vm *MockVM) eval() string {
	expr := strings.NewReplacer("×", "*", "÷", "/").Replace(vm.display)
	var a, b float64
	var op string
	for _, ch := range expr {
		switch ch {
		case '+', '-', '*', '/':
			op = string(ch)
		}
	}
	_ = a
	_ = b
	_ = op
	// Extremely small evaluator: only a+b / a-b / a*b / a/b.
	var cur, acc float64
	var pending byte
	has := false
	flush := func() {
		if !has {
			acc = cur
			has = true
			return
		}
		switch pending {
		case '+':
			acc += cur
		case '-':
			acc -= cur
		case '*':
			acc *= cur
		case '/':
			if cur != 0 {
				acc /= cur
			}
		}
	}
	num := ""
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c >= '0' && c <= '9' || c == '.' {
			num += string(c)
			continue
		}
		if num == "" {
			continue
		}
		cur = parseFloat(num)
		num = ""
		if pending == 0 {
			acc = cur
			has = true
		} else {
			flush()
		}
		if c == '+' || c == '-' || c == '*' || c == '/' {
			pending = c
		}
	}
	if num != "" {
		cur = parseFloat(num)
		if pending != 0 {
			flush()
		} else {
			acc = cur
		}
	}
	return trimFloat(acc)
}

func (vm *MockVM) notesApp() []Node {
	addID := vm.h(func() Node {
		vm.items = append(vm.items, fmt.Sprintf("নোট %d: %s", len(vm.items)+1, vm.text))
		vm.text = ""
		vm.sendFrame()
		return vm.buildTree()
	})
	var list []Node
	for i, it := range vm.items {
		list = append(list, N(fmt.Sprintf("note.%d", i), "Text", map[string]any{"text": it, "style": "body"}))
	}
	if len(list) == 0 {
		list = append(list, N("note.empty", "Text", map[string]any{"text": "কোনো নোট নেই", "style": "muted"}))
	}
	return []Node{
		N("note.input", "TextField", map[string]any{"text": vm.text, "placeholder": "নতুন নোট লিখুন..."}),
		N("btn.add", "Button", map[string]any{"label": "নোট যোগ করুন", "onTap": fmt.Sprintf("h:%d", addID)}),
		N("note.list", "Column", map[string]any{"gap": 6}, list...),
	}
}

func (vm *MockVM) posApp() []Node {
	products := []string{"মিনিকেট চাল ৫০কেজি", "সয়াবিন তেল ৫লি", "চিনি ১কেজি", "লবণ ১কেজি"}
	var list []Node
	for i, p := range products {
		idx := i
		name := p
		addID := vm.h(func() Node {
			vm.cart = append(vm.cart, name)
			vm.sendPatch([]map[string]any{{"op": "set", "id": "pos.cartCount", "prop": "text", "value": fmt.Sprintf("কার্ট: %d আইটেম", len(vm.cart))}})
			return vm.buildTree()
		})
		list = append(list, N(fmt.Sprintf("pos.item.%d", idx), "Row", map[string]any{"gap": 8},
			N(fmt.Sprintf("pos.name.%d", idx), "Text", map[string]any{"text": name}),
			N(fmt.Sprintf("pos.add.%d", idx), "Button", map[string]any{"label": "যোগ", "onTap": fmt.Sprintf("h:%d", addID)}),
		))
	}
	checkoutID := vm.h(func() Node {
		vm.send(HalCallMsg{Type: "hal.call", ID: 100, Dev: "printer", Op: "print",
			Args: mustJSON(map[string]any{"lines": vm.cart})})
		vm.cart = nil
		vm.sendFrame()
		return vm.buildTree()
	})
	return []Node{
		N("pos.cartCount", "Text", map[string]any{"text": fmt.Sprintf("কার্ট: %d আইটেম", len(vm.cart)), "style": "title"}),
		N("pos.list", "Column", map[string]any{"gap": 6}, list...),
		N("pos.checkout", "Button", map[string]any{"label": "চেকআউট ও প্রিন্ট", "onTap": fmt.Sprintf("h:%d", checkoutID)}),
	}
}

func (vm *MockVM) musicApp() []Node {
	tracks := []string{"আমার সোনার বাংলা", "গ্রাম ছাড়া ওই রাঙা মাটির পথ", "মেঘের কোলে রোদ হেসেছে"}
	var list []Node
	for i, t := range tracks {
		name := t
		playID := vm.h(func() Node {
			vm.playing = name
			vm.sendPatch([]map[string]any{{"op": "set", "id": "music.now", "prop": "text", "value": "চলছে: " + name}})
			return vm.buildTree()
		})
		list = append(list, N(fmt.Sprintf("music.track.%d", i), "Row", map[string]any{"gap": 8},
			N(fmt.Sprintf("music.title.%d", i), "Text", map[string]any{"text": t}),
			N(fmt.Sprintf("music.play.%d", i), "Button", map[string]any{"label": "▶", "onTap": fmt.Sprintf("h:%d", playID)}),
		))
	}
	now := "চলছে: —"
	if vm.playing != "" {
		now = "চলছে: " + vm.playing
	}
	return []Node{
		N("music.now", "Text", map[string]any{"text": now, "style": "title"}),
		N("music.list", "Column", map[string]any{"gap": 6}, list...),
	}
}

// helpers -----------------------------------------------------------------

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func parseFloat(s string) float64 {
	var f float64
	var frac float64 = 1
	seenDot := false
	for _, r := range s {
		if r == '.' {
			seenDot = true
			continue
		}
		if r < '0' || r > '9' {
			continue
		}
		if seenDot {
			frac /= 10
			f += float64(r-'0') * frac
		} else {
			f = f*10 + float64(r-'0')
		}
	}
	return f
}

func trimFloat(f float64) string {
	s := fmt.Sprintf("%.6f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// ensure time import is used (handshake timestamps are handled in session).
var _ = time.Now
