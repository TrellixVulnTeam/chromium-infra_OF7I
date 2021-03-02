package cli

import (
	"context"
	"flag"
	"infra/chromeperf/pinpoint"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"cloud.google.com/go/storage"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
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
