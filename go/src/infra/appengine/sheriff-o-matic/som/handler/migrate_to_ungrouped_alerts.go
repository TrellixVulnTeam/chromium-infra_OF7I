package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"infra/appengine/sheriff-o-matic/config"
	"infra/appengine/sheriff-o-matic/som/analyzer"
	"infra/appengine/sheriff-o-matic/som/model"
)

var (
	somTrees = []string{
		"android",
		"chrome_browser_release",
		"chromeos",
		"chromium",
		"chromium.clang",
		"chromium.gpu.fyi",
		"chromium.perf",
		"fuchsia",
		"ios",
	}
)

// alertPopulatorFunc and uuidGenerationFunc are for unit testing.
type alertPopulatorFunc func(c context.Context) error
type uuidGenerationFunc func() string

func generateUUID() string {
	return uuid.New().String()
}

// MigrateToUngroupedAlerts migrates annotation data in datastore
// to the new table when we switch off automatic grouping.
func MigrateToUngroupedAlerts(ctx *router.Context) {
	c, w := ctx.Context, ctx.Writer
	if err := migrateToUngroupedAlerts(c, config.EnableAutoGrouping, populateAlertsNonGrouping); err != nil {
		logging.Errorf(c, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("Successful."))
}

func migrateToUngroupedAlerts(c context.Context, autogrouping bool, apfn alertPopulatorFunc) error {
	// This is a potentially dangerous operation, since it replaces the whole
	// AnnotationNonGrouping table with new data.
	// To play safe, we do not expect this to be run when AnnotationNonGrouping is
	// in use.
	if !autogrouping {
		return errors.New("this should not be run when autogrouping is disabled")
	}
	if err := cleanAnnotationNonGroupingTable(c); err != nil {
		return err
	}
	if err := apfn(c); err != nil {
		return err
	}

	for _, tree := range somTrees {
		if err := migrateAnnotationsForTree(c, tree); err != nil {
			return err
		}
	}
	return nil
}

func migrateAnnotationsForTree(c context.Context, tree string) error {
	treeKey := datastore.MakeKey(c, "Tree", tree)

	// annotations is the list of current annotations for the tree.
	annotations := []*model.Annotation{}
	q := datastore.NewQuery("Annotation").Ancestor(treeKey)
	if err := datastore.GetAll(c, q, &annotations); err != nil {
		return err
	}

	// alerts is the list of current UNRESOLVED alerts for the tree.
	// We should not care about annotations associated with resolved alerts.
	alerts := []*model.AlertJSON{}
	q = datastore.NewQuery("AlertJSON").Ancestor(treeKey).Eq("Resolved", false)
	if err := datastore.GetAll(c, q, &alerts); err != nil {
		return err
	}

	annotations = filterAnnotationsByCurrentAlerts(annotations, alerts)

	// alertsNonGrouping is the list of new alerts we need to create annotations for.
	alertsNonGrouping := []*model.AlertJSONNonGrouping{}
	q = datastore.NewQuery("AlertJSONNonGrouping").Ancestor(treeKey)
	if err := datastore.GetAll(c, q, &alertsNonGrouping); err != nil {
		return err
	}

	annotationsNonGrouping, err := generateAnnotationsNonGrouping(c, annotations, alertsNonGrouping, generateUUID)
	if err != nil {
		return err
	}

	if err := datastore.Put(c, annotationsNonGrouping); err != nil {
		return err
	}
	return nil
}

func generateAnnotationsNonGrouping(c context.Context, annotations []*model.Annotation, alertsNonGrouping []*model.AlertJSONNonGrouping, gf uuidGenerationFunc) ([]*model.AnnotationNonGrouping, error) {
	annotationsNonGrouping := []*model.AnnotationNonGrouping{}
	stepNameToAlertKeyMap, err := generateStepNameToAlertKeyMap(alertsNonGrouping)
	if err != nil {
		return annotationsNonGrouping, err
	}
	// Process annotations that do not belong to a group
	for _, ann := range annotations {
		// ann.GroupID == "" means ann is neither a group nor belongs to a group.
		if ann.GroupID == "" {
			stepName, err := ann.GetStepName()
			if err != nil {
				return []*model.AnnotationNonGrouping{}, err
			}
			alertKeys, ok := stepNameToAlertKeyMap[stepName]
			if !ok {
				continue
			}
			newAnns, err := generateAnnotationsNonGroupingForSingleAnnotation(ann, alertKeys, gf)
			if err != nil {
				return []*model.AnnotationNonGrouping{}, err
			}
			annotationsNonGrouping = append(annotationsNonGrouping, newAnns...)
		}
	}

	// Process annotations that belong to a group
	newAnns, err := generateAnnotationsNonGroupingForGroupedAnnotations(annotations, stepNameToAlertKeyMap)
	if err != nil {
		return []*model.AnnotationNonGrouping{}, err
	}
	annotationsNonGrouping = append(annotationsNonGrouping, newAnns...)
	return annotationsNonGrouping, nil
}

func generateAnnotationsNonGroupingForGroupedAnnotations(annotations []*model.Annotation, stepNameToAlertKeyMap map[string][]string) ([]*model.AnnotationNonGrouping, error) {
	ret := []*model.AnnotationNonGrouping{}

	// Preprocess to group annotations by their groups.
	groupIDToGroupAnnotationMap := make(map[string]*model.Annotation)
	groupIDToChildAnnotationsMap := make(map[string][]*model.Annotation)
	for _, ann := range annotations {
		// This means ann is either a group or belongs to a group
		if ann.GroupID != "" {
			if ann.IsGroupAnnotation() {
				groupIDToGroupAnnotationMap[ann.Key] = ann
			} else {
				if _, ok := groupIDToChildAnnotationsMap[ann.GroupID]; !ok {
					groupIDToChildAnnotationsMap[ann.GroupID] = []*model.Annotation{}
				}
				groupIDToChildAnnotationsMap[ann.GroupID] = append(groupIDToChildAnnotationsMap[ann.GroupID], ann)
			}
		}
	}

	// Process group by group.
	for groupID, groupAnn := range groupIDToGroupAnnotationMap {
		// groupIDToChildAnnotationsMap[groupID] is guaranteed to exist, thanks to filterAnnotationsByCurrentAlerts.
		childAnns, _ := groupIDToChildAnnotationsMap[groupID]
		newGroupAnn := model.AnnotationNonGrouping(*groupAnn)
		ret = append(ret, &newGroupAnn)

		for _, ann := range childAnns {
			newAnns, err := generateAnnotationsNonGroupingForGroup(ann, stepNameToAlertKeyMap, newGroupAnn.Key)
			if err != nil {
				return []*model.AnnotationNonGrouping{}, err
			}
			ret = append(ret, newAnns...)
		}
	}
	return ret, nil
}

func generateAnnotationsNonGroupingForGroup(ann *model.Annotation, stepNameToAlertKeyMap map[string][]string, groupID string) ([]*model.AnnotationNonGrouping, error) {
	ret := []*model.AnnotationNonGrouping{}
	stepName, err := ann.GetStepName()
	if err != nil {
		return ret, err
	}
	alertKeys, ok := stepNameToAlertKeyMap[stepName]
	if !ok {
		return ret, nil
	}
	for _, alertKey := range alertKeys {
		newAnn := model.AnnotationNonGrouping(*ann)
		newAnn.Key = alertKey
		newAnn.KeyDigest = model.GenerateKeyDigest(alertKey)
		newAnn.GroupID = groupID
		ret = append(ret, &newAnn)
	}
	return ret, nil
}

func generateAnnotationsNonGroupingForSingleAnnotation(ann *model.Annotation, alertKeys []string, gf uuidGenerationFunc) ([]*model.AnnotationNonGrouping, error) {
	ret := []*model.AnnotationNonGrouping{}
	if len(alertKeys) == 1 {
		// Old annotation corresponds to a new annotation.
		// We don't need to create a group for this.
		newAnn := model.AnnotationNonGrouping(*ann)
		newAnn.Key = alertKeys[0]
		newAnn.KeyDigest = model.GenerateKeyDigest(newAnn.Key)
		ret = append(ret, &newAnn)
	} else {
		stepName, err := ann.GetStepName()
		if err != nil {
			return ret, err
		}

		// Create a new group
		groupName := fmt.Sprintf("Step %q failed in %d builders", stepName, len(alertKeys))
		groupID := gf()
		groupAnn := model.AnnotationNonGrouping(*ann)
		groupAnn.Key = groupID
		groupAnn.KeyDigest = model.GenerateKeyDigest(groupID)
		groupAnn.GroupID = groupName
		ret = append(ret, &groupAnn)

		//Add alerts to the group.
		for _, alertKey := range alertKeys {
			newAnn := &model.AnnotationNonGrouping{
				Tree:             groupAnn.Tree,
				Key:              alertKey,
				KeyDigest:        model.GenerateKeyDigest(alertKey),
				GroupID:          groupAnn.Key,
				ModificationTime: groupAnn.ModificationTime,
			}
			ret = append(ret, newAnn)
		}
	}
	return ret, nil
}

// generateStepNameToAlertKeyMap generates a mapping between step name and a list of alert ID.
func generateStepNameToAlertKeyMap(alertsNonGrouping []*model.AlertJSONNonGrouping) (map[string][]string, error) {
	m := make(map[string][]string)
	for _, alert := range alertsNonGrouping {
		stepName, err := alert.GetStepName()
		if err != nil {
			return m, err
		}
		if _, ok := m[stepName]; !ok {
			m[stepName] = []string{}
		}
		m[stepName] = append(m[stepName], alert.ID)
	}
	return m, nil
}

// filterAnnotationsByCurrentAlerts filters annotations by keeping only those associated with alerts.
func filterAnnotationsByCurrentAlerts(annotations []*model.Annotation, alerts []*model.AlertJSON) []*model.Annotation {
	alertMap := make(map[string]bool)
	groupMap := make(map[string]bool)
	for _, alert := range alerts {
		alertMap[alert.ID] = true
	}

	// Add relevant single annotations
	ret := []*model.Annotation{}
	for _, ann := range annotations {
		if _, ok := alertMap[ann.Key]; ok {
			ret = append(ret, ann)
			if ann.GroupID != "" {
				groupMap[ann.GroupID] = true
			}
		}
	}

	// Add relevant groups
	for _, ann := range annotations {
		if _, ok := groupMap[ann.Key]; ok {
			ret = append(ret, ann)
		}
	}
	return ret
}

func populateAlertsNonGrouping(c context.Context) error {
	if err := cleanAlertJSONNonGroupingTable(c); err != nil {
		return err
	}

	prevConfig := config.EnableAutoGrouping
	config.EnableAutoGrouping = false
	defer func() {
		config.EnableAutoGrouping = prevConfig
	}()

	a := analyzer.CreateAnalyzer(c)
	for _, tree := range somTrees {
		logging.Infof(c, "Populate alerts for tree %s", tree)
		if _, err := generateBigQueryAlerts(c, a, tree); err != nil {
			return err
		}
	}
	return nil
}

func cleanAnnotationNonGroupingTable(c context.Context) error {
	q := datastore.NewQuery("AnnotationNonGrouping")
	annotationsNonGrouping := []*model.AnnotationNonGrouping{}
	if err := datastore.GetAll(c, q, &annotationsNonGrouping); err != nil {
		return err
	}
	if err := datastore.Delete(c, annotationsNonGrouping); err != nil {
		return err
	}
	return nil
}

func cleanAlertJSONNonGroupingTable(c context.Context) error {
	q := datastore.NewQuery("AlertJSONNonGrouping")
	alertJSONsNonGrouping := []*model.AlertJSONNonGrouping{}
	if err := datastore.GetAll(c, q, &alertJSONsNonGrouping); err != nil {
		return err
	}
	if err := datastore.Delete(c, alertJSONsNonGrouping); err != nil {
		return err
	}
	return nil
}
