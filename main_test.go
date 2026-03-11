// ABOUTME: tests for gosortstructs core sorting logic
// ABOUTME: covers basic sort, reverse, anonymous fields, struct filter, nested structs, struct literals, and validate

package main

import (
    "bytes"
    "os"
    "testing"

    "github.com/dave/dst/decorator"
)

func runSort(t *testing.T, src string, cfg config) string {
    t.Helper()

    f, err := os.CreateTemp("", "gosortstructs_test_*.go")
    if err != nil {
        t.Fatalf("create temp file: %v", err)
    }
    defer os.Remove(f.Name())

    if _, err := f.WriteString(src); err != nil {
        t.Fatalf("write temp file: %v", err)
    }
    f.Close()

    cfg.file = f.Name()

    parsed, err := cfg.parse()
    if err != nil {
        t.Fatalf("parse: %v", err)
    }

    node, err := cfg.modify(parsed)
    if err != nil {
        t.Fatalf("modify: %v", err)
    }

    var buf bytes.Buffer
    if err := decorator.Fprint(&buf, node); err != nil {
        t.Fatalf("format: %v", err)
    }

    return buf.String()
}

func TestSort(t *testing.T) {
    cases := []struct {
        name string
        src  string
        cfg  config
        want string
    }{
        {
            name: "basic ascending sort",
            src: `package p

type Olympians struct {
	Zeus   string
	Apollo int
	Hermes bool
}
`,
            cfg: config{},
            want: `package p

type Olympians struct {
	Apollo int
	Hermes bool
	Zeus   string
}
`,
        },
        {
            name: "reverse sort",
            src: `package p

type Olympians struct {
	Apollo int
	Hermes bool
	Zeus   string
}
`,
            cfg: config{reverse: true},
            want: `package p

type Olympians struct {
	Zeus   string
	Hermes bool
	Apollo int
}
`,
        },
        {
            name: "anonymous fields pushed to end",
            src: `package p

import "sync"

type Olympians struct {
	sync.Mutex
	Zeus   string
	Apollo int
}
`,
            cfg: config{},
            want: `package p

import "sync"

type Olympians struct {
	Apollo int
	Zeus   string
	sync.Mutex
}
`,
        },
        {
            name: "filter by struct name",
            src: `package p

type Olympians struct {
	Zeus   string
	Apollo int
}
`,
            cfg: config{strct: "Olympians"},
            want: `package p

type Olympians struct {
	Apollo int
	Zeus   string
}
`,
        },
        {
            name: "struct literal sorted",
            src: `package p

func f() {
	_ = Olympians{
		Zeus:   "a",
		Apollo: 1,
		Hermes: true,
	}
}
`,
            cfg: config{},
            want: `package p

func f() {
	_ = Olympians{
		Apollo: 1,
		Hermes: true,
		Zeus:   "a",
	}
}
`,
        },
        {
            name: "slice of struct literals sorted",
            src: `package p

func f() {
	_ = []Olympians{
		{Zeus: "a", Apollo: 1},
		{Hermes: true, Apollo: 2},
	}
}
`,
            cfg: config{},
            want: `package p

func f() {
	_ = []Olympians{
		{Apollo: 1, Zeus: "a"},
		{Apollo: 2, Hermes: true},
	}
}
`,
        },
        {
            name: "nested anonymous struct sorted",
            src: `package p

type Pantheon struct {
	Inner struct {
		Zeus   string
		Apollo int
	}
	Name string
}
`,
            cfg: config{},
            want: `package p

type Pantheon struct {
	Inner struct {
		Apollo int
		Zeus   string
	}
	Name string
}
`,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := runSort(t, tc.src, tc.cfg)
            if got != tc.want {
                t.Errorf("got:\n%s\nwant:\n%s", got, tc.want)
            }
        })
    }
}

func TestSortStructNotFound(t *testing.T) {
    src := `package p

type Olympians struct {
	Zeus   string
	Apollo int
}
`
    f, err := os.CreateTemp("", "gosortstructs_test_*.go")
    if err != nil {
        t.Fatalf("create temp file: %v", err)
    }
    defer os.Remove(f.Name())
    if _, err := f.WriteString(src); err != nil {
        t.Fatalf("write temp file: %v", err)
    }
    f.Close()

    cfg := config{file: f.Name(), strct: "Titans"}
    parsed, err := cfg.parse()
    if err != nil {
        t.Fatalf("parse: %v", err)
    }
    if _, err := cfg.modify(parsed); err == nil {
        t.Errorf("expected error for non-existent struct name, got nil")
    }
}

func TestValidate(t *testing.T) {
    cases := []struct {
        name    string
        cfg     config
        wantErr bool
    }{
        {
            name:    "no file",
            cfg:     config{},
            wantErr: true,
        },
        {
            name:    "struct and line conflict",
            cfg:     config{file: "x.go", strct: "Olympians", line: "1,5"},
            wantErr: true,
        },
        {
            name:    "valid config",
            cfg:     config{file: "x.go"},
            wantErr: false,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.cfg.validate()
            if tc.wantErr && err == nil {
                t.Errorf("expected error, got nil")
            }
            if !tc.wantErr && err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        })
    }
}
