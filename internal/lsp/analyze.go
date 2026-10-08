// Package lsp implements a Language Server Protocol server for NilLang. It is
// shared by the `nilsp` binary (editor integration via stdio) and the HTTP
// endpoints used by the Studio.
package lsp

import (
	"fmt"
	"strings"
)

// Diagnostic is one compile/lint problem.
type Diagnostic struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	EndCol   int    `json:"endColumn,omitempty"`
	Severity string `json:"severity"` // error | warning | info | hint
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// CompletionItem is an autocomplete suggestion.
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          string `json:"kind"`
	Detail        string `json:"detail"`
	Documentation string `json:"documentation"`
	InsertText    string `json:"insertText"`
}

// HoverInfo describes a symbol.
type HoverInfo struct {
	Symbol      string   `json:"symbol"`
	Type        string   `json:"type"`
	Signature   string   `json:"signature"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
}

// Analyze runs the NilLang linter.
func Analyze(code string) []Diagnostic {
	diags := []Diagnostic{}
	lines := strings.Split(code, "\n")
	openBraces := 0
	openParens := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := i + 1
		openBraces += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
		openParens += strings.Count(trimmed, "(") - strings.Count(trimmed, ")")

		if strings.HasPrefix(trimmed, "চল ") && !strings.Contains(trimmed, "=") {
			diags = append(diags, Diagnostic{
				Line: lineNum, Column: len(trimmed), Severity: "warning",
				Message: "চল (let) স্টেটমেন্টে ইনিশিয়াল ভ্যালু দেওয়া সুপারিশকৃত।", Source: "nil-lsp",
			})
		}
		if strings.HasPrefix(trimmed, "ফাংশন ") && !strings.Contains(trimmed, "(") {
			diags = append(diags, Diagnostic{
				Line: lineNum, Column: len(trimmed), Severity: "error",
				Message: "ফাংশন ডিক্লেয়ারেশনে প্যারামিটার ব্র্যাকেট () আবশ্যক।", Source: "nil-lsp",
			})
		}
	}
	if openBraces != 0 {
		diags = append(diags, Diagnostic{
			Line: len(lines), Column: 1, Severity: "error",
			Message: fmt.Sprintf("ব্র্যাকেট ম্যাচিং অসমাপ্ত (আনক্লোজড ব্র্যাকেট: %d)", openBraces), Source: "nil-lsp",
		})
	}
	if openParens != 0 {
		diags = append(diags, Diagnostic{
			Line: len(lines), Column: 1, Severity: "error",
			Message: fmt.Sprintf("প্যারেন্থেসিস ম্যাচিং অসমাপ্ত (আনক্লোজড: %d)", openParens), Source: "nil-lsp",
		})
	}
	return diags
}

// Completions returns the built-in suggestion list.
func Completions() []CompletionItem {
	return []CompletionItem{
		{Label: "চল", Kind: "keyword", Detail: "let variable declaration", Documentation: "নীলাং-এ নতুন ভেরিয়েবল তৈরি করতে ব্যবহৃত হয়।", InsertText: "চল ${1:name} = ${2:value};"},
		{Label: "ফাংশন", Kind: "keyword", Detail: "function definition", Documentation: "নীলাং ফাংশন ডিক্লারেশন।", InsertText: "ফাংশন ${1:name}(${2:params}) {\n\t${3}\n}"},
		{Label: "যদি", Kind: "keyword", Detail: "if condition", Documentation: "শর্ত সাপেক্ষে কোড ব্লক এক্সিকিউশন।", InsertText: "যদি (${1:condition}) {\n\t${2}\n}"},
		{Label: "নাহলে", Kind: "keyword", Detail: "else block", Documentation: "যদি শর্ত মিথ্যা হয় তবে এই ব্লক চলবে।", InsertText: "নাহলে {\n\t${1}\n}"},
		{Label: "ফেরত", Kind: "keyword", Detail: "return statement", Documentation: "ফাংশন থেকে মান ফেরত পাঠাতে।", InsertText: "ফেরত ${1:value};"},
		{Label: "লুপ", Kind: "keyword", Detail: "for/while loop", Documentation: "পুনরাবৃত্তি লুপ ব্লক।", InsertText: "লুপ (${1:item} in ${2:list}) {\n\t${3}\n}"},
		{Label: "@Component", Kind: "component", Detail: "Alap UI Component", Documentation: "আলাপ ডিক্লারেটিভ ইউআই কম্পোনেন্ট ডেকোরেটর।", InsertText: "@Component\nফাংশন ${1:AppView}() {\n\tফেরত <Column>\n\t\t${2}\n\t</Column>;\n}"},
		{Label: "@State", Kind: "variable", Detail: "Reactive State Variable", Documentation: "কম্পোনেন্টের রিঅ্যাক্টিভ স্টেট ট্র্যাকার।", InsertText: "@State চল ${1:count} = ${2:0};"},
		{Label: "@Prop", Kind: "variable", Detail: "Component Property", Documentation: "প্যারেন্ট থেকে প্রাপ্ত প্রপার্টি।", InsertText: "@Prop চল ${1:title}: টেক্সট;"},
		{Label: "প্রিন্ট", Kind: "function", Detail: "print(message)", Documentation: "কনসোলে মেসেজ বা ডেটা আউটপুট দেখায়।", InsertText: "প্রিন্ট(${1:message});"},
		{Label: "নীলবাস.সেন্ড", Kind: "function", Detail: "NilBus.send(event, payload)", Documentation: "নীলবাস মেসেজ বাসে ইভেন্ট পোস্ট করে।", InsertText: "নীলবাস.সেন্ড(\"${1:service}\", \"${2:event}\", ${3:data});"},
	}
}

// Hover returns documentation for a symbol.
func Hover(symbol string) HoverInfo {
	if symbol == "" {
		symbol = "চল"
	}
	docs := map[string]HoverInfo{
		"চল": {Symbol: "চল", Type: "কীওয়ার্ড (Keyword)", Signature: "চল [নাম] = [মান]", Description: "নীলাং-এ নতুন ভেরিয়েবল ডিফাইন করার জন্য 'চল' ব্যবহার করা হয়।", Examples: []string{`চল বার্তা = "স্বাগতম!";`, "চল সংখ্যা = ১০;"}},
		"ফাংশন": {Symbol: "ফাংশন", Type: "ফাংশন ডিক্লেয়ারেশন (Keyword)", Signature: "ফাংশন [নাম]([প্যারামিটারসমূহ]) { ... }", Description: "নীলাং কোডব্লককে রিইউজেবল ফাংশন হিসেবে ডিক্লেয়ার করে।", Examples: []string{"ফাংশন যোগ(ক, খ) {\n    ফেরত ক + খ;\n}"}},
		"@Component": {Symbol: "@Component", Type: "আলাপ ডেকোরেটর (Decorator)", Signature: "@Component ফাংশন ViewName() { ... }", Description: "আলাপ ডিক্লারেটিভ ইউআই ইঞ্জিনে একটি ভিজ্যুয়াল কম্পোনেন্ট তৈরি করে।", Examples: []string{"@Component\nফাংশন NoteCard() { ফেরত <Card>...</Card>; }"}},
	}
	if info, ok := docs[symbol]; ok {
		return info
	}
	return HoverInfo{
		Symbol: symbol, Type: "সিস্টেম সিম্বল (Identifier)",
		Signature:   fmt.Sprintf("চল %s: এনি", symbol),
		Description: fmt.Sprintf("নীলাং রানটাইমে '%s' সিম্বল নিবন্ধিত আছে।", symbol),
		Examples:    []string{fmt.Sprintf("প্রিন্ট(%s);", symbol)},
	}
}
