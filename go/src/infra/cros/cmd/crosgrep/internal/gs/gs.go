package gs

import (
	"io"

	"go.chromium.org/luci/common/errors"
)

// DefaultMaxFileSize is one mebibyte. This is the default maximum number of bytes
// taken if no explicit limit is specified.
// The limit is intentionally small because crosgrep will download many log files locally
// and its important to be explicit about when large files should be downloaded.
const defaultMaxFileSize = 1024 * 1024

// GetReader retrieves an io.Reader associated with a Google Storage path.
// The default maximum size is one mebibyte.
func GetReader(client gsClient, gsPath string, maxSize int64) (io.Reader, error) {
	if maxSize == 0 {
		maxSize = defaultMaxFileSize
	}
	reader, err := client.NewReader(gsPath, 0, maxSize)
	if err != nil {
		return nil, errors.Annotate(err, "get reader").Err()
	}
	return reader, err
}

type gsClient interface {
	NewReader(path string, offset int64, length int64) (io.ReadCloser, error)
}
