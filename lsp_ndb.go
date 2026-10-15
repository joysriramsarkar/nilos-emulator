package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/joysriramsarkar/nilos-emulator/internal/lsp"
)

// ─── Nil-LSP Models ──────────────────────────────────────────────────

type LSPDiagnostic struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Message  string `json:"message"`
	Source   string `json:"source"`
}

type LSPCompletionItem struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"` // "keyword", "function", "variable", "component"
	Detail        string `json:"detail"`
	Documentation string `json:"documentation"`
	InsertText    string `json:"insertText"`
}

type LSPHoverInfo struct {
	Symbol      string   `json:"symbol"`
	Type        string   `json:"type"`
	Signature   string   `json:"signature"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
}

// ─── Nil-LSP API Handlers ────────────────────────────────────────────

func handleLSPAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code string `json:"code"`
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	diagnostics := []LSPDiagnostic{}
	for _, d := range lsp.Analyze(req.Code) {
		diagnostics = append(diagnostics, LSPDiagnostic{
			Line: d.Line, Column: d.Column, Severity: d.Severity, Message: d.Message, Source: d.Source,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"file":        req.File,
		"diagnostics": diagnostics,
		"errorsCount": len(diagnostics),
	})
}

func handleLSPComplete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	items := []LSPCompletionItem{
		{Label: "চল", Kind: "keyword", Detail: "let variable declaration", Documentation: "নীলাং-এ নতুন ভেরিয়েবল তৈরি করতে ব্যবহৃত হয়।", InsertText: "চল ${1:name} = ${2:value};"},
		{Label: "ফাংশন", Kind: "keyword", Detail: "function definition", Documentation: "নীলাং ফাংশন ডিক্লারেশন।", InsertText: "ফাংশন ${1:name}(${2:params}) {\n\t${3}\n}"},
		{Label: "যদি", Kind: "keyword", Detail: "if condition", Documentation: "শর্ত সাপেক্ষে কোড ব্লক এক্সিকিউশন।", InsertText: "যদি (${1:condition}) {\n\t${2}\n}"},
		{Label: "নাহলে", Kind: "keyword", Detail: "else block", Documentation: "যদি শর্ত মিথ্যা হয় তবে এই ব্লক চলবে।", InsertText: "নাহলে {\n\t${1}\n}"},
		{Label: "ফেরত", Kind: "keyword", Detail: "return statement", Documentation: "ফাংশন থেকে মান ফেরত পাঠাতে।", InsertText: "ফেরত ${1:value};"},
		{Label: "লুপ", Kind: "keyword", Detail: "for/while loop", Documentation: "পুনরাবৃত্তি লুপ ব্লক।", InsertText: "লুপ (${1:item} in ${2:list}) {\n\t${3}\n}"},
		{Label: "@Component", Kind: "component", Detail: "Alap UI Component", Documentation: "আলাপ ডিক্লারেটিভ ইউআই কম্পোনেন্ট ডেকোরেটর।", InsertText: "@Component\nফাংশন ${1:AppView}() {\n\tফেরত <Column>\n\t\t${2}\n\t</Column>;\n}"},
		{Label: "@State", Kind: "variable", Detail: "Reactive State Variable", Documentation: "কম্পোনেন্টের রিঅ্যাক্টিভ স্টেট ট্র্যাকার।", InsertText: "@State চল ${1:count} = ${2:0};"},
		{Label: "@Prop", Kind: "variable", Detail: "Component Property", Documentation: "প্যারেন্ট থেকে প্রাপ্ত প্রপার্টি।", InsertText: "@Prop চল ${1:title}: টেক্সট;"},
		{Label: "প্রিন্ট", Kind: "function", Detail: "print(message)", Documentation: "কনসোলে মেসেজ বা ডেটা আউটপুট দেখায়।", InsertText: "প্রিন্ট(${1:message});"},
		{Label: "নীলবাস.সেন্ড", Kind: "function", Detail: "NilBus.send(event, payload)", Documentation: "নীলবাস ডিস্ট্রিবিউটেড মেসেজ বাসে ইভেন্ট পোস্ট করে।", InsertText: "নীলবাস.সেন্ড(\"${1:service}\", \"${2:event}\", ${3:data});"},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": items,
	})
}

func handleLSPHover(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	sym := r.URL.Query().Get("symbol")
	if sym == "" {
		sym = "চল"
	}

	hoverDocs := map[string]LSPHoverInfo{
		"চল": {
			Symbol:      "চল",
			Type:        "কীওয়ার্ড (Keyword)",
			Signature:   "চল [নাম] = [মান]",
			Description: "নীলাং-এ নতুন ভেরিয়েবল ডিফাইন করার জন্য 'চল' ব্যবহার করা হয়। এটি জাভাস্ক্রিপ্টের 'let' বা রাস্টের 'let' এর সমতুল্য।",
			Examples:    []string{"চল বার্তা = \"স্বাগতম!\";", "চল সংখ্যা = ১০;"},
		},
		"ফাংশন": {
			Symbol:      "ফাংশন",
			Type:        "ফাংশন ডিক্লেয়ারেশন (Keyword)",
			Signature:   "ফাংশন [নাম]([প্যারামিটারসমূহ]) { ... }",
			Description: "নীলাং কোডব্লককে রিইউজেবল ফাংশন হিসেবে ডিক্লেয়ার করে।",
			Examples:    []string{"ফাংশন যোগ(ক, খ) {\n    ফেরত ক + খ;\n}"},
		},
		"@Component": {
			Symbol:      "@Component",
			Type:        "আলাপ ডেকোরেটর (Decorator)",
			Signature:   "@Component ফাংশন ViewName() { ... }",
			Description: "আলাপ ডিক্লারেটিভ ইউআই ইঞ্জিনে একটি ভিজ্যুয়াল উইজেট বা কম্পোনেন্ট তৈরি করে।",
			Examples:    []string{"@Component\nফাংশন NoteCard() { ফেরত <Card>...</Card>; }"},
		},
	}

	info, exists := hoverDocs[sym]
	if !exists {
		info = LSPHoverInfo{
			Symbol:      sym,
			Type:        "সিস্টেম সিম্বল (Identifier)",
			Signature:   fmt.Sprintf("চল %s: এনি", sym),
			Description: fmt.Sprintf("নীলাং রানটাইমে '%s' সিম্বল নিবন্ধিত আছে।", sym),
			Examples:    []string{fmt.Sprintf("প্রিন্ট(%s);", sym)},
		}
	}

	json.NewEncoder(w).Encode(info)
}
