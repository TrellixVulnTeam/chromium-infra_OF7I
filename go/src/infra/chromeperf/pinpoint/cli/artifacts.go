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
	"io"
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
	"gopkg.in/yaml.v2"
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

func (dam *downloadArtifactsMixin) doDownloadArtifacts(ctx context.Context, w io.Writer, httpClient *http.Client, workDir string, job *proto.Job) error {
	if !dam.downloadArtifacts || job.GetName() == "" {
		return nil
	}
	switch job.GetJobSpec().GetJobKind().(type) {
	case *proto.JobSpec_Bisection:
		return errors.Reason("Not implemented").Err()
	case *proto.JobSpec_Experiment:
		return dam.downloadExperimentArtifacts(ctx, w, httpClient, workDir, job)
	default:
		return errors.Reason("Unsupported Job Kind").Err()
	}
}

type artifactFile struct {
	Path   string `yaml:"path"`
	Source string `yaml:"source"`
}
type artifact struct {
	Selector string         `yaml:"selector"`
	Files    []artifactFile `yaml:"files"`
}

type changeConfig struct {
	Commit    string     `yaml:"commit"`
	Change    string     `yaml:"change"`
	ExtraArgs []string   `yaml:"extra_args"`
	Artifacts []artifact `yaml:"artifacts"`
}

type telemetryExperimentArtifactsManifest struct {
	JobID      string       `yaml:"job_id"`
	User       string       `yaml:"user"`
	Config     string       `yaml:"config"`
	Base       changeConfig `yaml:"base"`
	Experiment changeConfig `yaml:"experiment"`
}

func (dam *downloadArtifactsMixin) downloadExperimentArtifacts(ctx context.Context, w io.Writer, httpClient *http.Client, workDir string, job *proto.Job) error {
	id, err := pinpoint.LegacyJobID(job.Name)
	if err != nil {
		return errors.Annotate(err, "failed parsing pinpoint job name").Err()
	}
	dst, err := filepath.Abs(filepath.Join(workDir, id))
	if err != nil {
		return errors.Annotate(err, "failed getting absolute file path from %q %q", workDir, id).Err()
	}
	if err := promptRemove(os.Stdout, dst); err != nil {
		return errors.Annotate(err, "cannot download artifacts").Err()
	}

	urls := make(abExperimentURLs)
	m, err := urls.fromJob(job, dam.selectArtifacts)
	if err != nil {
		return errors.Annotate(err, "failed filtering urls").Err()
	}

	// Once we've validated the inputs, we'll proceed to downloading the
	// individual artifacts into a temporary directory, then rename it into the
	// destination directory.
	tmp, err := os.MkdirTemp(os.TempDir(), "pinpoint-cli-*")
	if err != nil {
		return errors.Annotate(err, "failed creating temporary directory").Err()
	}
	defer os.RemoveAll(tmp)
	isolatedclients := newIsolatedClientsCache(httpClient)
	if err := downloadIsolatedURLs(ctx, isolatedclients, tmp, urls); err != nil {
		return err
	}

	// At this point we'll emit the manifest file into the temporary directory,
	// so we can preserve some of the information from the Pinpoint job, mapping
	// the files to the relevant artifacts.
	manifest, err := yaml.Marshal(m)
	if err != nil {
		return errors.Annotate(err, "failed marshalling YAML for manifest").Err()
	}
	if err := os.WriteFile(filepath.Join(tmp, "manifest.yaml"), manifest, 0600); err != nil {
		return errors.Annotate(err, "failed writing manifest file").Err()
	}

	if err := os.Rename(tmp, dst); err != nil {
		return errors.Annotate(err, "failed renaming file from %q to %q", tmp, dst).Err()
	}
	fmt.Fprintf(w, "Downloaded all artifacts to: %q\n", dst)
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
		logging.Debugf(ctx, "Downloaded artifacts from %q", u)
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

type idStruct struct {
	commit *proto.GitilesCommit
	change *proto.GerritChange
	result *proto.ChangeResult
}

func generateResultMap(selector string, i idStruct) (abExperimentURLs, error) {
	// TODO: Maybe preserve the raw index information instead of relying on the path in the manifest.
	selectors := map[string]struct {
		label string
		key   string
	}{
		"test":  {"Test", "isolate"},
		"build": {"Build", "isolate"},
	}
	s, found := selectors[selector]
	if !found {
		return nil, errors.Reason("Unsupported selector: %s", selector).Err()
	}

	dirName := i.commit.GetGitHash()
	if i.change != nil {
		dirName += fmt.Sprintf("+%d", i.change.GetChange())
		if i.change.Patchset > 0 {
			dirName += fmt.Sprintf(".%d", i.change.GetPatchset())
		}
	}

	urls := abExperimentURLs{}
	for idx, a := range i.result.GetAttempts() {
		for _, e := range a.GetExecutions() {
			if e.GetLabel() == s.label {
				for _, d := range e.GetDetails() {
					if d.GetKey() == s.key {
						path := filepath.Join(dirName, selector, fmt.Sprintf("%02d", idx+1))
						if u, ok := urls[path]; ok {
							return nil, errors.Reason("URL %s and %s have conflicting key: %s", u, d.GetUrl(), path).Err()
						}
						urls[path] = d.GetUrl()
					}
				}
			}
		}
	}
	return urls, nil
}

func merge(to, from abExperimentURLs) error {
	for k, v := range from {
		if e, found := to[k]; found {
			return errors.Reason("key collision found for %q (current value = %q, merging value = %q)", k, e, v).Err()
		}
		to[k] = v
	}
	return nil
}

func (urls abExperimentURLs) fromJob(j *proto.Job, selector string) (*telemetryExperimentArtifactsManifest, error) {
	exp := j.GetJobSpec().GetExperiment()
	res := &telemetryExperimentArtifactsManifest{
		JobID:  j.Name,
		User:   j.CreatedBy,
		Config: j.JobSpec.Config,
		Base: changeConfig{
			Commit: j.JobSpec.GetExperiment().BaseCommit.GitHash,
			Change: func() string {
				if exp == nil {
					return ""
				}
				if exp.GetBasePatch() == nil {
					return ""
				}
				suffix := fmt.Sprintf("%d", exp.GetBasePatch().Change)
				if p := exp.GetBasePatch().Patchset; p > 0 {
					suffix += fmt.Sprintf(".%d", p)
				}
				return suffix
			}(),
			ExtraArgs: j.JobSpec.GetTelemetryBenchmark().GetExtraArgs(),
		},
		Experiment: changeConfig{
			Commit: j.JobSpec.GetExperiment().ExperimentCommit.GitHash,
			Change: func() string {
				if exp == nil {
					return ""
				}
				if exp.GetExperimentPatch() == nil {
					return ""
				}
				suffix := fmt.Sprintf("%d", exp.GetExperimentPatch().Change)
				if p := exp.GetExperimentPatch().Patchset; p > 0 {
					suffix += fmt.Sprintf("/%d", p)
				}
				return suffix
			}(),
			ExtraArgs: j.JobSpec.GetTelemetryBenchmark().GetExtraArgs(),
		},
	}
	for _, selector := range strings.Split(selector, ",") {
		for _, i := range []struct {
			change *changeConfig
			id     idStruct
		}{
			// Base configuration.
			{
				&res.Base,
				idStruct{
					exp.GetBaseCommit(),
					exp.GetBasePatch(),
					j.GetAbExperimentResults().GetAChangeResult(),
				},
			},
			// Experiment configuration.
			{
				&res.Experiment,
				idStruct{
					exp.GetExperimentCommit(),
					exp.GetExperimentPatch(),
					j.GetAbExperimentResults().GetBChangeResult(),
				},
			},
		} {
			m, err := generateResultMap(selector, i.id)

			if err != nil {
				return nil, err
			}
			err = merge(urls, m)
			if err != nil {
				return nil, err
			}
			a := artifact{
				Selector: selector,
			}
			for path, source := range m {
				a.Files = append(a.Files, artifactFile{Path: path, Source: source})
			}
			i.change.Artifacts = append(i.change.Artifacts, a)
		}
	}
	return res, nil
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
