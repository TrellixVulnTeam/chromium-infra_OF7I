// Copyright 2021 The Chromium OS Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.

// Package main implements a distributed worker model for uploading debug
// symbols to the crash service. This package will be called by recipes through
// CIPD and will perform the buisiness logic of the builder.
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	lgs "go.chromium.org/luci/common/gcloud/gs"
	"infra/cros/internal/gs"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Default server URLs for the crash service.
	prodUploadUrl    = "https://prod-crashsymbolcollector-pa.googleapis.com/v1"
	stagingUploadUrl = "https://staging-crashsymbolcollector-pa.googleapis.com/v1"
	// Time in milliseconds to sleep before retrying the task.
	sleepTime = time.Second

	// TODO(juahurta): add constants for crash file size limits
)

// These structs are used in the interactions with the crash API.
type crashSymbolFile struct {
	DebugFile string `json:"debug_file"`
	DebugId   string `json:"debug_id"`
}
type crashUploadInformation struct {
	UploadUrl string `json:"uploadUrl"`
	UploadKey string `json:"uploadKey"`
}
type crashSubmitSymbolBody struct {
	SymbolId crashSymbolFile `json:"symbol_id"`
}
type crashSubmitResposne struct {
	Result string `json:"result"`
}

type filterDuplicatesRequestBody struct {
	SymbolIds []crashSymbolFile `json:"symbol_ids"`
}

// The bulk status checker endpoint uses a different schema.
type filterSymbolFileInfo struct {
	DebugFile string `json:"debugFile"`
	DebugId   string `json:"debugId"`
}
type filterResponseStatusPair struct {
	SymbolId filterSymbolFileInfo `json:"symbolId"`
	Status   string               `json:"status"`
}
type filterResponseBody struct {
	Pairs []filterResponseStatusPair `json:"pairs"`
}

// taskConfig will contain the information needed to complete the upload task.
type taskConfig struct {
	symbolPath  string
	debugFile   string
	debugId     string
	dryRun      bool
	shouldSleep bool
}

type uploadDebugSymbols struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	gsPath      string
	workerCount uint64
	retryQuota  uint64
	staging     bool
	dryRun      bool
}

type crashConnectionInfo struct {
	key    string
	url    string
	client crashClient
}

type crashClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// retrieveApiKey fetches the crash API key stored in the local keystore.
func retrieveApiKey() (string, error) {
	// TODO(juahurta): Locate and verify the key wanted on the bot.
	apiKey := "HIDDEN KEY"
	return apiKey, nil
}

// TODO(b/197010274): add function to strip CFI lines if file is too large

// httpRequest  builds the http request and executes it using the globally
// defined Client variable.
func httpRequest(url, httpRequestType string, data io.Reader, headers http.Header, client crashClient) (*http.Response, error) {
	request, err := http.NewRequest(httpRequestType, url, data)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("nil http repsonse was recieved")
	}
	return response, nil
}

// crashRetrieveUploadInformation retrieves a unique upload url and key for the given symbol
// file.
func crashRetrieveUploadInformation(uploadInfo *crashUploadInformation, crash crashConnectionInfo) error {
	// Crash endpoint with and without the key.
	requestUrlWithoutKey := crash.url + "/uploads:create"
	requestUrlWithKey := fmt.Sprintf("%s?key=%s", requestUrlWithoutKey, crash.key)
	response, err := httpRequest(requestUrlWithKey, http.MethodPost, nil, nil, crash.client)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("request to %s failed with status %d", requestUrlWithoutKey, response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	err = json.Unmarshal(body, &uploadInfo)
	if err != nil {
		return err
	}

	return nil
}

// crashUploadSymbol uploads the file to crash storage using the unique URL provided.
func crashUploadSymbol(symbolPath, uploadUrl string, crash crashConnectionInfo) error {
	// URL used in error message, actual url exposes keys and non-authenticated
	// upload urls.
	nonExposingUrl := "https://storage.googleapis.com/crashsymboldropzone/"
	// Open the file to be sent in the PUT request.
	file, err := os.Open(symbolPath)
	if err != nil {
		return err
	}
	defer file.Close()

	response, err := httpRequest(uploadUrl, http.MethodPut, file, nil, crash.client)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("request to %s failed with status %d", nonExposingUrl, response.StatusCode)
	}

	return nil
}

// crashSubmitSymbolUpload confirms the upload with the crash service.
func crashSubmitSymbolUpload(uploadKey string, task taskConfig, crash crashConnectionInfo) error {
	// Crash endpoints with and without the keys obsfusicated.
	requestUrlWithoutKey := crash.url + "/uploads/"
	requestUrlWithKey := fmt.Sprintf("%s%s:complete?key=%s", requestUrlWithoutKey, uploadKey, crash.key)

	bodyInfo := crashSubmitSymbolBody{
		crashSymbolFile{
			task.debugFile,
			task.debugId,
		},
	}

	body, err := json.Marshal(bodyInfo)
	if err != nil {
		return err
	}

	response, err := httpRequest(requestUrlWithKey, http.MethodPost, bytes.NewReader(body), nil, crash.client)
	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("request to %s failed with status %d", requestUrlWithoutKey, response.StatusCode)
	}
	// TODO(juahurta): Check body of the response for a success message
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	var uploadStatus crashSubmitResposne
	err = json.Unmarshal(responseBody, &uploadStatus)
	if err != nil {
		return err
	}

	if uploadStatus.Result == "RESULT_UNSPECIFIED" {
		return fmt.Errorf("upload could not be completed")
	}

	return nil
}

// upload will perform the upload of the symbol file to the crash service.
func upload(task *taskConfig, crash crashConnectionInfo) bool {
	message := bytes.Buffer{}
	uploadLogger := log.New(&message, fmt.Sprintf("Uploading %s\n", task.debugFile), logFlags)
	if task.dryRun {
		LogOut("Would have uploaded %s to crash", task.debugFile)
		return true
	}

	var uploadInfo crashUploadInformation
	err := crashRetrieveUploadInformation(&uploadInfo, crash)

	if err != nil {
		LogOut("error: %s", err.Error())
		return false
	}
	uploadLogger.Printf("%s: SUCCESS\n", crash.url)

	err = crashUploadSymbol(task.symbolPath, uploadInfo.UploadUrl, crash)

	if err != nil {
		LogOut("error: %s", err.Error())
		return false
	}
	uploadLogger.Printf("%s: SUCCESS\n", "https://storage.googleapis.com/crashsymboldropzone/")

	err = crashSubmitSymbolUpload(uploadInfo.UploadKey, *task, crash)

	if err != nil {
		LogOut("error: %s", err.Error())
		return false
	}
	uploadLogger.Printf("%s: SUCCESS\n", crash.url)
	LogOut(message.String())
	return true
}

// getFilename takes a file path, relative or full, and returns the filename.
func getFilename(filePath string) (string, error) {
	basePath := filepath.Base(filePath)

	if basePath == "." {
		return filePath, fmt.Errorf("error: given filepath %s is invalid", filePath)
	}

	return basePath, nil
}

// getCmdUploadDebugSymbols builds the CLI command and captures flags.
func getCmdUploadDebugSymbols(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "upload <options>",
		ShortDesc: "Upload debug symbols to crash.",
		CommandRun: func() subcommands.CommandRun {
			b := &uploadDebugSymbols{}
			b.authFlags = authcli.Flags{}
			b.Flags.StringVar(&b.gsPath, "gs-path", "", ("[Required] Url pointing to the GS " +
				"bucket storing the tarball."))
			b.Flags.Uint64Var(&b.workerCount, "worker-count", 64, ("Number of worker threads" +
				" to spawn."))
			b.Flags.Uint64Var(&b.retryQuota, "retry-quota", 200, ("Number of total upload retries" +
				" allowed."))
			b.Flags.BoolVar(&b.staging, "staging", false, ("Specifies if the builder" +
				" should push to the staging crash service or prod."))
			b.Flags.BoolVar(&b.dryRun, "dry-run", true, ("Specified whether network" +
				" operations should be dry ran."))
			b.authFlags.Register(b.GetFlags(), authOpts)
			return b
		}}
}

// generateClient handles the authentication of the user then generation of the
// client to be used by the gs module.
func generateClient(ctx context.Context, authOpts auth.Options) (*gs.ProdClient, error) {
	LogOut("Generating authenticated client for Google Storage access.")
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		return nil, err
	}

	gsClient, err := gs.NewProdClient(ctx, authedClient)
	if err != nil {
		return nil, err
	}
	return gsClient, err
}

// downloadZippedSymbols will download the tarball from google storage which contains all
// of the symbol files to be uploaded. Once downloaded it will return the local
// filepath to tarball.
func downloadZippedSymbols(client gs.Client, gsPath, workDir string) (string, error) {
	var destPath string

	if strings.HasSuffix(gsPath, ".tgz") {
		destPath = filepath.Join(workDir, "debug.tgz")
	} else {
		destPath = filepath.Join(workDir, "debug.tar.xz")
	}

	LogOut("Downloading compressed symbols from %s and storing in %s", gsPath, destPath)
	return destPath, client.Download(lgs.Path(gsPath), destPath)
}

// unzipSymbols will take the local path of the fetched tarball and then unpack it.
// It will then return a list of file paths pointing to the unpacked symbol
// files.

func unzipSymbols(inputPath, outputPath string) error {
	LogOut("Decompressing files from %s and storing %s", inputPath, outputPath)

	if strings.HasSuffix(inputPath, ".tar.xz") {
		// use tar CLI to extract xz tools
		cmd := exec.Command("unxz", inputPath)
		err := cmd.Run()
		if err != nil {
			return err
		}
		return nil
	}
	srcReader, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer srcReader.Close()

	destWriter, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer destWriter.Close()

	gzipReader, err := gzip.NewReader(srcReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	_, err = io.Copy(destWriter, gzipReader)

	return err
}

// unpackTarball will take the local path of the fetched tarball and then unpack
// it. It will then return a list of file paths pointing to the unpacked symbol
// files. Searches for .so.sym files.
func unpackTarball(inputPath, outputDir string) ([]string, error) {
	LogOut("Untarring files from %s and storing symbols in %s", inputPath, outputDir)
	retArray := []string{}

	// Open locally stored .tar file.
	srcReader, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer srcReader.Close()

	tarReader := tar.NewReader(srcReader)

	// Iterate through the tar file saving only the debug symbols.
	for {
		header, err := tarReader.Next()
		// End of file reached, terminate the loop smoothly.
		if err == io.EOF {
			break
		}
		// An error occurred fetching the next header.
		if err != nil {
			return nil, err
		}
		// The header indicates it's a file. Store and save the file if it is a
		// symbol file.
		if header.FileInfo().Mode().IsRegular() {
			// Check if the file is a symbol file.
			filename, err := getFilename(header.Name)
			if err != nil {
				return nil, err
			}
			if filepath.Ext(filename) != ".sym" {
				continue
			}

			destFilePath := filepath.Join(outputDir, filename)
			destFile, err := os.Create(destFilePath)
			if err != nil {
				return nil, err
			}

			retArray = append(retArray, destFilePath)

			// Write contents of the symbol file to local storage.
			_, err = io.Copy(destFile, tarReader)
			if err != nil {
				return nil, err
			}
		}
	}

	return retArray, err
}

// generateConfigs will take a list of strings with containing the paths to the
// unpacked symbol files. It will return a list of generated task configs
// alongside the communication channels to be used.

func generateConfigs(symbolFiles []string, retryQuota uint64, dryRun bool, crash crashConnectionInfo) ([]taskConfig, error) {
	LogOut("Generating %d task configs", len(symbolFiles))

	// The task should only sleep on retry.
	shouldSleep := false

	tasks := make([]taskConfig, len(symbolFiles))

	// Generate task configurations.
	for index, filepath := range symbolFiles {
		var debugId string

		// debugFile is used in the body of the call to the crash service.
		debugFilename, err := getFilename(filepath)
		if err != nil {
			return nil, err
		}

		// Get id from from the first line of the file.
		file, err := os.Open(filepath)
		if err != nil {
			return nil, err
		}
		lineScanner := bufio.NewScanner(file)

		if lineScanner.Scan() {
			line := strings.Split(lineScanner.Text(), " ")

			// 	The first line of the syms file will read like:
			// 	  MODULE Linux arm F4F6FA6CCBDEF455039C8DE869C8A2F40 blkid
			if len(line) != 5 {
				return nil, fmt.Errorf("error: incorrect first line format for symbol file %s", debugFilename)
			}

			debugId = strings.ReplaceAll(line[3], "-", "")
		}
		err = file.Close()
		if err != nil {
			return nil, err
		}

		tasks[index] = taskConfig{filepath, debugFilename, debugId, dryRun, shouldSleep}
	}

	// Filter out already uploaded debug symbols.
	tasks, err := filterTasksAlreadyUploaded(tasks, dryRun, crash)
	if err != nil {
		return nil, err
	}
	LogOut("%d of %d symbols were duplicates. %d symbols will be sent for upload.", (len(symbolFiles) - len(tasks)), len(symbolFiles), len(tasks))

	return tasks, nil
}

// filterTasksAlreadyUploaded will send a batch request to the crash service for
// file upload status. It will then filter out symbols that have already been
// uploaded, reducing our upload time.
// Variable name formatting comes from API definition.

func filterTasksAlreadyUploaded(tasks []taskConfig, dryRun bool, crash crashConnectionInfo) ([]taskConfig, error) {
	LogOut("Filtering out previously uploaded symbol files.")

	var responseInfo filterResponseBody
	symbols := filterDuplicatesRequestBody{SymbolIds: []crashSymbolFile{}}

	// Generate body for api call.
	for _, task := range tasks {
		symbols.SymbolIds = append(symbols.SymbolIds, crashSymbolFile{task.debugFile, task.debugId})
	}
	postBody, err := json.Marshal(symbols)
	if err != nil {
		LogErr(err.Error())
		return nil, err
	}

	header := http.Header{}
	header.Add("Content-Type", "application/json")
	response, err := httpRequest(fmt.Sprintf(crash.url+"/symbols:checkStatuses?key=%s", crash.key), http.MethodPost, bytes.NewReader(postBody), nil, crash.client)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return tasks, fmt.Errorf("request to %s failed with status %d", crash.client, response.StatusCode)
	}

	// Parse the response from the API.
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	err = json.Unmarshal(body, &responseInfo)
	if err != nil {
		return nil, err
	}

	// Convert to map for easier filtering.
	responseMap := make(map[string]string)

	for _, pair := range responseInfo.Pairs {
		responseMap[pair.SymbolId.DebugFile] = pair.Status
	}

	// Return a list of tasks that include the non-uploaded symbol files only.
	filtered := []taskConfig{}
	for _, task := range tasks {
		if responseMap[task.debugFile] != "FOUND" {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

// uploadSymbols is the main loop that will spawn goroutines that will handle
// the upload tasks. Should its worker fail it's upload and we have retries
// left, send the task to the end of the channel's buffer.
func uploadSymbols(tasks []taskConfig, maximumWorkers, retryQuota uint64,
	staging bool, crash crashConnectionInfo) (int, error) {
	// Number of tasks to process.
	tasksLeftToComplete := uint64(len(tasks))

	// If there are less tasks to complete than allotted workers, reduce the
	// worker count.
	if maximumWorkers > tasksLeftToComplete {
		maximumWorkers = tasksLeftToComplete
	}

	// This buffered channel will act as a queue for us to pull tasks from.
	taskQueue := make(chan taskConfig, tasksLeftToComplete)

	// Fill channel with tasks.
	for _, task := range tasks {
		taskQueue <- task
	}

	// currentWorkerCount will track how many workers are.
	currentWorkerCount := uint64(0)

	var waitgroup sync.WaitGroup

	// This is the main driver loop for the distributed worker design.
	for {
		// All tasks have been completed, close the channel and exit the loop.
		if tasksLeftToComplete == 0 {
			// Close the task queue.
			close(taskQueue)
			// Wait for all goroutines to finish then exit function.
			// All goroutines should be done by now but this is here just in case.
			waitgroup.Wait()
			return 0, nil
		}

		// Exceeded the allotted number of retries.
		if retryQuota == 0 {
			return 1, fmt.Errorf("error: too many retries taken")
		}

		if currentWorkerCount >= maximumWorkers {
			continue
		}

		// Perform a non-blocking check for a task in the queue.
		select {
		// If there is a task in the queue, create a worker to handle it.
		case task := <-taskQueue:
			atomic.AddUint64(&currentWorkerCount, uint64(1))
			waitgroup.Add(1)

			// Spawn a worker to handle the task.
			go func() {
				defer waitgroup.Done()
				if task.shouldSleep {
					time.Sleep(sleepTime)
				}

				// Since golang is pass by value we don't need to wory about sending
				// crash here.
				uploadSuccessful := upload(&task, crash)

				// If the task failed, toss the task to the end of the queue.
				if !uploadSuccessful {
					task.shouldSleep = true
					taskQueue <- task
					// Decrement the retryQuota we have left.
					atomic.AddUint64(&retryQuota, ^uint64(0))
				} else {
					// If the worker completed the task successfully, decrement the
					// tasksLeftToComplete counter.
					atomic.AddUint64(&tasksLeftToComplete, ^uint64(0))
				}
				// Remove a worker from the current pool.
				atomic.AddUint64(&currentWorkerCount, ^uint64(0))
			}()
		default:
			continue
		}
	}
}

// validate checks the values of the required flags and returns an error they
// aren't populated. Since multiple flags are required, the error message may
// include multiple error statements.
func (b *uploadDebugSymbols) validate() error {
	errStr := ""
	if b.gsPath == "" {
		return fmt.Errorf("error: -gs-path value is required")
	}
	if !strings.HasPrefix(b.gsPath, "gs://") {
		return fmt.Errorf("error: -gs-path must point to a google storage location. E.g. gs://some-bucket/debug.tgz")
	}
	if !(strings.HasSuffix(b.gsPath, ".tgz") || strings.HasSuffix(b.gsPath, ".tar.xz")) {
		return fmt.Errorf("error: -gs-path must point to a compressed tar file. %s", b.gsPath)
	}
	if b.workerCount <= 0 {
		return fmt.Errorf("error: -worker-count value must be greater than zero")
	}
	if b.retryQuota == 0 {
		return fmt.Errorf("error: -retry-count value may not be zero")
	}

	if errStr != "" {
		return fmt.Errorf(errStr)
	}
	return nil
}

func initCrashConnectionInfo(crash *crashConnectionInfo, staging bool) error {
	crash.client = &http.Client{}

	if staging {
		crash.url = stagingUploadUrl
	} else {
		crash.url = prodUploadUrl
	}

	apiKey, err := retrieveApiKey()
	if err != nil {
		return err
	}

	crash.key = apiKey
	return nil
}

// Run is the function to be called by the CLI execution. TODO(b/197010274):
// Move business logic into a separate function so Run() can be tested fully.
func (b *uploadDebugSymbols) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ret := SetUp(b, a, args, env)
	if ret != 0 {
		return ret
	}

	var crash crashConnectionInfo
	err := initCrashConnectionInfo(&crash, b.staging)
	if err != nil {
		LogErr(err.Error())
		return 1
	}

	// Generate authenticated http client.
	ctx := context.Background()
	authOpts, err := b.authFlags.Options()
	if err != nil {
		LogErr(err.Error())
		return 1
	}
	authClient, err := generateClient(ctx, authOpts)
	if err != nil {
		LogErr(err.Error())
		return 1
	}
	// Create local dir and file for tarball to live in.
	workDir, err := ioutil.TempDir("", "tarball")
	LogOut("Creating working folder at %s", workDir)
	if err != nil {
		LogErr(err.Error())
		return 1
	}
	symbolDir, err := ioutil.TempDir(workDir, "symbols")
	if err != nil {
		LogErr(err.Error())
		return 1
	}

	tarbalPath := filepath.Join(workDir, "debug.tar")
	defer os.RemoveAll(workDir)

	zippedSymbolsPath, err := downloadZippedSymbols(authClient, b.gsPath, workDir)
	if err != nil {
		LogErr(err.Error())
		return 1
	}
	err = unzipSymbols(zippedSymbolsPath, tarbalPath)
	if err != nil {
		LogErr(err.Error())
		return 1
	}

	symbolFiles, err := unpackTarball(tarbalPath, symbolDir)
	if err != nil {
		LogErr(err.Error())
		return 1
	}

	tasks, err := generateConfigs(symbolFiles, b.retryQuota, b.dryRun, crash)
	if err != nil {
		LogErr(err.Error())
		return 1
	}

	retcode, err := uploadSymbols(tasks, b.workerCount, b.retryQuota, b.staging, crash)
	if err != nil {
		LogErr(err.Error())
		return 1
	}
	LogOut("Exiting with retcode: %d", retcode)
	return retcode
}
