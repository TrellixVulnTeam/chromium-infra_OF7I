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
	"os"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/subcommands"
	"google.golang.org/api/option"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// commonOpts is the common options for all subcommands.
type commonOpts struct {
	svcAcctJSON        string
	cloudProjectID     string
	dataset            string
	tableName          string
	scanIntervalMinute int
}

// RegisterFlags set the common options to the subcommands' flag set.
func (c *commonOpts) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&c.svcAcctJSON, "service-account-json", "", "Path to JSON file with service account credentials to use")
	f.StringVar(&c.cloudProjectID, "cloud-project-id", "chrome-fleet-analytics", "ID of the cloud project to upload metrics data to")
	f.StringVar(&c.dataset, "dataset", "", "Dataset name of the BigQuery tables")
	f.StringVar(&c.tableName, "table", "", "BigQuery table name")
	f.IntVar(&c.scanIntervalMinute, "scan-interval-minute", 10, "Scan interval in minute")
}

// BqInserter gets the BigQuery inserter object of the table specified.
func (c *commonOpts) BqInserter() (*bigquery.Inserter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bqc, err := bigquery.NewClient(ctx, c.cloudProjectID, option.WithCredentialsFile(c.svcAcctJSON))
	if err != nil {
		return nil, fmt.Errorf("get BigQuery inserter: %s", err)
	}
	return bqc.Dataset(c.dataset).Table(c.tableName).Inserter(), nil
}

func main() {
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&podStatCmd{}, "")

	flag.Parse()
	rc := int(subcommands.Execute(context.Background()))
	log.Printf("Exited with %d", rc)
	os.Exit(rc)
}

func getK8sClientSet() (*kubernetes.Clientset, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("get K8s client set: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("get K8s client set: %s", err)
	}
	return clientset, nil
}

func getClusterName(c *kubernetes.Clientset) (string, error) {
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

func reportToBigQuery(r *bigquery.Inserter, items []bigquery.ValueSaver, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := r.Put(ctx, items); err != nil {
		log.Printf("Put record to bigquery failed: %s", err)
	} else {
		log.Printf("Put %d record(s) to bigquery", len(items))
	}
}
