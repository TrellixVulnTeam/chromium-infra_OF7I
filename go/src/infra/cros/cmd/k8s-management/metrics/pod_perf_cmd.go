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
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	k8sclientset "k8s.io/metrics/pkg/client/clientset/versioned"
	k8smetrics "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

type podPerfCmd struct {
	commonOpts
	namespace   string
	application string
}

func (*podPerfCmd) Name() string { return "pod-perf" }
func (*podPerfCmd) Synopsis() string {
	return "Get pod performance (CPU & RAM) usage in the specified namespace."
}
func (c *podPerfCmd) Usage() string {
	return fmt.Sprintf("%s: %s\n\n", c.Name(), c.Synopsis())
}

func (c *podPerfCmd) SetFlags(f *flag.FlagSet) {
	c.commonOpts.RegisterFlags(f)
	f.StringVar(&c.namespace, "namespace", "skylab", "pod namespace")
	f.StringVar(&c.application, "application", "", "pod application, e.g. skylab-drone-prod")
}

func (c *podPerfCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := c.startPodPerf(); err != nil {
		log.Printf("Pod perf: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

func (c *podPerfCmd) startPodPerf() error {
	inserter, err := c.commonOpts.BqInserter()
	if err != nil {
		return fmt.Errorf("start pod perf metrics: %s", err)
	}

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("start pod perf metrics: %s", err)
	}
	clientset, err := k8sclientset.NewForConfig(k8sConfig)
	if err != nil {
		return fmt.Errorf("start pod perf metrics: %s", err)
	}
	mi := clientset.MetricsV1beta1().PodMetricses(c.namespace)

	podFilter := ""
	if c.application != "" {
		podFilter = fmt.Sprintf("app=%s", c.application)
	}

	for {
		reportToBigQuery(inserter, getPodPerf(mi, podFilter), 5*time.Second)
		time.Sleep((time.Duration)(c.scanIntervalMinute) * time.Minute)
	}
}

type podPerfRecord struct {
	timestamp time.Time
	name      string
	cpuMili   int64
	memoryMiB int64
}

func (i *podPerfRecord) Save() (row map[string]bigquery.Value, insertID string, err error) {
	row = map[string]bigquery.Value{
		"timestamp":  i.timestamp,
		"name":       i.name,
		"cpu_mili":   i.cpuMili,
		"memory_MiB": i.memoryMiB,
	}
	return row, "", nil
}

func getPodPerf(p k8smetrics.PodMetricsInterface, filter string) []bigquery.ValueSaver {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	podMetricsList, err := p.List(ctx, k8smetav1.ListOptions{LabelSelector: filter})
	if err != nil {
		log.Printf("Pod perf metric error: %s", err)
		return nil
	}
	log.Printf("Get perf metric of %d pods", len(podMetricsList.Items))
	var items []bigquery.ValueSaver
	for _, pm := range podMetricsList.Items {
		// Sum up the usage of all containers of the current pod.
		cpu := k8sresource.Quantity{}
		memory := k8sresource.Quantity{}
		for _, c := range pm.Containers {
			cpu.Add(*c.Usage.Cpu())
			memory.Add(*c.Usage.Memory())
		}

		items = append(items, &podPerfRecord{
			timestamp: pm.Timestamp.Time,
			name:      pm.GetName(),
			cpuMili:   cpu.MilliValue(),
			memoryMiB: memory.Value() >> 20, // From Bytes to MiB.
		})
	}
	return items
}
