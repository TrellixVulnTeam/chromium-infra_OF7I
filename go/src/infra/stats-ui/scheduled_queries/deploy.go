// The deploy command deploys Scheduled Queries into BigQuery
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	datatransfer "cloud.google.com/go/bigquery/datatransfer/apiv1"
	"google.golang.org/api/iterator"
	datatransferpb "google.golang.org/genproto/googleapis/cloud/bigquery/datatransfer/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	log.SetFlags(0)
	ctx := context.Background()

	projectID := flag.String("project_id", "", "project ID")
	flag.Usage = func() {
		fmt.Println("Usage: deploy [--flags] file ...")
		flag.PrintDefaults()
		fmt.Println(`
Files contain the queries to schedule and should have the following headers:
--name:<name>         Display name of the scheduled query. If the scheduled
                      query exists, it will be replaced if there are any
                      changes.
--schedule:<schedule> Schedule for the query. Consult BigQuery documentation for
                      allowed formats.

Note: If the name is changed, a new schedule job will be created and the old one
will need to be deleted manually`)
	}
	flag.Parse()

	if *projectID == "" {
		fmt.Println("no project specified")
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Println("no files specified")
		os.Exit(1)
	}

	queries, err := scheduledQueries(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	// Creates a client.
	client, err := datatransfer.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	configs, err := fetchTransferConfigs(ctx, client, *projectID)
	if err != nil {
		log.Fatalf("Failed to get scheduled queries: %v", err)
	}

	for _, query := range queries {
		if config, ok := configs[query.name]; ok {
			if config.Schedule == query.schedule && config.Params.Fields["query"].GetStringValue() == query.query {
				log.Printf("%s: no changes found", query.name)
			} else {
				config, err := updateScheduledQuery(ctx, client, config, query)
				if err != nil {
					log.Fatalf("%s: error updating scheduled query [%v]", query.name, err)
				}
				log.Printf("%s: updated, next run at %s", query.name, config.NextRunTime.AsTime())
			}
		} else {
			config, err := addScheduledQuery(ctx, client, *projectID, query)
			if err != nil {
				log.Fatalf("%s: error creating scheduled query [%v]", query.name, err)
			}
			log.Printf("%s: created, next run at %s", query.name, config.NextRunTime.AsTime())
		}
	}

}

func fetchTransferConfigs(ctx context.Context, client *datatransfer.Client, projectID string) (map[string]*datatransferpb.TransferConfig, error) {
	req := &datatransferpb.ListTransferConfigsRequest{
		Parent:        fmt.Sprintf("projects/%s", projectID),
		DataSourceIds: []string{"scheduled_query"},
	}
	it := client.ListTransferConfigs(ctx, req)
	configs := make(map[string]*datatransferpb.TransferConfig)
	for {
		config, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if _, exists := configs[config.DisplayName]; exists {
			log.Fatalf("Multiple scheduled queries with display name [%v].  Please fix and run again.", config.DisplayName)
		}
		configs[config.DisplayName] = config
	}
	return configs, nil
}

func addScheduledQuery(ctx context.Context, client *datatransfer.Client, projectID string, query scheduledQuery) (*datatransferpb.TransferConfig, error) {
	req := &datatransferpb.CreateTransferConfigRequest{
		Parent: fmt.Sprintf("projects/%s/locations/US", projectID),
		TransferConfig: &datatransferpb.TransferConfig{
			DisplayName:  query.name,
			DataSourceId: "scheduled_query",
			Schedule:     query.schedule,
			EmailPreferences: &datatransferpb.EmailPreferences{
				EnableFailureEmail: true,
			},
			Params: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"query": structpb.NewStringValue(query.query),
				},
			},
		},
	}
	return client.CreateTransferConfig(ctx, req)
}

func updateScheduledQuery(ctx context.Context, client *datatransfer.Client, config *datatransferpb.TransferConfig, query scheduledQuery) (*datatransferpb.TransferConfig, error) {
	req := &datatransferpb.UpdateTransferConfigRequest{
		TransferConfig: &datatransferpb.TransferConfig{
			Name:     config.Name,
			Schedule: query.schedule,
			Params: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"query": structpb.NewStringValue(query.query),
				},
			},
		},
	}
	updateMask, err := fieldmaskpb.New(req, "transfer_config.schedule", "transfer_config.params")
	if err != nil {
		return nil, err
	}
	req.UpdateMask = updateMask
	return client.UpdateTransferConfig(ctx, req)
}

type scheduledQuery struct {
	name     string
	query    string
	schedule string
}

var nameRegex = regexp.MustCompile(`--name:([^\n]+)\n`)
var scheduleRegex = regexp.MustCompile(`--schedule:([^\n]+)\n`)

func scheduledQueries(files []string) ([]scheduledQuery, error) {
	queries := make([]scheduledQuery, 0, len(files))
	for _, file := range files {
		bytes, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		query := scheduledQuery{query: string(bytes)}

		name := nameRegex.FindStringSubmatch(query.query)
		if name == nil || len(name) < 2 {
			return nil, fmt.Errorf("%s has no name specified", file)
		}
		query.name = strings.Trim(name[1], " ")

		schedule := scheduleRegex.FindStringSubmatch(query.query)
		if schedule == nil || len(schedule) < 2 {
			return nil, fmt.Errorf("%s has no schedule specified", file)
		}
		query.schedule = strings.Trim(schedule[1], " ")

		queries = append(queries, query)
	}
	return queries, nil
}
