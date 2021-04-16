// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package repo contains functions for interacting with manifests and the
// repo tool.
package repo

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"
)

const (
	attrRegexpTemplate = `%s="([^\"]*)"`
	tagRegexpTempate   = "<%s[^(<>)]*>"
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
func findProjectTag(project *Project, rawManifest string) string {
	projectRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "project"))
	for _, tag := range projectRegexp.FindAllString(rawManifest, -1) {
		p := &Project{}

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

func (e *Default) attrMap() map[string]string {
	return map[string]string{
		"remote":   e.RemoteName,
		"revision": e.Revision,
		"sync-j":   e.SyncJ,
	}
}

func (e *Remote) attrMap() map[string]string {
	return map[string]string{
		"fetch":    e.Fetch,
		"name":     e.Name,
		"revision": e.Revision,
		"alias":    e.Alias,
	}
}

func (e *Project) attrMap() map[string]string {
	return map[string]string{
		"path":     e.Path,
		"name":     e.Name,
		"revision": e.Revision,
		"upstream": e.Upstream,
		"remote":   e.RemoteName,
		"groups":   e.Groups,
		"sync-c":   e.SyncC,
	}
}

func updateElement(manifest, elt string, attrs map[string]string) string {
	var newElt string
	for k, v := range attrs {
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
func UpdateManifestElements(reference *Manifest, rawManifest []byte) ([]byte, error) {
	manifest := string(rawManifest)

	// We use xml.Unmarshal to avoid the complexities of a
	// truly exhaustive regex, which would need to include logic for <annotation> tags nested
	// within a <project> tag (which are needed to determine the project type).
	parsedManifest := Manifest{}
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
	}
	manifest = updateElement(manifest, defaultTags[0], reference.Default.attrMap())

	// Sync <remote> tag(s) to reference.
	remoteRegexp := regexp.MustCompile(fmt.Sprintf(tagRegexpTempate, "remote"))
	remoteTags := remoteRegexp.FindAllString(manifest, -1)
	usedRemotes := 0
	for _, remoteTag := range remoteTags {
		remoteName := getAttr(remoteTag, "name")
		if referenceRemote := reference.GetRemoteByName(remoteName); referenceRemote != nil {
			manifest = updateElement(manifest, remoteTag, referenceRemote.attrMap())
			usedRemotes += 1
		}
	}
	if usedRemotes < len(reference.Remotes) {
		return nil, fmt.Errorf("reference contained remote(s) not present in the manifest")
	}

	// Sync <project> tag(s) to reference.
	usedProjects := 0
	for _, project := range parsedManifest.Projects {
		projectTag := findProjectTag(&project, manifest)
		projectPath := getAttr(projectTag, "path")
		if referenceProject, _ := reference.GetProjectByPath(projectPath); referenceProject != nil {
			manifest = updateElement(manifest, projectTag, referenceProject.attrMap())
			usedProjects += 1
		}
	}
	if usedProjects < len(reference.Projects) {
		return nil, fmt.Errorf("reference contained project(s) not present in the manifest")
	}

	// Remove trailing space in start tags.
	manifest = regexp.MustCompile(`\s+>`).ReplaceAllString(manifest, ">")

	return []byte(manifest), nil
}
