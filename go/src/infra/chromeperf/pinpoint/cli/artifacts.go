// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint/proto"

	"go.chromium.org/luci/client/downloader"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/isolatedclient"
	"go.chromium.org/luci/common/logging"
)

type downloadArtifactsMixin struct {
	downloadArtifacts bool
	selectArtifacts   string
}

func (dam *downloadArtifactsMixin) RegisterFlags(flags *flag.FlagSet, userCfg userConfig) {
	flags.BoolVar(&dam.downloadArtifacts, "download-artifacts", userCfg.DownloadArtifacts, text.Doc(`
		If set, artifacts are downloaded to the -work-dir.  Note that files
		will NOT be overwritten if they exist already (an error will be
		printed).  Override default from user configuration file.
	`))
	flags.StringVar(&dam.selectArtifacts, "select-artifacts", userCfg.SelectArtifacts, text.Doc(`
		Ignored unless -download-artifacts is set; if set, only selected artifacts
		will be download.
	`))
}

func (dam *downloadArtifactsMixin) doDownloadArtifacts(ctx context.Context, httpClient *http.Client, workDir string, job *proto.Job) error {
	if !dam.downloadArtifacts || job.GetName() == "" {
		return nil
	}
	switch job.GetJobSpec().GetJobKind().(type) {
	case *proto.JobSpec_Bisection:
		return errors.Reason("Not implemented").Err()
	case *proto.JobSpec_Experiment:
		return dam.downloadExperimentArtifacts(ctx, httpClient, workDir, job)
	default:
		return errors.Reason("Unsupported Job Kind").Err()
	}
}

func (dam *downloadArtifactsMixin) downloadExperimentArtifacts(ctx context.Context, httpClient *http.Client, workDir string, job *proto.Job) error {
	urls := make(abExperimentURLs)
	if err := urls.fromJob(job, dam.selectArtifacts); err != nil {
		return errors.Annotate(err, "failed filtering urls").Err()
	}

	tmp, err := ioutil.TempDir(os.TempDir(), "pinpoint-cli-*")
	if err != nil {
		return errors.Annotate(err, "failed creating temporary directory").Err()
	}
	defer os.RemoveAll(tmp)
	isolatedclients := newIsolatedClientsCache(httpClient)
	if err := downloadIsolatedURLs(ctx, isolatedclients, tmp, urls); err != nil {
		return err
	}
	id, err := pinpoint.LegacyJobID(job.Name)
	if err != nil {
		return errors.Annotate(err, "failed parsing pinpoint job name").Err()
	}
	dst, err := filepath.Abs(filepath.Join(workDir, id))
	if err != nil {
		return errors.Annotate(err, "failed getting absolute file path from %q %q", workDir, id).Err()
	}
	if err := os.Rename(tmp, dst); err != nil {
		return errors.Annotate(err, "failed renaming file from %q to %q", tmp, dst).Err()
	}
	logging.Infof(ctx, "Downloaded artifacts %q", dst)
	return nil
}

func downloadIsolatedURLs(ctx context.Context, clients *isolatedClientsCache, base string, urls map[string]string) errors.MultiError {
	g := sync.WaitGroup{}
	errs := make(chan error)
	// Extract a downloader function which we'll run in a goroutine.
	downloader := func(path, u string, errs chan error, g *sync.WaitGroup) {
		logging.Debugf(ctx, "Downloading %q", u)
		defer logging.Debugf(ctx, "Done: %q", u)
		defer g.Done()
		obj, err := fromIsolatedURL(u)
		if err != nil {
			errs <- errors.Annotate(err, "failed parsing isolated url: %s", u).Err()
			return
		}
		dst := filepath.Join(base, path)
		if err := os.MkdirAll(dst, 0700); err != nil {
			errs <- errors.Annotate(err, "failed creating directory: %s", dst).Err()
			return
		}
		d := downloader.New(ctx, clients.Get(obj), isolated.HexDigest(obj.digest), dst, nil)
		d.Start()
		if err := d.Wait(); err != nil {
			errs <- errors.Annotate(err, "failed downloading artifacts from isolated: %s", u).Err()
			return
		}
		return
	}
	for path, u := range urls {
		g.Add(1)
		go downloader(path, u, errs, &g)
	}

	// Wait for all the downloader goroutines to finish, then close the errors channel.
	go func() {
		g.Wait()
		close(errs)
	}()

	// Handle all the errors from the channel until it's closed.
	res := errors.MultiError{}
	for err := range errs {
		res = append(res, err)
	}

	if len(res) == 0 {
		return nil
	}
	return res
}

type isolatedClientsCache struct {
	httpclient *http.Client
	clients    map[string]*isolatedclient.Client
}

func newIsolatedClientsCache(client *http.Client) *isolatedClientsCache {
	return &isolatedClientsCache{
		httpclient: client,
		clients:    make(map[string]*isolatedclient.Client),
	}
}

func (c *isolatedClientsCache) Get(obj *isolatedObject) *isolatedclient.Client {
	if clt, ok := c.clients[obj.clientKey()]; ok {
		return clt
	}
	clt := isolatedclient.NewClient(
		obj.host,
		isolatedclient.WithNamespace(obj.namespace),
		isolatedclient.WithAuthClient(c.httpclient),
	)
	c.clients[obj.clientKey()] = clt
	return clt
}

type abExperimentURLs map[string]string

func (urls abExperimentURLs) fromJob(j *proto.Job, selector string) error {
	for _, selector := range strings.Split(selector, ",") {
		if err := urls.appendResult(
			selector,
			j.GetJobSpec().GetExperiment().GetBaseCommit(),
			j.GetJobSpec().GetExperiment().GetBasePatch(),
			j.GetAbExperimentResults().GetAChangeResult(),
		); err != nil {
			return err
		}
		if err := urls.appendResult(
			selector,
			j.GetJobSpec().GetExperiment().GetExperimentCommit(),
			j.GetJobSpec().GetExperiment().GetExperimentPatch(),
			j.GetAbExperimentResults().GetBChangeResult(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (urls abExperimentURLs) appendResult(selector string, c *proto.GitilesCommit, p *proto.GerritChange, r *proto.ChangeResult) error {
	selectors := map[string]struct {
		label string
		key   string
	}{
		"test":  {"Test", "isolate"},
		"build": {"Build", "isolate"},
	}
	s, ok := selectors[selector]
	if !ok {
		return errors.Reason("Unsupported selector: %s", selector).Err()
	}

	dirName := c.GetGitHash()
	if p != nil {
		dirName += fmt.Sprintf("+%d/%d", p.GetChange(), p.GetPatchset())
	}

	for idx, a := range r.GetAttempts() {
		for _, e := range a.GetExecutions() {
			if e.GetLabel() == s.label {
				for _, d := range e.GetDetails() {
					if d.GetKey() == s.key {
						path := filepath.Join(dirName, selector, fmt.Sprintf("%02d", idx))
						if u, ok := urls[path]; ok {
							return errors.Reason("URL %s and %s have conflicting key: %s", u, d.GetUrl(), path).Err()
						}
						urls[path] = d.GetUrl()
					}
				}
			}
		}
	}
	return nil
}

type isolatedObject struct {
	host, namespace, digest string
}

func (o *isolatedObject) clientKey() string {
	return fmt.Sprintf("%s/%s", o.host, o.namespace)
}

func fromIsolatedURL(u string) (*isolatedObject, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	namespace := uu.Query().Get("namespace")
	if namespace == "" {
		namespace = "default-gzip"
	}
	return &isolatedObject{
		host:      fmt.Sprintf("%s://%s", uu.Scheme, uu.Host),
		namespace: namespace,
		digest:    uu.Query().Get("digest"),
	}, nil
}
