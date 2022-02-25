package bugs

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/common/tsmon/types"
)

var (
	// BugsCreatedCounter is the metric that counts the number of bugs
	// created by Weetbix, by project and bug-filing system.
	BugsCreatedCounter = metric.NewCounter("weetbix/bug_updater/bugs_created",
		"The number of bugs created by auto-bug filing, "+
			"by LUCI Project and bug-filing system.",
		&types.MetricMetadata{
			Units: "bugs",
		},
		// The LUCI project.
		field.String("project"),
		// The bug-filing system. Either "monorail" or "buganizer".
		field.String("bug_system"),
	)

	BugsUpdatedCounter = metric.NewCounter("weetbix/bug_updater/bugs_updated",
		"The number of bugs updated by auto-bug filing, "+
			"by LUCI Project and bug-filing system.",
		&types.MetricMetadata{
			Units: "bugs",
		},
		// The LUCI project.
		field.String("project"),
		// The bug-filing system. Either "monorail" or "buganizer".
		field.String("bug_system"))
)
