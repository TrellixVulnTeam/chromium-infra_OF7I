package cli

import (
	"context"
	"flag"
	"fmt"
	"infra/chromeperf/pinpoint"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/protobuf/encoding/prototext"
)

// downloadResultsToDir copies the results associated with the provided job to the dstDir.
// The file that is written is returned.
func downloadResultsToDir(ctx context.Context, gcs *storage.Client, dstDir string, result *pinpoint.ResultFile) (string, error) {
	bucket, path := result.GcsBucket, result.Path
	dstFile := filepath.Join(dstDir, filepath.Base(path))
	if _, err := os.Stat(dstFile); !os.IsNotExist(err) {
		return "", errors.Reason("cannot download result to %v: that file already exists", dstFile).Err()
	}

	src, err := gcs.Bucket(bucket).Object(path).NewReader(ctx)
	if err != nil {
		return "", errors.Annotate(err, "error requesting gs://%v/%v", bucket, path).Err()
	}
	defer src.Close()

	tmp, err := ioutil.TempFile("", filepath.Base(path))
	if err != nil {
		return "", err
	}
	// On early error, make sure to clean up after ourselves; these will error
	// out if things are successful, but that shouldn't cause a problem.
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := io.Copy(tmp, src); err != nil {
		return "", errors.Annotate(err, "error downloading gs://%v/%v to file %q", bucket, path, tmp.Name()).Err()
	}
	if err := tmp.Close(); err != nil {
		return "", errors.Annotate(err, "error flushing %v", tmp.Name()).Err()
	}
	if err := os.Rename(tmp.Name(), dstFile); err != nil {
		return "", err
	}
	return dstFile, nil
}

type downloadResultsMixin struct {
	resultsDir      string
	downloadResults bool
	openResults     bool
}

// TODO(chowski): if we are actually going to use mixins a lot, probably should
// add some support in the pinpointCommand wrapper type somehow.
func (drm *downloadResultsMixin) RegisterFlags(flags *flag.FlagSet) {
	flags.BoolVar(&drm.downloadResults, "download-results", false, text.Doc(`
		If set, results are downloaded to the -results-dir.
		Note that files will NOT be overwritten if they exist already (an error
		will be printed).
	`))
	flags.StringVar(&drm.resultsDir, "results-dir", os.TempDir(), text.Doc(`
		Ignored unless -download-results is set;
		the directory to store results in.
	`))
	flags.BoolVar(&drm.openResults, "open-results", false, text.Doc(`
		Ignored unless -download-results is set;
		if set, the results will automatically be opened in a browser.
		Requires xdg-open to be installed.
	`))
}

func (drm *downloadResultsMixin) doDownloadResults(ctx context.Context, job *pinpoint.Job) error {
	if !drm.downloadResults || job.GetName() == "" {
		return nil
	}
	if job.GetState() != pinpoint.Job_SUCCEEDED {
		logging.Infof(ctx, "Can't download results: must be in state SUCCEEDED, got %s", job.GetState())
		return nil
	}

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		return errors.Annotate(err, "couldn't connect to Google Cloud Storage (GCS)").Err()
	}
	defer gcs.Close()

	var errs errors.MultiError
	for _, result := range job.GetResultFiles() {
		dstFile, err := downloadResultsToDir(ctx, gcs, drm.resultsDir, result)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		logging.Infof(ctx, "Downloaded result file %v", dstFile)

		if drm.openResults {
			if err := exec.Command("xdg-open", dstFile).Run(); err != nil {
				// Doesn't count as a fatal error, just is inconvenient.
				logging.Errorf(ctx, "Couldn't open file with xdg-open: %v", err)
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

type waitForJobMixin struct {
	wait bool

	// TODO(dberris): Centralise the logging and allow for quiet mode.
	quiet bool
}

func (wjm *waitForJobMixin) RegisterFlags(flags *flag.FlagSet) {
	flags.BoolVar(&wjm.wait, "wait", false, text.Doc(`
		When enabled, will wait for a job to complete.
	`))
	flags.BoolVar(&wjm.quiet, "quiet", false, text.Doc(`
		Suppress progress output when waiting.
	`))
}

// waitForJob can return a nil job in case `j` is also nil, and return a valid
// pointer to a Job in case there's an error, indicating partial success.
func (wjm *waitForJobMixin) waitForJob(
	ctx context.Context,
	c pinpoint.PinpointClient,
	j *pinpoint.Job,
	o io.Writer,
) (*pinpoint.Job, error) {
	if !wjm.wait || j == nil {
		return j, nil
	}
	req := &pinpoint.GetJobRequest{Name: pinpoint.LegacyJobName(j.Name)}
	poll := time.NewTicker(10 * time.Second)
	defer poll.Stop()

	lastJob := j
	for {
		resp, err := c.GetJob(ctx, req)
		if err != nil {
			return j, errors.Annotate(err, "failed during GetJob").Err()
		}
		if !wjm.quiet && lastJob.GetLastUpdateTime().AsTime() != resp.GetLastUpdateTime().AsTime() {
			out := prototext.MarshalOptions{Multiline: true}.Format(lastJob)
			fmt.Fprintln(o, out)
			fmt.Fprintln(o, "--------------------------------")
		}
		lastJob = resp
		if s := lastJob.State; s != pinpoint.Job_RUNNING && s != pinpoint.Job_PENDING {
			if !wjm.quiet {
				fmt.Fprintf(o, "Final state for job %q: %v\n", lastJob.Name, s)
			}
			break
		}
		select {
		case <-ctx.Done():
			return lastJob, errors.Annotate(ctx.Err(), "polling for job wait cancelled").Err()
		case <-poll.C:
			// loop back around and retry.
		}
	}
	return lastJob, nil
}
