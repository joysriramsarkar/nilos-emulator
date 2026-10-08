// Command nilosd is the NilLang runtime worker. The emulator may launch one
// instance per running app, speaking the NilHost Protocol over stdin/stdout.
//
// When the real NilLang toolchain is available, nilosd delegates to it; here
// it hosts the built-in mock VM so the emulator is fully functional without
// the upstream compiler.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joysriramsarkar/nilos-emulator/internal/runtime"
)

func main() {
	var (
		appDir = flag.String("app", "", "app install directory (contains payload/)")
		entry  = flag.String("entry", "", "app entry path relative to install dir")
		id     = flag.String("id", "", "app id")
		name   = flag.String("name", "", "app display name")
	)
	flag.Parse()

	if *id == "" {
		*id = os.Getenv("NILEMU_APP")
	}
	if *appDir == "" {
		*appDir = os.Getenv("NILEMU_APPDIR")
	}
	if *name == "" {
		*name = *id
	}

	app := runtime.AppInfo{ID: *id, Name: *name, Entry: *entry, Dir: *appDir}
	if err := runtime.RunMockVM(app, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "nilosd:", err)
		os.Exit(1)
	}
}
