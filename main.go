package main

// usage
// gosortstruct --file main.go --reverse --struct Node

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
)

type config struct {
	file    string
	strct   string // TODO: this should be a []string
	reverse bool
	fset    *token.FileSet
	source  string
}

func (c *config) parse() (*ast.File, error) {
	c.fset = token.NewFileSet()
	var src interface{}
	return parser.ParseFile(c.fset, c.file, src, parser.ParseComments)
}

func (c *config) process(node *ast.File) (*ast.File, error) {
	sortStructs := func(x ast.Node) bool {
		t, ok := x.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if t.Type == nil {
			return true
		}

		name := t.Name.Name
		if len(c.strct) > 0 && name != c.strct {
			return true
		}

		s, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// TODO: do away with this
		if c.reverse {
			sort.Slice(s.Fields.List, func(i, j int) bool {
				return s.Fields.List[i].Names[0].Name > s.Fields.List[j].Names[0].Name
			})
		} else {
			sort.Slice(s.Fields.List, func(i, j int) bool {
				return s.Fields.List[i].Names[0].Name < s.Fields.List[j].Names[0].Name
			})
		}

		return false
	}

	ast.Inspect(node, sortStructs)
	return node, nil
}

func (c *config) format(node *ast.File) (string, error) {
	var buf bytes.Buffer
	err := format.Node(&buf, c.fset, node)
	if err != nil {
		panic(err)
	}
	return buf.String(), err
}

func main() {
	var (
		flagFile    = flag.String("file", "", "file name to be processed")
		flagReverse = flag.Bool("reverse", false, "reverse alphabetical sort")
		flagStruct  = flag.String("struct", "", "struct to sort")
		// TODO: add --rewrite flag which updates the file being processed
	)
	flag.Parse()

	cfg := config{
		file:    *flagFile,
		reverse: *flagReverse,
		strct:   *flagStruct,
	}

	node, err := cfg.parse()
	if err != nil {
		panic(err)
	}

	node, err = cfg.process(node)
	if err != nil {
		panic(err)
	}

	out, err := cfg.format(node)
	if err != nil {
		panic(err)
	}

	fmt.Println(out)
}
