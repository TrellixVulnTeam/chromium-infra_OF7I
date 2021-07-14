package gitiles

import (
	"context"
	"time"

	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	initialDelay = 2 * time.Second
	maxDelay     = 16 * time.Second
	retries      = 5
)

type retriableClient struct {
	client       Client
	retryFactory retry.Factory
}

// NewRetriableClient creates Gitiles client that automatically retries with
// exponential backoff if error was insufficient quota.
func NewRetriableClient(c Client) *retriableClient {
	retryFactory := func() retry.Iterator {
		return &retry.ExponentialBackoff{
			Limited: retry.Limited{
				Retries: retries,
				Delay:   initialDelay,
			},

			MaxDelay: maxDelay,
		}
	}
	return &retriableClient{
		client:       c,
		retryFactory: retryFactory,
	}
}

// Log retrieves commit log.
func (r *retriableClient) Log(ctx context.Context, in *gitilesProto.LogRequest, opts ...grpc.CallOption) (out *gitilesProto.LogResponse, err error) {
	err2 := retry.Retry(ctx, r.retryFactory, func() error {
		out, err = r.client.Log(ctx, in, opts...)
		if shouldRetry(err) {
			return err
		}
		return nil
	}, nil)
	if out == nil && err == nil {
		return nil, err2
	}
	return
}

// Refs retrieves repo refs.
func (r *retriableClient) Refs(ctx context.Context, in *gitilesProto.RefsRequest, opts ...grpc.CallOption) (out *gitilesProto.RefsResponse, err error) {
	err2 := retry.Retry(ctx, r.retryFactory, func() error {
		out, err = r.client.Refs(ctx, in, opts...)
		if shouldRetry(err) {
			return err
		}
		return nil
	}, nil)
	if out == nil && err == nil {
		return nil, err2
	}
	return
}

// Archive retrieves archived contents of the project.
//
// An archive is a shallow bundle of the contents of a repository.
//
// DEPRECATED: Use DownloadFile to obtain plain text files.
// TODO(pprabhu): Migrate known users to DownloadFile and delete this RPC.
func (r *retriableClient) Archive(ctx context.Context, in *gitilesProto.ArchiveRequest, opts ...grpc.CallOption) (out *gitilesProto.ArchiveResponse, err error) {
	err2 := retry.Retry(ctx, r.retryFactory, func() error {
		out, err = r.client.Archive(ctx, in, opts...)
		if shouldRetry(err) {
			return err
		}
		return nil
	}, nil)
	if out == nil && err == nil {
		return nil, err2
	}
	return
}

// DownloadFile retrieves a file from the project.
func (r *retriableClient) DownloadFile(ctx context.Context, in *gitilesProto.DownloadFileRequest, opts ...grpc.CallOption) (out *gitilesProto.DownloadFileResponse, err error) {
	err2 := retry.Retry(ctx, r.retryFactory, func() error {
		out, err = r.client.DownloadFile(ctx, in, opts...)
		if shouldRetry(err) {
			return err
		}
		return nil
	}, nil)
	if out == nil && err == nil {
		return nil, err2
	}
	return
}

// Projects retrieves list of available Gitiles projects
func (r *retriableClient) Projects(ctx context.Context, in *gitilesProto.ProjectsRequest, opts ...grpc.CallOption) (out *gitilesProto.ProjectsResponse, err error) {
	err2 := retry.Retry(ctx, r.retryFactory, func() error {
		out, err = r.client.Projects(ctx, in, opts...)
		if shouldRetry(err) {
			return err
		}
		return nil
	}, nil)
	if out == nil && err == nil {
		return nil, err2
	}
	return
}

func shouldRetry(err error) bool {
	s, ok := status.FromError(err)
	if !ok {
		return false
	}
	return s.Code() == codes.ResourceExhausted
}
