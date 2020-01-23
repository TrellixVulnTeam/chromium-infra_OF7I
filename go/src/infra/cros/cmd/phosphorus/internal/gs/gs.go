package gs

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	gcgs "go.chromium.org/luci/common/gcloud/gs"
)

// DirWriter Mockable interface for writing whole directories at once
type DirWriter interface {
	WriteDir() error
}

type prodDirWriter struct {
	// The directory to be written from
	localRootDir string
	// The directory to be written to
	gsRootDir gcgs.Path

	// Mockable means of carrying out file-level writes
	client AuthedClient
}

var _ DirWriter = &prodDirWriter{}

// AuthedClient Mockable wrapper around the core "spin up subWriter" flow
type AuthedClient interface {
	NewWriter(p Path) (io.Writer, error)
}

type realAuthedClient struct {
	client gcgs.Client
}

var _ AuthedClient = &realAuthedClient{}

func (c *realAuthedClient) NewWriter(p Path) (io.Writer, error) {
	return c.client.NewWriter(gcgs.Path(p))
}

// NewDirWriter creates an object which can write a directory and its subdirectories to the given Google Storage path
func NewDirWriter(localPath string, gsPath Path, client AuthedClient) (DirWriter, error) {
	if err := verifyPaths(localPath, string(gsPath)); err != nil {
		return nil, err
	}
	return &prodDirWriter{
		localRootDir: localPath,
		gsRootDir:    gcgs.Path(gsPath),
		client:       client,
	}, nil
}

func verifyPaths(localPath string, gsPath string) error {
	problems := []string{}
	if _, err := os.Stat(localPath); err != nil {
		problems = append(problems, "invalid local path")
	} else if _, err := os.Open(localPath); err != nil {
		problems = append(problems, "unreadable local path")
	}
	if _, err := url.Parse(gsPath); err != nil {
		problems = append(problems, "invalid GS path")
	}
	if len(problems) > 0 {
		return errors.Reason("path errors: %s", strings.Join(problems, ", ")).Err()
	}
	return nil
}

// Path Google Storage path, to file or directory
type Path gcgs.Path

// WriteDir Write all files in the subtree of the local path to the corresponding places
//  in the subtree of the GS path
func (w *prodDirWriter) WriteDir() error {
	writeOne := func(src string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(w.localRootDir, src)
		if err != nil {
			return err
		}
		dest := Path(w.gsRootDir.Concat(relPath))
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		writer, err := w.client.NewWriter(dest)
		if err != nil {
			return err
		}
		bs := make([]byte, info.Size())
		if _, err = f.Read(bs); err != nil {
			return err
		}
		if _, err = writer.Write(bs); err != nil {
			return errors.Annotate(err, "writing from %s to %s", src, dest).Err()
		}
		return nil
	}
	return filepath.Walk(w.localRootDir, writeOne)
}

func newAuthenticatedTransport(ctx context.Context, f *authcli.Flags) (http.RoundTripper, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create authenticated transport").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	return a.Transport()
}

// NewAuthedClient Create a client with the given auth flags
func NewAuthedClient(ctx context.Context, f *authcli.Flags) (AuthedClient, error) {
	t, err := newAuthenticatedTransport(ctx, f)
	if err != nil {
		return nil, err
	}
	cli, err := gcgs.NewProdClient(ctx, t)
	if err != nil {
		return nil, errors.Annotate(err, "creating authenticated GS client").Err()
	}
	return &realAuthedClient{client: cli}, nil
}
