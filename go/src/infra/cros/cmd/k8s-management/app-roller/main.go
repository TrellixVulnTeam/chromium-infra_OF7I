// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command app-roller generates latest application YAML file and applies it to
// the K8s cluster.
// Because there are internal data in our k8s applications, we use a YAML file
// to pass them to app-roller. The YAML file looks like:
//
// - name: my_cool_app
//   source: https://chrome.googlesource.com/path/to/the/app/yaml/template
//   # clusters lists the K8s cluster which will run this app. When not
//   # specified, the app will run on all clusters.
//   clusters:
//   - <API_server_IP:API_server_port>
//   images:
//   - name: my_cool_image1
//     repo: gcr.io/project/image1
//     official_tag_regex: ^official-\d+$
//     tag: prod  # default tag is 'latest-official'
//   - name: my_cool_image2
//     repo: gcr.io/project/image2
//     official_tag_regex: ^official-\d+$
//
// The regex must begin with '^' and end with '$' in order to match the whole
// tag string strictly.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/jdxcode/netrc"
	"gopkg.in/yaml.v2"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("app roller: %s", err)
	}
	log.Printf("app-roller: Done")
}

func innerMain() error {
	var (
		serviceAccountJSON = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use")
		netrcPath          = flag.String("netrc", "", "Path to .netrc file used to access the Gerrit server")
		appsYAMLURL        = flag.String("apps-yaml-url", "", "URL to a yaml file which includes all applications data")
	)
	flag.Parse()

	content, err := os.ReadFile(*serviceAccountJSON)
	if err != nil {
		return fmt.Errorf("read credential %q: %s", *serviceAccountJSON, err)
	}
	auth := google.NewJSONKeyAuthenticator(string(content))

	var nr *netrc.Netrc
	if *netrcPath == "" {
		log.Printf("No netrc specified, continue without authorization against the source server")
	} else {
		nr, err = netrc.Parse(*netrcPath)
		if err != nil {
			return fmt.Errorf("parse netrc %q: %s", *netrcPath, err)
		}
	}
	downloader := &netrcClient{nr}
	apps, err := loadApps(downloader, *appsYAMLURL)
	if err != nil {
		return fmt.Errorf("load apps yaml %q: %s", *appsYAMLURL, err)
	}

	cluster, err := getClusterName()
	if err != nil {
		return err
	}
	ch := make(chan string, len(apps))
	var wg sync.WaitGroup
	for _, a := range apps {
		if len(a.Clusters) > 0 && !stringInSlice(cluster, a.Clusters) {
			log.Printf("Skip the rolling of %q to %q", a, cluster)
			continue
		}

		wg.Add(1)
		go func(a app) {
			defer wg.Done()
			if err := rolloutApp(a, auth, downloader); err != nil {
				log.Printf("Apply %q: %s", a, err)
				ch <- fmt.Sprintf("%q", a)
			}
		}(a)
	}
	wg.Wait()
	close(ch)

	var names []string
	for n := range ch {
		names = append(names, n)
	}
	if len(names) > 0 {
		return fmt.Errorf("failed to roll-out: %s", strings.Join(names, ", "))
	}
	return nil
}

// stringInSlice returns true if a string in a slice, otherwise false.
func stringInSlice(s string, slice []string) bool {
	for _, ss := range slice {
		if s == ss {
			return true
		}
	}
	return false
}

// loadApps load applications data from a yaml file.
func loadApps(d downloader, fileURL string) ([]app, error) {
	log.Printf("Download the applications config file from %q", fileURL)
	content, err := d.download(fileURL)
	if err != nil {
		return nil, fmt.Errorf("load apps from %q: %s", fileURL, err)
	}
	var apps []app
	if err := yaml.Unmarshal([]byte(content), &apps); err != nil {
		return nil, fmt.Errorf("load apps from %q: %s", fileURL, err)
	}
	return apps, nil
}

// rolloutApp generates application YAML file and apply to K8s.
func rolloutApp(a app, auth authn.Authenticator, d downloader) error {
	yamlTemplate, err := d.download(a.Source)
	if err != nil {
		return fmt.Errorf("roll out app %q: %s", a, err)
	}
	imageMap, err := resolveImages(a.Images, auth)
	if err != nil {
		return fmt.Errorf("roll out app %q: %s", a, err)
	}
	content, err := genAppYaml(yamlTemplate, imageMap)
	if err != nil {
		return fmt.Errorf("roll out app %q: %s", a, err)
	}
	yamlDocs, err := splitYAMLDoc(content)
	if err != nil {
		return fmt.Errorf("roll out app %q: %s", a, err)
	}
	for _, d := range yamlDocs {
		if err := applyToK8s(d); err != nil {
			return fmt.Errorf("roll out app %q: %s", a, err)
		}
	}
	return nil
}

// resolveImages resolves all images to their official tags of the app.
func resolveImages(images []image, auth authn.Authenticator) (map[string]string, error) {
	m := map[string]string{}
	for _, img := range images {
		obj, err := parseImage(img)
		if err != nil {
			return nil, fmt.Errorf("resolve images (%q): %s", img, err)
		}
		if _, ok := m[img.Name]; ok {
			return nil, fmt.Errorf("resolve images (%q): duplicate image name %q", img, img.Name)
		}
		officialTag, err := resolveImageToOfficial(obj, auth)
		if err != nil {
			return nil, fmt.Errorf("resolve images (%q): %s", img, err)
		}
		log.Printf("Resolved %q to %q", img, officialTag)
		m[img.Name] = officialTag
	}
	return m, nil
}

// genAppYaml generates real YAML file for an application.
func genAppYaml(yamlTemplate string, imageMap map[string]string) (string, error) {
	t, err := template.New("base").Parse(yamlTemplate)
	if err != nil {
		return "", fmt.Errorf("gen YAML: %s", err)
	}

	var buf bytes.Buffer
	t.Execute(&buf, imageMap)
	return buf.String(), nil
}

// app is an application which has a configuration template downloading from a
// remote source server and a series of container images.
type app struct {
	Name     string
	Source   string
	Clusters []string
	Images   []image `yaml:"images"`
}

func (a app) String() string { return a.Name }

// image is an official container image of an application.
type image struct {
	Name string
	// Repo is the container image repo, e.g.
	// "gcr.io/chromeos-drone-images/drone".
	Repo string
	// OfficialTagRegex is the regex of the image official tag.
	// The regex must begin with '^' and end with '$' in order to match the
	// whole tag string strictly.
	OfficialTagRegex string `yaml:"official_tag_regex"`
	// Tag is the we monitor, e.g. "prod". We will resolve it to a tag matches
	// the official tag regex.
	// Default tag is "latest-official".
	Tag string
}

func (i *image) String() string {
	return fmt.Sprintf("%s(%s:%s)", i.Name, i.Repo, i.Tag)
}

// parsedImage is a parsed image which has initialized objects.
type parsedImage struct {
	repo  imageRepo
	regex *regexp.Regexp
	tag   string
}

// parseImage parses an image and returns a parsedImage object.
func parseImage(img image) (*parsedImage, error) {
	if !strings.HasPrefix(img.OfficialTagRegex, "^") || !strings.HasSuffix(img.OfficialTagRegex, "$") {
		return nil, fmt.Errorf("parse image %q: the regex %q must start with ^ and end with $", img, img.OfficialTagRegex)
	}
	re, err := regexp.Compile(img.OfficialTagRegex)
	if err != nil {
		return nil, fmt.Errorf("parse image %q: %s", img, err)
	}

	// Set the default tag if it's not specified.
	tag := img.Tag
	if tag == "" {
		tag = latestOfficial
	}

	return &parsedImage{
		repo:  &gcrRepo{img.Repo},
		regex: re,
		tag:   tag,
	}, nil
}

// latestOfficial is the default image tag for an app.
const latestOfficial = "latest-official"

// resolveToOfficial resolves the image tag to a tag matching the official tag
// regex.
// There's no guarantee which one will be returned when there are multiple tags
// matches the official tag regex.
func resolveImageToOfficial(img *parsedImage, auth authn.Authenticator) (string, error) {
	allTags, err := img.repo.allTagsOnImage(auth, img.tag)
	if err != nil {
		return "", fmt.Errorf("resolve to official: %s", err)
	}
	for _, t := range allTags {
		if img.regex.Match([]byte(t)) {
			r := fmt.Sprintf("%s:%s", img.repo, t)
			return r, nil
		}
	}
	return "", fmt.Errorf("resolve to official: no official tag on image")
}

// downloader is the interface for a client which can download the YAML template
// from a server.
type downloader interface {
	// download downloads a file specified by url and returns it's content.
	download(url string) (content string, err error)
}

// netrcClient is a http client which can download files from a HTTP server
// using netrc auth.
type netrcClient struct{ nr *netrc.Netrc }

// download implements the download method of downloader interface.
func (n *netrcClient) download(strURL string) (string, error) {
	u, err := url.Parse(strURL)
	if err != nil {
		return "", fmt.Errorf("download %q: %s", strURL, err)
	}
	q := u.Query()
	q.Set("format", "TEXT")
	u.RawQuery = q.Encode()

	log.Printf("Downloading %q", strURL)
	c := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("download %q: %s", u, err)
	}
	n.setAuth(req, u.Hostname())

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("download from %q: %s", u, err)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return "", fmt.Errorf("download %q: status code %d", u, resp.StatusCode)
	}
	if err != nil {
		return "", fmt.Errorf("download %q: %s", u, err)
	}
	content, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return "", fmt.Errorf("download %q: %s", u, err)
	}
	return string(content), nil
}

// setAuth set basic auth to the request.
func (n *netrcClient) setAuth(req *http.Request, hostname string) {
	if n.nr == nil {
		log.Printf("No netrc specified, continue access %q without authorization", hostname)
		return
	}
	m := n.nr.Machine(hostname)
	if m != nil {
		req.SetBasicAuth(m.Get("login"), m.Get("password"))
	} else {
		log.Printf("Machine %q not found in netrc, continue without authorization", hostname)
	}
}

// imageRepo is the interface for a remote container image repo.
type imageRepo interface {
	// allTagsOnImage returns all tags of a image.
	allTagsOnImage(auth authn.Authenticator, tag string) (tags []string, err error)
}

type gcrRepo struct {
	name string
}

func (g gcrRepo) String() string { return g.name }

// allTagsOnImage implements the allTagsOnImage method of imageRepo interface.
func (g *gcrRepo) allTagsOnImage(auth authn.Authenticator, tag string) ([]string, error) {
	repo, err := name.NewRepository(g.name)
	if err != nil {
		return nil, fmt.Errorf("all tags on image %q:%q: %s", g, tag, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tags, err := google.List(repo, google.WithAuth(auth), google.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("all tags on image %q:%q: %s", g, tag, err)
	}
	for _, m := range tags.Manifests {
		for _, t := range m.Tags {
			if t == tag {
				return m.Tags, nil
			}
		}
	}
	return nil, fmt.Errorf("all tags on image %q: no images had the tag %q", g, tag)
}

// applyToK8s applies the generated YAML to K8s.
func applyToK8s(generatedYAML string) error {
	// TODO(guocb): log to BigQuery.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := k8sApply(ctx, generatedYAML); err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}
	return nil
}

// getClusterName gets the name of current K8s cluster.
func getClusterName() (string, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("get cluster name: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return "", fmt.Errorf("get cluster name: %s", err)
	}
	// We use the API server info (i.e. 'IP:port') as the cluster name.
	// See https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/discovery/discovery_client.go#L160
	// for how to get the API server info.
	v := &k8sMetaV1.APIVersions{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clientset.RESTClient().Get().AbsPath(clientset.LegacyPrefix).Do(ctx).Into(v); err != nil {
		return "", fmt.Errorf("get cluster name: %s", err)
	}
	if len(v.ServerAddressByClientCIDRs) == 0 {
		return "", fmt.Errorf("no data in ServerAddressByClientCIDRs")
	}
	return v.ServerAddressByClientCIDRs[0].ServerAddress, nil
}

// splitYAMLDoc splits the input YAML file content into multiple YAML documents
// separated by '---'.
func splitYAMLDoc(content string) ([]string, error) {
	dec := yaml.NewDecoder(bytes.NewReader([]byte(content)))
	var docs []string
	for {
		var v interface{}
		err := dec.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("split YAML doc: %s", err)
		}
		s, err := yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("split YAML doc: %s", err)
		}
		docs = append(docs, string(s))
	}
	return docs, nil
}
