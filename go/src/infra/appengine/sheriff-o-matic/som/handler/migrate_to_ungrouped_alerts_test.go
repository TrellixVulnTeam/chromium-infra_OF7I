package handler

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
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
	Convey("test generate annotations", t, func() {
		gf := func() string {
			return "group 1"
		}

		annotations := []*model.Annotation{
			{
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
				ID: "tree:project:bucket:builder1:step1:0",
			},
			{
				ID: "tree:project:bucket:builder1:step2:0",
			},
			{
				ID: "tree:project:bucket:builder2:step2:0",
			},
		}

		expected := []*model.AnnotationNonGrouping{
			{
				Key:        "tree:project:bucket:builder1:step1:0",
				KeyDigest:  "9571f9904d25259886f8c1c74c743b5b8201e1ca",
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
				Key:       "tree:project:bucket:builder1:step2:0",
				KeyDigest: "5c2976aae380eb2de0b09c20a8bfe947a03d8483",
				GroupID:   "group 1",
			},
			{
				Key:       "tree:project:bucket:builder2:step2:0",
				KeyDigest: "67ea5df8c714f4046ac970ba16236a86180312b8",
				GroupID:   "group 1",
			},
		}
		newAnnotations, err := generateAnnotationsNonGrouping(context.Background(), annotations, alerts, gf)
		So(err, ShouldBeNil)
		So(newAnnotations, ShouldResemble, expected)
	})
}
