package handler

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"infra/appengine/sheriff-o-matic/config"
)

// MigrateToUngroupedAlerts migrates annotation data in datastore
// to the new table when we switch off automatic grouping.
func MigrateToUngroupedAlerts(ctx *router.Context) {
	c, w := ctx.Context, ctx.Writer
	logging.Infof(c, "Running migration script to disable grouping")

	// This is a potentially dangerous operation, since it replaces the whole
	// AnnotationNonGrouping table with new data.
	// To play safe, we do not expect this to be run when AnnotationNonGrouping is
	// in use.
	if !config.EnableAutoGrouping {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("This should not be run when autogrouping is disabled."))
		return
	}
	w.Write([]byte("Successful."))
}
