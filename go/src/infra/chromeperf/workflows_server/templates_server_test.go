// Copyright 2020 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"infra/chromeperf/workflows"

	. "github.com/smartystreets/goconvey/convey"
	configProto "go.chromium.org/luci/common/proto/config"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/impl/memory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

const configSingleTemplate = `
		templates: {
			name: "chromium-telemetry-bisect-v1"
			display_name: "Chromium Telemetry benchmark bisection (v1)"
			description: "Bisects a Telemetry benchmark+metric+story for culprits between two CLs"
			inputs: [
				{
					name: "benchmark"
					kind: TYPE_STRING
					cardinality: CARDINALITY_REQUIRED
					options: {
						name: "regex_validator"
						value {
							[type.googleapis.com/google.protobuf.Value] {
								string_value: "^[a-zA-Z0-9\\._\\-]+$"
							}
						}
					}
				},
				{
					name: "configuration"
					kind: TYPE_STRING
					cardinality: CARDINALITY_REQUIRED
				},
				{
					name: "story_tags"
					kind: TYPE_STRING
					cardinality: CARDINALITY_OPTIONAL
				},
				{
					name: "metric"
					kind: TYPE_STRING
					cardinality: CARDINALITY_REQUIRED
					number: 4
				},
				{
					name: "commit_range"
					kind: TYPE_MESSAGE
					cardinality: CARDINALITY_REQUIRED
					type_url: "api.pinpoint.cr.dev/GitilesCommitRange"
				}
			]
			task_options: {
				fields: [
					{
						key: "benchmark"
						value {
							string_value: "{benchmark}"
						}
					},
					{
						key: "configuration"
						value {
							string_value: "{configuration}"
						}
					},
					{
						key: "start_git_hash"
						value {
							string_value: "{commit_range.start_git_hash}"
						}
					},
					{
						key: "end_git_hash"
						value {
							string_value: "{commit_range.end_git_hash}"
						}
					}
				]
			}
			graph_creation_module: "chromeperf.pinpoint.bisector"
			cria_readers: [
				'project-pinpoint-api-users'
			]
		}`

func TestValidConfigurations(t *testing.T) {
	ctx := context.Background()
	l := bufconn.Listen(bufSize)
	s := grpc.NewServer()

	// Define the same client to use throughout the test.
	dialer := func(context.Context, string) (net.Conn, error) {
		return l.Dial()
	}
	// Also set-up a "mock" luci-config HTTP service which we will perform requests against.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %v", r)
		// This will just serve the contents of the file.
		fmt.Fprintln(w, configSingleTemplate)
	}))
	defer ts.Close()

	// Create a configV1 client, with the httpClient we've already seeded.
	configSets := map[config.Set]memory.Files{
		configSetName: map[string]string{},
	}
	mockConfig := func(body string) {
		configSets[configSetName][workflowTemplatesFile] = body
	}
	workflows.RegisterWorkflowTemplatesServer(s, &workflowTemplatesServer{luciConfigClient: memory.New(configSets)})
	go func() {
		if err := s.Serve(l); err != nil {
			log.Fatalf("Server startup failed.")
		}
	}()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed creating a connection: %v", err)
	}
	defer conn.Close()
	client := workflows.NewWorkflowTemplatesClient(conn)

	Convey("Given a valid configuration defined with one template", t, func() {
		mockConfig(configSingleTemplate)

		Convey("When we attempt to validate the contents", func() {
			resp, err := client.ValidateConfig(
				ctx, &configProto.ValidationRequestMessage{
					ConfigSet: "test-validation",
					Path:      "/test/validation/path",
					Content:   []byte(configSingleTemplate),
				},
			)

			Convey("Then we get a non-error response", func() {
				So(err, ShouldBeNil)
				So(resp.Messages, ShouldBeEmpty)
			})
		})

		Convey("When we list the templates", func() {
			resp, err := client.ListWorkflowTemplates(
				ctx, &workflows.ListWorkflowTemplatesRequest{
					PageSize: 10,
				},
			)

			Convey("Then we find that the defined template is in the list", func() {
				So(err, ShouldBeNil)
				So(resp.WorkflowTemplates, ShouldNotBeEmpty)
			})
		})

		Convey("When we get the template by name", func() {

			Convey("Then we get a non-error response", nil)

			Convey("And we find that the defined template is retrieved", nil)

		})

	})

	Convey("Given a valid configuration with more templates", t, func() {

		Convey("When we attempt to validate the contents", func() {

			Convey("Then we get a non-error response", nil)

		})

		Convey("When we list the templates", func() {

			Convey("Then we find that the defined templates are in the list", nil)

		})

		Convey("When we get the templates by name", func() {

			Convey("Then we get a non-error response", nil)

			Convey("And we find that the defined templates are retrieved", nil)

		})

	})

}

func TestInvalidConfigurations(t *testing.T) {

	Convey("Given a configuration with ill-formed text protobufs", t, func() {

		Convey("When we attempt to validate the contents", func() {

			Convey("Then we get a validation response with an ERROR severity", nil)

		})

	})

	Convey("Given a configuration with missing input fields", t, func() {

		Convey("When we attempt to validate the contents", func() {

			Convey("Then we get a validation response with an ERROR severity", nil)

		})

	})

}
