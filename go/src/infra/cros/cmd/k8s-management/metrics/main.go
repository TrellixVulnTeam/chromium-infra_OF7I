// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
	k8sApiCoreV1 "k8s.io/api/core/v1"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sTypedCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var (
	svcAcctJSON        = flag.String("service-account-json", "", "Path to JSON file with service account credentials to use")
	cloudProjectID     = flag.String("cloud-project-id", "cros-lab-servers", "ID of the cloud project to upload metrics data to")
	dataset            = flag.String("dataset", "k8s_workloads", "Dataset name of the BigQuery tables")
	tableName          = flag.String("table", "pod_info", "BigQuery table name")
	scanIntervalMinute = flag.Int("scan-interval-minute", 10, "Scan interval in minute")
	podNamespace       = flag.String("pod-namespace", "skylab", "Namespace of pods to count")
)

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

func main() {
	if err := innerMain(); err != nil {
		log.Fatalf("K8s metrics: %s", err)
	}
}

func innerMain() error {
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := bigquery.NewClient(ctx, *cloudProjectID, option.WithCredentialsFile(*svcAcctJSON))
	if err != nil {
		return err
	}
	r := c.Dataset(*dataset).Table(*tableName).Inserter()
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}
	cn, err := clusterName(clientset)
	if err != nil {
		return fmt.Errorf("get cluster name: %s", err)
	}
	log.Printf("cluster name: %q", cn)
	reportMetricsLoop(cn, clientset.CoreV1().Pods(*podNamespace), r)
	return nil
}

func clusterName(c *kubernetes.Clientset) (string, error) {
	// We use the API server info (i.e. 'IP:port') as the cluster name.
	// See https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/discovery/discovery_client.go#L160
	// for how to get the API server info.
	v := &k8sMetaV1.APIVersions{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.RESTClient().Get().AbsPath(c.LegacyPrefix).Do(ctx).Into(v); err != nil {
		return "", fmt.Errorf("get cluster name: %s", err)
	}
	if len(v.ServerAddressByClientCIDRs) == 0 {
		return "", errors.New("no data in ServerAddressByClientCIDRs")
	}
	return v.ServerAddressByClientCIDRs[0].ServerAddress, nil
}

func reportMetricsLoop(clusterName string, p k8sTypedCoreV1.PodInterface, r *bigquery.Inserter) {
	for {
		reportToBigQuery(r, getPodMetrics(clusterName, p))
		time.Sleep((time.Duration)(*scanIntervalMinute) * time.Minute)
	}
}

func getPodMetrics(clusterName string, p k8sTypedCoreV1.PodInterface) []*record {
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

	items := []*record{}
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

func reportToBigQuery(r *bigquery.Inserter, items []*record) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Put(ctx, items); err != nil {
		log.Printf("Put record to bigquery failed: %s", err)
	} else {
		log.Printf("Put %d record(s) to bigquery", len(items))
	}
}
