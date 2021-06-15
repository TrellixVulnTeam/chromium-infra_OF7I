package base

import (
	"os"

	"github.com/maruel/subcommands"
)

// CrosgrepBQProjectEnvvar is the environment variable to use for bigquery project.
const crosgrepBQProjectEnvvar = "CROSGREP_BQ_PROJECT"

// Command is the common subcommand for crosgrep commands.
// It contains information like the logging verbosity and the current bigquery billing project// that is used for the underlying SQL query.
type Command struct {
	subcommands.CommandRunBase
	verbose   bool
	BQProject string
}

// InitFlags sets up the common flags for a command.
func (c *Command) InitFlags() {
	c.Flags.StringVar(&c.BQProject, "bq-project", "", "BigQuery Project for use in queries, falls back to CROSGREP_BQ_PROJECT envvar")
	c.Flags.BoolVar(&c.verbose, "verbose", false, `Set the verbosity of diagnostic messages.`)
}

// Verbose returns whether the command is intended to run with verbose logging
// enabled or not.
func (c *Command) Verbose() bool {
	return c.verbose
}

// GetBQProject returns the cloud project for bigquery explicitly specified on the command line
// or taken from the CROSGREP_BQ_PROJECT environment variable if no flag is provided.
func (c *Command) GetBQProject() string {
	if c.BQProject == "" {
		return os.Getenv(crosgrepBQProjectEnvvar)
	}
	return c.BQProject
}
