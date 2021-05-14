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
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/tools/go/buildutil"
)

// stores flags and internal config
type config struct {
	file     string
	fset     *token.FileSet
	reverse  bool
	source   string
	strct    string
	modified io.Reader
	write    bool
	line     string
	start    int
	end      int
}

type offset struct {
}

// simple wrapper to facilitate sorting
// of anonymous fields i.e have field names
type structType struct {
	Name string
	node *dst.Field
	// Comment *ast.CommentGroup
	// Doc     *ast.CommentGroup
	// Tag     *ast.CommentGroup
}

func main() {
	if err := start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func start() error {
	var (
		flagFile     = flag.String("file", "", "file name to be processed")
		flagReverse  = flag.Bool("reverse", false, "reverse alphabetical sort")
		flagStruct   = flag.String("struct", "", "struct to sort")
		flagWrite    = flag.Bool("write", false, "write result to source file (overwrite)")
		flagModified = flag.Bool("modified", false, "read from stdin")
		flagLine     = flag.String("line", "", "line number of the struct to be processed")
	)
	flag.Parse()

	cfg := config{
		file:    *flagFile,
		reverse: *flagReverse,
		strct:   *flagStruct,
		write:   *flagWrite,
		line:    *flagLine,
	}

	if len(*flagLine) != 0 {
		splitted := strings.Split(*flagLine, ",")
		cfg.start, _ = strconv.Atoi(splitted[0])
		cfg.end, _ = strconv.Atoi(splitted[1])
	}

	// read from stdin (for use by editors)
	if *flagModified {
		cfg.modified = os.Stdin
	}

	err := cfg.validate()
	if err != nil {
		return err
	}

	astFile, err := cfg.parse()
	if err != nil {
		return err
	}

	dec := decorator.NewDecorator(cfg.fset)
	f, err := dec.DecorateFile(astFile)
	if err != nil {
		panic(err)
	}

	node, err := cfg.modify(f)
	if err != nil {
		fmt.Printf("modify err: %+v", err)
		return err
	}

	fmt.Printf("%+v", node)
	if err := decorator.Print(node); err != nil {
		fmt.Printf("print err: %+v", err)
		panic(err)
	}

	// out, err := cfg.format(node)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println(out)
	return nil
}

// func (c *config) findSelection(node ast.Node) (int, int, error) {
// }

func (c *config) validate() error {
	if c.file == "" {
		return errors.New("no file passed")
	}

	if len(c.line) != 0 && len(c.strct) != 0 {
		return errors.New("pass either --struct or --line")
	}

	return nil
}

func (c *config) parse() (*ast.File, error) {
	c.fset = token.NewFileSet()
	var src interface{}

	// reads from stdin
	if c.modified != nil {
		archive, err := buildutil.ParseOverlayArchive(c.modified)
		if err != nil {
			return nil, fmt.Errorf("failed to parse stdin: %+v", err)
		}

		fc, ok := archive[c.file]
		if ok {
			return nil, fmt.Errorf("couldn't find %s in archive: %+v", c.file, err)
		}
		src = fc
	}

	return parser.ParseFile(c.fset, c.file, src, parser.ParseComments)
}

// https://golang.org/src/go/ast/filter.go
func fieldName(x interface{}) *dst.Ident {
	switch t := x.(type) {
	case *dst.Ident:
		return t
	case *dst.SelectorExpr:
		if _, ok := t.X.(*dst.Ident); ok {
			return t.Sel
		}
	case *dst.StarExpr:
		return fieldName(t.X)
	}
	return nil
}

func (c *config) modify(f *dst.File) (*dst.File, error) {
	var foundOne bool
	sortStructs := func(x dst.Node) bool {
		var anon = []structType{}
		t, ok := x.(*dst.TypeSpec)
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
		s, ok := t.Type.(*dst.StructType)
		if !ok {
			return true
		}

		// now that the current node is indeed a struct
		// if line number is provided, do an early return
		// if this is not the struct we're interested in

		// startLNo := s.Decorations().Start
		// endLNo := s.Decorations().End
		// if len(c.line) != 0 {
		// 	if !(startLNo <= c.start && c.end <= endLNo) {
		// 		return true
		// 	}
		// }

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

		// will throw out of bounds for structs which
		// have no anonymous fields, keep a check here
		if len(anon) != 0 {
			// fmt.Println(anon[0].Name)
		}

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

		// for i := range s.Fields.List {
		// 	// fmt.Println(s.Fields.List[i].Doc.Text())
		// 	// fmt.Printf("%d: %+v\n", i, s.Fields.List[i].Doc.Text())
		// }

		// tmp := copyField(s.Fields.List[1])
		// s.Fields.List[1] = copyField(s.Fields.List[0])
		// s.Fields.List[0] = copyField(tmp)

		// fmt.Println(s.Fields.List[0].Names)
		// fmt.Println(s.Fields.List[1].Names)
		// fmt.Println(s.Fields.List[0].Doc.Text())
		// fmt.Println(s.Fields.List[1].Doc.Text())
		// fmt.Println(c.fset.Position(s.Fields.List[0].Doc.Pos()).Line)
		// fmt.Println(c.fset.Position(s.Fields.List[1].Doc.Pos()).Line)
		// fmt.Println(c.fset.Position(s.Fields.List[0].Doc.Pos()).Line)

		return true
	}

	dst.Inspect(f, sortStructs)

	if c.strct != "" && !foundOne {
		return f, errors.New("no struct found")
	}

	return f, nil
}

func copyField(node *dst.Field) *dst.Field {
	return &dst.Field{
		Names: node.Names,
		Tag:   node.Tag,
		Type:  node.Type.(dst.Expr),
	}
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
