package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"sort"
)

func main() {
	src := `package main
type Example struct {
	Name String
	Age Int
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	ast.Inspect(file, func(x ast.Node) bool {
		s, ok := x.(*ast.StructType)
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

	var buf bytes.Buffer
	err = format.Node(&buf, fset, file)
	if err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
}
