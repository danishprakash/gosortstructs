package main

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

type Test struct {
	Name string
	age  int
}

func formatOutput(fset *token.FileSet, file *ast.File) []byte {
	var buf bytes.Buffer
	err := format.Node(&buf, fset, file)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func main() {
	// read file
	// pass struct name
	var structName = flag.String("struct", "", "struct to sort")
	flag.Parse()

	src := `package main
type Example struct {
	Name String
	Age Int
}

type Hotel struct {
	Rating int
	Location String
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	ast.Inspect(file, func(x ast.Node) bool {
		t, ok := x.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if t.Type == nil {
			return true
		}

		name := t.Name.Name
		if len(*structName) > 0 && name != *structName {
			return true
		}

		s, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// fmt.Println(s.Fields.NumFields())
		sort.Slice(s.Fields.List, func(i, j int) bool {
			return s.Fields.List[i].Names[0].Name < s.Fields.List[j].Names[0].Name
		})

		// for _, field := range s.Fields.List {
		// 	fmt.Printf("Field: %s\n", field.Names[0].Name)
		// }

		return false
	})

	fmt.Println(string(formatOutput(fset, file)))
}
