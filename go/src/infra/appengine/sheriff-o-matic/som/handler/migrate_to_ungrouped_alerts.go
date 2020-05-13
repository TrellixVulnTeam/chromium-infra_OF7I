package handler

import (
	"context"
	"errors"
	"net/http"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"infra/appengine/sheriff-o-matic/config"
	"infra/appengine/sheriff-o-matic/som/analyzer"
	"infra/appengine/sheriff-o-matic/som/model"
)

// alertPopulatorFunc is for unit testing.
type alertPopulatorFunc func(c context.Context) error

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

	q := datastore.NewQuery("Annotation")
	annotations := []*model.Annotation{}
	if err := datastore.GetAll(c, q, &annotations); err != nil {
		return err
	}
	annotationsNonGrouping, err := generateAnnotationsNonGrouping(c, annotations)
	if err != nil {
		return err
	}

	if err := datastore.Put(c, annotationsNonGrouping); err != nil {
		return err
	}
	return nil
}

func generateAnnotationsNonGrouping(c context.Context, annotations []*model.Annotation) ([]*model.AnnotationNonGrouping, error) {
	annotationsNonGrouping := []*model.AnnotationNonGrouping{}
	// TODO(crbug.com/1043371): Implement this function.
	return annotationsNonGrouping, nil
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

	trees := []string{"android", "chrome_browser_release", "chromeos", "chromium", "chromium.clang", "chromium.gpu.fyi", "chromium.perf", "fuchsia", "ios"}
	a := analyzer.CreateAnalyzer(c)
	for _, tree := range trees {
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
