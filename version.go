package main

import (
	"fmt"
	"io"
)

// Version information, injected at build time.
var Version = "development version"

// PrintVersion prints version information to w.
func PrintVersion(w io.Writer) {
	fmt.Fprintln(w, "RHMAP fh-system-dump-tool", Version)
}
