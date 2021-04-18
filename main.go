package main

// usage
// gosortstruct --file main.go --struct Node --write --reverse

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"sort"
)

// stores flags and internal config
type config struct {
	file    string
	fset    *token.FileSet
	reverse bool
	source  string
	strct   string
	write   bool
}

// simple wrapper to facilitate sorting
// of anonymous fields i.e have field names
type structType struct {
	Name string
	node *ast.Field
}

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func start() error {
	var (
		flagFile    = flag.String("file", "", "file name to be processed")
		flagReverse = flag.Bool("reverse", false, "reverse alphabetical sort")
		flagStruct  = flag.String("struct", "", "struct to sort")
		flagWrite   = flag.Bool("write", false, "write result to source file (overwrite)")
	)
	flag.Parse()

	cfg := config{
		file:    *flagFile,
		reverse: *flagReverse,
		strct:   *flagStruct,
		write:   *flagWrite,
	}

	err := cfg.validate()
	if err != nil {
		return err
	}

	node, err := cfg.parse()
	if err != nil {
		return err
	}

	node, err = cfg.process(node)
	if err != nil {
		return err
	}

	out, err := cfg.format(node)
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func (c *config) validate() error {
	if c.file == "" {
		return errors.New("no file passed")
	}

	return nil
}

func (c *config) parse() (*ast.File, error) {
	c.fset = token.NewFileSet()
	var src interface{}
	return parser.ParseFile(c.fset, c.file, src, parser.ParseComments)
}

// https://golang.org/src/go/ast/filter.go
func fieldName(x interface{}) *ast.Ident {
	switch t := x.(type) {
	case *ast.Ident:
		return t
	case *ast.SelectorExpr:
		if _, ok := t.X.(*ast.Ident); ok {
			return t.Sel
		}
	case *ast.StarExpr:
		return fieldName(t.X)
	}
	return nil
}

func (c *config) process(node *ast.File) (*ast.File, error) {
	var foundOne bool
	sortStructs := func(x ast.Node) bool {
		var anon = []structType{}
		t, ok := x.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if t.Type == nil {
			return true
		}

		name := t.Name.Name

		// if --struct is passed and no matches
		// found, return appropriate response
		if c.strct != "" && name == c.strct {
			foundOne = true
		}

        // to get names of anon fields
		s, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// separate out anonymous fields
		for i := len(s.Fields.List) - 1; i >= 0; i-- {
			if s.Fields.List[i].Names == nil {
				anon = append(anon, structType{
					fieldName(s.Fields.List[i].Type).Name,
					s.Fields.List[i],
				})
				s.Fields.List = append(s.Fields.List[:i], s.Fields.List[i+1:]...)
			}
		}

		// will through out of bounds for structs which
		// have no anonymous fields, keep a check here
		if len(anon) != 0 {
			// fmt.Println(anon[0].Name)
		}

		// gather standard and anonymous structs separately
		// condition := func(f []*ast.Ident) bool { return f == nil }
		// s.Fields.List = filter(s, condition)

		sortFunc := func(i, j int) bool {
			return s.Fields.List[i].Names[0].Name < s.Fields.List[j].Names[0].Name
		}

		revSortFunc := func(i, j int) bool {
			return s.Fields.List[i].Names[0].Name > s.Fields.List[j].Names[0].Name
		}

		anonRevSortFunc := func(i, j int) bool {
			return anon[i].Name > anon[j].Name
		}

		anonSortFunc := func(i, j int) bool {
			return anon[i].Name < anon[j].Name
		}

        // TODO: --reverse sort on structs with anon
        // fields tend to have a newline separation
		// sort anonymous fields separately
		if c.reverse {
			sort.Slice(s.Fields.List, revSortFunc)
			if anon != nil {
				sort.Slice(anon, anonRevSortFunc)
			}
		} else {
			sort.Slice(s.Fields.List, sortFunc)
			if anon != nil {
				sort.Slice(anon, anonSortFunc)
			}
		}

		// push back sorted anonymous fields
		if len(anon) != 0 {
			for _, f := range anon {
				s.Fields.List = append(s.Fields.List, f.node)
			}
		}

		return true
	}

	ast.Inspect(node, sortStructs)

	if c.strct != "" && !foundOne {
		return node, errors.New("no struct found")
	}

	return node, nil
}

func (c *config) format(node *ast.File) (string, error) {
	var buf bytes.Buffer
	err := format.Node(&buf, c.fset, node)
	if err != nil {
		panic(err)
	}

	if c.write {
		err = ioutil.WriteFile(c.file, buf.Bytes(), 0)
		if err != nil {
			panic(err)
		}
	}

	return buf.String(), err
}
