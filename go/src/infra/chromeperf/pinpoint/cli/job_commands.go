package cli

import (
	"context"
	"fmt"
	"infra/chromeperf/pinpoint"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/encoding/prototext"
)

type listJobs struct {
	baseCommandRun
	filter string
}

func cmdListJobs(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "list-jobs [-filter='']",
		ShortDesc: "Lists jobs tracked by Pinpoint",
		LongDesc: text.Doc(`
			Prints out a list of jobs tracked by Pinpoint to stdout, possibly
			constrained by the filter.
		`),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &listJobs{}
		}),
	}
}

func (lj *listJobs) RegisterFlags(p Param) {
	lj.baseCommandRun.RegisterFlags(p)
	// TODO(dberris): Link to documentation about supported fields in the filter.
	lj.Flags.StringVar(&lj.filter, "filter", "", text.Doc(`
		Optional filter to apply to restrict the set of jobs listed. See
		https://aip.dev/160 for details on the filter syntax.
	`))
}

func (lj *listJobs) Run(ctx context.Context, a subcommands.Application, args []string) error {
	c, err := lj.pinpointClient(ctx)
	if err != nil {
		return err
	}

	req := &pinpoint.ListJobsRequest{Filter: lj.filter}
	resp, err := c.ListJobs(ctx, req)
	if err != nil {
		return errors.Annotate(err, "failed during ListJobs").Err()
	}
	out := prototext.MarshalOptions{Multiline: true}.Format(resp)
	fmt.Println(out)
	return nil
}

type getJob struct {
	baseCommandRun
	name string
}

func cmdGetJob(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "get-job -name='...'",
		ShortDesc: "Prints information about a Pinpoint job",
		LongDesc: text.Doc(`
			Prints out information about a Job.
		`),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &getJob{}
		}),
	}
}

func (gj *getJob) RegisterFlags(p Param) {
	gj.baseCommandRun.RegisterFlags(p)
	gj.Flags.StringVar(&gj.name, "name", "", text.Doc(`
		Required; the name of the job to get information about.
		Example: "-name=XXXXXXXXXXXXXX"
	`))
}

func (gj *getJob) Run(ctx context.Context, a subcommands.Application, args []string) error {
	if gj.name == "" {
		return errors.Reason("must set -name").Err()
	}
	c, err := gj.pinpointClient(ctx)
	if err != nil {
		return err
	}

	req := &pinpoint.GetJobRequest{Name: pinpoint.LegacyJobName(gj.name)}
	resp, err := c.GetJob(ctx, req)
	if err != nil {
		return errors.Annotate(err, "failed during GetJob").Err()
	}
	out := prototext.MarshalOptions{Multiline: true}.Format(resp)
	fmt.Println(out)
	return nil
}

type waitJob struct {
	baseCommandRun
	name  string
	quiet bool
}

func cmdWaitJob(p Param) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "wait-job -name='...' [-quiet]",
		ShortDesc: "Polls the Pinpoint backend until the job stops running",
		LongDesc: text.Doc(`
			wait-job blocks until the provided Pinpoint job is finished, i.e.
			until the job is no longer RUNNING nor PENDING. -quiet disables
			informational text output.
		`),
		CommandRun: wrapCommand(p, func() pinpointCommand {
			return &waitJob{}
		}),
	}
}

func (wj *waitJob) RegisterFlags(p Param) {
	wj.baseCommandRun.RegisterFlags(p)
	wj.Flags.StringVar(&wj.name, "name", "", text.Doc(`
		Required; the name of the job to poll.
		Example: "-name=XXXXXXXXXXXXXX"
	`))
	wj.Flags.BoolVar(&wj.quiet, "quiet", false, text.Doc(`
		Disable informational text output; errors are still printed.
	`))
}

func (wj *waitJob) Run(ctx context.Context, a subcommands.Application, args []string) error {
	if wj.name == "" {
		return errors.Reason("must set -name").Err()
	}
	c, err := wj.pinpointClient(ctx)
	if err != nil {
		return err
	}

	req := &pinpoint.GetJobRequest{Name: pinpoint.LegacyJobName(wj.name)}
	poll := time.NewTicker(10 * time.Second)
	defer poll.Stop()

	// TODO(chowski): if we wait too long, our JWT could expire.
	// We should auto-renew that somehow.
	var lastUpdateTime time.Time
	for {
		resp, err := c.GetJob(ctx, req)
		if err != nil {
			return errors.Annotate(err, "failed during GetJob").Err()
		}
		if updateTime := resp.LastUpdateTime.AsTime(); lastUpdateTime != updateTime && !wj.quiet {
			lastUpdateTime = updateTime
			out := prototext.MarshalOptions{Multiline: true}.Format(resp)
			fmt.Println(out)
			fmt.Println("--------------------------------")
		}

		if s := resp.State; s != pinpoint.Job_RUNNING && s != pinpoint.Job_PENDING {
			if !wj.quiet {
				fmt.Printf("Final state for job %q: %v\n", resp.Name, s)
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Annotate(ctx.Err(), "polling for wait-job cancelled").Err()
		case <-poll.C:
			// loop back around and retry.
		}
	}
}
