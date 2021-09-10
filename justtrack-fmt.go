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
	inFile   = flag.String("r", "", "read from file")
	outFile  = flag.String("w", "", "write to file instead of stdout")
)

func report(err error) {
	scanner.PrintError(os.Stderr, err)

	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: justtrack-fmt file [flags]\n")
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

	var out io.Writer

	// default write to stdout
	out = os.Stdout

	var in io.Reader

	// default read from stdin
	in = os.Stdin

	if *inFile != "" {
		// read from file
		inf, err := os.Open(*inFile)
		if err != nil {
			report(err)

			return
		}
		defer inf.Close()

		if !isValidFile(inf) {
			report(fmt.Errorf("%s not valid file", *inFile))

			return
		}

		in = inf
	}

	// write to file
	if *outFile != "" {
		outf, err := os.OpenFile(*outFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			report(err)

			return
		}
		defer outf.Close()

		var tmp bytes.Buffer

		out = &tmp

		defer func() {
			// delete content of file
			if err = outf.Truncate(0); err != nil {
				report(err)

				return
			}

			if _, err = outf.Seek(0, 0); err != nil {
				report(err)

				return
			}

			if _, err := outf.Write(tmp.Bytes()); err != nil {
				report(err)

				return
			}
		}()
	}

	if *gf != "" {
		outGf, err := exec.Command(*gf, *inFile).Output()
		if err != nil {
			report(err)

			return
		}

		in = bytes.NewReader(outGf)
	}

	if err := FormatFile(in, out); err != nil {
		report(err)
	}
}
