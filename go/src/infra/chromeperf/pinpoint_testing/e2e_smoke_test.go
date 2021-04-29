package e2e_smoke_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"infra/chromeperf/pinpoint"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	. "github.com/smartystreets/goconvey/convey"
)

// userEmail is the email address of the entity running the smoke tests.
const userEmail = "e2e@smoke.test"

// testPaths collects relevant paths for running a test suite.
type testPaths struct {
	pinpointCLI      string
	grpcServer       string
	fakelegacyServer string
}

func compile(intoDir string) (testPaths, error) {
	var paths testPaths
	for _, bin := range []struct {
		name, packagePath string
		setPath           *string
	}{
		{"pinpoint_cli.exe", "infra/chromeperf/cmd/pinpoint", &paths.pinpointCLI},
		{"grpc_pinpoint.exe", "infra/chromeperf/pinpoint_server", &paths.grpcServer},
		{"fakelegacy_pinpoint.exe", "infra/chromeperf/pinpoint/fakelegacy/bin", &paths.fakelegacyServer},
	} {
		outPath := filepath.Join(intoDir, bin.name)
		*bin.setPath = outPath

		out, err := exec.Command("go", "build", "-o", outPath, bin.packagePath).CombinedOutput()
		if err != nil {
			return testPaths{}, errors.Reason("failed to build %q; output was:\n\n%s", bin.name, out).Err()
		}
	}
	return paths, nil
}

func startProcess(onError func(error), path string, args ...string) (cancel func(), _ error) {
	cmd := exec.Command(path, args...)
	output := new(bytes.Buffer)
	cmd.Stdout = output
	cmd.Stderr = output

	name := filepath.Base(path)
	if err := cmd.Start(); err != nil {
		return nil, errors.Annotate(err, "failed to start %q", name).Err()
	}

	exited := make(chan struct{})
	go func() {
		defer close(exited)

		if err := cmd.Wait(); err != nil {
			err = errors.Reason("unexpected unclean process exit: process %q exited with %v; output was:\n\n%s", name, err, output).Err()
			onError(err)
		}
	}()
	return func() {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			// If we failed to signal the process, pass an error to the caller
			// return rather than waiting for the process to exit.
			onError(errors.Annotate(err, "failed to signal process").Err())
			return
		}
		<-exited
	}, nil
}

func setup(t *testing.T) (testPaths, error) {
	dir, err := ioutil.TempDir("", "pinpoint_e2e_smoke_test")
	if err != nil {
		return testPaths{}, err
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	fmt.Printf("Compiling into %q...\n", dir)
	return compile(dir)
}

// start invokes the relevant servers in the background. The cleanup function
// must always be called, even if an error is also returned.
func start(ctx context.Context, t *testing.T, paths testPaths) (cleanup func(), execCLI func(args ...string) (string, error), _ error) {
	unexpectedError := func(err error) {
		t.Errorf("Test invariant failure: %v", err)
	}

	var cleanups []func()
	cleanup = func() {
		for _, cl := range cleanups {
			cl()
		}
	}

	const (
		fakelegacyPort = 1123
		grpcEndpoint   = "localhost:60800"
	)
	for _, p := range []struct {
		path string
		args []string
	}{{
		paths.grpcServer, []string{
			"--legacy_pinpoint_service", fmt.Sprintf("http://localhost:%d", fakelegacyPort),
			"--hardcoded_user_email", userEmail,
		},
	}, {
		paths.fakelegacyServer, []string{
			"--port", fmt.Sprint(fakelegacyPort),
			"--template-dir", "../pinpoint/fakelegacy/templates",
		},
	}} {
		cancel, err := startProcess(unexpectedError, p.path, p.args...)
		if err != nil {
			return cleanup, nil, err
		}
		cleanups = append(cleanups, cancel)
	}

	ctx, cf := context.WithTimeout(ctx, 5*time.Second)
	defer cf()
	if err := waitForServices(ctx, grpcEndpoint); err != nil {
		return cleanup, nil, err
	}

	execCLI = func(args ...string) (string, error) {
		args = append(args, "--endpoint", grpcEndpoint)
		out, err := exec.Command(paths.pinpointCLI, args...).CombinedOutput()
		if err != nil {
			return "", errors.Reason("`pinpoint %q` failed with status %v; output was:\n\n%s", args, err, out).Err()
		}
		return string(out), nil
	}
	return cleanup, execCLI, nil
}

func waitForServices(ctx context.Context, grpcEndpoint string) error {
	// Using shorter backoff periods allows for faster connections as we reset
	// the services and redial (due to how GoConvey runs the tests).
	params := grpc.WithConnectParams(grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  time.Millisecond,
			Multiplier: 1.01,
			MaxDelay:   10 * time.Millisecond,
		},
	})
	conn, err := grpc.DialContext(ctx, grpcEndpoint, params, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return errors.Annotate(err, "failed to dial gRPC endpoint %v", grpcEndpoint).Err()
	}
	defer conn.Close()
	client := pinpoint.NewPinpointClient(conn)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Verify that the fakelegacy service is up. We rely on looking up a
	// non-existent job to give us NotFound to indicate that a request made it
	// all the way to fakelegacy and back.
	for {
		_, err := client.GetJob(ctx, &pinpoint.GetJobRequest{Name: "jobs/legacy-10000000000000"})
		status, ok := status.FromError(err)
		if !ok {
			return errors.Annotate(err, "unexpected error from GetJob").Err()
		}
		switch code := status.Code(); code {
		default:
			return errors.Annotate(err, "unexpected error code %v returned from GetJobs", code).Err()
		case codes.NotFound:
			// The fakelegacy service responded with this code, so everything is up!
			return nil
		case codes.Internal:
			// We infer that the fakelegacy service couldn't be contacted;
			// sleep and retry.
		}
		select {
		case <-ctx.Done():
			return errors.Annotate(ctx.Err(), "timed out waiting for gRPC service at %v", grpcEndpoint).Err()
		case <-ticker.C:
			continue
		}
	}
}

func extractJobID(t *testing.T, out string) string {
	lastSlash := strings.LastIndexByte(out, '/')
	if lastSlash == -1 {
		t.Fatalf("couldn't find URL-like path in CLI output:\n\n%s", out)
	}
	return strings.TrimSpace(out[lastSlash+1:])

}

func TestScheduleJobFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		// We rely on sending an interrupt to other processes, which is not
		// supported on Windows (https://golang.org/issue/6720).
		t.Skip("Test is unsupported on windows")
	}
	paths, err := setup(t)
	if err != nil {
		t.Fatal(err)
	}

	// Fix the user config to a relative path.
	if err := os.Setenv("PINPOINT_USER_CONFIG", "testdata/sample-user-config.yaml"); err != nil {
		t.Fatal(err)
	}

	Convey("With a fresh set of servers", t, func() {
		ctx, cf := context.WithCancel(context.Background())
		defer cf()

		cleanup, execCLI, err := start(ctx, t, paths)
		defer cleanup()
		So(err, ShouldBeNil)

		Convey("list-jobs is empty", func() {
			out, err := execCLI("list-jobs")
			So(err, ShouldBeNil)
			So(out, ShouldEqual, "\n")
		})

		Convey("creating a new telemetry experiment succeeds", func() {
			out, err := execCLI(
				"experiment-telemetry-start",
				"-base-commit", "abcefg",
				"-exp-cl", "1234/5",
				"-benchmark", "jetstream2",
				"-story", "JetStream2",
				"-cfg", "linux-perf",
			)
			So(err, ShouldBeNil)
			jobID := extractJobID(t, out)

			Convey("get-job shows the new job", func() {
				out, err = execCLI("get-job", "-name", jobID)
				So(err, ShouldBeNil)
			})
			Convey("list-jobs shows the new job", func() {
				out, err = execCLI("list-jobs")
				So(err, ShouldBeNil)
				So(out, ShouldContainSubstring, pinpoint.LegacyJobName(jobID))
			})
		})

		Convey("creating a new telemetry experiment with presets works", func() {
			out, err := execCLI(
				"experiment-telemetry-start",
				"-presets-file", "testdata/sample-presets.yaml",
				"-base-commit", "abcdef",
				"-exp-cl", "1234/5",
				"-preset", "sample",
			)
			So(err, ShouldBeNil)
			jobID := extractJobID(t, out)

			Convey("get-job shows the new job", func() {
				out, err = execCLI("get-job", "-name", jobID)
				So(err, ShouldBeNil)
			})
		})

		Convey("config shows the expected flags for a user config", func() {
			out, err := exec.Command(paths.pinpointCLI, "config").CombinedOutput()
			So(err, ShouldBeNil)
			So(string(out), ShouldContainSubstring, "testdata/sample-presets.yaml")
		})
	})
}
