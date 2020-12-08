package datastore

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"

	fleet "infra/libs/fleet/protos"
	ufs "infra/libs/fleet/protos/go"
)

func mockAsset(id, lab string) *fleet.ChopsAsset {
	return &fleet.ChopsAsset{
		Id: id,
		Location: &ufs.Location{
			Lab:      lab,
			Row:      "Phobos-3",
			Rack:     "Deimos-0",
			Shelf:    "Olympus-Mons",
			Position: "Amazonis-Planitia",
		},
	}
}

func mockAssetInfo(id string) *ufs.AssetInfo {
	return &ufs.AssetInfo{
		AssetTag:           id,
		SerialNumber:       "1998A%26AT...15..249U",
		CostCenter:         "Mimas",
		GoogleCodeName:     "Lapetus",
		Model:              "Titan",
		BuildTarget:        "Rhea",
		ReferenceBoard:     "Enceladus",
		EthernetMacAddress: "DE:AD:BE:EF:00:FF",
		Sku:                "OrionArm",
	}
}

func assertLocationEqual(a, b *ufs.Location) {
	So(a.GetLab(), ShouldEqual, b.GetLab())
	So(a.GetRow(), ShouldEqual, b.GetRow())
	So(a.GetRack(), ShouldEqual, b.GetRack())
	So(a.GetShelf(), ShouldEqual, b.GetShelf())
	So(a.GetPosition(), ShouldEqual, b.GetPosition())
}

func TestAddAsset(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	asset1 := mockAsset("45673456237895", "lab1")
	asset2 := mockAsset("45673456237896", "lab2")
	asset3 := mockAsset("45673456237897", "lab3")
	asset4 := mockAsset("", "")
	Convey("Add asset to datastore", t, func() {
		Convey("Add asset to datastore", func() {
			req := []*fleet.ChopsAsset{asset1}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.ID, ShouldEqual, asset1.GetId())
			// Verify it can be fetched.
			reqGet := []string{asset1.GetId()}
			res = GetAssetsByID(ctx, reqGet)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.ID, ShouldEqual, asset1.GetId())
			res = GetAssetStatesByID(ctx, reqGet)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			stateStr := res[0].StateEntity.State.GetState().String()
			So(stateStr, ShouldEqual, "STATE_ONBOARDING")
		})
		Convey("Add existing asset to datastore", func() {
			req := []*fleet.ChopsAsset{asset2}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			So(res[0].Entity.ID, ShouldEqual, asset2.GetId())

			// Verify state update
			res = GetAssetStatesByID(ctx, []string{asset2.GetId()})
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			stateStr := res[0].StateEntity.State.GetState().String()
			So(stateStr, ShouldEqual, "STATE_ONBOARDING")

			// Verify adding existing asset
			req2 := []*fleet.ChopsAsset{asset2, asset3}
			res2, err := AddAssets(ctx, req2)
			So(err, ShouldBeNil)
			So(res2, ShouldNotBeNil)
			So(res2, ShouldHaveLength, 2)
			So(res2[0].Err, ShouldNotBeNil)
			So(res2[0].Err.Error(), ShouldContainSubstring, "Asset exists in the database")
			So(res2[1].Err, ShouldBeNil)
			So(res2[1].Entity.ID, ShouldEqual, asset3.GetId())

			// Verify state is changed for successfully added new asset
			res = GetAssetStatesByID(ctx, []string{asset2.GetId(), asset3.GetId()})
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			stateStr = res[0].StateEntity.State.GetState().String()
			So(stateStr, ShouldEqual, "STATE_ONBOARDING")
			stateStr = res[1].StateEntity.State.GetState().String()
			So(stateStr, ShouldEqual, "STATE_ONBOARDING")
		})
		Convey("Add asset without ID to datastore", func() {
			req := []*fleet.ChopsAsset{asset4}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
		})
	})
}

func TestAddAssetInfo(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	assetInfo1 := mockAssetInfo("45673456237895")
	assetInfo2 := mockAssetInfo("45673456237896")
	assetInfo3 := mockAssetInfo("45673456237896")
	assetInfo3.CostCenter = "Huygens Gap"
	assetInfo4 := mockAssetInfo("")
	Convey("Add AssetInfo to datastore", t, func() {
		Convey("Add AssetInfo to datastore", func() {
			req := []*ufs.AssetInfo{assetInfo1, assetInfo2}
			res := AddAssetInfo(ctx, req)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			So(res[0].Entity, ShouldNotBeNil)
			So(res[1].Entity, ShouldNotBeNil)
			So(res[0].Entity.AssetTag, ShouldEqual, assetInfo1.GetAssetTag())
			So(res[1].Entity.AssetTag, ShouldEqual, assetInfo2.GetAssetTag())
		})
		Convey("Add AssetInfo to datastore with duplicates", func() {
			req := []*ufs.AssetInfo{assetInfo1, assetInfo2, assetInfo3}
			res := AddAssetInfo(ctx, req)
			So(res, ShouldHaveLength, 3)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			So(res[2].Err, ShouldBeNil)
			So(res[0].Entity, ShouldNotBeNil)
			So(res[1].Entity, ShouldNotBeNil)
			So(res[2].Entity, ShouldNotBeNil)
			So(res[0].Entity.AssetTag, ShouldEqual, assetInfo1.GetAssetTag())
			So(res[1].Entity.AssetTag, ShouldEqual, assetInfo2.GetAssetTag())
			So(res[2].Entity.AssetTag, ShouldEqual, assetInfo3.GetAssetTag())
			// Return contains same entity on both as only one was insterted
			So(res[2].Entity, ShouldEqual, res[1].Entity)
		})
		Convey("Add AssetInfo without ID to datastore", func() {
			req := []*ufs.AssetInfo{assetInfo1, assetInfo2, assetInfo4, assetInfo4}
			res := AddAssetInfo(ctx, req)
			So(res, ShouldHaveLength, 4)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			So(res[2].Err, ShouldNotBeNil)
			So(res[3].Err, ShouldNotBeNil)
			So(res[0].Entity, ShouldNotBeNil)
			So(res[1].Entity, ShouldNotBeNil)
			So(res[2].Entity, ShouldBeNil)
			So(res[3].Entity, ShouldBeNil)
			So(res[0].Entity.AssetTag, ShouldEqual, assetInfo1.GetAssetTag())
			So(res[1].Entity.AssetTag, ShouldEqual, assetInfo2.GetAssetTag())
		})
	})
}

func TestGetAssetInfo(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	ids1 := []string{"45673456237895", "45673456237896",
		"45673456237897", "45673456237898"}
	req1 := []*ufs.AssetInfo{mockAssetInfo(ids1[0]),
		mockAssetInfo(ids1[1]),
		mockAssetInfo(ids1[2]),
		mockAssetInfo(ids1[3])}
	ids2 := []string{"45673456237899", ""}
	Convey("Get AssetInfo from datastore", t, func() {
		Convey("Get existing and non-exiting AssetInfo from datastore", func() {
			res := AddAssetInfo(ctx, req1)
			So(res, ShouldHaveLength, 4)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			So(res[2].Err, ShouldBeNil)
			So(res[3].Err, ShouldBeNil)
			ids1 = append(ids1, "45673456237899")
			ai := GetAssetInfo(ctx, ids1)
			So(ai, ShouldHaveLength, 5)
			So(ai[4].Err, ShouldNotBeNil)
			So(ai[0].Entity.AssetTag, ShouldEqual, ids1[0])
			So(ai[1].Entity.AssetTag, ShouldEqual, ids1[1])
			So(ai[2].Entity.AssetTag, ShouldEqual, ids1[2])
			So(ai[3].Entity.AssetTag, ShouldEqual, ids1[3])
			So(&ai[0].Entity.Info, ShouldResemble, req1[0])
			So(&ai[1].Entity.Info, ShouldResemble, req1[1])
			So(&ai[2].Entity.Info, ShouldResemble, req1[2])
			So(&ai[3].Entity.Info, ShouldResemble, req1[3])
		})
		Convey("Get AssetInfo from datastore with invalid keys", func() {
			ai := GetAssetInfo(ctx, ids2)
			So(ai, ShouldHaveLength, 2)
			So(ai[0].Err, ShouldNotBeNil)
			So(ai[1].Err, ShouldNotBeNil)
		})
	})
}

func TestUpdateAsset(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	asset1 := mockAsset("45673456237895", "lab1")
	asset2 := mockAsset("45673456237896", "lab2")
	asset3 := mockAsset("45673456237896", "lab3")
	asset4 := mockAsset("", "")
	Convey("Update asset on datastore", t, func() {
		Convey("Update non-existing asset to datastore", func() {
			req := []*fleet.ChopsAsset{asset1}
			res, err := UpdateAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
			So(res[0].Err.Error(), ShouldContainSubstring, "No such asset in the database")
		})
		Convey("Update existing asset to datastore", func() {
			req := []*fleet.ChopsAsset{asset2}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldBeNil)
			// Verify location is updated.
			reqUpdate := []*fleet.ChopsAsset{asset3}
			res2, err := UpdateAssets(ctx, reqUpdate)
			So(err, ShouldBeNil)
			So(res2, ShouldHaveLength, 1)
			So(res2[0].Err, ShouldBeNil)
			So(res2[0].Asset.GetLocation().GetLab(), ShouldEqual, asset3.GetLocation().GetLab())
		})
		Convey("Update asset without ID to datastore", func() {
			req := []*fleet.ChopsAsset{asset4}
			res, err := UpdateAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].Err, ShouldNotBeNil)
		})
	})
}

func TestGetAllAssets(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	asset1 := mockAsset("45673456237895", "lab1")
	asset2 := mockAsset("45673456237896", "lab2")
	asset3 := mockAsset("45673456237897", "lab1")
	asset4 := mockAsset("45673456237898", "lab2")
	Convey("Get all assets from datastore", t, func() {
		req := []*fleet.ChopsAsset{asset1, asset2}
		res, err := AddAssets(ctx, req)
		So(err, ShouldBeNil)
		So(res, ShouldHaveLength, 2)
		So(res[0].Err, ShouldBeNil)
		So(res[1].Err, ShouldBeNil)
		// Verify
		assets, err := GetAllAssets(ctx, false)
		So(err, ShouldBeNil)
		So(assets, ShouldHaveLength, 2)
		want := []string{"45673456237895", "45673456237896"}
		get := []string{assets[0].GetId(), assets[1].GetId()}
		So(get, ShouldResemble, want)
	})
	Convey("Get all asset tags [keys only] from datastore", t, func() {
		req := []*fleet.ChopsAsset{asset4, asset3}
		res, err := AddAssets(ctx, req)
		So(err, ShouldBeNil)
		So(res, ShouldHaveLength, 2)
		So(res[0].Err, ShouldBeNil)
		So(res[1].Err, ShouldBeNil)
		// Verify
		// Query the assets added above
		assets, err := GetAllAssets(ctx, true)
		So(err, ShouldBeNil)
		// Depending on the order of execution, we might get more than
		// 2 assets. Verify that location is empty for both
		So(len(assets), ShouldBeGreaterThanOrEqualTo, 2)
		emptyLocation := &ufs.Location{}
		for _, a := range assets {
			So(a.GetLocation(), ShouldResemble, emptyLocation)
		}
	})
}

func TestGetAssetsByID(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	asset1 := mockAsset("45673456237895", "lab1")
	asset2 := mockAsset("45673456237896", "lab2")
	Convey("Get assets from datastore", t, func() {
		Convey("Batch get asset from ID", func() {
			req := []*fleet.ChopsAsset{asset1, asset2}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)
			req1 := []string{asset1.GetId(), asset2.GetId()}
			res1 := GetAssetsByID(ctx, req1)
			So(res1, ShouldHaveLength, 2)
			So(res1[0].Err, ShouldBeNil)
			So(res1[1].Err, ShouldBeNil)
			ast1, err := res1[0].Entity.ToChopsAsset()
			So(err, ShouldBeNil)
			ast2, err := res1[1].Entity.ToChopsAsset()
			So(err, ShouldBeNil)
			assertLocationEqual(ast1.GetLocation(), asset1.GetLocation())
			assertLocationEqual(ast2.GetLocation(), asset2.GetLocation())
		})
		Convey("Get non-existent asset", func() {
			req1 := []string{"45673456237897", ""}
			res1 := GetAssetsByID(ctx, req1)
			So(res1, ShouldHaveLength, 2)
			So(res1[1].Err, ShouldNotBeNil)
		})
	})
}

func TestDeleteAsset(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	asset1 := mockAsset("45673456237895", "lab1")
	asset2 := mockAsset("45673456237896", "lab2")
	Convey("Delete assets from datastore", t, func() {
		Convey("Batch delete existing assets", func() {
			req := []*fleet.ChopsAsset{asset1, asset2}
			res, err := AddAssets(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].Err, ShouldBeNil)
			So(res[1].Err, ShouldBeNil)

			req1 := []string{asset1.GetId(), asset2.GetId(), "45673456237897"}
			res1 := DeleteAsset(ctx, req1)
			So(res1, ShouldNotBeNil)
			So(res1, ShouldHaveLength, 3)
			a1 := getAssetOpResultByID(res1, asset1.GetId())
			So(a1.Err, ShouldBeNil)
			a2 := getAssetOpResultByID(res1, asset2.GetId())
			So(a2.Err, ShouldBeNil)
			a3 := getAssetOpResultByID(res1, "45673456237897")
			So(a3.Err.Error(), ShouldContainSubstring, "Asset not found")

			// Verify entity & state are both removed
			res2 := GetAssetsByID(ctx, req1)
			So(res2, ShouldHaveLength, 3)
			So(res2[0].Err, ShouldNotBeNil)
			So(res2[1].Err, ShouldNotBeNil)
			So(res2[2].Err, ShouldNotBeNil)
			res2 = GetAssetStatesByID(ctx, req1)
			So(res2, ShouldHaveLength, 3)
			So(res2[0].Err, ShouldNotBeNil)
			So(res2[1].Err, ShouldNotBeNil)
			So(res2[2].Err, ShouldNotBeNil)
		})
	})
}

func getAssetOpResultByID(res []*AssetOpResult, ID string) *AssetOpResult {
	for _, r := range res {
		if r.Entity.ID == ID {
			return r
		}
	}
	return nil
}
