package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/token"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/thoas/go-funk"
)

const (
	EMBEDDEDS = iota
	LOGGER
	ID
	FUNCS
	STRUCTS
	SCALARS
	TIMESTAMPS
)

const (
	idFieldName     = "Id"
	loggerFieldName = "logger"
	// number of different categories we have to sort inside of a struct.
	structFieldCategoryCount = 7
)

var timestampNames = []string{"CreatedAt", "UpdatedAt"}

type JustTrackFmtConfig struct {
	// Formatter that will be run before and after main shortening process. If empty,
	// defaults to goimports (if found), otherwise gofmt.
	BaseFormatterCmd string
}
type JustTrackFmt struct {

	// Some extra params around the base formatter generated from the BaseFormatterCmd
	// argument in the config.
	baseFormatter     string
	baseFormatterArgs []string
	config            JustTrackFmtConfig
}

func NewJustTrackFmt(config JustTrackFmtConfig) *JustTrackFmt {
	var formatterComponents []string

	if config.BaseFormatterCmd == "" {
		_, err := exec.LookPath("gofumpt")
		if err != nil {
			formatterComponents = []string{"gofmt"}
		} else {
			formatterComponents = []string{"gofumpt"}
		}
	} else {
		formatterComponents = strings.Split(config.BaseFormatterCmd, " ")
	}

	jtf := &JustTrackFmt{
		config:        config,
		baseFormatter: formatterComponents[0],
	}

	if len(formatterComponents) > 1 {
		jtf.baseFormatterArgs = formatterComponents[1:]
	} else {
		jtf.baseFormatterArgs = []string{}
	}

	return jtf

}

func (j JustTrackFmt) Format(content []byte) ([]byte, error) {
	f, err := decorator.Parse(content)
	if err != nil {
		return nil, err
	}
	// no declarations - nothing to do
	if f.Decls == nil || len(f.Decls) == 0 {
		return content, nil
	}
	// loop over all declarations
	for _, d := range f.Decls {
		// function declarations
		// we want empty lines before return/continue/break
		if f, ok := d.(*dst.FuncDecl); ok {
			addEmptyLineBeforeReturn(f)

			continue
		}

		genDecl, ok := d.(*dst.GenDecl)
		if !ok {
			continue
		}

		// import statements
		if genDecl.Tok == token.IMPORT {
			removeUnnecessaryImportNames(genDecl)

			continue
		}

		// sort var blocks
		if genDecl.Tok == token.VAR {
			sortSpecs(genDecl.Specs)

			continue
		}

		// sort const
		// if iota ignore the block because we don't want to mess this up
		if genDecl.Tok == token.CONST {
			foundIota := false
			for _, i := range genDecl.Specs {
				if v, ok := i.(*dst.ValueSpec); ok {
					if len(v.Values) == 1 {
						if i, ok := v.Values[0].(*dst.Ident); ok {
							if i.String() == "iota" {
								foundIota = true

								break
							}
						}
					}
				}
			}

			if !foundIota {
				sortSpecs(genDecl.Specs)
			}

			continue
		}

		for _, s := range genDecl.Specs {
			ts, ok := s.(*dst.TypeSpec)
			if !ok {
				continue
			}

			// sort interfaces
			if i, ok := ts.Type.(*dst.InterfaceType); ok {
				groupAndSortFieldList(i.Methods.List)

				continue
			}

			s, ok := ts.Type.(*dst.StructType)
			if !ok {
				continue
			}

			groupAndSortFieldList(s.Fields.List)
		}
	}

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	if err := decorator.Fprint(foo, f); err != nil {
		return nil, err
	}
	if err := foo.Flush(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func sortSpecs(specs []dst.Spec) {
	sort.Slice(specs, func(i, j int) bool {
		left := specs[i].(*dst.ValueSpec)
		right := specs[j].(*dst.ValueSpec)

		return compareStrings(left.Names[0].Name, right.Names[0].Name)
	})
}

func groupAndSortFieldList(l []*dst.Field) {
	// there is nothing to sort when the struct only got 1 field or less
	if len(l) <= 1 {
		return
	}

	embeddeds := make([]*dst.Field, 0)
	id := make([]*dst.Field, 0)
	logger := make([]*dst.Field, 0)
	timestamps := make([]*dst.Field, 0)
	rest := make([]*dst.Field, 0)

	for _, i := range l {
		// nested struct
		if s, ok := i.Type.(*dst.StructType); ok {
			groupAndSortFieldList(s.Fields.List)
		}

		// embeddeds don't have names
		if len(i.Names) == 0 {
			embeddeds = append(embeddeds, i)

			continue
		}

		// identifier of the struct
		if i.Names[0].Name == idFieldName {
			id = append(id, i)

			continue
		}

		// logger
		if i.Names[0].Name == loggerFieldName {
			logger = append(logger, i)

			continue
		}

		// timestamps
		if funk.ContainsString(timestampNames, i.Names[0].Name) {
			timestamps = append(timestamps, i)

			continue
		}

		rest = append(rest, i)
	}

	sortStructFieldsByName(id)
	sortStructFieldsByName(embeddeds)
	sortStructFieldsByName(logger)
	sortStructFieldsByName(timestamps)
	sortStructFieldsByName(rest)

	merged := make([]*dst.Field, 0)
	merged = append(merged, embeddeds...)
	merged = append(merged, logger...)
	merged = append(merged, id...)
	merged = append(merged, rest...)
	merged = append(merged, timestamps...)

	for i := range merged {
		l[i] = merged[i]
	}
}

func sortStructFieldsByName(list []*dst.Field) {
	if len(list) < 2 {
		return
	}

	sort.Slice(list, func(i, j int) bool {
		var right, left string

		// embeddeds don't have a name
		if list[i].Names == nil {
			left = fmt.Sprintf("%s", list[i].Type)
		} else {
			left = list[i].Names[0].Name
		}

		// embeddeds don't have a name
		if list[j].Names == nil {
			right = fmt.Sprintf("%s", list[j].Type)
		} else {
			right = list[j].Names[0].Name
		}

		return compareStrings(left, right)
	})
}

func compareStrings(left, right string) bool {
	right = strings.ToLower(right)
	left = strings.ToLower(left)

	return left < right
}

func addEmptyLineBeforeReturn(f *dst.FuncDecl) {
	dst.Inspect(f, func(n dst.Node) bool {
		// we are only interested in block statements ({})
		blkStm, ok := n.(*dst.BlockStmt)
		if !ok {
			return true
		}

		if len(blkStm.List) == 0 {
			return true
		}

		// loop over block statements and insert empty line before return/break/continue
		// if not the only statement in the block
		// not interested in the first statement
		// override `Before` with empty line
		for _, l := range blkStm.List[1:] {
			switch l.(type) {
			case *dst.ReturnStmt, *dst.BranchStmt:
				l.Decorations().Before = dst.EmptyLine
			}
		}

		return true
	})
}

// remove import names when `-` is just replaced with `_`
func removeUnnecessaryImportNames(str *dst.GenDecl) {
	for _, i := range str.Specs {
		im, ok := i.(*dst.ImportSpec)
		if !ok {
			continue
		}

		// does not have import name
		if im.Name == nil {
			continue
		}

		path := im.Path.Value
		path = strings.TrimRight(path, "\"")
		path = strings.ReplaceAll(path, "-", "_")
		packageName := filepath.Base(path)

		// remove unnecessary import name
		if packageName == im.Name.Name {
			im.Name = nil
		}
	}
}
