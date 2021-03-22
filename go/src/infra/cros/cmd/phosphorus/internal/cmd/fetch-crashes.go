// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"infra/cros/cmd/phosphorus/internal/tls"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"
	tlsapi "go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FetchCrashes subcommand: fetches crashes from a DUT, optionally uploading them.
var FetchCrashes = &subcommands.Command{
	UsageLine: "fetch-crashes -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Fetch crashes from a DUT, optionally uploading them to http://crash/.",
	LongDesc: `Fetch crashes from a DUT, optionally uploading them to http://crash/.

Uses the TLS FetchCrashes API to retrieve crashes from a specified DUT and,
depending on settings in the input proto, may upload them to http://crash/ for
internal debugging.`,
	CommandRun: func() subcommands.CommandRun {
		c := &fetchCrashesRun{}
		c.Flags.StringVar(&c.InputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.FetchCrashesRequest")
		c.Flags.StringVar(&c.OutputPath, "output_json", "", "Path to write JSON encoded test_platform.phosphorus.FetchCrashesResponse to")
		return c
	},
}

type fetchCrashesRun struct {
	CommonRun
}

// Run is the main entry point to (and wrapper around) the FetchCrashes subcommand.
func (c *fetchCrashesRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.ValidateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}

	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, args, env); err != nil {
		logApplicationError(ctx, a, err)
		return 1
	}
	return 0
}

const (
	// outputSubDir is the directory to which we write files indicating uploaded crash info.
	outputSubDir = "test_runner"
	// uploadedFile is the file to which we write info about uploaded crashes (to enable easier
	// debugging via output logs, e.g. in stainless).
	uploadedFile = "uploaded_crashes.txt"
)

// baseCrashNamePat is a regex that matches the base portion of a crash file. For example, in
// the following examples, it matches "chrome.20201202.130102.12345.4567"
// chrome.20201202.130102.12345.4567.log.gz
// chrome.20201202.130102.12345.4567.dmp
// chrome.20201202.130102.12345.4567.i915_error_state.log.xz
var baseCrashNamePat = regexp.MustCompile(`^[^.]+\.\d+\.\d+\.\d+\.\d+`)

// innerRun reads in the JSON PB input, runs the actual fetch-crashes command, and serializes the output.
func (c *fetchCrashesRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	r := &phosphorus.FetchCrashesRequest{}
	if err := ReadJSONPB(c.InputPath, r); err != nil {
		return err
	}
	if err := validateFetchCrashesRequest(r); err != nil {
		return err
	}

	if d := r.Deadline.AsTime(); !d.IsZero() {
		var c context.CancelFunc
		logging.Infof(ctx, "Running with deadline %s (current time: %s)", d, time.Now().UTC())
		ctx, c = context.WithDeadline(ctx, d)
		defer c()
	}

	resp, err := runTLSFetchCrashes(ctx, r)
	if err != nil {
		return err
	}

	return WriteJSONPB(c.OutputPath, resp)
}

// validateFetchCrashesRequest ensures that all required parameters are present in |r|.
func validateFetchCrashesRequest(r *phosphorus.FetchCrashesRequest) error {
	missingArgs := getCommonMissingArgs(r.Config)

	if r.DutHostname == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

type fullCrash struct {
	// info is the metadata for the crash.
	info *tlsapi.CrashInfo
	// blob is the *combined* blobs for the crash. That is, if FetchCrashes
	// sent a file across 3 CrashBlob protos, it will only be represented
	// in one here.
	blobs []*tlsapi.CrashBlob
}

// runTLSFetchCrashes is the core of the implementation of fetch_crashes: it runs the FetchCrashes TLS API, assembles
// its output, and (if requested) uploads the crashes.
func runTLSFetchCrashes(ctx context.Context, r *phosphorus.FetchCrashesRequest) (*phosphorus.FetchCrashesResponse, error) {
	req := tlsapi.FetchCrashesRequest{
		Dut:       r.DutHostname,
		FetchCore: false,
	}

	logging.Infof(ctx, "Starting TLS")
	tlsServer, err := tls.StartBackground(fmt.Sprintf("0.0.0.0:%d", droneTLWPort))
	if err != nil {
		return nil, errors.Annotate(err, "run TLS Provision").Err()
	}
	defer tlsServer.Stop()

	logging.Infof(ctx, "Connecting to TLS")
	conn, err := grpc.Dial(tlsServer.Address(), grpc.WithInsecure())
	if err != nil {
		return nil, errors.Annotate(err, "connect to TLS").Err()
	}
	defer conn.Close()

	cl := tlsapi.NewCommonClient(conn)

	fetchResp := &phosphorus.FetchCrashesResponse{State: phosphorus.FetchCrashesResponse_SUCCEEDED}

	logging.Infof(ctx, "Calling FetchCrashes")
	stream, err := cl.FetchCrashes(ctx, &req)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return fetchResp, nil
		}
		return nil, errors.Annotate(err, "run TLS FetchCrashes").Err()
	}

	logging.Infof(ctx, "Finding existing crashes")
	rtdCrashes, err := findRTDCrashes(ctx, r.Config.Task.ResultsDir)
	if err != nil {
		logging.Errorf(ctx, "Failed to get preexisting crashes: ", err)
	}

	outDir := filepath.Join(r.Config.Task.ResultsDir, outputSubDir)

	logging.Infof(ctx, "Reading FetchCrashes response")
	var lastCrashID int64 = -1
	var crashInfo *tlsapi.CrashInfo
	// crashBlobs maps the blob's filename to the full blob struct. It is
	// used to combine files split across multiple blobs.
	crashBlobs := map[string]*tlsapi.CrashBlob{}
readStream:
	for {
		resp, err := stream.Recv()
		switch err {
		case nil:
			if lastCrashID != resp.CrashId {
				// We've started to receive a new crash, so we've gotten all of the prior one.
				// Process it.
				if lastCrashID != -1 {
					summary, err := processCrash(ctx, crashInfo, crashBlobs, r, outDir)
					if err != nil {
						logging.Errorf(ctx, "Failed to process crash: %s. Will continue to process others, if any.", err)
					} else {
						fetchResp.Crashes = append(fetchResp.Crashes, summary)
					}
				}
				lastCrashID = resp.CrashId
				crashInfo = nil
				crashBlobs = map[string]*tlsapi.CrashBlob{}
			}
			switch x := resp.Data.(type) {
			case *tlsapi.FetchCrashesResponse_Crash:
				if crashInfo != nil {
					logging.Errorf(ctx, "Found two CrashInfos for crash %d. Second exec: %s", lastCrashID, x.Crash.ExecName)
				} else {
					crashInfo = x.Crash
					logging.Infof(ctx, "Starting to process crash %d (exec %s)", lastCrashID, crashInfo.ExecName)
				}
			case *tlsapi.FetchCrashesResponse_Blob:
				// Reassemble the blob into one proto.
				if crashBlobs[x.Blob.Filename] == nil {
					crashBlobs[x.Blob.Filename] = x.Blob
				} else {
					crashBlobs[x.Blob.Filename].Blob = append(crashBlobs[x.Blob.Filename].Blob, x.Blob.Blob...)
				}
			case *tlsapi.FetchCrashesResponse_Core:
				logging.Errorf(ctx, "Unexpected coredump for crash %d. Ignoring.", lastCrashID)
			default:
				logging.Errorf(ctx, "Unexpected crash response of type %T for crash %d", x, lastCrashID)
			}
		case io.EOF:
			// Process the last crash, if any.
			if lastCrashID != -1 {
				summary, err := processCrash(ctx, crashInfo, crashBlobs, r, outDir)
				if err != nil {
					logging.Errorf(ctx, "Failed to process crash %d: %s.", lastCrashID, err)
				} else {
					fetchResp.Crashes = append(fetchResp.Crashes, summary)
				}
			}
			break readStream
		default:
			if status.Code(err) == codes.Unimplemented {
				return fetchResp, nil
			}
			return nil, errors.Annotate(err, "read TLS FetchCrashes response").Err()
		}
	}

	logging.Infof(ctx, "Finding missing crashes")
	fetchResp.CrashesRtdOnly, fetchResp.CrashesTlsOnly = findMissingCrashes(rtdCrashes, fetchResp.Crashes)

	if err := writeProcessedCrashDetails(ctx, outDir, fetchResp.Crashes); err != nil {
		// Don't return an error here, because we still successfully
		// processed the crashes.
		logging.Errorf(ctx, "Failed to write output details: %s", err)
	}

	// If we uploaded crashes, remove them now to prevent them from being
	// uploaded again.
	if r.UploadCrashes {
		logging.Infof(ctx, "Removing uploaded crashes")
		if err := removeCrashes(ctx, r, cl); err != nil {
			// Don't return an error here, because we still successfully
			// processed the crashes.
			logging.Errorf(ctx, "Failed to clean up crashes: %s", err)
		}
	}

	logging.Infof(ctx, "Completing successfully")

	return fetchResp, nil
}

// findMissingCrashes compares the crashes found in during this run to those written by the rtd.
// Returns:
//   |missing| - list of crashes found in rtdCrashes but not in crashes
//   |extra| - list of crashes found in crashes but not in rtdCrashes
func findMissingCrashes(rtdCrashes map[string]bool, crashes []*phosphorus.CrashSummary) (missing []string, extra []string) {
	for _, c := range crashes {
		if c.FilenameBase == "" {
			continue
		}
		if _, ok := rtdCrashes[c.FilenameBase]; !ok {
			extra = append(extra, c.FilenameBase)
		}
	}
	for k := range rtdCrashes {
		found := false
		for _, c := range crashes {
			if k == c.FilenameBase {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, k)
		}
	}
	return
}

// removeCrashes empties out all accessible crash directories on the DUT
func removeCrashes(ctx context.Context, r *phosphorus.FetchCrashesRequest, cl tlsapi.CommonClient) error {
	// Some of these directories may not exist (e.g. if the user isn't logged in), and that's okay.
	crashDirs := []string{"/var/spool/crash/", "/home/chronos/crash/", "/home/root/*/crash/", "/home/chronos/u-*/crash/"}

	req := &tlsapi.ExecDutCommandRequest{
		Name:    r.DutHostname,
		Command: fmt.Sprintf(`/usr/bin/find %s -maxdepth 1 -type f -delete`, strings.Join(crashDirs, " ")),
	}

	stream, err := cl.ExecDutCommand(ctx, req)
	if err != nil {
		return errors.Annotate(err, "remove crash files").Err()
	}

	// Ensure command finishes, but don't check exit code -- the find may fail if some directories don't exist, and that's okay.
readStream:
	for {
		_, err := stream.Recv()
		switch err {
		case nil:
			// do nothing
		case io.EOF:
			break readStream
		default:
			return errors.Annotate(err, "running ExecDutCommand").Err()
		}
	}
	return nil
}

// writeIndividualCrash writes an individual crash to |dir|/crashes, returning the base part of the crash name written
func writeIndividualCrash(ctx context.Context, info *tlsapi.CrashInfo, crashBlobs map[string]*tlsapi.CrashBlob, dir string) (string, error) {
	crashDir := filepath.Join(dir, "crashes")
	if err := os.MkdirAll(crashDir, 0755); err != nil {
		return "", errors.Annotate(err, "create output directory").Err()
	}
	// The metadata file name is not present in the CrashInfo file, so compute
	// it from the name of a crash blob.
	var metaName string
	var base string

	for _, b := range crashBlobs {
		path := filepath.Join(crashDir, b.Filename)
		if err := ioutil.WriteFile(path, b.Blob, 0644); err != nil {
			return "", errors.Annotate(err, "write blob %s", b.Filename).Err()
		}
		if base == "" {
			base = baseCrashNamePat.FindString(b.Filename)
			if base != "" {
				metaName = base + ".meta"
			}
		}
	}
	if metaName == "" {
		return "", errors.New("didn't extract metadata file name")
	}
	var metaContents string
	metaContents += "exec_name=" + info.ExecName + "\n"
	metaContents += "prod=" + info.Prod + "\n"
	metaContents += "ver=" + info.Ver + "\n"
	metaContents += "sig=" + info.Sig + "\n"
	metaContents += "in_progress_integration_test=" + info.InProgressIntegrationTest + "\n"
	metaContents += "collector=" + info.Collector + "\n"
	for _, meta := range info.Fields {
		metaContents += fmt.Sprintf("%s=%s\n", meta.Key, meta.Text)
	}
	if err := ioutil.WriteFile(filepath.Join(crashDir, metaName), []byte(metaContents), 0644); err != nil {
		return "", errors.Annotate(err, "write metadata %s", metaName).Err()
	}
	return base, nil
}

// writeProcessedCrashDetails writes details of the processed crashes to the output directory for debugging purposes
// (e.g. for browsing in stainless).
func writeProcessedCrashDetails(ctx context.Context, dir string, crashes []*phosphorus.CrashSummary) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Annotate(err, "create output directory").Err()
	}
	outFile := filepath.Join(dir, uploadedFile)

	// If the file doesn't exist, create it. Append if it exists.
	f, err := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Annotate(err, "open output file").Err()
	}
	defer f.Close()

	now := time.Now()
	var firstErr error
	for _, crash := range crashes {
		// Add timestamp of test run to make lines more unique (in case
		// there are multiple crashes from one executable, for
		// example). Use the same timestamp for all crashes from a run
		// to make it (slightly) more clear that it's the timestamp at
		// which the file was written, rather than the timstamp at
		// which the crash happened.
		if _, err := fmt.Fprintf(f, "%s: ", now.Format(time.RFC3339)); err != nil {
			// Continue looping in case later data can be written
			if firstErr == nil {
				firstErr = err
			}
			logging.Errorf(ctx, "Failed to write time: %s", err)
		}
		// Add jsonified proto.
		marshaler := jsonpb.Marshaler{}
		if err := marshaler.Marshal(f, crash); err != nil {
			// Continue looping in case later protos can be serialized.
			if firstErr == nil {
				firstErr = err
			}
			logging.Errorf(ctx, "Failed to write JSON pb: %s", err)
		}
		if _, err := fmt.Fprint(f, "\n"); err != nil {
			// Continue looping in case later data can be written
			if firstErr == nil {
				firstErr = err
			}
			logging.Errorf(ctx, "Failed to write newline: %s", err)
		}
	}

	return firstErr
}

// processCrash evaluates a fully-received crash, writes it to |dir|/crashes, uploads
// it if requested, and gives back an appropriate CrashSummary.
func processCrash(ctx context.Context, info *tlsapi.CrashInfo, crashBlobs map[string]*tlsapi.CrashBlob, r *phosphorus.FetchCrashesRequest, dir string) (*phosphorus.CrashSummary, error) {
	logging.Infof(ctx, "Processing full crash for exec %s (upload: %t)", info.ExecName, r.UploadCrashes)
	var url string
	if r.UploadCrashes {
		var blobs []*tlsapi.CrashBlob
		for _, b := range crashBlobs {
			blobs = append(blobs, b)
		}

		var err error
		url, err = uploadCrash(ctx, r.GetConfig().GetFetchCrashesStep(), fullCrash{info: info, blobs: blobs})
		if err != nil {
			return nil, errors.Annotate(err, "uploading crash for %s", info.ExecName).Err()
		}
	}

	base, err := writeIndividualCrash(ctx, info, crashBlobs, dir)
	if err != nil {
		logging.Errorf(ctx, "Failed to write crash: ", err)
	}

	summary := &phosphorus.CrashSummary{
		ExecName:                  info.ExecName,
		UploadUrl:                 url,
		Sig:                       info.Sig,
		InProgressIntegrationTest: info.InProgressIntegrationTest,
		FilenameBase:              base,
	}
	return summary, nil
}

// findRTDCrashes returns a list of crashes found by the RTD.
// Specifically, each item in the map maps the basename of a crash (e.g.
// chrome.20201202.130102.12345.4567) to the path to a .meta file for that crash
// (or, if the crash has no .meta file associated with it, any file associated
// with the crash).
func findRTDCrashes(ctx context.Context, rootDir string) (map[string]bool, error) {
	// Globs to directories where we know crashes to show up.
	crashGlobs := []string{
		"autoserv_test/sysinfo/var/spool/crash/",
		"autoserv_test/*/sysinfo/var/log/chrome/Crash Reports/",
		"autoserv_test/tast/results/crashes/", // no .meta files typically found here
		"autoserv_test/debug/",
		"provision_*/crashinfo.*/",
	}

	// Create a set of basenames w/out extensions
	crashes := make(map[string]bool)
	for _, g := range crashGlobs {
		dirs, err := filepath.Glob(filepath.Join(rootDir, g))
		// Per go docs, "The only possible returned error is ErrBadPattern, when pattern is malformed."
		if err != nil {
			return nil, errors.Annotate(err, "invalid glob").Err()
		}
		for _, d := range dirs {
			if err := addCrashesInDir(ctx, d, &crashes); err != nil {
				return nil, err
			}
		}
	}
	return crashes, nil
}

// addCrasheInDir finds all crashes in |d| and adds them to |crashes|
func addCrashesInDir(ctx context.Context, d string, crashes *map[string]bool) error {
	files, err := ioutil.ReadDir(d)
	if err != nil {
		return errors.Annotate(err, "reading directory %s", d).Err()
	}
	for _, f := range files {
		if !f.IsDir() {
			base := baseCrashNamePat.FindString(f.Name())
			if base != "" {
				(*crashes)[base] = true
			}
		}
	}
	return nil
}

// uploadCrash uploads the given crash to provided crash server url.
// It returns the URL at which the uploaded crash can be found.
func uploadCrash(ctx context.Context, config *phosphorus.FetchCrashesStep, crash fullCrash) (string, error) {
	buf, contentType, err := formatCrashForUpload(ctx, crash)
	if err != nil {
		return "", errors.Annotate(err, "build POST form").Err()
	}

	// Attempt to compress the data, falling back to sending uncompressed
	// data if compression fails.
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	didCompress := true
	if _, err := zw.Write(buf.Bytes()); err != nil {
		logging.Errorf(ctx, "Failed to compress data. Sending uncompressed.")
		didCompress = false
	} else if err := zw.Close(); err != nil {
		logging.Errorf(ctx, "Failed to compress data. Sending uncompressed.")
		didCompress = false
	}

	toUpload := compressed
	if !didCompress {
		toUpload = buf
	}

	// Do the Post request. cannot use http.Post() because it doesn't let us
	// specify the Content-Encoding header.
	req, err := http.NewRequest(http.MethodPost, config.CrashServerReportUrl, &toUpload)
	if err != nil {
		return "", errors.Annotate(err, "creating upload request").Err()
	}
	req.Header.Set("Content-Type", contentType)
	if didCompress {
		req.Header.Set("Content-Encoding", "gzip")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Annotate(err, "uploading crash").Err()
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.Annotate(err, "server returned %s", resp.Status).Err()
	}
	var id strings.Builder
	if _, err := io.Copy(&id, resp.Body); err != nil {
		return "", errors.Annotate(err, "server succeeded but could not read response body").Err()
	}
	resp.Body.Close()

	return config.CrashServerViewUrl + "/" + id.String(), nil
}

// formatCrashForUpload takes the provided crash and turns it into a byte slice suitable for POSTing to crash/.
// The format should match that of Sender::CreateCrashFormData in platform2/crash-reporter
func formatCrashForUpload(ctx context.Context, crash fullCrash) (buf bytes.Buffer, contentType string, err error) {
	w := multipart.NewWriter(&buf)
	defer w.Close()

	// create a new slice of fields to upload, including those that are
	// separated out in the proto. We use a new slice because we don't want
	// to edit the existing one in the proto.
	fieldsToUpload := []*tlsapi.CrashMetadata{
		{Key: "exec_name", Text: crash.info.ExecName},
		{Key: "prod", Text: crash.info.Prod},
		{Key: "ver", Text: crash.info.Ver},
		{Key: "sig", Text: crash.info.Sig},
		{Key: "in_progress_integration_test", Text: crash.info.InProgressIntegrationTest},
		{Key: "collector", Text: crash.info.Collector},
		// Add a special key to indicate that this was uploaded from a
		// hardware test run (to enable filtering of these crashes).
		{Key: "hwtest_suite_run", Text: "true"},
	}
	fieldsToUpload = append(fieldsToUpload, crash.info.Fields...)

	for _, meta := range fieldsToUpload {
		var fw io.Writer
		if fw, err = w.CreateFormField(meta.Key); err != nil {
			err = errors.Annotate(err, "add %s field", meta.Key).Err()
			return
		}
		if _, err = io.WriteString(fw, meta.Text); err != nil {
			err = errors.Annotate(err, "write %s field", meta.Text).Err()
			return
		}
	}

	var fw io.Writer
	for _, b := range crash.blobs {
		if fw, err = w.CreateFormFile(b.Key, b.Filename); err != nil {
			err = errors.Annotate(err, "add %s blob", b.Filename).Err()
			return
		}
		if _, err = io.Copy(fw, bytes.NewReader(b.Blob)); err != nil {
			err = errors.Annotate(err, "write %s field", b.Filename).Err()
			return
		}
	}

	contentType = w.FormDataContentType()
	return
}
