// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/subcommands"
	k8sApiCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sTypedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type podStatCmd struct {
	commonOpts
	namespace string
}

func (*podStatCmd) Name() string { return "pod-stat" }
func (*podStatCmd) Synopsis() string {
	return "Get pod statistics of version, distribution on nodes, etc."
}
func (c *podStatCmd) Usage() string {
	return fmt.Sprintf("%s: %s\n\n", c.Name(), c.Synopsis())
}
func (c *podStatCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.RegisterFlags(f)
	f.StringVar(&c.namespace, "namespace", "skylab", "pod namespace")
}

func (c *podStatCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := c.startPodStat(); err != nil {
		log.Printf("Pod stat: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (c *podStatCmd) startPodStat() error {
	inserter, err := c.commonOpts.BqInserter()
	if err != nil {
		return fmt.Errorf("start pod perf metrics: %s", err)
	}

	clientset, err := getK8sClientSet()
	if err != nil {
		return fmt.Errorf("new metric client: %s", err)
	}
	cn, err := getClusterName(clientset)
	if err != nil {
		return fmt.Errorf("new metric client: %s", err)
	}
	log.Printf("Cluster name: %q", cn)

	reportPodStatLoop(cn, clientset.CoreV1().Pods(c.namespace), inserter, (time.Duration)(c.scanIntervalMinute)*time.Minute)
	return nil
}

func reportPodStatLoop(clusterName string, p k8sTypedCoreV1.PodInterface, r *bigquery.Inserter, interval time.Duration) {
	for {
		reportToBigQuery(r, getPodStat(clusterName, p), 5*time.Second)
		time.Sleep(interval)
	}
}

// We don't save each pod data to BigQuery. Instead, we count pods which have
// the same application, version, etc.
type podGroup struct {
	application string
	version     string
	state       string
	nodeName    string
}

type record struct {
	podGroup
	cluster   string
	timestamp time.Time
	count     int
}

func (i *record) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = map[string]bigquery.Value{
		"cluster":     i.cluster,
		"timestamp":   i.timestamp,
		"application": i.application,
		"version":     i.version,
		"state":       i.state,
		"node_name":   i.nodeName,
		"count":       i.count,
	}
	return row, "", nil
}

func getPodStat(clusterName string, p k8sTypedCoreV1.PodInterface) []bigquery.ValueSaver {
	now := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pods, err := p.List(ctx, k8sMetaV1.ListOptions{})
	if err != nil {
		log.Printf("List pods: %s", err)
		return nil
	}
	stat := make(map[podGroup]int)
	for _, p := range pods.Items {
		m := p.ObjectMeta
		application := m.Labels["app"]
		// Deployment and DaemonSet has different way to indicate a version.
		var ver string
		if h, ok := m.Labels["pod-template-hash"]; ok {
			ver = h
		}
		if h, ok := m.Labels["controller-revision-hash"]; ok {
			ver = h
		}
		state := podState(&p)
		nodeName := p.Spec.NodeName
		b := podGroup{application: application, version: ver, state: state, nodeName: nodeName}
		stat[b]++
	}

	items := []bigquery.ValueSaver{}
	for b, c := range stat {
		i := record{cluster: clusterName, timestamp: now, count: c, podGroup: b}
		items = append(items, &i)
	}
	return items
}

func podState(pod *k8sApiCoreV1.Pod) string {
	// See below link for the implementation details.
	// https://github.com/kubernetes/kubernetes/blob/v1.2.0/pkg/kubectl/resource_printer.go#L561-L590
	state := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		state = pod.Status.Reason
	}
	if pod.DeletionTimestamp != nil {
		state = "Terminating"
	}
	return state
}
