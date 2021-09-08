package main

import (
	"flag"
	"fmt"
	"go/scanner"
	"os"
	"strings"

	"github.com/SamuelTheJackson/justtrack-fmt"
)

var (
	exitCode = 0
	file     = flag.String("f", "", "read from file instead of stdin")
)

func report(err error) {
	scanner.PrintError(os.Stderr, err)

	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: justtrack-fmt [flags]\n")
	flag.PrintDefaults()
}

func isValidFile(f *os.File) bool {
	name := f.Name()

	if strings.HasPrefix(name, ".") {
		return false
	}

	if !strings.HasSuffix(name, ".go") {
		return false
	}

	return true
}

func main() {
	justtrackMain()

	os.Exit(exitCode)
}

func justtrackMain() {
	flag.Usage = usage
	flag.Parse()

	// default read from stdin
	in := os.Stdin

	// read from file
	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			report(err)

			return
		}
		defer f.Close()

		if !isValidFile(f) {
			report(fmt.Errorf("%s not valid file", *file))

			return
		}

		in = f
	}

	if err := justtrack_fmt.FormatFile(in, os.Stdout); err != nil {
		report(err)
	}
}
