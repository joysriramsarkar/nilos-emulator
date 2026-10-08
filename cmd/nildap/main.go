// Command nildap is the NilOS Debug Adapter (DAP). Editors launch it and speak
// the Debug Adapter Protocol over stdio.
//
// Modes:
//
//	nildap --exec "nil dap path/to/main.nil"   # proxy to the real NilLang VM debugger
//	nildap --nil-bin <nil> --program main.nil  # shortcut for the above
//	nildap                                      # standalone mock program
//	nildap --device <id> --app <id>             # attach to a running emulator app
//
// The real NilLang VM (github.com/joysriramsarkar/nilLang) implements the VM
// debug hooks and the `nil dap` server; this adapter fronts it so a single
// configuration works for editors and the Studio.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/joysriramsarkar/nilos-emulator/internal/dap"
)

func main() {
	device := flag.String("device", "", "device id to attach to")
	app := flag.String("app", "", "app id to attach to")
	server := flag.String("server", envOr("NILEMU_SERVER", "http://127.0.0.1:8085"), "control API base URL")
	execCmd := flag.String("exec", "", "proxy DAP to this command's stdio (e.g. \"nil dap main.nil\")")
	nilBin := flag.String("nil-bin", envOr("NIL_BIN", "nil"), "path to the nil binary")
	program := flag.String("program", "", "NilLang source file to debug with the real VM")
	flag.Parse()

	// Real-VM proxy mode.
	cmdLine := *execCmd
	if cmdLine == "" && *program != "" {
		cmdLine = dapCommandLine(*nilBin, *program)
	}
	if cmdLine != "" {
		if err := runProxy(cmdLine, os.Stdin, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, "nildap:", err)
			os.Exit(1)
		}
		return
	}

	var backend dap.Backend
	if *device != "" && *app != "" {
		backend = dap.NewHTTPBackend(*server, *device, *app)
	} else {
		backend = dap.NewMockBackend()
	}

	srv := dap.NewServer(os.Stdin, os.Stdout, backend)
	if err := srv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "nildap:", err)
		os.Exit(1)
	}
}

// dapCommandLine builds the command that runs the real NilLang VM debugger.
func dapCommandLine(nilBin, program string) string {
	if strings.ContainsAny(nilBin, " \t") {
		nilBin = `"` + nilBin + `"`
	}
	if strings.ContainsAny(program, " \t") {
		program = `"` + program + `"`
	}
	return fmt.Sprintf("%s dap %s", nilBin, program)
}

// runProxy runs a DAP command with our streams attached, making nildap a
// transparent bridge to the real NilLang debugger. The child's exit code is
// propagated unchanged.
func runProxy(cmdLine string, in io.Reader, out, errOut io.Writer) error {
	parts := splitCommandLine(cmdLine)
	if len(parts) == 0 {
		return fmt.Errorf("empty --exec command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}

// splitCommandLine splits on whitespace but honours double quotes so paths with
// spaces (very common on Windows) survive intact.
func splitCommandLine(s string) []string {
	var (
		parts []string
		cur   strings.Builder
		inQ   bool
	)
	for _, r := range s {
		switch {
		case r == '"':
			inQ = !inQ
		case (r == ' ' || r == '\t' || r == '\n') && !inQ:
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
