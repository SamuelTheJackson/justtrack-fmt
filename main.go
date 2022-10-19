package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	// Flags
	baseFormatterCmd = kingpin.Flag(
		"base-formatter",
		"Base formatter to use").Default("").String()

	// Args
	paths = kingpin.Arg(
		"paths",
		"Paths to format",
	).Strings()
	writeOutput = kingpin.Flag(
		"write-output",
		"Write output to source instead of stdout").Short('w').Default("false").Bool()
)

func main() {
	kingpin.Parse()

	err := run()

	if err != nil {
		log.Fatalln(err)
	}

}

func run() error {
	config := JustTrackFmtConfig{
		BaseFormatterCmd: *baseFormatterCmd,
	}

	formatter := NewJustTrackFmt(config)

	if len(*paths) == 0 {
		contents, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		result, err := formatter.Format(contents)
		if err != nil {
			return err
		}

		if err := handleOutput("", contents, result); err != nil {
			return err
		}
	} else {
		for _, path := range *paths {

			switch info, err := os.Stat(path); {
			case err != nil:
				return err
			case info.IsDir():
				// Path is a directory- walk it
				err = filepath.Walk(
					path,
					func(subPath string, subInfo os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						if !subInfo.IsDir() && strings.HasSuffix(subPath, ".go") {
							// Shorten file and generate output
							contents, result, err := processFile(formatter, subPath)
							if err != nil {
								return err
							}
							err = handleOutput(subPath, contents, result)
							if err != nil {
								return err
							}
						}

						return nil
					},
				)
				if err != nil {
					return err
				}
			default:
				// Path is a file
				contents, result, err := processFile(formatter, path)
				if err != nil {
					return err
				}
				err = handleOutput(path, contents, result)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func processFile(formatter *JustTrackFmt, path string) ([]byte, []byte, error) {
	log.Debugf("Processing file %s", path)

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	result, err := formatter.Format(contents)

	return contents, result, err
}

func handleOutput(path string, contents []byte, result []byte) error {
	if contents == nil {
		return nil
	} else if *writeOutput {
		if path == "" {
			return errors.New("no path to write out to")
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if bytes.Equal(contents, result) {
			log.Debugf("contents unchanged, skipping write")

			return nil
		} else {
			log.Debugf("contents changed, writing output to %s", path)

			return ioutil.WriteFile(path, result, info.Mode())
		}
	} else {
		fmt.Print(string(result))

		return nil
	}
}
