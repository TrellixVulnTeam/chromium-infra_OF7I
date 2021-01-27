// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package handler implements HTTP server that handles requests to default module.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"infra/appengine/sheriff-o-matic/som/model"
	"infra/monitoring/messages"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/router"
)

const (
	// Maximum number of alerts to autoresolve at once to datastore to avoid exceedding datasize limits.
	maxAlertsAutoResolveCount = 100
	// model.RevisionSummaryJSONs this recent will be returned
	recentRevisions = time.Hour * 24 * 7
	// model.AlertJSONs this recently resolved will be returned
	recentResolved = time.Hour * 24 * 3
	// resolved alerts will expire after this time
	resolvedAlertExpiration = time.Hour * 24 * 7
)

var (
	// ErrUnrecognizedTree indicates that a request specified an unrecognized tree.
	ErrUnrecognizedTree = fmt.Errorf("unrecognized tree name")
)

// GetAlerts handles API requests for alerts.
func GetAlerts(ctx *router.Context, unresolved bool, resolved bool) *messages.AlertsSummary {
	c, w, p := ctx.Context, ctx.Writer, ctx.Params

	tree := p.ByName("tree")

	var q *datastore.Query
	alertResults := []*model.AlertJSON{}
	revisionSummaryResults := []*model.RevisionSummaryJSON{}
	if unresolved {
		q = datastoreCreateAlertQuery().Ancestor(datastore.MakeKey(c, "Tree", tree)).Eq("Resolved", false)
		err := datastoreGetAlertsByQuery(c, &alertResults, q)
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}

		q = datastore.NewQuery("RevisionSummaryJSON")
		q = q.Ancestor(datastore.MakeKey(c, "Tree", tree))
		q = q.Gt("Date", clock.Get(c).Now().Add(-recentRevisions))

		err = datastore.GetAll(c, q, &revisionSummaryResults)
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}
	}

	resolvedResults := []*model.AlertJSON{}
	if resolved {
		q = datastoreCreateAlertQuery().Ancestor(datastore.MakeKey(c, "Tree", tree)).Eq("Resolved", true)
		q = q.Gt("Date", clock.Get(c).Now().Add(-recentResolved))

		err := datastoreGetAlertsByQuery(c, &resolvedResults, q)
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}
	}

	alertsSummary := &messages.AlertsSummary{
		RevisionSummaries: make(map[string]*messages.RevisionSummary),
	}
	if len(alertResults) >= 1 {
		alertsSummary.Alerts = make([]*messages.Alert, len(alertResults))
	}
	if len(resolvedResults) >= 1 {
		alertsSummary.Resolved = make([]*messages.Alert, len(resolvedResults))
	}

	for i, alertJSON := range alertResults {
		err := json.Unmarshal(alertJSON.Contents, &alertsSummary.Alerts[i])
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}

		t := messages.EpochTime(alertJSON.Date.Unix())
		if alertsSummary.Timestamp == 0 || t > alertsSummary.Timestamp {
			alertsSummary.Timestamp = t
		}
	}

	for i, alertJSON := range resolvedResults {
		err := json.Unmarshal(alertJSON.Contents, &alertsSummary.Resolved[i])
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}

		t := messages.EpochTime(alertJSON.Date.Unix())
		if alertsSummary.Timestamp == 0 || t > alertsSummary.Timestamp {
			alertsSummary.Timestamp = t
		}
	}

	for _, summaryJSON := range revisionSummaryResults {
		summary := &messages.RevisionSummary{}
		err := json.Unmarshal(summaryJSON.Contents, summary)
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return nil
		}
		alertsSummary.RevisionSummaries[summaryJSON.ID] = summary
	}
	return alertsSummary
}

func getAlerts(ctx *router.Context, unresolved bool, resolved bool) {
	c, w := ctx.Context, ctx.Writer
	alertsSummary := GetAlerts(ctx, unresolved, resolved)
	data, err := json.MarshalIndent(alertsSummary, "", "\t")
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// GetAlertsHandler handles API requests for all alerts and revision summaries.
func GetAlertsHandler(ctx *router.Context) {
	getAlerts(ctx, true, true)
}

// GetUnresolvedAlertsHandler handles API requests for unresolved alerts
// and revision summaries.
func GetUnresolvedAlertsHandler(ctx *router.Context) {
	getAlerts(ctx, true, false)
}

// GetResolvedAlertsHandler handles API requests for resolved alerts.
func GetResolvedAlertsHandler(ctx *router.Context) {
	getAlerts(ctx, false, true)
}

func convertAlertJSONsToAlertJSONsNonGrouping(alertJSONs []*model.AlertJSON, alertJSONsNonGrouping *[]*model.AlertJSONNonGrouping) {
	*alertJSONsNonGrouping = make([]*model.AlertJSONNonGrouping, len(alertJSONs))
	for i, alertJSON := range alertJSONs {
		tmp := model.AlertJSONNonGrouping(*alertJSON)
		(*alertJSONsNonGrouping)[i] = &tmp
	}
}

func convertAlertJSONsNonGroupingToAlertJSONs(alertJSONsNonGrouping []*model.AlertJSONNonGrouping, alertJSONs *[]*model.AlertJSON) {
	*alertJSONs = make([]*model.AlertJSON, len(alertJSONsNonGrouping))
	for i, alertJSONNonGrouping := range alertJSONsNonGrouping {
		tmp := model.AlertJSON(*alertJSONNonGrouping)
		(*alertJSONs)[i] = &tmp
	}
}

func datastorePutAlertJSONs(c context.Context, alertJSONs []*model.AlertJSON) error {
	alertJSONsNonGrouping := []*model.AlertJSONNonGrouping{}
	convertAlertJSONsToAlertJSONsNonGrouping(alertJSONs, &alertJSONsNonGrouping)
	return datastore.Put(c, alertJSONsNonGrouping)
}

func datastorePutAlertJSON(c context.Context, alertJSON *model.AlertJSON) error {
	alertJSONs := []*model.AlertJSON{alertJSON}
	return datastorePutAlertJSONs(c, alertJSONs)
}

func datastoreGetAlertJSONs(c context.Context, alertJSONs []*model.AlertJSON) {
	alertJSONsNonGrouping := []*model.AlertJSONNonGrouping{}
	convertAlertJSONsToAlertJSONsNonGrouping(alertJSONs, &alertJSONsNonGrouping)
	datastore.Get(c, alertJSONsNonGrouping)
	convertAlertJSONsNonGroupingToAlertJSONs(alertJSONsNonGrouping, &alertJSONs)
}

func datastoreCreateAlertQuery() *datastore.Query {
	return datastore.NewQuery("AlertJSONNonGrouping")
}

func datastoreGetAlertsByQuery(c context.Context, alertJSONs *[]*model.AlertJSON, q *datastore.Query) error {
	alertJSONsNonGrouping := []*model.AlertJSONNonGrouping{}
	err := datastore.GetAll(c, q, &alertJSONsNonGrouping)
	if err != nil {
		return err
	}
	convertAlertJSONsNonGroupingToAlertJSONs(alertJSONsNonGrouping, alertJSONs)
	return nil
}

func putAlertsDatastore(c context.Context, tree string, alertsSummary *messages.AlertsSummary, autoResolve bool) error {
	treeKey := datastore.MakeKey(c, "Tree", tree)
	now := clock.Now(c).UTC()

	// Search for existing entities to preserve resolved status.
	alertJSONs := []*model.AlertJSON{}
	alertMap := make(map[string]*messages.Alert)
	for _, alert := range alertsSummary.Alerts {
		alertJSONs = append(alertJSONs, &model.AlertJSON{
			ID:           alert.Key,
			Tree:         treeKey,
			Resolved:     false,
			AutoResolved: false,
		})
		alertMap[alert.Key] = alert
	}

	// Add/modify alerts.
	err := datastore.RunInTransaction(c, func(c context.Context) error {
		// Get any existing keys to preserve resolved status, assign updated content.
		datastoreGetAlertJSONs(c, alertJSONs)
		for i, alert := range alertsSummary.Alerts {
			alertJSONs[i].Date = now
			var err error
			alertJSONs[i].Contents, err = json.Marshal(alert)
			if err != nil {
				return err
			}
			// Unresolve autoresolved alerts.
			// TODO (nqmtuan): Check if we can simplify the condition.
			if alertJSONs[i].Resolved && alertJSONs[i].AutoResolved {
				alertJSONs[i].Resolved = false
				alertJSONs[i].AutoResolved = false
			}
		}
		return datastorePutAlertJSONs(c, alertJSONs)
	}, nil)
	if err != nil {
		return err
	}

	if autoResolve {
		// Ideally this request would be performed in a transaction, but it can exceed the datastore API request size limit.
		alertJSONs = []*model.AlertJSON{}
		q := datastoreCreateAlertQuery().Ancestor(treeKey).Eq("Resolved", false)
		openAlerts := []*model.AlertJSON{}
		err = datastoreGetAlertsByQuery(c, &openAlerts, q)
		if err != nil {
			return err
		}
		for _, alert := range openAlerts {
			if _, modified := alertMap[alert.ID]; !modified {
				alert.Resolved = true
				alert.AutoResolved = true
				alert.ResolvedDate = now
				alertJSONs = append(alertJSONs, alert)

				// Avoid really large datastore Put.
				if len(alertJSONs) > maxAlertsAutoResolveCount {
					err = datastorePutAlertJSONs(c, alertJSONs)
					if err != nil {
						return err
					}
					alertJSONs = []*model.AlertJSON{}
				}
			}
		}
		err = datastorePutAlertJSONs(c, alertJSONs)
		if err != nil {
			return err
		}
	}

	revisionSummaryJSONs := make([]model.RevisionSummaryJSON,
		len(alertsSummary.RevisionSummaries))
	i := 0
	for key, summary := range alertsSummary.RevisionSummaries {
		revisionSummaryJSONs[i].ID = key
		revisionSummaryJSONs[i].Tree = treeKey
		revisionSummaryJSONs[i].Date = now
		revisionSummaryJSONs[i].Contents, err = json.Marshal(summary)
		if err != nil {
			return err
		}

		i++
	}

	return datastore.Put(c, revisionSummaryJSONs)
}

// FlushOldAlertsHandler deletes old resolved alerts from datastore.
func FlushOldAlertsHandler(ctx *router.Context) {
	c, w := ctx.Context, ctx.Writer

	numDeleted, err := flushOldAlerts(c)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	s := fmt.Sprintf("deleted %d alerts", numDeleted)
	logging.Debugf(c, s)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s))
}

func flushOldAlerts(c context.Context) (int, error) {
	q := datastore.NewQuery("AlertJSONNonGrouping").Eq("Resolved", true)
	q = q.Lt("ResolvedDate", clock.Get(c).Now().Add(-resolvedAlertExpiration))
	q = q.KeysOnly(true)

	results := []*model.AlertJSONNonGrouping{}
	err := datastore.GetAll(c, q, &results)
	if err != nil {
		return 0, fmt.Errorf("error fetching alerts to delete: %s", err)
	}

	err = datastore.Delete(c, results)
	if err != nil {
		return 0, fmt.Errorf("error deleting alerts: %s", err)
	}

	return len(results), nil
}
