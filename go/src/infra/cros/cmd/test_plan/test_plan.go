package main

import (
	"errors"
	"fmt"
	"infra/cros/internal/testplan"
	"infra/tools/dirmd"

	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	luciflag "go.chromium.org/luci/common/flag"
)

func errToCode(a subcommands.Application, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

var app = &subcommands.DefaultApplication{
	Name:  "test_plan",
	Title: "A tool to generate ChromeOS test plans from config in DIR_METADATA files.",
	Commands: []*subcommands.Command{
		cmdGenerate,
		cmdValidate,
		subcommands.CmdHelp,
	},
}

var cmdGenerate = &subcommands.Command{
	UsageLine: "generate -root ROOT -cl CL1 [-cl CL2] -out OUTPUT",
	ShortDesc: "generate a test plan",
	LongDesc: text.Doc(`
		Generate a test plan.

		Computes test config from "DIR_METADATA" files and generates a test plan
		based on the files the patchsets are touching.
	`),
	CommandRun: func() subcommands.CommandRun {
		r := &generateRun{}
		r.Flags.StringVar(&r.root, "root", "", "Path to the root directory")
		r.Flags.Var(luciflag.StringSlice(&r.cls), "cl", text.Doc(`
			CL URL for the patchsets being tested. Must be specified at least once.

			Example: https://chromium-review.googlesource.com/c/chromiumos/platform2/+/123456
		`))
		r.Flags.StringVar(&r.out, "out", "", "Path to the output test plan")

		return r
	},
}

type generateRun struct {
	subcommands.CommandRunBase
	root string
	cls  []string
	out  string
}

func (r *generateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run(a, args, env))
}

func (r *generateRun) run(a subcommands.Application, args []string, env subcommands.Env) error {
	if r.root == "" {
		return errors.New("-root is required")
	}

	if len(r.cls) == 0 {
		return errors.New("-cl must be specified at least once")
	}

	if r.out == "" {
		return errors.New("-out is required")
	}

	return fmt.Errorf("generate not implemented")
}

var cmdValidate = &subcommands.Command{
	UsageLine: "validate -root ROOT TARGET1 [TARGET2...]",
	ShortDesc: "validate metadata files",
	LongDesc: text.Doc(`
		Validate metadata files.

		Validation logic on "DIR_METADATA" files specific to ChromeOS test planning.
		Note that validation is done on computed metadata, not directly on
		"DIR_METADATA" files; required fields do not need to be explicitly specified in
		all files, as long as they are present on computed targets.

		Each positional argument should be a path to a directory to compute and validate
		metadata for.

		The subcommand returns a non-zero exit code if any of the files is invalid.
	`),
	CommandRun: func() subcommands.CommandRun {
		r := &validateRun{}
		r.Flags.StringVar(&r.root, "root", "", "Path to the root directory, typically the root of the ChromeOS checkout")
		return r
	},
}

type validateRun struct {
	subcommands.CommandRunBase
	root string
}

func (r *validateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return errToCode(a, r.run(a, args, env))
}

func (r *validateRun) run(a subcommands.Application, args []string, env subcommands.Env) error {
	if r.root == "" {
		return errors.New("-root is required")
	}

	mapping, err := dirmd.ReadComputed(r.root, args...)
	if err != nil {
		return err
	}

	if err = testplan.ValidateMapping(mapping); err != nil {
		return err
	}

	return nil
}

func main() {
	os.Exit(subcommands.Run(app, nil))
}
