// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manifestutil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"infra/cros/internal/repo"

	"go.chromium.org/luci/common/errors"
)

const (
	attrRegexpTemplate = `%s="([^\"]*)"`
	tagRegexpTempate   = "<%s[^(<>)]*>"
	// Matches <tag/> and <tag><subtag .../></tag>.
	// Slight limitation: does not match <tag><tag></tag></tag>. But this
	// shouldn't be an issue for manifests.
	fullTagRegexpTemplate = `[ \t]*<%[1]s[^(<>)]*?(\/>|>[\s\S]*?<\/%[1]s>)\n*`
)

func delAttr(tag, attr string) string {
	// Regex for finding attribute. Include leading whitespace.
	attrRegex := regexp.MustCompile(fmt.Sprintf(`\s*`+attrRegexpTemplate, attr))
	return attrRegex.ReplaceAllString(tag, ``)
}

func getAttr(tag, attr string) string {
	// Regex for finding attribute.
	attrRegex := regexp.MustCompile(fmt.Sprintf(attrRegexpTemplate, attr))

	// Attribute with current value.
	if currAttr := attrRegex.FindStringSubmatch(tag); currAttr == nil {
		return ""
	} else {
		return currAttr[1]
	}
}

func setAttr(tag, attr, value string) string {
	// Regex for finding attribute.
	attrRegex := regexp.MustCompile(fmt.Sprintf(attrRegexpTemplate, attr))
	// Attribute with new value.
	newAttr := fmt.Sprintf(`%s="%s"`, attr, value)

	// Attribute with current value.
	currAttr := attrRegex.FindString(tag)
	if currAttr != "" { // Attr exists, replace value.
		return attrRegex.ReplaceAllString(tag, newAttr)
	}
	// Attr does not exist, add attribute to end of [start] tag.
	endRegex := regexp.MustCompile(`(\s*/?>)`)
	return endRegex.ReplaceAllString(tag, " "+newAttr+"$1")
}

// Given a Project struct, find the corresponding start tag in
// a raw XML file. Empty string indicates no match.
func findProjectTag(project *repo.Project, rawManifest string) string {
	projectRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "project"))
	for _, tag := range projectRegexp.FindAllString(rawManifest, -1) {
		p := &repo.Project{}

		// If tag is not a singleton, add empty end tag for unmarshalling purposes.
		var err error
		if tag[len(tag)-2:] != "/>" {
			err = xml.Unmarshal([]byte(tag+"</project>"), p)
		} else {
			err = xml.Unmarshal([]byte(tag), p)
		}
		if err != nil {
			continue
		}

		// Together, Name and Path form a unique identifier.
		// If Path is blank, Name is (or at least ought to be) a unique identifier.
		if project.Name == p.Name && (p.Path == "" || project.Path == p.Path) {
			return tag
		}
	}
	return ""
}

// Given a Project struct, find the corresponding tag in a raw XML file
// (including the end tag). Empty string indicates no match.
func findFullProjectTag(project *repo.Project, rawManifest string) string {
	projectRegexp := regexp.MustCompile(fmt.Sprintf(fullTagRegexpTemplate, "project"))
	for _, tag := range projectRegexp.FindAllString(rawManifest, -1) {
		p := &repo.Project{}
		err := xml.Unmarshal([]byte(tag), p)
		if err != nil {
			continue
		}

		// Together, Name and Path form a unique identifier.
		// If Path is blank, Name is (or at least ought to be) a unique identifier.
		if project.Name == p.Name && (p.Path == "" || project.Path == p.Path) {
			return tag
		}
	}
	return ""
}

func updateElement(manifest, elt string, attrs map[string]string) string {
	var newElt string
	// Sort the keys so that this function is deterministic (for testing).
	ks := []string{}
	for k := range attrs {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	for _, k := range ks {
		v := attrs[k]
		// If the attribute is empty, delete it from the element.
		if v == "" {
			newElt = delAttr(elt, k)
			manifest = strings.ReplaceAll(manifest, elt, newElt)
		} else {
			newElt = setAttr(elt, k, v)
			manifest = strings.ReplaceAll(manifest, elt, newElt)
		}
		elt = newElt
	}
	return manifest
}

// UpdateManifestElements updates manifest elements in place (so as to make the
// minimal changes possible to a manifest file) according to a reference
// manifest.
// The intended use case is to read a manifest from disk into a Manifest struct,
// modify the manifest in memory, and write the changes back to disk.
//
// Currently supports Default, Remote, and Project elements (not Annotation).
//
// The raw manifest will be updated at the element-level: if an element in the
// raw manifest matches an element in the reference manifest, all attributes
// will be set to the values in the reference element. The 'name' field is
// used as a unique identifier for <remote> elements and the 'path' field for
// <project> elements.
// The function will return an error if there is more than one <default> element
// in the raw manifest. The function will also return an error if elements in
// the reference manifest do not exist in the raw manifest.
// This function will NOT remove elements from the manifest just because they
// do not exist in the reference manifest.
func UpdateManifestElements(reference *repo.Manifest, rawManifest []byte) ([]byte, error) {
	return updateManifestElements(reference, rawManifest, false)
}

// UpdateManifestElementsStrict does the same thing as UpdateManifestElements
// except that it removes projects not present in the reference manifest.
func UpdateManifestElementsStrict(reference *repo.Manifest, rawManifest []byte) ([]byte, error) {
	return updateManifestElements(reference, rawManifest, true)
}

func updateManifestElements(reference *repo.Manifest, rawManifest []byte, deleteMissing bool) ([]byte, error) {
	manifest := string(rawManifest)

	// We use xml.Unmarshal to avoid the complexities of a
	// truly exhaustive regex, which would need to include logic for <annotation> tags nested
	// within a <project> tag (which are needed to determine the project type).
	parsedManifest := repo.Manifest{}
	err := xml.Unmarshal(rawManifest, &parsedManifest)
	if err != nil {
		return nil, errors.Annotate(err, "failed to unmarshal manifest").Err()
	}
	parsedManifest.ResolveImplicitLinks()

	// Sync <default> tag to reference.
	defaultRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "default"))
	defaultTags := defaultRegexp.FindAllString(manifest, -1)
	if len(defaultTags) > 1 {
		return nil, fmt.Errorf("manifest has more than one <default> tag")
	} else if len(defaultTags) == 1 {
		manifest = updateElement(manifest, defaultTags[0], reference.Default.AttrMap())
	} else {
		if reference.Default.RemoteName != "" || reference.Default.Revision != "" || reference.Default.SyncJ != "" {
			return nil, fmt.Errorf("reference contained default(s), manifest did not")
		}
	}

	// Sync <remote> tag(s) to reference.
	remoteRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "remote"))
	remoteTags := remoteRegexp.FindAllString(manifest, -1)
	usedRemotes := 0
	for _, remoteTag := range remoteTags {
		remoteName := getAttr(remoteTag, "name")
		if referenceRemote := reference.GetRemoteByName(remoteName); referenceRemote != nil {
			manifest = updateElement(manifest, remoteTag, referenceRemote.AttrMap())
			usedRemotes += 1
		}
	}
	if usedRemotes < len(reference.Remotes) {
		return nil, fmt.Errorf("reference contained remote(s) not present in the manifest")
	}

	// Sync <project> tag(s) to reference.
	usedProject := map[string]bool{}
	for _, project := range parsedManifest.Projects {
		projectTag := findProjectTag(&project, manifest)
		projectPath := getAttr(projectTag, "path")
		if referenceProject, _ := reference.GetProjectByPath(projectPath); referenceProject != nil {
			manifest = updateElement(manifest, projectTag, referenceProject.AttrMap())
			usedProject[project.Path] = true
		}
	}
	if deleteMissing {
		for _, project := range parsedManifest.Projects {
			if _, used := usedProject[project.Path]; !used {
				tag := findFullProjectTag(&project, manifest)
				manifest = strings.Replace(manifest, tag, "", 1)
			}
		}
	}
	if len(usedProject) < len(reference.Projects) {
		return nil, fmt.Errorf("reference contained project(s) not present in the manifest")
	}

	// Remove trailing space in start tags.
	manifest = regexp.MustCompile(`\s+>`).ReplaceAllString(manifest, ">")

	return []byte(manifest), nil
}

// UpdateManifestElementsInFile performs the same operation as UpdateManifestElements
// but operates on a specific manifest file, handling all input/output.
// Returns whether or not the file contents changed, and a potential error.
func UpdateManifestElementsInFile(path string, reference *repo.Manifest) (bool, error) {
	data, err := LoadManifestFromFileRaw(path)
	if err != nil {
		return false, err
	}

	if newData, err := UpdateManifestElements(reference, data); err != nil {
		return false, err
	} else if err := ioutil.WriteFile(path, newData, 0644); err != nil {
		return false, errors.Annotate(err, "failed to write manifest").Err()
	} else {
		return !bytes.Equal(data, newData), nil
	}
}
