package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type IDEExtensionInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TargetIDE   string   `json:"targetIde"` // "VSCode", "AndroidStudio"
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	DownloadURL string   `json:"downloadUrl"`
	Icon        string   `json:"icon"`
	BundleFile  string   `json:"bundleFile"`
}

func GetAvailableExtensions() []IDEExtensionInfo {
	return []IDEExtensionInfo{
		{
			ID:          "nilang-vscode-pro",
			Name:        "NilLang & NilOS Studio for Visual Studio Code",
			TargetIDE:   "VS Code (v1.85+)",
			Version:     "2.0.0",
			Author:      "Joy Sriram Sarkar & Nilang Core Team",
			Description: "নীলাং ভাষা, আলাপ UI ফ্রেমওয়ার্ক, NDB ডিবাগার ও নীল ওএস এমুলেটর লাইভ প্রিভিউ এক্সটেনশন।",
			Features: []string{
				"সিনট্যাক্স হাইলাইটিং ও বাংলা কীওয়ার্ড সাপোর্ট",
				"Nil-LSP ইন্টিগ্রেশন (অটোকমপ্লিট ও লিন্টার)",
				"NDB রিমোট ডিবাগার প্রোটোকল",
				"নীল ওএস ডিভাইস এমুলেটর ১-ক্লিক লঞ্চার",
				"আলাপ ডিক্লারেটিভ UI স্নsnippetস",
			},
			DownloadURL: "/api/extensions/vscode/download",
			Icon:        "🔷",
			BundleFile:  "nilang-vscode-extension-2.0.0.zip",
		},
		{
			ID:          "nilang-android-studio",
			Name:        "NilOS Device Bridge for Android Studio & IntelliJ",
			TargetIDE:   "Android Studio / IntelliJ IDEA 2024+",
			Version:     "2.0.0",
			Author:      "Nilang Core Team",
			Description: "অ্যান্ড্রয়েড স্টুডিওতে সরাসরি নীল ওএস ডিভাইস এমুলেটর ইন্টিগ্রেশন, Logcat ও NilBus মনিটর।",
			Features: []string{
				"Android Studio Device Manager-এ NilPhone/NilPad এমুলেটর রেজিস্ট্রেশন",
				"NilBus IPC রিয়েল-টাইম লগিং",
				"Ed25519 সাইন্ড .nilax প্যাকেজ বিল্ডার",
				"Headless QEMU কার্নেল স্ট্রিম ভিউয়ার",
			},
			DownloadURL: "/api/extensions/android-studio/download",
			Icon:        "🤖",
			BundleFile:  "nilos-android-studio-plugin-2.0.0.zip",
		},
	}
}

func handleExtensionsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	exts := GetAvailableExtensions()
	json.NewEncoder(w).Encode(exts)
}

func handleVSCodeDownload(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	packageJSON := `{
  "name": "nilang-vscode-pro",
  "displayName": "NilLang & NilOS Studio Extension",
  "description": "Full language support, Nil-LSP, NDB debugger and live NilOS device emulator for VS Code.",
  "version": "2.0.0",
  "publisher": "joysriramsarkar",
  "engines": {
    "vscode": "^1.85.0"
  },
  "categories": ["Programming Languages", "Debuggers", "Linters", "Snippets"],
  "contributes": {
    "languages": [{
      "id": "nilang",
      "aliases": ["NilLang", "nil", "নীলাং"],
      "extensions": [".nil", ".nabc", ".nilax"],
      "configuration": "./language-configuration.json"
    }],
    "grammars": [{
      "language": "nilang",
      "scopeName": "source.nilang",
      "path": "./syntaxes/nilang.tmLanguage.json"
    }],
    "debuggers": [{
      "type": "ndb",
      "label": "NDB (Nil Debugger)",
      "program": "./bin/ndb-adapter.js"
    }],
    "commands": [
      {
        "command": "nilos.launchEmulator",
        "title": "NilOS: Launch Device Emulator Studio"
      },
      {
        "command": "nilos.connectNDB",
        "title": "NilOS: Connect NDB Remote Debugger"
      }
    ]
  },
  "scripts": {
    "vscode:prepublish": "npm run compile",
    "compile": "tsc -p ./"
  }
}`

	langConfig := `{
  "comments": {
    "lineComment": "//",
    "blockComment": ["/*", "*/"]
  },
  "brackets": [
    ["{", "}"],
    ["[", "]"],
    ["(", ")"]
  ],
  "autoClosingPairs": [
    { "open": "{", "close": "}" },
    { "open": "[", "close": "]" },
    { "open": "(", "close": ")" },
    { "open": "\"", "close": "\"" },
    { "open": "'", "close": "'" }
  ]
}`

	tmLanguage := `{
  "$schema": "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json",
  "name": "NilLang",
  "patterns": [
    {
      "name": "keyword.control.nilang",
      "match": "\\b(চল|ফাংশন|যদি|নাহলে|ফেরত|লুপ|import|export|class|let|fn|if|else|return|for|while)\\b"
    },
    {
      "name": "entity.name.type.decorator.nilang",
      "match": "@(Component|State|Prop|Watch|Computed)\\b"
    },
    {
      "name": "string.quoted.double.nilang",
      "begin": "\"",
      "end": "\"",
      "patterns": [{ "name": "constant.character.escape.nilang", "match": "\\\\." }]
    },
    {
      "name": "constant.numeric.nilang",
      "match": "\\b[0-9]+(\\.[0-9]+)?\\b"
    },
    {
      "name": "comment.line.double-slash.nilang",
      "match": "//.*$"
    }
  ],
  "scopeName": "source.nilang"
}`

	readmeMD := fmt.Sprintf(`# NilLang & NilOS Studio VS Code Extension (v2.0.0)

Official Visual Studio Code extension for NilLang and NilOS Studio Emulator.

## Installation
1. Open VS Code.
2. Press 'Ctrl+Shift+P' and select 'Extensions: Install from VSIX...'.
3. Select this bundle package.

## Features
- Full syntax highlighting for Bengali and English keywords ('চল', 'ফাংশন', 'যদি', 'নাহলে', '@Component', '@State')
- Nil-LSP connection to http://localhost:8085/api/lsp
- NDB remote debugging with breakpoints and call stack
- Live NilOS Emulator integration

Generated: %s
`, time.Now().Format(time.RFC1123))

	files := map[string]string{
		"package.json":                   packageJSON,
		"language-configuration.json":    langConfig,
		"syntaxes/nilang.tmLanguage.json": tmLanguage,
		"README.md":                      readmeMD,
	}

	for path, content := range files {
		f, err := zw.Create(path)
		if err == nil {
			f.Write([]byte(content))
		}
	}
	zw.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"nilang-vscode-extension-2.0.0.zip\"")
	w.Write(buf.Bytes())
}

func handleAndroidStudioDownload(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	pluginXML := `<?xml version="1.0" encoding="UTF-8"?>
<idea-plugin>
    <id>org.nilang.nilos.studio</id>
    <name>NilOS Device Bridge &amp; NilLang Studio</name>
    <version>2.0.0</version>
    <vendor email="core@nilang.org" url="https://nilang.org">Nilang Core Team</vendor>

    <description><![CDATA[
      Android Studio & IntelliJ IDEA plugin for NilOS Virtual Device Emulator and NilBus debugging.
    ]]></description>

    <depends>com.intellij.modules.platform</depends>

    <extensions defaultExtensionNs="com.intellij">
        <fileType name="NilLang File" implementationClass="org.nilang.plugin.NilFileType" fieldName="INSTANCE" language="NilLang" extensions="nil;nabc"/>
        <syntaxHighlighter key="NilLang" implementationClass="org.nilang.plugin.NilSyntaxHighlighter"/>
        <toolWindow id="NilOS Emulator" secondary="true" icon="/icons/nilos.svg" anchor="right" factoryClass="org.nilang.plugin.NilOSToolWindowFactory"/>
    </extensions>

    <actions>
        <action id="NilOS.LaunchEmulator" class="org.nilang.plugin.LaunchEmulatorAction" text="Launch NilOS Device Emulator" description="Launch NilOS 2.0 Studio Emulator">
            <add-to-group group-id="ToolsMenu" anchor="last"/>
        </action>
    </actions>
</idea-plugin>`

	readmeMD := fmt.Sprintf(`# NilOS Device Bridge Plugin for Android Studio & IntelliJ (v2.0.0)

Official Android Studio / IntelliJ IDEA plugin for NilOS Virtual Device integration.

## Installation
1. Open Android Studio.
2. Go to 'Settings' -> 'Plugins' -> ⚙️ -> 'Install Plugin from Disk...'.
3. Select this ZIP file and restart IDE.

## Features
- Direct NilPhone & NilPad integration in Android Studio Device Manager
- Real-time NilBus IPC Logcat stream
- One-click launch of NilOS 2.0 Studio Emulator (http://localhost:8085/device.html)

Generated: %s
`, time.Now().Format(time.RFC1123))

	files := map[string]string{
		"META-INF/plugin.xml": pluginXML,
		"README.md":           readmeMD,
	}

	for path, content := range files {
		f, err := zw.Create(path)
		if err == nil {
			f.Write([]byte(content))
		}
	}
	zw.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"nilos-android-studio-plugin-2.0.0.zip\"")
	w.Write(buf.Bytes())
}
