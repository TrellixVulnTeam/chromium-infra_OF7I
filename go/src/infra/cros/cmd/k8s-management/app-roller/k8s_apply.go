// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// k8sApply applies a YAML content to the K8s cluster the program running on.
// The implementation is mostly inspired by
// https://ymmt2005.hatenablog.com/entry/2020/04/14/An_example_of_using_dynamic_client_of_k8s.io/client-go
func k8sApply(ctx context.Context, content string) (*change, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}
	obj := &unstructured.Unstructured{}
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := decUnstructured.Decode([]byte(content), nil, obj)
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dyn.Resource(mapping.Resource)
	}

	c, err := putResource(ctx, dr, obj)
	if err != nil {
		return nil, fmt.Errorf("apply to k8s: %s", err)
	}
	return c, nil
}

// putResource puts (i.e. create or patch) the target resource.
func putResource(ctx context.Context, dr dynamic.ResourceInterface, obj *unstructured.Unstructured) (*change, error) {
	// We try to get the existing resource from K8s to check if the input will
	// cause changes. However, we cannot compare them directly because the one
	// we get from K8s may includes many default fields which are not in the
	// input. Additionally, K8s may do some data converting, e.g. convert
	// '1000m' to '1', which make the comparison impossible.
	name := obj.GetName()
	before, err := dr.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get the resource %q (not created yet?): %s", name, err)
		log.Printf("Trying to create %q", name)
		// We run only one instance of app-roller in each K8s cluster and we
		// ensure K8s only have one running instance of app-roller to make sure
		// there's no race condition.
		after, err := dr.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("put resource: create %q: %s", name, err)
		}
		c, err := diffResource(nil, after)
		if err != nil {
			return nil, fmt.Errorf("put resource %q: %s", name, err)
		}
		return c, nil
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("put resource %q: %s", name, err)
	}
	// Dry run that patch to check if there's change.
	c, err := patchResource(ctx, name, dr, before, data, true)
	if err != nil {
		return nil, fmt.Errorf("put resource %q: %s", name, err)
	}
	if c == nil {
		return nil, nil
	}
	// Really patch the resource.
	c, err = patchResource(ctx, name, dr, before, data, false)
	if err != nil {
		return nil, fmt.Errorf("put resource %q: %s", name, err)
	}
	return c, nil
}

// patchResource patches the target resource and returns the changes.
func patchResource(ctx context.Context, name string, dr dynamic.ResourceInterface, before *unstructured.Unstructured, data []byte, dryRun bool) (*change, error) {
	// Patch with force to overwrite all changes made by others, otherwise there
	// might be conflicts which cannot be resolved automatically.
	force := true
	po := metav1.PatchOptions{FieldManager: "k8s_app_roller", Force: &force}
	if dryRun {
		po.DryRun = []string{"All"}
	}
	after, err := dr.Patch(ctx, name, types.ApplyPatchType, data, po)
	if err != nil {
		return nil, fmt.Errorf("patchResource (dry run: %v): %s", dryRun, err)
	}
	c, err := diffResource(before, after)
	if err != nil {
		return nil, fmt.Errorf("patchResource (dry run: %v): %s", dryRun, err)
	}
	return c, nil
}

// diffResource compares the resources and returns the difference.
func diffResource(before, after *unstructured.Unstructured) (*change, error) {
	filterNoisyFields(before)
	filterNoisyFields(after)

	js0, err := json.Marshal(before)
	if err != nil {
		return nil, fmt.Errorf("get changes (before) %s", err)
	}
	js1, err := json.Marshal(after)
	if err != nil {
		return nil, fmt.Errorf("get changes (after) %s", err)
	}

	kn := after.GetKind() + "/" + after.GetName()
	d := cmp.Diff(before, after)
	if d == "" {
		log.Printf("Nothing changed on %q", kn)
		return nil, nil
	}
	log.Printf("Changes of %q: (-before, +after)\n%s", kn, d)
	return &change{
		timestamp: time.Now(),
		namespace: after.GetNamespace(),
		resource:  kn,
		before:    string(js0),
		after:     string(js1),
		diff:      d,
	}, nil
}

// filterNoisyFields filters out fields that constantly changing even there's no
// real changes.
func filterNoisyFields(obj *unstructured.Unstructured) {
	if obj == nil {
		return
	}
	obj.Object["metadata"].(map[string]interface{})["managedFields"] = nil
	obj.Object["metadata"].(map[string]interface{})["resourceVersion"] = nil
}

// change is the structured data of resource change.
type change struct {
	timestamp time.Time
	namespace string
	resource  string
	before    string
	after     string
	diff      string
}
