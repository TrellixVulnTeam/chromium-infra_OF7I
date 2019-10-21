package multibot

import (
	"fmt"
	"infra/libs/skylab/dutstate"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"

	"go.chromium.org/luci/common/errors"
)

func validateMultiBotHostInfo(message *skylab_local_state.MultiBotHostInfo) error {
	if message == nil {
		return fmt.Errorf("nil message")
	}

	var missingArgs []string

	if message.HostInfo == nil {
		missingArgs = append(missingArgs, "host info")
	}

	if message.DutName == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// WriteHostInfo takes a MultiBotHostInfo, extracts the AutotestHostInfo proto
// and DUT name from it, and writes that proto to a file in the specified
// directory in a JSON-encoded format.
func WriteHostInfo(message *skylab_local_state.MultiBotHostInfo, dir string) error {
	if err := validateMultiBotHostInfo(message); err != nil {
		return err
	}
	p := dutstate.HostInfoFilePath(dir, message.DutName)

	if err := writeJSONPb(p, message.HostInfo); err != nil {
		return errors.Annotate(err, "write host info").Err()
	}

	return nil
}

// writeJSONPb writes a JSON encoded proto to outFile.
func writeJSONPb(outFile string, payload proto.Message) error {
	dir := filepath.Dir(outFile)
	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dir, 0777); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}

	w, err := os.Create(outFile)
	if err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	defer w.Close()

	marshaler := jsonpb.Marshaler{}
	if err := marshaler.Marshal(w, payload); err != nil {
		return errors.Annotate(err, "write JSON pb").Err()
	}
	return nil
}
