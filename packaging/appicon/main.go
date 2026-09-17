// Command appicon writes the application icon files used by the packaging
// scripts. Run via `make build`; nothing binary needs to be committed.
package main

import (
	"flag"
	"fmt"
	"os"

	"micoff/internal/icon"
)

func main() {
	icns := flag.String("icns", "", "write macOS .icns to this path")
	ico := flag.String("ico", "", "write Windows .ico to this path")
	preview := flag.String("png", "", "write a 512px PNG preview to this path")
	flag.Parse()

	write := func(path string, data []byte) {
		if path == "" {
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	write(*icns, icon.ICNS(icon.App))
	write(*ico, icon.AppICO())
	write(*preview, icon.PNG(icon.App(512)))
}
