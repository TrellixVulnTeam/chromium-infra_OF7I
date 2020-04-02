// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package manifest defines structure of YAML files with target definitions.
package manifest

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"go.chromium.org/luci/common/errors"
)

// Manifest is a definition of what to build, how and where.
//
// Comments here describe the structure of the manifest file on disk. In the
// loaded form all paths use filepath.Separator as a directory separator.
type Manifest struct {
	// Name is the name of this target, required.
	//
	// When building Docker images it is an image name (without registry or any
	// tags).
	Name string `yaml:"name"`

	// ManifestDir is a directory that contains this manifest file.
	//
	// Populated when it is loaded.
	ManifestDir string `yaml:"-"`

	// Extends is a unix-style path (relative to this YAML file) to a manifest
	// used as a base.
	//
	// Optional.
	//
	// Such base manifests usually contain definitions shared by many files, such
	// as "imagepins" and "infra".
	//
	// Dicts are merged (recursively), lists are joined (base entries first).
	Extends string `yaml:"extends,omitempty"`

	// Dockerfile is a unix-style path to the image's Dockerfile, relative to this
	// YAML file.
	//
	// Presence of this field indicates that the manifest describes how to build
	// a docker image. If its missing, docker related subcommands won't work.
	//
	// All images referenced in this Dockerfile are resolved into concrete digests
	// via an external file. See ImagePins field for more information.
	Dockerfile string `yaml:"dockerfile,omitempty"`

	// ContextDir is a unix-style path to the directory to use as a basis for
	// the build. The path is relative to this YAML file.
	//
	// All files there end up available to the remote builder (e.g. a docker
	// daemon will see this directory as a context directory when building
	// the image).
	//
	// All symlinks there are resolved to their targets. Only +w and +x file mode
	// bits are preserved (all files have 0444 mode by default, +w adds additional
	// 0200 bit and +x adds additional 0111 bis). All other file metadata (owners,
	// setuid bits, modification times) are ignored.
	//
	// The default value depends on whether Dockerfile is set. If it is, then
	// ContextDir defaults to the directory with Dockerfile. Otherwise the context
	// directory is assumed to be empty.
	ContextDir string `yaml:"contextdir,omitempty"`

	// InputsDir is an optional directory that can be used to reference files
	// consumed by build steps (as "${inputsdir}/path").
	//
	// Unlike ContextDir, its full content does not automatically end up in the
	// output.
	InputsDir string `yaml:"inputsdir,omitempty"`

	// ImagePins is a unix-style path to the YAML file with pre-resolved mapping
	// from (docker image, tag) pair to the corresponding docker image digest.
	//
	// The path is relative to the manifest YAML file. It should point to a YAML
	// file with the following structure:
	//
	//    pins:
	//      - image: <img>
	//        tag: <tag>
	//        digest: sha256:<sha256>
	//      - image: <img>
	//        tag: <tag>
	//        digest: sha256:<sha256>
	//      ...
	//
	// See dockerfile.Pins struct for more details.
	//
	// This file will be used to rewrite the input Dockerfile to reference all
	// images (in "FROM ..." lines) only by their digests. This is useful for
	// reproducibility of builds.
	//
	// Only following forms of "FROM ..." statement are allowed:
	//  * FROM <image> [AS <name>] (assumes "latest" tag)
	//  * FROM <image>[:<tag>] [AS <name>] (resolves the given tag)
	//  * FROM <image>[@<digest>] [AS <name>] (passes the definition through)
	//
	// In particular ARGs in FROM line (e.g. "FROM base:${CODE_VERSION}") are
	// not supported.
	//
	// If not set, the Dockerfile must use only digests to begin with, i.e.
	// all FROM statements should have form "FROM <image>@<digest>".
	//
	// Ignored if Dockerfile field is not set.
	ImagePins string `yaml:"imagepins,omitempty"`

	// Deterministic is true if Dockerfile (with all "FROM" lines resolved) can be
	// understood as a pure function of inputs in ContextDir, i.e. it does not
	// depend on the state of the world.
	//
	// Examples of things that make Dockerfile NOT deterministic:
	//   * Using "apt-get" or any other remote calls to non-pinned resources.
	//   * Cloning repositories from "master" ref (or similar).
	//   * Fetching external resources using curl or wget.
	//
	// When building an image marked as deterministic, the builder will calculate
	// a hash of all inputs (including resolve Dockerfile itself) and check
	// whether there's already an image built from them. If there is, the build
	// will be skipped completely and the existing image reused.
	//
	// Images marked as non-deterministic are always rebuilt and reuploaded, even
	// if nothing in ContextDir has changed.
	Deterministic *bool `yaml:"deterministic,omitempty"`

	// Infra is configuration of the build infrastructure to use: Google Storage
	// bucket, Cloud Build project, etc.
	//
	// Keys are names of presets (like "dev", "prod"). What preset is used is
	// controlled via "-infra" command line flag (defaults to "dev").
	Infra map[string]Infra `yaml:"infra"`

	// Build defines a series of local build steps.
	//
	// Each step may add more files to the context directory. The actual
	// `contextdir` directory on disk won't be modified. Files produced here are
	// stored in a temp directory and the final context directory is constructed
	// from the full recursive copy of `contextdir` and files emitted here.
	Build []*BuildStep `yaml:"build,omitempty"`
}

// Infra contains configuration of build infrastructure to use: Google Storage
// bucket, Cloud Build project, etc.
type Infra struct {
	// Storage specifies Google Storage location to store *.tar.gz tarballs
	// produced after executing all local build steps.
	//
	// Expected format is "gs://<bucket>/<prefix>". Tarballs will be stored as
	// "gs://<bucket>/<prefix>/<name>/<sha256>.tar.gz", where <name> comes from
	// the manifest and <sha256> is a hex sha256 digest of the tarball.
	//
	// The bucket should exist already. Its contents is trusted, i.e. if there's
	// an object with desired <sha256>.tar.gz there already, it won't be replaced.
	//
	// Required when using Cloud Build.
	Storage string `yaml:"storage"`

	// Registry is a Cloud Registry to push images to e.g. "gcr.io/something".
	//
	// If empty, images will be built and then just discarded (will not be pushed
	// anywhere). Useful to verify Dockerfile is working without accumulating
	// cruft.
	Registry string `yaml:"registry"`

	// CloudBuild contains configuration of Cloud Build infrastructure.
	CloudBuild CloudBuildConfig `yaml:"cloudbuild"`
}

// rebaseOnTop implements "extends" logic.
func (i *Infra) rebaseOnTop(b Infra) {
	setIfEmpty(&i.Storage, b.Storage)
	setIfEmpty(&i.Registry, b.Registry)
	i.CloudBuild.rebaseOnTop(b.CloudBuild)
}

// CloudBuildConfig contains configuration of Cloud Build infrastructure.
type CloudBuildConfig struct {
	Project string `yaml:"project"` // name of Cloud Project to use for builds
	Docker  string `yaml:"docker"`  // version of "docker" tool to use for builds
}

// rebaseOnTop implements "extends" logic.
func (c *CloudBuildConfig) rebaseOnTop(b CloudBuildConfig) {
	setIfEmpty(&c.Project, b.Project)
	setIfEmpty(&c.Docker, b.Docker)
}

// BuildStep is one local build operation.
//
// It takes a local checkout and produces one or more output files put into
// the context directory.
//
// This struct is a "case class" with union of all supported build step kinds.
// The chosen "case" is returned by Concrete() method.
type BuildStep struct {
	// Fields common to two or more build step kinds.

	// Dest specifies a location to put the result into.
	//
	// Applies to `copy`, `go_build` and `go_gae_bundle` steps.
	//
	// Usually prefixed with "${contextdir}/" to indicate it is relative to
	// the context directory.
	//
	// Optional in the original YAML, always populated after Manifest is parsed.
	// See individual *BuildStep structs for defaults.
	Dest string `yaml:"dest,omitempty"`

	// Disjoint set of possible build kinds.
	//
	// To add a new step kind:
	//   1. Add a new embedded struct here with definition of the step.
	//   2. Add methods to implement ConcreteBuildStep.
	//   3. Add one more entry to a slice in wireStep(...) below.
	//   4. Add the actual step implementation to builder/step*.go.
	//   5. Add one more type switch to Builder.Build() in builder/builder.go.

	CopyBuildStep        `yaml:",inline"` // copy a file or directory into the output
	GoBuildStep          `yaml:",inline"` // build go binary using "go build"
	RunBuildStep         `yaml:",inline"` // run a command that modifies the checkout
	GoGAEBundleBuildStep `yaml:",inline"` // bundle Go source code for GAE

	manifest *Manifest         // the manifest that defined this step
	index    int               // zero-based index of the step in its parent manifest
	concrete ConcreteBuildStep // pointer to one of *BuildStep above
}

// ConcreteBuildStep is implemented by various *BuildStep structs.
type ConcreteBuildStep interface {
	String() string // used for human logs only, doesn't have to encode all details

	isEmpty() bool                                        // true if the struct is not populated
	initStep(bs *BuildStep, dirs map[string]string) error // populates 'bs' and self
}

// Concrete returns a pointer to some concrete populated *BuildStep.
func (bs *BuildStep) Concrete() ConcreteBuildStep { return bs.concrete }

// CopyBuildStep indicates we want to copy a file or directory.
//
// Doesn't materialize copies on disk, just puts them directly into the output
// file set.
type CopyBuildStep struct {
	// Copy is a path to copy files from.
	//
	// Should start with either "${contextdir}/", "${inputsdir}/" or
	// "${manifestdir}/" to indicate the root path.
	//
	// Can either be a directory or a file. Whatever it is, it will be put into
	// the output as Dest. By default Dest is "${contextdir}/<basename of Copy>"
	// (i.e. we copy Copy into the root of the context dir).
	Copy string `yaml:"copy,omitempty"`
}

func (s *CopyBuildStep) String() string { return fmt.Sprintf("copy %q", s.Copy) }

func (s *CopyBuildStep) isEmpty() bool { return s.Copy == "" }

func (s *CopyBuildStep) initStep(bs *BuildStep, dirs map[string]string) (err error) {
	if s.Copy, err = renderPath("copy", s.Copy, dirs); err != nil {
		return err
	}
	if bs.Dest == "" {
		bs.Dest = "${contextdir}/" + filepath.Base(s.Copy)
	}
	if bs.Dest, err = renderPath("dest", bs.Dest, dirs); err != nil {
		return err
	}
	return
}

// GoBuildStep indicates we want to build a go command binary.
//
// Doesn't materialize the build output on disk, just puts it directly into the
// output file set.
type GoBuildStep struct {
	// GoBinary specifies a go command binary to build.
	//
	// This is a path (relative to GOPATH) to some 'main' package. It will be
	// built roughly as:
	//
	//  $ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build <go_binary> -o <dest>
	//
	// Where <dest> is taken from Dest and it must be under the context directory.
	// It is set to "${contextdir}/<go package name>" by default.
	GoBinary string `yaml:"go_binary,omitempty"`
}

func (s *GoBuildStep) String() string { return fmt.Sprintf("go build %q", s.GoBinary) }

func (s *GoBuildStep) isEmpty() bool { return s.GoBinary == "" }

func (s *GoBuildStep) initStep(bs *BuildStep, dirs map[string]string) (err error) {
	if bs.Dest == "" {
		bs.Dest = "${contextdir}/" + path.Base(s.GoBinary)
	}
	bs.Dest, err = renderPath("dest", bs.Dest, dirs)
	return
}

// RunBuildStep indicates we want to run some arbitrary command.
//
// The command may modify the checkout or populate the context dir.
type RunBuildStep struct {
	// Run indicates a command to run along with all its arguments.
	//
	// Strings that start with "${contextdir}/", "${inputsdir}/" or
	// "${manifestdir}/" will be rendered as absolute paths.
	Run []string `yaml:"run,omitempty"`

	// Cwd is a working directory to run the command in.
	//
	// Default is ${contextdir}.
	Cwd string `yaml:"cwd,omitempty"`

	// Outputs is a list of files or directories to put into the output.
	//
	// They are something that `run` should be generating.
	//
	// They are expected to be under "${contextdir}". A single output entry
	// "${contextdir}/generated/file" is equivalent to a copy step that "picks up"
	// the generated file:
	//   - copy: ${contextdir}/generated/file
	//     dest: ${contextdir}/generated/file
	//
	// If outputs are generated outside of the context directory, use `copy` steps
	// explicitly.
	Outputs []string
}

func (s *RunBuildStep) String() string { return fmt.Sprintf("run %q in %q", s.Run, s.Cwd) }

func (s *RunBuildStep) isEmpty() bool { return len(s.Run) == 0 && s.Cwd == "" && len(s.Outputs) == 0 }

func (s *RunBuildStep) initStep(bs *BuildStep, dirs map[string]string) (err error) {
	if len(s.Run) == 0 {
		return errors.Reason("bad `run` value: must not be empty").Err()
	}

	for i, val := range s.Run {
		if isTemplatedPath(val) {
			rel, err := renderPath(fmt.Sprintf("run[%d]", i), val, dirs)
			if err != nil {
				return err
			}
			// We are going to pass these arguments to a command with different cwd,
			// need to make sure they are absolute.
			if s.Run[i], err = filepath.Abs(rel); err != nil {
				return errors.Annotate(err, "bad `run[%d]` %q", i, rel).Err()
			}
		}
	}

	if s.Cwd == "" {
		s.Cwd = "${contextdir}"
	}
	if s.Cwd, err = renderPath("cwd", s.Cwd, dirs); err != nil {
		return err
	}

	for i, out := range s.Outputs {
		if s.Outputs[i], err = renderPath(fmt.Sprintf("output[%d]", i), out, dirs); err != nil {
			return err
		}
	}

	return
}

// GoGAEBundleBuildStep can be used to prepare a tarball with Go GAE app source.
//
// Given an input directory that points to some `main` go package, it:
//   * Non-recursively copies *.go files there to `Dest`.
//   * Recursively copies all other files there to `Dest`.
//   * Copies all *.go code with transitive dependencies to `_gopath/src/`.
//   * Puts the import path of the package into `Dest/.gaedeploy.json`.
//
// This ensures "gcloud app deploy" eventually can upload all *.go files needed
// to deploy a module.
type GoGAEBundleBuildStep struct {
	// GoGAEBundle is path to some 'main' go package to bundle.
	GoGAEBundle string `yaml:"go_gae_bundle,omitempty"`
}

func (s *GoGAEBundleBuildStep) String() string { return fmt.Sprintf("go gae bundle %q", s.GoGAEBundle) }

func (s *GoGAEBundleBuildStep) isEmpty() bool { return s.GoGAEBundle == "" }

func (s *GoGAEBundleBuildStep) initStep(bs *BuildStep, dirs map[string]string) (err error) {
	if s.GoGAEBundle, err = renderPath("go_gae_bundle", s.GoGAEBundle, dirs); err != nil {
		return
	}
	bs.Dest, err = renderPath("dest", bs.Dest, dirs)
	return
}

// Load loads the manifest from the given path, traversing all "extends" links.
//
// After the manifest is loaded, its fields (like ContextDir) can be manipulated
// (e.g. to set defaults), after which all "${dir}/" references in build steps
// must be resolved by a call to RenderSteps.
func Load(path string) (*Manifest, error) {
	return loadRecursive(path, 0)
}

// parse reads the manifest and populates paths there.
//
// If cwd is not empty, rebases all relative paths in it on top of it.
//
// Does not traverse "extends" links.
func parse(r io.Reader, cwd string) (*Manifest, error) {
	body, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "failed to read the manifest body").Err()
	}
	out := Manifest{}
	if err = yaml.Unmarshal(body, &out); err != nil {
		return nil, errors.Annotate(err, "failed to parse the manifest").Err()
	}
	if err := out.initBase(cwd); err != nil {
		return nil, err
	}
	return &out, nil
}

// loadRecursive implements Load by tracking how deep we go as a simple
// protection against recursive "extends" links.
func loadRecursive(path string, fileCount int) (*Manifest, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, errors.Annotate(err, "when opening manifest file").Err()
	}
	defer r.Close()

	m, err := parse(r, filepath.Dir(path))
	switch {
	case err != nil:
		return nil, errors.Annotate(err, "when parsing %q", path).Err()
	case m.Extends == "":
		return m, nil
	case fileCount > 10:
		return nil, errors.Reason("too much nesting").Err()
	}

	base, err := loadRecursive(m.Extends, fileCount+1)
	if err != nil {
		return nil, errors.Annotate(err, "when loading %q", path).Err()
	}
	m.rebaseOnTop(base)
	return m, nil
}

// initBase initializes pointers in steps and rebases paths.
//
// Doesn't yet touch actual bodies of steps, they will be initialized later
// when the whole manifest tree is loaded, see RenderSteps.
func (m *Manifest) initBase(cwd string) error {
	if err := validateName(m.Name); err != nil {
		return errors.Annotate(err, `bad "name" field`).Err()
	}
	m.ManifestDir = cwd
	normPath(&m.Extends, cwd)
	normPath(&m.Dockerfile, cwd)
	normPath(&m.ContextDir, cwd)
	normPath(&m.InputsDir, cwd)
	normPath(&m.ImagePins, cwd)
	if m.ContextDir == "" && m.Dockerfile != "" {
		m.ContextDir = filepath.Dir(m.Dockerfile)
	}
	for k, v := range m.Infra {
		if err := validateInfra(v); err != nil {
			return errors.Annotate(err, "in infra section %q", k).Err()
		}
	}
	for i, b := range m.Build {
		if err := wireStep(b, m, i); err != nil {
			return errors.Annotate(err, "bad build step #%d", i+1).Err()
		}
	}
	return nil
}

// RenderSteps replaces "${dir}/" in paths in steps with actual values.
func (m *Manifest) RenderSteps() error {
	for _, b := range m.Build {
		dirs := map[string]string{
			"contextdir":  m.ContextDir,
			"inputsdir":   m.InputsDir,
			"manifestdir": b.manifest.ManifestDir,
		}
		if err := b.concrete.initStep(b, dirs); err != nil {
			return errors.Annotate(err, "bad build step #%d in %q", b.index+1, b.manifest.ManifestDir).Err()
		}
	}
	return nil
}

// rebaseOnTop implements "extends" logic.
func (m *Manifest) rebaseOnTop(b *Manifest) {
	m.Extends = "" // resolved now

	setIfEmpty(&m.Dockerfile, b.Dockerfile)
	setIfEmpty(&m.ContextDir, b.ContextDir)
	setIfEmpty(&m.InputsDir, b.InputsDir)
	setIfEmpty(&m.ImagePins, b.ImagePins)
	if m.Deterministic == nil && b.Deterministic != nil {
		cpy := *b.Deterministic
		m.Deterministic = &cpy
	}

	// Rebase all entries already present in 'm' on top of entries in 'b'.
	for k, v := range m.Infra {
		if base, ok := b.Infra[k]; ok {
			v.rebaseOnTop(base)
			m.Infra[k] = v
		}
	}
	// Copy all entries in 'b' that are not in 'm'.
	for k, v := range b.Infra {
		if _, ok := m.Infra[k]; !ok {
			if m.Infra == nil {
				m.Infra = make(map[string]Infra, 1)
			}
			m.Infra[k] = v
		}
	}

	// Steps are just joined (base ones first).
	m.Build = append(b.Build, m.Build...)
}

func setIfEmpty(a *string, b string) {
	if *a == "" {
		*a = b
	}
}

// validateName validates "name" field in the manifest.
func validateName(t string) error {
	const forbidden = "\\:@"
	switch {
	case t == "":
		return errors.Reason("can't be empty, it's required").Err()
	case strings.ContainsAny(t, forbidden):
		return errors.Reason("%q contains forbidden symbols (any of %q)", t, forbidden).Err()
	default:
		return nil
	}
}

func validateInfra(i Infra) error {
	if i.Storage != "" {
		url, err := url.Parse(i.Storage)
		if err != nil {
			return errors.Annotate(err, "bad storage %q", i.Storage).Err()
		}
		switch {
		case url.Scheme != "gs":
			return errors.Reason("bad storage %q, only gs:// is supported currently", i.Storage).Err()
		case url.Host == "":
			return errors.Reason("bad storage %q, bucket name is missing", i.Storage).Err()
		}
	}
	return nil
}

// wireStep initializes `concrete` and `manifest` pointers in the step.
//
// Doesn't touch any other fields.
func wireStep(bs *BuildStep, m *Manifest, index int) error {
	set := make([]ConcreteBuildStep, 0, 1)
	for _, s := range []ConcreteBuildStep{
		&bs.CopyBuildStep,
		&bs.GoBuildStep,
		&bs.RunBuildStep,
		&bs.GoGAEBundleBuildStep,
	} {
		if !s.isEmpty() {
			set = append(set, s)
		}
	}
	// One and only one substruct should be populated.
	switch {
	case len(set) == 0:
		return errors.Reason("unrecognized or empty").Err()
	case len(set) > 1:
		return errors.Reason("ambiguous").Err()
	default:
		bs.manifest = m
		bs.index = index
		bs.concrete = set[0]
		return nil
	}
}

func normPath(p *string, cwd string) {
	if *p != "" {
		*p = filepath.FromSlash(*p)
		if cwd != "" {
			*p = filepath.Join(cwd, *p)
		}
	}
}

// isTemplatedPath is true if 'p' starts with "${<something>}[/]".
func isTemplatedPath(p string) bool {
	parts := strings.SplitN(p, "/", 2)
	return strings.HasPrefix(parts[0], "${") && strings.HasSuffix(parts[0], "}")
}

// renderPath verifies `p` starts with "${<something>}[/]", replaces it with
// dirs[<something>], and normalizes the result.
func renderPath(title, p string, dirs map[string]string) (string, error) {
	if p == "" {
		return "", errors.Reason("bad `%s`: must not be empty", title).Err()
	}

	// Helper for error messages.
	keys := func() string {
		ks := make([]string, 0, len(dirs))
		for k := range dirs {
			ks = append(ks, fmt.Sprintf("${%s}", k))
		}
		sort.Strings(ks)
		return strings.Join(ks, " or ")
	}

	parts := strings.SplitN(p, "/", 2)
	if !strings.HasPrefix(parts[0], "${") || !strings.HasSuffix(parts[0], "}") {
		return "", errors.Reason("bad `%s`: must start with %s", title, keys()).Err()
	}

	switch val, ok := dirs[strings.TrimSuffix(strings.TrimPrefix(parts[0], "${"), "}")]; {
	case !ok:
		return "", errors.Reason("bad `%s`: unknown dir variable %s, expecting %s", title, parts[0], keys()).Err()
	case val == "":
		return "", errors.Reason("bad `%s`: dir variable %s it not set", title, parts[0]).Err()
	case len(parts) == 1:
		return val, nil
	default:
		return filepath.Join(val, filepath.FromSlash(parts[1])), nil
	}
}
