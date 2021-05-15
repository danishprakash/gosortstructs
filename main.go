package main

// usage
// gosortstruct --file main.go --struct Node --write --reverse

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
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
	dec      *decorator.Decorator
	df       *dst.File
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
// of anonymous fields (not having field names)
type structType struct {
	Name string
	node *dst.Field
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
		return fmt.Errorf("failed to validate command line flags: %+v", err)
	}

	f, err := cfg.parse()
	if err != nil {
		return fmt.Errorf("failed to parse source: %+v", err)
	}

	node, err := cfg.modify(f)
	if err != nil {
		return fmt.Errorf("failed to modify source: %+v", err)
	}

	err = cfg.format(node)
	if err != nil {
		return fmt.Errorf("failed to format source: %+v", err)
	}

	return nil
}

func (c *config) validate() error {
	if c.file == "" {
		return errors.New("no file passed")
	}

	if len(c.line) != 0 && len(c.strct) != 0 {
		return errors.New("pass either --struct or --line")
	}

	return nil
}

func (c *config) parse() (*dst.File, error) {
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

	astFile, err := parser.ParseFile(c.fset, c.file, src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	c.dec = decorator.NewDecorator(c.fset)
	c.df, err = c.dec.DecorateFile(astFile)
	if err != nil {
		panic(err)
	}

	return c.df, nil
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

// modify Inspects the (a)dst node and sorts the
// fields of the struct based on specified flags
func (c *config) modify(f *dst.File) (*dst.File, error) {
	var foundOne bool
	sortStructs := func(x dst.Node) bool {
		var anon = []structType{}

		// we need TypeSpec so as to
		// parse the name of the struct
		t, ok := x.(*dst.TypeSpec)
		if !ok {
			return true
		}

		if t.Type == nil {
			return true
		}

		// if --struct is passed and no matches
		// found, return appropriate response
		if c.strct != "" && t.Name.Name == c.strct {
			foundOne = true
		}

		// if this is the struct we want to modify
		// get the StructType type and begin modification
		s, ok := t.Type.(*dst.StructType)
		if !ok {
			return true
		}

		// now that the current node is indeed a struct
		// if line number is provided, return if no match
		if len(c.line) != 0 {
			// convert dst node to ast and get position
			startLNo := c.fset.Position(c.dec.Ast.Nodes[x].Pos()).Line
			endLNo := c.fset.Position(c.dec.Ast.Nodes[x].End()).Line
			if !(startLNo <= c.start && c.end <= endLNo) {
				return true
			}
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

		// less functions for sort.Slice()

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

		// append sorted anonymous fields (segregation)
		if len(anon) != 0 {
			for _, f := range anon {
				s.Fields.List = append(s.Fields.List, f.node)
			}
		}

		return true
	}

	dst.Inspect(f, sortStructs)

	if c.strct != "" && !foundOne {
		return f, errors.New("no struct found")
	}

	return f, nil
}

func (c *config) format(node *dst.File) error {
	var buf bytes.Buffer
	err := decorator.Fprint(&buf, node)
	if err != nil {
		return err
	}

	if c.write {
		err = ioutil.WriteFile(c.file, buf.Bytes(), 0)
		if err != nil {
			return err
		}
	}

	fmt.Println(buf.String())
	return nil
}
