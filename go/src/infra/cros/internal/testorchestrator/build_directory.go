package testorchestrator

import (
	"context"
	"errors"
	"fmt"
	"path"

	"cloud.google.com/go/storage"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/luci/luciexe/build"
	"google.golang.org/api/iterator"
)

func ListBuildDirectory(
	ctx context.Context,
	client *storage.Client,
	gcsPath *chromiumos.GcsPath,
) (objects []string, err error) {
	step, _ := build.StartStep(ctx, "list build directory")
	defer func() { step.End(err) }()

	bucket := client.Bucket(gcsPath.Bucket)
	query := &storage.Query{
		Prefix: gcsPath.Path,
	}

	it := bucket.Objects(ctx, query)
	names := []string{}

	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		names = append(names, attrs.Name)
	}

	step.SetSummaryMarkdown(fmt.Sprintf("Found %d objects in gs://%s", len(names), path.Join(gcsPath.Bucket, gcsPath.Path)))

	return names, nil
}
