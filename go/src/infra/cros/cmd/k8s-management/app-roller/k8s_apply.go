// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"fmt"

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
func k8sApply(ctx context.Context, content string) error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}
	obj := &unstructured.Unstructured{}
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, gvk, err := decUnstructured.Decode([]byte(content), nil, obj)
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dyn.Resource(mapping.Resource)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}

	if _, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: "k8s_app_roller"}); err != nil {
		return fmt.Errorf("apply to k8s: %s", err)
	}
	return nil
}
