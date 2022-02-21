// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	goavro "github.com/linkedin/goavro/v2"
	"sigs.k8s.io/yaml"
)

//go:embed avro_schema.yaml
var yamlSchema []byte

var codecOnce sync.Once
var codec *goavro.Codec
var codecErr error

// avroCodec returns codec used to write ninja log with AVRO format.
func avroCodec() (*goavro.Codec, error) {
	codecOnce.Do(func() {
		jsonSchema, err := yaml.YAMLToJSON(yamlSchema)
		if err != nil {
			codecErr = fmt.Errorf("failed to convert %s: %w", yamlSchema, err)
			return
		}

		codec, err = goavro.NewCodec(string(jsonSchema))
		if err != nil {
			codecErr = fmt.Errorf("failed to create codec: %w", err)
			return
		}
	})

	return codec, codecErr
}

// This is overridden in test.
var timeNow = time.Now

// toAVRO returns ninja log passed to AVRO codec.
func toAVRO(info *NinjaLog) map[string]interface{} {
	weightedTime := WeightedTime(info.Steps)
	steps := Dedup(info.Steps)

	buildID := info.Metadata.BuildID
	if buildID == 0 {
		// Set random number if buildID is not set.
		// This is mainly for ninjalog from chromium developer.
		binary.Read(rand.Reader, binary.BigEndian, &buildID)
	}

	os := "UNKNOWN"
	// Parse platform as it is returned from python's platform.system().
	switch platform := info.Metadata.Platform; {
	case platform == "Windows" || strings.Contains(platform, "CYGWIN"):
		os = "WIN"
	case platform == "Linux":
		os = "LINUX"
	case platform == "Darwin":
		os = "MAC"
	}

	buildConfigs := make([]map[string]interface{}, 0, len(info.Metadata.BuildConfigs))
	for k, v := range info.Metadata.BuildConfigs {
		buildConfigs = append(buildConfigs, map[string]interface{}{
			"key":   k,
			"value": v,
		})
	}

	// Configuring order is matter for same key.
	sort.SliceStable(buildConfigs, func(i, j int) bool {
		return buildConfigs[i]["key"].(string) < buildConfigs[j]["key"].(string)
	})

	logEntries := make([]map[string]interface{}, 0, len(steps))

	for _, s := range steps {
		outputs := append(s.Outs, s.Out)
		sort.Strings(outputs)
		logEntries = append(logEntries, map[string]interface{}{
			"outputs":               outputs,
			"start_duration_sec":    s.Start.Seconds(),
			"end_duration_sec":      s.End.Seconds(),
			"weighted_duration_sec": weightedTime[s.Out].Seconds(),
		})
	}

	return map[string]interface{}{
		"targets":       info.Metadata.getTargets(),
		"build_id":      buildID,
		"os":            os,
		"step_name":     info.Metadata.StepName,
		"jobs":          info.Metadata.Jobs,
		"cpu_core":      int(info.Metadata.CPUCore),
		"build_configs": buildConfigs,
		"log_entries":   logEntries,
		"created_at":    timeNow(),
	}

}

// WriteNinjaLogToGCS upload ninja log to GCS in avro format.
func WriteNinjaLogToGCS(ctx context.Context, info *NinjaLog, bucket, filename string) (rerr error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); rerr == nil {
			rerr = err
		}
	}()

	bkt := client.Bucket(bucket)
	obj := bkt.Object(filename)
	gcsw := obj.NewWriter(ctx)
	defer func() {
		if err := gcsw.Close(); rerr == nil {
			rerr = err
		}
	}()

	codec, err := avroCodec()
	if err != nil {
		return err
	}

	ocfw, err := goavro.NewOCFWriter(goavro.OCFConfig{
		W:     gcsw,
		Codec: codec,
	})
	if err != nil {
		return err
	}

	return ocfw.Append([]interface{}{toAVRO(info)})
}
