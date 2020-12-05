package rtd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"infra/cros/cmd/prototype-rts/internal/docker"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/api/test/rtd/v1"
	"go.chromium.org/luci/common/logging"
)

// Orchestrator manages the lifecycle of an RTD container and its invocations.
type Orchestrator struct {
	volumeHostDir string
	container     docker.Docker
}

// StartRTDContainer starts an RTD container, possibly a totally fake one, and
// returns once it's running.
func (o *Orchestrator) StartRTDContainer(ctx context.Context, imageURI string) error {
	logging.Infof(ctx, "Starting RTD container")
	var err error
	if o.volumeHostDir, err = ioutil.TempDir(os.TempDir(), "rtd-volume"); err != nil {
		return err
	}
	if err = o.container.Pull(ctx, imageURI); err != nil {
		return err
	}
	if err = o.container.CreateVolume(ctx, o.volumeHostDir); err != nil {
		return err
	}
	if err = o.container.Run(ctx, imageURI); err != nil {
		return err
	}
	return nil
}

// StopRTDContainer stops a running RTD container.
func (o *Orchestrator) StopRTDContainer(ctx context.Context) error {
	if !o.container.IsRunning() {
		return fmt.Errorf("container isn't running; nothing to stop")
	}
	logging.Infof(ctx, "Stopping RTD container")
	if err := o.container.Stop(ctx); err != nil {
		return err
	}
	return nil
}

// Invoke runs an RTD Invocation against the running RTD container.
func (o *Orchestrator) Invoke(ctx context.Context, progressSinkPort, tlsPort int32, rtdCmd string) error {
	if !o.container.IsRunning() {
		return fmt.Errorf("container hasn't been started yet; can't invoke")
	}
	// TODO: needs more work
	i := &rtd.Invocation{
		ProgressSinkClientConfig: &rtd.ProgressSinkClientConfig{
			Port: progressSinkPort,
		},
		TestLabServicesConfig: &rtd.TLSClientConfig{
			TlsPort:    tlsPort,
			TlsAddress: "127.0.0.1",
		},
		Duts: []*rtd.DUT{
			{
				TlsDutName: "my-little-dutty",
			},
		},
		Requests: []*rtd.Request{
			{
				Name: "request_dummy-pass",
				Test: "remoteTestDrivers/tnull/tests/dummy-pass",
			},
		},
	}
	invocationFile, err := writeInvocationToFile(ctx, i, o.volumeHostDir, docker.VolumeContainerDir)
	if err != nil {
		return err
	}

	dockerCmd := strings.Fields(strings.Trim(rtdCmd, "\""))
	dockerCmd = append(dockerCmd, "--input", invocationFile)

	if err := o.container.Exec(ctx, dockerCmd); err != nil {
		return err
	}
	return nil
}

func writeInvocationToFile(ctx context.Context, i *rtd.Invocation, volumeHostDir, volumeContainerDir string) (string, error) {
	b, err := proto.Marshal(i)
	if err != nil {
		return "", err
	}
	filename := "invocation.binaryproto"
	hostFile := path.Join(volumeHostDir, filename)
	containerFile := path.Join(volumeContainerDir, filename)
	if err = ioutil.WriteFile(hostFile, b, 0664); err != nil {
		return "", err
	}
	marsh := jsonpb.Marshaler{EmitDefaults: true, Indent: "  "}
	strForm, err := marsh.MarshalToString(i)
	if err != nil {
		return "", err
	}
	logging.Infof(ctx, "Wrote RTD's input Invocation binaryproto to %v", hostFile)
	logging.Infof(ctx, "Contents of this Invocation message in jsonpb form are:\n%v", strForm)
	return containerFile, nil
}
