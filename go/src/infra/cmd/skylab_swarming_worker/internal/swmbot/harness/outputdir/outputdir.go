// Package outputdir implements Swarming exported results directory creation
// and exposing those dirs as envvars
package outputdir

import (
	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"os"

	"go.chromium.org/luci/common/errors"
)

const clientOutdir = "LOG_DATA_DIR"

// Open creates the derived output directories and aliases for them. These do
//  not need to be closed by the harness.
func Open(i *swmbot.Info) error {
	clientDirPath := i.LogDataDir()
	if err := os.MkdirAll(clientDirPath, 0755); err != nil {
		return errors.Annotate(err, "open log-data dir %s", clientDirPath).Err()
	}
	if err := os.Setenv(clientOutdir, clientDirPath); err != nil {
		return errors.Annotate(err, "set output dir envvar %s: %s",
			clientOutdir, clientDirPath).Err()
	}
	return nil
}
