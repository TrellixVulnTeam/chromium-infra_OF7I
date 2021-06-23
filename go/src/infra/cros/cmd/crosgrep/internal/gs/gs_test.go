package gs

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestGetReader tests reading a string out of a file.
func TestGetReader(t *testing.T) {
	f := &fakeGSClient{
		content: "b",
	}
	reader, err := GetReader(
		f,
		"a",
		0,
	)
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff("b", readString(reader)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
	if f.contentPrefixLength != 1024*1024 {
		t.Errorf("unexpected contentPrefixLength %d", f.contentPrefixLength)
	}
}

func readString(reader io.Reader) string {
	out, err := ioutil.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(out)
}

// FakeGSClient is a fake that emulates the interface of a Google storage client.
// FakeGSClient always returns a reader pointing at the same content regardless of
// the path requested.
type fakeGSClient struct {
	content string
	// ContentPrefixLength is the length of the prefix to be read.
	// It is set when constructing a new reader.
	contentPrefixLength int64
}

// NewReader constructs a new reaer pointing at the content embedded inside a fakeGSClient instance.
func (f *fakeGSClient) NewReader(p string, offset int64, length int64) (io.ReadCloser, error) {
	f.contentPrefixLength = length
	return ioutil.NopCloser(strings.NewReader(f.content)), nil
}
