# gosortstructs
sorts structs

##### What
A command line tool which uses AST to sort fields of a Go struct for easier readability and better diffs. This tool is meant to be used as an underlying tool for text editors ([vim-plugin](https://github.com/danishprakash/vim-gosortstructs)). The resultant Go code conforms with gofmt but is eventually opinionated.

##### How
- By default, this program alphabetically sorts the struct(s) in the specified file.
- If an anonymous field is part of the struct, they are separately sorted and grouped at the end of the struct.

```
type Hotel struct {
   *Founder
   Rating   int
   Location string
   *Organization

}
```
to

```
type Hotel struct {
	Location       string
	Rating         int
	*Founder
	*Organization
}
```

##### Advantages
- In large codebases, it makes it easier to identify all new fields that are added to a struct if they are sorted alphabetically.
- If you've intermixed fields as in the above example, your struct tags would be wrongly indented which hampers readability.

##### Installation

```sh
go install danishpraka.sh/gosortstructs@latest
```

#### Usage

```sh
$ gosortstructs --help
Usage of gosortstructs:
  -file string
        file name to be processed
  -reverse
        reverse alphabetical sort
  -struct string
        struct to sort
  -write
        write result to source file (overwrite)
  -line
        position of the struct/cursor (to be used programmatically)
```



#### License
MIT License

Copyright (c) [Danish Prakash](https://github.com/danishprakash)
