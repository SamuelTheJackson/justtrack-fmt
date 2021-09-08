package justtrack_fmt

import (
	"fmt"
	"go/token"
	"io"
	"io/ioutil"
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
	STRUCTS
	SCALARS
	TIMESTAMPS
)

const (
	idFieldName     = "Id"
	loggerFieldName = "logger"
	// number of different categories we have to sort inside of a struct.
	structFieldCategoryCount = 6
)

var timestampNames = []string{"CreatedAt", "UpdatedAt"}

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

func FormatFile(in io.Reader, out io.Writer) error {
	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	f, err := decorator.Parse(src)
	if err != nil {
		return err
	}

	// no declarations - nothing to do
	if f.Decls == nil {
		return nil
	}

	// loop over all declarations
	for _, d := range f.Decls {
		// function declarations
		// we want empty lines before return/continue/break
		if f, ok := d.(*dst.FuncDecl); ok {
			addEmptyLineBeforeReturn(f)

			continue
		}

		str, ok := d.(*dst.GenDecl)
		if !ok {
			continue
		}

		// import statements
		if str.Tok == token.IMPORT {
			removeUnnecessaryImportNames(str)

			continue
		}

		// sort var blocks
		if str.Tok == token.VAR {
			sortSpecs(str.Specs)

			continue
		}

		// sort const
		// if iota ignore the block because we don't want to mess this up
		if str.Tok == token.CONST {
			foundIota := false
			for _, i := range str.Specs {
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
				sortSpecs(str.Specs)
			}

			continue
		}

		for _, s := range str.Specs {
			dstS, ok := s.(*dst.TypeSpec)
			if !ok {
				continue
			}

			l, ok := dstS.Type.(*dst.StructType)
			if !ok {
				continue
			}

			groupAndSortFieldList(l.Fields.List)
		}

		if err != nil {
			return err
		}
	}

	if err := decorator.Fprint(out, f); err != nil {
		return err
	}

	return nil
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

	fieldList := make([][]*dst.Field, structFieldCategoryCount)

	for i := range fieldList {
		fieldList[i] = make([]*dst.Field, 0)
	}

	for _, i := range l {
		// nested struct
		if s, ok := i.Type.(*dst.StructType); ok {
			groupAndSortFieldList(s.Fields.List)
		}

		// embeddeds don't have names
		if len(i.Names) == 0 {
			fieldList[EMBEDDEDS] = append(fieldList[EMBEDDEDS], i)

			continue
		}

		// identifier of the struct
		if i.Names[0].Name == idFieldName {
			fieldList[ID] = append(fieldList[ID], i)

			continue
		}

		// logger
		if i.Names[0].Name == loggerFieldName {
			fieldList[LOGGER] = append(fieldList[LOGGER], i)

			continue
		}

		// timestamps
		if funk.ContainsString(timestampNames, i.Names[0].Name) {
			fieldList[TIMESTAMPS] = append(fieldList[TIMESTAMPS], i)

			continue
		}

		// scalars
		if is, ok := i.Type.(*dst.Ident); ok {
			if is.Obj == nil {
				fieldList[SCALARS] = append(fieldList[SCALARS], i)

				continue
			}
		}

		fieldList[STRUCTS] = append(fieldList[STRUCTS], i)
	}

	counter := 0

	for _, r := range fieldList {
		if len(r) == 0 {
			continue
		}

		sortStructFieldsByName(r)

		from := counter
		to := from + len(r)

		copy(l[from:to], r)

		counter += len(r)
	}
}

func sortStructFieldsByName(list []*dst.Field) {
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

	return right > left
}
