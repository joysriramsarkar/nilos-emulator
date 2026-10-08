// Command nilsp is the NilLang Language Server. Editors connect over stdio and
// speak LSP (Content-Length framed JSON-RPC).
package main

import (
	"fmt"
	"os"

	"github.com/joysriramsarkar/nilos-emulator/internal/lsp"
)

func main() {
	srv := lsp.NewServer(os.Stdin, os.Stdout)
	if err := srv.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "nilsp:", err)
		os.Exit(1)
	}
}
