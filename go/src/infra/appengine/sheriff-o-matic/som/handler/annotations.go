// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/grpc"

	"infra/appengine/sheriff-o-matic/som/client"
	"infra/appengine/sheriff-o-matic/som/model"
	monorailv3 "infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/gae/service/info"
	"go.chromium.org/luci/gae/service/memcache"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/router"
)

const (
	annotationsCacheKey = "annotation-metadata"
	// annotations will expire after this amount of time
	annotationExpiration = time.Hour * 24 * 10
	// maxMonorailQuerySize is the maximum number of bugs per monorail query.
	maxMonorailQuerySize = 100
)

// AnnotationsIssueClient is for testing purpose
type AnnotationsIssueClient interface {
	SearchIssues(context.Context, *monorailv3.SearchIssuesRequest, ...grpc.CallOption) (*monorailv3.SearchIssuesResponse, error)
	MakeIssue(ctx context.Context, in *monorailv3.MakeIssueRequest, opts ...grpc.CallOption) (*monorailv3.Issue, error)
}

// AnnotationHandler handles annotation-related requests.
type AnnotationHandler struct {
	Bqh                 *BugQueueHandler
	MonorailIssueClient AnnotationsIssueClient
}

// MonorailBugData wrap around monorailv3.Issue to send to frontend.
type MonorailBugData struct {
	BugID     string `json:"id,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status,omitempty"`
}

// AnnotationResponse ... The Annotation object extended with cached bug data.
type AnnotationResponse struct {
	model.Annotation
	BugData map[string]MonorailBugData `json:"bug_data"`
}

func convertAnnotationsNonGroupingToAnnotations(annotationsNonGrouping []*model.AnnotationNonGrouping, annotations *[]*model.Annotation) {
	*annotations = make([]*model.Annotation, len(annotationsNonGrouping))
	for i, annotationNonGrouping := range annotationsNonGrouping {
		tmp := model.Annotation(*annotationNonGrouping)
		(*annotations)[i] = &tmp
	}
}

func convertAnnotationsToAnnotationsNonGrouping(annotations []*model.Annotation) []*model.AnnotationNonGrouping {
	annotationsNonGrouping := make([]*model.AnnotationNonGrouping, len(annotations))
	for i, annotation := range annotations {
		tmp := model.AnnotationNonGrouping(*annotation)
		annotationsNonGrouping[i] = &tmp
	}
	return annotationsNonGrouping
}

func datastoreGetAnnotation(c context.Context, annotation *model.Annotation) error {
	annotationNonGrouping := model.AnnotationNonGrouping(*annotation)
	err := datastore.Get(c, &annotationNonGrouping)
	if err != nil {
		return err
	}
	*annotation = model.Annotation(annotationNonGrouping)
	return nil
}

func datastorePutAnnotation(c context.Context, annotation *model.Annotation) error {
	annotations := []*model.Annotation{annotation}
	return datastorePutAnnotations(c, annotations)
}

func datastorePutAnnotations(c context.Context, annotations []*model.Annotation) error {
	annotationsNonGrouping := convertAnnotationsToAnnotationsNonGrouping(annotations)
	return datastore.Put(c, annotationsNonGrouping)
}

func datastoreCreateAnnotationQuery() *datastore.Query {
	return datastore.NewQuery("AnnotationNonGrouping")
}

func datastoreGetAnnotationsByQuery(c context.Context, annotations *[]*model.Annotation, q *datastore.Query) error {
	annotationsNonGrouping := []*model.AnnotationNonGrouping{}
	err := datastore.GetAll(c, q, &annotationsNonGrouping)
	if err != nil {
		return err
	}
	convertAnnotationsNonGroupingToAnnotations(annotationsNonGrouping, annotations)
	return nil
}

func datastoreDeleteAnnotations(c context.Context, annotations []*model.Annotation) error {
	annotationsNonGrouping := convertAnnotationsToAnnotationsNonGrouping(annotations)
	return datastore.Delete(c, annotationsNonGrouping)
}

func convertBugData(bugData *monorailv3.Issue) (MonorailBugData, error) {
	projectID, bugID, err := client.ParseMonorailIssueName(bugData.Name)
	if err != nil {
		return MonorailBugData{}, err
	}
	return MonorailBugData{
		BugID:     bugID,
		ProjectID: projectID,
		Status:    bugData.Status.Status,
		Summary:   bugData.Summary,
	}, nil
}

// Convert data from model.Annotation type to AnnotationResponse type by populating monorail data.
func makeAnnotationResponse(annotations *model.Annotation, meta map[string]*monorailv3.Issue) (*AnnotationResponse, error) {
	bugs := make(map[string]MonorailBugData)
	for _, b := range annotations.Bugs {
		if bugData, ok := meta[b.BugID]; ok {
			mbd, err := convertBugData(bugData)
			if err != nil {
				return nil, err
			}
			bugs[b.BugID] = mbd
		}
	}
	return &AnnotationResponse{*annotations, bugs}, nil
}

func filterAnnotations(annotations []*model.Annotation, activeKeys map[string]interface{}) []*model.Annotation {
	ret := []*model.Annotation{}
	groups := map[string]interface{}{}

	// Process annotations not belonging to a group
	for _, a := range annotations {
		if _, ok := activeKeys[a.Key]; ok {
			ret = append(ret, a)
			if a.GroupID != "" {
				groups[a.GroupID] = nil
			}
		}
	}

	// Process annotations belonging to a group
	for _, a := range annotations {
		if _, ok := groups[a.Key]; ok {
			ret = append(ret, a)
		}
	}
	return ret
}

// GetAnnotationsHandler retrieves a set of annotations.
func (ah *AnnotationHandler) GetAnnotationsHandler(ctx *router.Context, activeKeys map[string]interface{}) {
	c, w, p := ctx.Context, ctx.Writer, ctx.Params

	tree := p.ByName("tree")

	q := datastoreCreateAnnotationQuery()

	if tree != "" {
		q = q.Ancestor(datastore.MakeKey(c, "Tree", tree))
	}

	annotations := []*model.Annotation{}
	datastoreGetAnnotationsByQuery(c, &annotations, q)

	annotations = filterAnnotations(annotations, activeKeys)

	meta, err := ah.getAnnotationsMetaData(ctx)

	if err != nil {
		logging.Errorf(c, "while fetching annotation metadata")
	}

	response := make([]*AnnotationResponse, len(annotations))
	for i, a := range annotations {
		resp, err := makeAnnotationResponse(a, meta)
		if err != nil {
			errStatus(c, w, http.StatusInternalServerError, err.Error())
			return
		}
		response[i] = resp
	}

	data, err := json.Marshal(response)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (ah *AnnotationHandler) getAnnotationsMetaData(ctx *router.Context) (map[string]*monorailv3.Issue, error) {
	c := ctx.Context
	item, err := memcache.GetKey(c, annotationsCacheKey)
	val := make(map[string]*monorailv3.Issue)

	if err == memcache.ErrCacheMiss {
		logging.Warningf(c, "No annotation metadata in memcache, refreshing...")
		val, err = ah.refreshAnnotations(ctx, nil)

		if err != nil {
			return nil, err
		}
	} else {
		if err = json.Unmarshal(item.Value(), &val); err != nil {
			logging.Errorf(c, "while unmarshaling metadata in getAnnotationsMetaData")
			return nil, err
		}
	}
	return val, nil
}

// RefreshAnnotationsHandler refreshes the set of annotations.
func (ah *AnnotationHandler) RefreshAnnotationsHandler(ctx *router.Context) {
	c, w := ctx.Context, ctx.Writer

	bugMap, err := ah.refreshAnnotations(ctx, nil)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	data, err := json.Marshal(bugMap)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// Builds a map keyed by projectId (i.e "chromium", "fuchsia"), value contains
// the chunks for the projectId. Each chunk contains at most 100 bugID.
// Monorail only returns a maximum of 100 bugs at a time, so we need to break queries into chunks.
func createProjectChunksMapping(bugs []model.MonorailBug, chunkSize int) map[string][][]string {
	projectIDToBugIDMap := make(map[string][]string)
	for _, bug := range bugs {
		if bugList, ok := projectIDToBugIDMap[bug.ProjectID]; ok {
			projectIDToBugIDMap[bug.ProjectID] = append(bugList, bug.BugID)
		} else {
			projectIDToBugIDMap[bug.ProjectID] = []string{bug.BugID}
		}
	}
	result := make(map[string][][]string)
	for projectID, bugIDs := range projectIDToBugIDMap {
		result[projectID] = breakToChunks(bugIDs, chunkSize)
	}
	return result
}

func generateQueryFromChunk(chunk []string) string {
	bits := make([]string, len(chunk))
	for i, bugID := range chunk {
		bits[i] = "id=" + bugID
	}
	return strings.Join(bits, " OR ")
}

func createSearchIssueRequests(c context.Context, projectChunkMap map[string][][]string) []*monorailv3.SearchIssuesRequest {
	reqs := []*monorailv3.SearchIssuesRequest{}
	for projectID, chunkList := range projectChunkMap {
		for _, chunk := range chunkList {
			query := generateQueryFromChunk(chunk)
			projectResourceName := client.GetMonorailProjectResourceName(projectID)
			req := &monorailv3.SearchIssuesRequest{
				Projects: []string{projectResourceName},
				Query:    query,
			}
			reqs = append(reqs, req)
		}
	}
	return reqs
}

func breakToChunks(items []string, chunkSize int) [][]string {
	var result [][]string
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		result = append(result, items[i:end])
	}
	return result
}

func filterDuplicateBugs(bugs []model.MonorailBug) []model.MonorailBug {
	bugIds := map[string]interface{}{}
	filteredBugs := []model.MonorailBug{}
	for _, bug := range bugs {
		if _, exist := bugIds[bug.BugID]; !exist {
			bugIds[bug.BugID] = nil
			filteredBugs = append(filteredBugs, bug)
		}
	}
	return filteredBugs
}

func (ah *AnnotationHandler) searchIssues(c context.Context, req *monorailv3.SearchIssuesRequest) (*monorailv3.SearchIssuesResponse, error) {
	logging.Infof(c, "Query monorail for bugs: %v", req)
	resp, err := ah.MonorailIssueClient.SearchIssues(c, req)
	if err == nil {
		logging.Infof(c, "Got %d bugs", len(resp.Issues))
	}
	return resp, err
}

// Update the cache for annotation bug data.
func (ah *AnnotationHandler) refreshAnnotations(ctx *router.Context, a *model.Annotation) (map[string]*monorailv3.Issue, error) {
	c := ctx.Context
	q := datastoreCreateAnnotationQuery()
	results := []*model.Annotation{}
	datastoreGetAnnotationsByQuery(c, &results, q)

	// Monorail takes queries of the format id:1,2,3 (gets bugs with those ids).
	if a != nil {
		results = append(results, a)
	}

	allBugs := []model.MonorailBug{}
	for _, annotation := range results {
		allBugs = append(allBugs, annotation.Bugs...)
	}

	allBugs = filterDuplicateBugs(allBugs)
	projectChunkMap := createProjectChunksMapping(allBugs, maxMonorailQuerySize)
	reqs := createSearchIssueRequests(c, projectChunkMap)
	m := make(map[string]*monorailv3.Issue)
	lock := sync.Mutex{}
	err := parallel.WorkPool(8, func(taskC chan<- func() error) {
		for i := 0; i < len(reqs); i++ {
			req := reqs[i]
			taskC <- func() error {
				resp, err := ah.searchIssues(c, req)
				if err != nil {
					logging.Errorf(c, "error getting bugs from monorail: %v", err)
					return err
				}
				for _, b := range resp.Issues {
					_, bugID, err := client.ParseMonorailIssueName(b.Name)
					if err != nil {
						return err
					}
					// TODO (crbug.com/1127471) Key should also include projectID
					lock.Lock()
					m[bugID] = b
					lock.Unlock()
				}
				return nil
			}
		}
	})

	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	item := memcache.NewItem(c, annotationsCacheKey).SetValue(bytes)

	err = memcache.Set(c, item)

	if err != nil {
		return nil, err
	}

	return m, nil
}

type postRequest struct {
	XSRFToken string           `json:"xsrf_token"`
	Data      *json.RawMessage `json:"data"`
}

// PostAnnotationsHandler handles updates to annotations.
func (ah *AnnotationHandler) PostAnnotationsHandler(ctx *router.Context) {
	c, w, r, p := ctx.Context, ctx.Writer, ctx.Request, ctx.Params

	tree := p.ByName("tree")
	action := p.ByName("action")
	if action != "add" && action != "remove" {
		errStatus(c, w, http.StatusBadRequest, "unrecognized annotation action")
		return
	}

	req := &postRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		errStatus(c, w, http.StatusBadRequest, fmt.Sprintf("while decoding request: %s", err))
		return
	}

	if err = xsrf.Check(c, req.XSRFToken); err != nil {
		errStatus(c, w, http.StatusForbidden, err.Error())
		return
	}

	// Extract the annotation key from the otherwise unparsed body.
	rawJSON := struct{ Key string }{}
	if err = json.Unmarshal([]byte(*req.Data), &rawJSON); err != nil {
		errStatus(c, w, http.StatusBadRequest, fmt.Sprintf("while decoding request: %s", err))
	}

	key := rawJSON.Key

	annotation := &model.Annotation{
		Tree:      datastore.MakeKey(c, "Tree", tree),
		KeyDigest: fmt.Sprintf("%x", sha1.Sum([]byte(key))),
		Key:       key,
	}

	err = datastoreGetAnnotation(c, annotation)
	if action == "remove" && err != nil {
		logging.Errorf(c, "while getting %s: %s", key, err)
		errStatus(c, w, http.StatusNotFound, fmt.Sprintf("Annotation %s not found", key))
		return
	}

	needRefresh := false
	if info.AppID(c) != "" && info.AppID(c) != "app" {
		c = appengine.WithContext(c, r)
	}
	// The annotation probably doesn't exist if we're adding something.
	data := bytes.NewReader([]byte(*req.Data))
	if action == "add" {
		needRefresh, err = annotation.Add(c, data)
	} else if action == "remove" {
		needRefresh, err = annotation.Remove(c, data)
	}

	if err != nil {
		errStatus(c, w, http.StatusBadRequest, err.Error())
		return
	}

	err = r.Body.Close()
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	err = datastorePutAnnotation(c, annotation)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	var m map[string]*monorailv3.Issue
	// Refresh the annotation cache on a write. Note that we want the rest of the
	// code to still run even if this fails.
	if needRefresh {
		logging.Infof(c, "Refreshing annotation metadata, due to a stateful modification.")
		m, err = ah.refreshAnnotations(ctx, annotation)
		if err != nil {
			logging.Errorf(c, "while refreshing annotation cache on post: %s", err)
		}
	} else {
		m, err = ah.getAnnotationsMetaData(ctx)
		if err != nil {
			logging.Errorf(c, "while getting annotation metadata: %s", err)
		}
	}

	annotationResp, err := makeAnnotationResponse(annotation, m)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	resp, err := json.Marshal(annotationResp)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// FlushOldAnnotationsHandler culls obsolete annotations from the datastore.
// TODO (crbug.com/1079068): Perhaps we want to revisit flush annotation logic.
func FlushOldAnnotationsHandler(ctx *router.Context) {
	c, w := ctx.Context, ctx.Writer

	numDeleted, err := flushOldAnnotations(c)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	s := fmt.Sprintf("deleted %d annotations", numDeleted)
	logging.Debugf(c, s)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(s))
}

func flushOldAnnotations(c context.Context) (int, error) {
	q := datastoreCreateAnnotationQuery()
	q = q.Lt("ModificationTime", clock.Get(c).Now().Add(-annotationExpiration))
	q = q.KeysOnly(true)

	results := []*model.Annotation{}
	err := datastoreGetAnnotationsByQuery(c, &results, q)
	if err != nil {
		return 0, fmt.Errorf("while fetching annotations to delete: %s", err)
	}

	for _, ann := range results {
		logging.Debugf(c, "Deleting %#v\n", ann)
	}

	err = datastoreDeleteAnnotations(c, results)
	if err != nil {
		return 0, fmt.Errorf("while deleting annotations: %s", err)
	}

	return len(results), nil
}

// FileBugHandler files a new bug in monorail.
func (ah *AnnotationHandler) FileBugHandler(ctx *router.Context) {
	c, w, r := ctx.Context, ctx.Writer, ctx.Request

	req := &postRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		errStatus(c, w, http.StatusBadRequest, fmt.Sprintf("while decoding request: %s", err))
		return
	}

	if err = xsrf.Check(c, req.XSRFToken); err != nil {
		errStatus(c, w, http.StatusForbidden, err.Error())
		return
	}

	rawJSON := struct {
		Summary     string
		Description string
		ProjectID   string
		Cc          []string
		Priority    string
		Labels      []string
	}{}
	if err = json.Unmarshal([]byte(*req.Data), &rawJSON); err != nil {
		errStatus(c, w, http.StatusBadRequest, fmt.Sprintf("while decoding request: %s", err))
	}

	sa, err := info.ServiceAccount(c)
	if err != nil {
		logging.Errorf(c, "failed to get service account: %v", err)
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	user := auth.CurrentIdentity(c)
	description := fmt.Sprintf("Filed by %s on behalf of %s\n\n%s", sa, user.Email(),
		rawJSON.Description)

	reqCCUsers := make([]*monorailv3.Issue_UserValue, len(rawJSON.Cc))
	for i, user := range rawJSON.Cc {
		reqCCUsers[i] = &monorailv3.Issue_UserValue{
			User: "users/" + user,
		}
	}

	reqLabels := make([]*monorailv3.Issue_LabelValue, len(rawJSON.Labels))
	for i, label := range rawJSON.Labels {
		reqLabels[i] = &monorailv3.Issue_LabelValue{
			Label: label,
		}
	}

	issueReq := &monorailv3.MakeIssueRequest{
		Parent:      client.GetMonorailProjectResourceName(rawJSON.ProjectID),
		Description: description,
		Issue: &monorailv3.Issue{
			Summary: rawJSON.Summary,
			Status: &monorailv3.Issue_StatusValue{
				Status: "Untriaged",
			},
			Labels:  reqLabels,
			CcUsers: reqCCUsers,
		},
		NotifyType: monorailv3.NotifyType_EMAIL,
	}
	processIssueRequest(c, rawJSON.ProjectID, issueReq)
	res, err := ah.MonorailIssueClient.MakeIssue(c, issueReq)
	if err != nil {
		logging.Errorf(c, "error inserting new Issue: %v", err)
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	out, err := json.Marshal(res)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	logging.Infof(c, "%v", out)
	w.Header().Set("Content-Type", "applications/json")
	w.Write(out)
}

func processIssueRequest(c context.Context, projectID string, req *monorailv3.MakeIssueRequest) error {
	if projectID == "chromium" {
		// For chromium project, we need to specify type
		fieldName, err := client.GetMonorailTypeField(c, projectID)
		if err != nil {
			return err
		}
		req.Issue.FieldValues = []*monorailv3.FieldValue{
			{
				Field: fieldName,
				Value: "Bug",
			},
		}
	}
	if projectID == "angleproject" {
		// For angleproject, the status is "New" instead of "Untriaged"
		req.Issue.Status.Status = "New"
		// Angleproject has different way of assigning priority
		mapping := map[string]string{
			"Pri-0": "Critical",
			"Pri-1": "High",
			"Pri-2": "Medium",
			"Pri-3": "Low",
		}
		fieldName, err := client.GetMonorailPriorityField(c, projectID)
		if err != nil {
			return err
		}
		for i, label := range req.Issue.Labels {
			if val, ok := mapping[label.Label]; ok {
				// Remove label
				n := len(req.Issue.Labels)
				req.Issue.Labels[i] = req.Issue.Labels[n-1]
				req.Issue.Labels = req.Issue.Labels[:n-1]
				req.Issue.FieldValues = []*monorailv3.FieldValue{
					{
						Field: fieldName,
						Value: val,
					},
				}
				break
			}
		}
		// Angleproject require type field
		typeFieldName, err := client.GetMonorailTypeField(c, projectID)
		if err != nil {
			return err
		}
		req.Issue.FieldValues = append(req.Issue.FieldValues, &monorailv3.FieldValue{
			Field: typeFieldName,
			Value: "Defect",
		})
	}
	return nil
}
