package rtd

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/api/test/rtd/v1"
	"go.chromium.org/luci/common/logging"
)

var (
	containerRunning = false
)

// StartRTDContainer starts an RTD container, possibly a totally fake one, and
// returns once it's running.
func StartRTDContainer(ctx context.Context) error {
	if containerRunning {
		return fmt.Errorf("container already started; can't start another one")
	}
	logging.Infof(ctx, "Starting RTD container")
	logging.Infof(ctx, "TODO: `docker pull` the RTD container")
	logging.Infof(ctx, "TODO: `docker start` the RTD container")
	containerRunning = true
	return nil
}

// StopRTDContainer stops a running RTD container.
func StopRTDContainer(ctx context.Context) error {
	if !containerRunning {
		return fmt.Errorf("container isn't running; nothing to stop")
	}
	logging.Infof(ctx, "Stopping RTD container")
	logging.Infof(ctx, "TODO: `docker stop` the RTD container")
	containerRunning = false
	return nil
}

// Invoke runs an RTD Invocation against the running RTD container.
func Invoke(ctx context.Context, progressSinkPort, tlsPort int32) error {
	if !containerRunning {
		return fmt.Errorf("container hasn't been started yet; can't invoke")
	}
	// TODO: needs more work
	i := &rtd.Invocation{
		ProgressSinkClientConfig: &rtd.ProgressSinkClientConfig{
			Port: progressSinkPort,
		},
		TestLabServicesConfig: &rtd.TLSClientConfig{
			TlsPort: tlsPort,
		},
		Duts: []*rtd.DUT{
			{
				TlsDutName: "my-little-dutty",
			},
		},
		Name: "request_dummy-pass",
		Test: "remoteTestDrivers/tnull/tests/dummy-pass",
	}
	invocationFile, err := writeInvocationToFile(ctx, i)
	if err != nil {
		return err
	}
	logging.Infof(ctx, "TODO: `docker exec` against the container, with --input %v", invocationFile)
	logging.Infof(ctx, "Sleeping for a bit to simulate waiting for the Invocation to complete")
	time.Sleep(10 * time.Second)
	return nil
}

func writeInvocationToFile(ctx context.Context, i *rtd.Invocation) (string, error) {
	b, err := proto.Marshal(i)
	if err != nil {
		return "", err
	}
	tmp, err := ioutil.TempDir("", "start-rtd")
	if err != nil {
		return "", err
	}
	file := path.Join(tmp, "invocation.binaryproto")
	if err = ioutil.WriteFile(file, b, 0664); err != nil {
		return "", err
	}
	marsh := jsonpb.Marshaler{EmitDefaults: true, Indent: "  "}
	strForm, err := marsh.MarshalToString(i)
	if err != nil {
		return "", err
	}
	logging.Infof(ctx, "Wrote RTD's input Invocation binaryproto to %v", file)
	logging.Infof(ctx, "Contents of this Invocation message in jsonpb form are:\n%v", strForm)
	return file, nil
}
