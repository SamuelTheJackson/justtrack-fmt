package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/scanner"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	exitCode = 0
	gf       = flag.String("g", "", "set gofumpt binary path")
	rFile    = flag.String("r", "", "read from file instead of stdin")
	wFile    = flag.String("w", "", "write to file instead of stdout")
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
	var in io.Reader
	in = os.Stdin

	var out io.Writer

	// default write to stdout
	out = os.Stdout

	// read from file
	if *rFile != "" {
		f, err := os.Open(*rFile)
		if err != nil {
			report(err)

			return
		}
		defer f.Close()

		if !isValidFile(f) {
			report(fmt.Errorf("%s not valid file", *rFile))

			return
		}

		in = f
	}

	// write to file
	if *wFile != "" {
		f, err := os.OpenFile(*wFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			report(err)

			return
		}
		defer f.Close()

		var tmp bytes.Buffer

		out = &tmp

		defer func() {
			// delete content of file
			if err = f.Truncate(0); err != nil {
				report(err)

				return
			}

			if _, err = f.Seek(0, 0); err != nil {
				report(err)

				return
			}

			if _, err := f.Write(tmp.Bytes()); err != nil {
				report(err)

				return
			}
		}()
	}

	if *gf != "" {
		cmd := exec.Command(*gf, *rFile)
		if *rFile == "" {
			cmd.Stdin = in
		}

		out, err := cmd.Output()
		if err != nil {
			report(err)

			return
		}

		in = bytes.NewReader(out)

	}

	if err := FormatFile(in, out); err != nil {
		report(err)
	}
}
