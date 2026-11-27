package main

import (
	"flag"
	"fmt"
	"io"
)

// printUsage writes the help of the program: starting the server with the
// flags of fs, and the device commands, which are not flags and so would not
// show up otherwise.
func printUsage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, "usage:\n  household-app [flags]   start the server (prepares the data first)\n")
	_, _ = fmt.Fprint(w, deviceCommands)
	_, _ = fmt.Fprint(w, "\nflags of the server:\n")
	out := fs.Output()
	fs.SetOutput(w)
	fs.PrintDefaults()
	fs.SetOutput(out)
}
