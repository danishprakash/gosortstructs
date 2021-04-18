# gosortstructs
sorts structs

##### Why
I wrote this as an assistive binary for a vim-plugin which would automatically sort structs in a given go file on a save event based a certain heuristic. The resultant go code after running this conforms with gofmt but is eventually subjective in style.

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
- In large codebases, it makes it easier to identify what all fields are added to a struct since they'll be sorted alphabetically.
- If you've intermixed fields as in the above example, your tags would be wrongly indented which makes for a not so good picture.

##### Installation

```sh
go get -u github.com/danishprakash/gosortstructs
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
```
