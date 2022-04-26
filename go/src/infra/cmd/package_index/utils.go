package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/logging"

	kpb "infra/cmd/package_index/kythe/proto"
)

// A list of path prefixes to set a different root and path, used by
// kythe/grimoire to find external headers.
var rootModifiers = []string{
	"src/third_party/depot_tools/win_toolchain",
	"src/build/linux/debian_sid_amd64-sysroot",
}

// Substrings of arguments that should be removed from compile commands on Windows.
var unwantedArgSubstringsWin = []string{
	// These Skia header path defines throw errors in the Windows indexer for
	// some reason.
	"-DSK_USER_CONFIG_HEADER",
	"-DSK_GPU_WORKAROUNDS_HEADER",
}

// removeFilepathsFiles removes all .filepaths files within the specified root dir.
func removeFilepathsFiles(ctx context.Context, root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		logging.Errorf(ctx, "Cannot convert root directory path %s to an absolute path", root)
		return err
	}

	r, err := os.Open(root)
	if err != nil {
		logging.Errorf(ctx, "Cannot open root directory %s", root)
		return err
	}
	defer r.Close()

	files, err := r.Readdir(-1)
	if err != nil {
		logging.Errorf(ctx, "Cannot read directory %s", r.Name)
		return err
	}

	for _, f := range files {
		if f.Mode().IsRegular() && filepath.Ext(f.Name()) == ".filepaths" {
			fPath := filepath.Join(root, f.Name())
			err = os.Remove(fPath)
			if err != nil {
				logging.Errorf(ctx, "Cannot remove file at %s", fPath)
				return err
			}
		}
	}
	return nil
}

// convertGnPath converts GN paths into output-directory-relative paths.
//
// gnPath begins with a //, which represents the root of the repository.
// The expectation is that outDir always contains src/.
func convertGnPath(ctx context.Context, gnPath, outDir string) (string, error) {
	if !strings.HasPrefix(outDir, "src") {
		panic("Directory does not start with src")
	}

	paths := strings.Split(gnPath[2:], "/")
	paths = append([]string{"src"}, paths...)
	s, err := filepath.Rel(outDir, path.Join(paths...))
	if err != nil {
		logging.Errorf(ctx, "Cannot convert path %v to a relative path", paths)
		return "", err
	}
	return s, nil
}

// convertPathToForwardSlashes converts a path to use forward slashes.
func convertPathToForwardSlashes(pth string) string {
	if runtime.GOOS == "windows" {
		return strings.ReplaceAll(pth, "\\", "/")
	}
	return pth
}

// normalizePath returns a cleaned path join of dir and pth.
func normalizePath(dir, pth string) string {
	return path.Clean(filepath.Join(dir, pth))
}

// injectUnitBuildDetails adds BuildDetail information from buildConfig into unitProto.
func injectUnitBuildDetails(ctx context.Context, unitProto *kpb.CompilationUnit, buildConfig string) {
	// If there is already BuildDetails, we need to reuse it.
	for i, anyDetails := range unitProto.GetDetails() {
		if anyDetails.GetTypeUrl() == "kythe.io/proto/kythe.proto.BuildDetails" {
			buildDetails := &kpb.BuildDetails{}
			if err := ptypes.UnmarshalAny(anyDetails, buildDetails); err != nil {
				panic(fmt.Sprintf("Failed to parse unit details: %v", err))
			}
			buildDetails.BuildConfig = buildConfig
			any, err := ptypes.MarshalAny(buildDetails)
			if err != nil {
				panic(fmt.Sprintf("Failed to pack Any details: %v", err))
			}
			any.TypeUrl = "kythe.io/proto/kythe.proto.BuildDetails"
			unitProto.Details[i] = any
			return
		}
	}

	// BuildDetails wasn't found, create a new one.
	details := &kpb.BuildDetails{}
	details.BuildConfig = buildConfig

	any, err := ptypes.MarshalAny(details)
	if err != nil {
		panic(fmt.Sprintf("Failed to pack Any details: %v", err))
	}
	any.TypeUrl = "kythe.io/proto/kythe.proto.BuildDetails"

	unitProto.Details = append(unitProto.Details, any)
}

// findImports looks for all import statements and returns absolute path
// to all imported files.
//
// Args:
//   regex: compiled regex that matches import statement. It can have only
//   one group which yields import filename
//   fpath: path to file that will be inspected, absolute path
//   importPaths: list of import directories, should be absolute path
//
// Returns:
//   set containing all imports
//
// For example, if content of .proto file is following:
// import "foo.proto"
// import weak "bar.proto"
//
// and working directory is "/tmp" and regex is protoImportRe
// this function will return set("/tmp/foo.proto", "/tmp/bar.proto").
func findImports(ctx context.Context, regex *regexp.Regexp, fpath string, importPaths []string) (imports []string) {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		logging.Warningf(ctx, "File %s does not exist, returning empty import set", fpath)
		return imports
	}

	contents, err := ioutil.ReadFile(fpath)
	if err != nil {
		panic(fmt.Sprintf("Cannot read file %s: %v", fpath, err))
	}

	found := false
	for _, imp := range regex.FindAllStringSubmatch(string(contents), -1) {
		for _, importPath := range importPaths {
			p := path.Join(importPath, imp[len(imp)-1])
			if _, err := os.Stat(p); err == nil {
				imports = append(imports, filepath.Clean(p))
				found = true
				break
			}
		}
		if !found {
			logging.Infof(ctx, "Could not find import %s for file %s", imp, fpath)
			logging.Infof(ctx, "%v", importPaths)
		}
	}
	return imports
}

// setVnameForFile returns the appropriate VName for filepath.
//
// Specifically, this checks if the file should be put in a special corpus
// (e.g. the one for the Windows SDK), and if so overrides defaultCorpus
// and moves the windows path to root.
func setVnameForFile(ctx context.Context, vnameProto *kpb.VName, filepath, defaultCorpus string) {
	if strings.Contains(filepath, "\\") {
		panic("Filepath contains \\")
	}

	vnameProto.Corpus = defaultCorpus
	vnameProto.Path = filepath
	for _, prefix := range rootModifiers {
		if strings.HasPrefix(filepath, prefix+"/") {
			vnameProto.Path = filepath[len(prefix)+1:]
			vnameProto.Root = prefix
			break
		}
	}
}

// isUnwantedWinArg checks if a given arg should be removed for
// compatibility with the Windows indexer.
func isUnwantedWinArg(arg string) bool {
	for _, substr := range unwantedArgSubstringsWin {
		if strings.Contains(arg, substr) {
			return true
		}
	}
	return false
}
