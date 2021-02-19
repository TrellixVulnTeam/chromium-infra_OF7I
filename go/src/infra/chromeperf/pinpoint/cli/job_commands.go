package cli

import (
	"fmt"
	"infra/chromeperf/pinpoint"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"google.golang.org/protobuf/encoding/prototext"
)

type listJobs struct {
	baseCommandRun
	filter string
}

func cmdListJobs(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "list-jobs [--filter='']",
		ShortDesc: "Lists jobs tracked by Pinpoint",
		LongDesc: text.Doc(`
			Prints out a list of jobs tracked by Pinpoint to stdout, possibly
			constrained by the filter.
		`),
		CommandRun: func() subcommands.CommandRun {
			lj := &listJobs{}
			lj.RegisterDefaultFlags(p)
			// TODO(dberris): Link to documentation about supported fields in the filter.
			lj.Flags.StringVar(&lj.filter, "filter", "", text.Doc(`
				Optional filter to apply to restrict the set of jobs listed. See
				https://aip.dev/160 for details on the filter syntax.
			`))
			return lj
		},
	}
}

func (lj *listJobs) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, lj, env)
	c, err := lj.pinpointClient(ctx)
	if err != nil {
		return lj.done(a, fmt.Errorf("failed to create a Pinpoint client: %s", err))
	}

	req := &pinpoint.ListJobsRequest{Filter: lj.filter}
	resp, err := c.ListJobs(ctx, req)
	if err != nil {
		return lj.done(a, fmt.Errorf("failed during ListJobs: %v", err))
	}
	out := prototext.MarshalOptions{Multiline: true}.Format(resp)
	fmt.Println(out)
	return lj.done(a, nil)
}
