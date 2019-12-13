// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package userinput

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
)

func TestGetRequestFromFiles(t *testing.T) {
	Convey("test happy path for getRequestFromFiles", t, func() {
		content := `host1,powerunit_hostname=host1-rpm1,powerunit_outlet=.A1,pool=fake_pool
host2,powerunit_hostname=host2-rpm1,powerunit_outlet=.A2`
		fname := "happy"
		p, err := writeTempFile([]byte(content), fname)
		So(err, ShouldBeNil)
		defer os.Remove(p)

		req, err := GetRequestFromFiles(p)
		So(err, ShouldBeNil)
		So(len(req.GetDutProperties()), ShouldEqual, 2)
		dps := req.GetDutProperties()
		dpHost1 := getDutPropertyByHostname(dps, "host1")
		So(dpHost1.GetPool(), ShouldEqual, "fake_pool")
		So(dpHost1.GetRpm().GetPowerunitHostname(), ShouldEqual, "host1-rpm1")
		So(dpHost1.GetRpm().GetPowerunitOutlet(), ShouldEqual, ".A1")

		dpHost2 := getDutPropertyByHostname(dps, "host2")
		So(dpHost2.GetPool(), ShouldEqual, "")
		So(dpHost2.GetRpm().GetPowerunitHostname(), ShouldEqual, "host2-rpm1")
		So(dpHost2.GetRpm().GetPowerunitOutlet(), ShouldEqual, ".A2")
	})

	Convey("test partial rpm is passed in", t, func() {
		content := `host1,powerunit_hostname=host1-rpm1`
		fname := "partial_rpm"
		p, err := writeTempFile([]byte(content), fname)
		So(err, ShouldBeNil)
		defer os.Remove(p)

		req, err := GetRequestFromFiles(p)
		So(err, ShouldBeNil)
		So(len(req.GetDutProperties()), ShouldEqual, 1)
		dps := req.GetDutProperties()
		dpHost1 := getDutPropertyByHostname(dps, "host1")
		So(dpHost1.GetPool(), ShouldEqual, "")
		So(dpHost1.GetRpm(), ShouldNotBeNil)
		So(dpHost1.GetRpm().GetPowerunitHostname(), ShouldEqual, "host1-rpm1")
		So(dpHost1.GetRpm().GetPowerunitOutlet(), ShouldEqual, "")
	})

	Convey("test only pool is passed in", t, func() {
		content := `host1,pool=fake_pool`
		fname := "only_pool"
		p, err := writeTempFile([]byte(content), fname)
		So(err, ShouldBeNil)
		defer os.Remove(p)

		req, err := GetRequestFromFiles(p)
		So(err, ShouldBeNil)
		So(len(req.GetDutProperties()), ShouldEqual, 1)
		dps := req.GetDutProperties()
		dpHost1 := getDutPropertyByHostname(dps, "host1")
		So(dpHost1.GetPool(), ShouldEqual, "fake_pool")
		So(dpHost1.GetRpm(), ShouldBeNil)
	})

	Convey("test duplicated settings for getRequestFromFiles", t, func() {
		contents := map[string]string{
			"duplicated_pool":               `host1,pool=fake_pool,pool=fake_pool_2`,
			"duplicated_powerunit_hostname": `host1,powerunit_hostname=h1,powerunit_hostname=h2`,
			"duplicated_powerunit_outlet":   `host1,powerunit_outlet=o1,powerunit_outlet=o2`,
		}
		ps := writeBatchTempFiles(contents)
		So(len(ps), ShouldEqual, len(contents))
		for _, p := range ps {
			defer os.Remove(p)
			_, err := GetRequestFromFiles(p)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "already setup")
		}
	})
}

func getDutPropertyByHostname(duts []*fleet.DutProperty, h string) *fleet.DutProperty {
	for _, d := range duts {
		if d.GetHostname() == h {
			return d
		}
	}
	return nil
}

func writeBatchTempFiles(contents map[string]string) []string {
	ps := make([]string, 0, len(contents))
	for fn, c := range contents {
		p, err := writeTempFile([]byte(c), fn)
		if err != nil {
			return nil
		}
		ps = append(ps, p)
	}
	return ps
}
