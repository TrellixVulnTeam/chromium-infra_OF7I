//go:generate go run $GOFILE -output ../options.go
//go:generate go fmt ../options.go

// Generates options.go, which implements git options.
//
// To modify options.go, modify the Options declared in this file and run
// go generate.
//
// Each command that can take options (e.g. "clone") has a generated interface,
// so only options valid for that command can be passed.
//
// Each option (e.g. "depth") has a generated function that returns the option,
// which can be passed to a command: `git.Clone(..., git.Depth(10))`
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"runtime"
	"strings"
	"text/template"

	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/data/text"
)

// Command describes a git command.
type Command struct {
	// Name of the Command, e.g. "clone". Should be lower case.
	Name string
}

// Option describes an option passed to a git command.
//
// Note that most fields and methods are exported to allow use in templates.
type Option struct {
	// Name of the Option, as passed to git. Should be lower case. Should not
	// contain a "--" prefix. For example, "depth".
	Name string
	// Type of the Option, for example int or bool.
	Type reflect.Kind
	// Commands the Option is valid for.
	Cmds []*Command
	// Documentation for the Option.
	Doc string
}

// FuncName returns the name of the Go function to build an option.
//
// "-" are removed, and the name is converted to camelcase with the first letter
// capitalized. For example, "depth" -> "Depth", "dry-run" -> "DryRun".
func (o Option) FuncName() string {
	return strings.ReplaceAll(strings.Title(o.Name), "-", "")
}

// StructName returns FuncName with "Opt" added as a suffix.
func (o Option) StructName() string {
	return o.FuncName() + "Opt"
}

// LowerName returns the name of the Option in lower case.
func (o Option) LowerName() string {
	return strings.ToLower(o.Name)
}

// IsBoolOpt returns true if Option has a Bool type.
func (o Option) IsBoolOpt() bool {
	return o.Type == reflect.Bool
}

// Commands that accept options.
var pushRefCmd = &Command{Name: "pushRef"}
var fetchCmd = &Command{Name: "fetch"}
var cloneCmd = &Command{Name: "clone"}

// Options to generate.
var options = []Option{
	{
		Name: "force",
		Type: reflect.Bool,
		Cmds: []*Command{pushRefCmd},
		Doc: text.Doc(
			`Force overrides checks that a command might run before modifying ` +
				`state, see individual command documentation for exact behavior.`,
		),
	},
	{
		Name: "dry-run",
		Type: reflect.Bool,
		Cmds: []*Command{pushRefCmd},
		Doc:  `DryRun shows what a command would do without actually running it.`,
	},
	{
		Name: "depth",
		Type: reflect.Int,
		Cmds: []*Command{fetchCmd, cloneCmd},
		Doc:  `Depth creates a shallow clone, and limits fetches in a repository created by a shallow clone.`,
	},
	{
		Name: "no-tags",
		Type: reflect.Bool,
		Cmds: []*Command{fetchCmd, cloneCmd},
		Doc:  `NoTags disables downloading tags during fetch or clone commands.`,
	},
}

// genInterfaces writes interfaces for each unique Command in options to w.
func genInterfaces(w io.Writer, options []Option) error {
	interfaceTmpl := `
type {{.Name}}Opt interface {
	{{.Name}}OptArgs() []string
}
	`

	t, err := template.New("interface").Parse(interfaceTmpl)
	if err != nil {
		return err
	}

	// cmdNames is a set of all unique Commands in options. nameToCmd maps a
	// Command.Name to Command. Use a set of names and a map so that Commands
	// can be generated in order.
	cmdNames := stringset.New(0)
	nameToCmd := make(map[string]*Command)

	for _, opt := range options {
		for _, cmd := range opt.Cmds {
			cmdNames.Add(cmd.Name)
			nameToCmd[cmd.Name] = cmd
		}
	}

	for _, name := range cmdNames.ToSortedSlice() {
		if err = t.Execute(w, nameToCmd[name]); err != nil {
			return err
		}
	}

	return nil
}

// genOptions writes the implementation of each of options to w.
//
// Each Option has a struct that implements the interface for each of its
// Commands, and a function to build a new instance of the struct.
func genOptions(w io.Writer, options []Option) error {
	optionsTmpl := `
type {{.StructName}} struct {
	val {{.Type}}
}

// {{.Doc}}
{{- if .IsBoolOpt}}
func {{.FuncName}}If(enabled {{.Type}}) *{{.StructName}} {
	return &{{.StructName}}{val: enabled}
}

// {{.Doc}}
func {{.FuncName}}() *{{.StructName}} {
	return &{{.StructName}}{val: true}
}
{{else}}
func {{.FuncName}}(val {{.Type}}) *{{.StructName}} {
	return &{{.StructName}}{val: val}
}
{{end}}

func (o {{$.StructName}}) args()[]string {
{{- if .IsBoolOpt}}
	if o.val {
		return []string{"--{{.LowerName}}"}
	}

	return []string{}
{{else}}
	return []string{"--{{.LowerName}}", fmt.Sprint(o.val)}
{{end}}
}

{{range .Cmds}}
func (o {{$.StructName}}) {{.Name}}OptArgs() []string {return o.args()}
{{end}}
`

	t, err := template.New("struct and func").Parse(optionsTmpl)
	if err != nil {
		return err
	}

	for _, opt := range options {
		if err = t.Execute(w, opt); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	output := flag.String("output", "", "Path to write generated options to")
	flag.Parse()

	// Get the name of this file, to direct edits of the generated file to this
	// file instead.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) returned !ok")
	}

	f, err := os.Create(*output)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	fmt.Fprintf(
		f, `
// Code generated by %[1]s. DO NOT EDIT MANUALLY.
//
// Edit %[1]s and run "go generate" to modify.

package git

import (
	"fmt"
)
`,
		path.Base(filename),
	)

	if err = genInterfaces(f, options); err != nil {
		panic(err)
	}

	if err = genOptions(f, options); err != nil {
		panic(err)
	}
}
