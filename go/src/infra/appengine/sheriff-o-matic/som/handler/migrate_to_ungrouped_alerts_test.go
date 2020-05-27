package handler

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"infra/appengine/sheriff-o-matic/som/model"
)

func fakepopulateAlertsNonGrouping(c context.Context) error {
	return nil
}

func TestMigrateToUngroupedAlerts(t *testing.T) {
	Convey("test migrate to ungrouped alerts should not run if autogrouping is off", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, false, fakepopulateAlertsNonGrouping)
		So(err, ShouldNotBeNil)
	})

	Convey("test migrate to ungrouped alerts", t, func() {
		c := gaetesting.TestingContext()
		err := migrateToUngroupedAlerts(c, true, fakepopulateAlertsNonGrouping)
		So(err, ShouldBeNil)
	})
}

func TestFilterAnnotationsByCurrentAlerts(t *testing.T) {
	Convey("test filter annotations by current alerts", t, func() {
		annotations := []*model.Annotation{
			{
				Key: "alert 1",
			},
			{
				Key: "alert 2",
			},
			{
				Key:     "alert 3",
				GroupID: "group id 1",
			},
			{
				Key:     "group id 1",
				GroupID: "group name 1",
			},
			{
				Key:     "alert 4",
				GroupID: "group id 2",
			},
			{
				Key:     "group id 2",
				GroupID: "group name 2",
			},
		}

		alerts := []*model.AlertJSON{
			{
				ID: "alert 1",
			},
			{
				ID: "alert 3",
			},
		}

		filteredAnnotation := filterAnnotationsByCurrentAlerts(annotations, alerts)
		expected := []*model.Annotation{
			{
				Key: "alert 1",
			},
			{
				Key:     "alert 3",
				GroupID: "group id 1",
			},
			{
				Key:     "group id 1",
				GroupID: "group name 1",
			},
		}
		So(filteredAnnotation, ShouldResemble, expected)
	})
}

func TestGenerateAnnotationsNonGrouping(t *testing.T) {
	gf := func() string {
		return "group 1"
	}
	c := gaetesting.TestingContext()
	Convey("test generate annotations for standalone annotations", t, func() {
		annotations := []*model.Annotation{
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree.step1",
				SnoozeTime: 123,
				Bugs: []model.MonorailBug{
					{
						BugID:     "123",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 1",
					},
				},
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree.step2",
				SnoozeTime: 456,
				Bugs: []model.MonorailBug{
					{
						BugID:     "456",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 2",
					},
				},
			},
		}

		alerts := []*model.AlertJSONNonGrouping{
			{
				ID: "tree$!project$!bucket$!builder1$!step1$!0",
			},
			{
				ID: "tree$!project$!bucket$!builder1$!step2$!0",
			},
			{
				ID: "tree$!project$!bucket$!builder2$!step2$!0",
			},
		}

		expected := []*model.AnnotationNonGrouping{
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree$!project$!bucket$!builder1$!step1$!0",
				KeyDigest:  "5e898632e69015f37557efbdc8314d7afa316883",
				SnoozeTime: 123,
				Bugs: []model.MonorailBug{
					{
						BugID:     "123",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 1",
					},
				},
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "group 1",
				KeyDigest:  "6d83066ffb09acd6bbdf7b1dbb6688283b8f7a02",
				GroupID:    "Step \"step2\" failed in 2 builders",
				SnoozeTime: 456,
				Bugs: []model.MonorailBug{
					{
						BugID:     "456",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 2",
					},
				},
			},
			{
				Tree:      datastore.MakeKey(c, "Tree", "tree"),
				Key:       "tree$!project$!bucket$!builder1$!step2$!0",
				KeyDigest: "b56302326d7b4cd5dbee9d506aaad19f50143315",
				GroupID:   "group 1",
			},
			{
				Tree:      datastore.MakeKey(c, "Tree", "tree"),
				Key:       "tree$!project$!bucket$!builder2$!step2$!0",
				KeyDigest: "66d36fb329c2af86fa19a695825877517c56b6ff",
				GroupID:   "group 1",
			},
		}
		newAnnotations, err := generateAnnotationsNonGrouping(context.Background(), annotations, alerts, gf)
		So(err, ShouldBeNil)
		So(newAnnotations, ShouldResemble, expected)
	})

	Convey("test generate annotations for grouped annotations", t, func() {
		annotations := []*model.Annotation{
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree.step1",
				SnoozeTime: 123,
				Bugs: []model.MonorailBug{
					{
						BugID:     "123",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 1",
					},
				},
				GroupID: "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree.step2",
				SnoozeTime: 456,
				Bugs: []model.MonorailBug{
					{
						BugID:     "456",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 2",
					},
				},
				GroupID: "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
				GroupID:    "My group",
				SnoozeTime: 789,
				Bugs: []model.MonorailBug{
					{
						BugID:     "789",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 3",
					},
				},
			},
		}

		alerts := []*model.AlertJSONNonGrouping{
			{
				ID: "tree$!project$!bucket$!builder1$!step1$!0",
			},
			{
				ID: "tree$!project$!bucket$!builder1$!step2$!0",
			},
			{
				ID: "tree$!project$!bucket$!builder2$!step2$!0",
			},
		}

		expected := []*model.AnnotationNonGrouping{
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
				GroupID:    "My group",
				SnoozeTime: 789,
				Bugs: []model.MonorailBug{
					{
						BugID:     "789",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 3",
					},
				},
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree$!project$!bucket$!builder1$!step1$!0",
				KeyDigest:  "5e898632e69015f37557efbdc8314d7afa316883",
				SnoozeTime: 123,
				Bugs: []model.MonorailBug{
					{
						BugID:     "123",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 1",
					},
				},
				GroupID: "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
			},

			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree$!project$!bucket$!builder1$!step2$!0",
				KeyDigest:  "b56302326d7b4cd5dbee9d506aaad19f50143315",
				SnoozeTime: 456,
				Bugs: []model.MonorailBug{
					{
						BugID:     "456",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 2",
					},
				},
				GroupID: "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
			},
			{
				Tree:       datastore.MakeKey(c, "Tree", "tree"),
				Key:        "tree$!project$!bucket$!builder2$!step2$!0",
				KeyDigest:  "66d36fb329c2af86fa19a695825877517c56b6ff",
				SnoozeTime: 456,
				Bugs: []model.MonorailBug{
					{
						BugID:     "456",
						ProjectID: "chromium",
					},
				},
				Comments: []model.Comment{
					{
						Text: "Comment 2",
					},
				},
				GroupID: "e2d935ac-c623-4c10-b1e3-73bb54584f8f",
			},
		}
		newAnnotations, err := generateAnnotationsNonGrouping(context.Background(), annotations, alerts, gf)
		So(err, ShouldBeNil)
		So(newAnnotations, ShouldResemble, expected)
	})
}
