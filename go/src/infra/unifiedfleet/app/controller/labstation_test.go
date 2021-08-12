package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"google.golang.org/grpc/codes"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/external"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func TestUpdateLabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("UpdateLabstation", t, func() {
		Convey("UpdateLabstation - Non-existent labstation", func() {
			labstation1 := mockLabstation("labstation-1", "machine-1")
			// Labstation doesn't exist. Must return error
			res, err := UpdateLabstation(ctx, labstation1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Failed to get existing Labstation")
			So(res, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-1")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 0)
			_, err = GetMachineLSE(ctx, "labstation-1")
			So(err, ShouldNotBeNil)
		})
		Convey("UpdateLabstation - Delete machine, mask update", func() {
			// Reset a machine by setting machines to nil and machines update mask
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-2",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation2 := mockLabstation("labstation-2", "machine-2")
			res, err := CreateLabstation(ctx, labstation2)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation2)
			labstation2 = mockLabstation("labstation-2", "")
			// Attempt to delete machine. Should fail.
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("machines"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "InvalidArgument")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-2")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-2")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			labstation3, err := GetMachineLSE(ctx, "labstation-2")
			So(err, ShouldBeNil)
			So(labstation3.GetMachines(), ShouldResemble, []string{"machine-2"})
		})
		Convey("UpdateLabstation - Delete machine", func() {
			// Reset a machine in maskless update.
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-3", "machine-3")
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation1 = mockLabstation("labstation-3", "")
			// Attempt to delete the machine in maskless update. Should fail.
			res, err = UpdateLabstation(ctx, labstation1, nil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Empty Machine ID")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-3")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-3")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			labstation3, err := GetMachineLSE(ctx, "labstation-3")
			So(err, ShouldBeNil)
			So(labstation3.GetMachines(), ShouldResemble, []string{"machine-3"})
		})
		Convey("UpdateLabstation - Reset rpm using update mask", func() {
			// Delete rpm using update mask and setting rpm name to nil
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-4", "machine-4")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = &chromeosLab.OSRPM{
				PowerunitName:   "rpm-4",
				PowerunitOutlet: ".A4",
			}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			// rpm of labstation2 is nil by default.
			labstation2 := mockLabstation("labstation-4", "machine-4")
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.rpm.host"))
			So(err, ShouldBeNil)
			So(res.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm(), ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].OldValue, ShouldEqual, "rpm-4")
			So(changes[1].NewValue, ShouldEqual, "")
			So(changes[2].OldValue, ShouldEqual, ".A4")
			So(changes[2].NewValue, ShouldEqual, "")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-4")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-4")
			So(err, ShouldBeNil)
			So(labstation3.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm(), ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "hosts/labstation-4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
		})
		Convey("UpdateLabstation - Reset rpm outlet using update mask", func() {
			// Reset rpm outlet using a mask update. Should fail.
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-5", "machine-5")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = &chromeosLab.OSRPM{
				PowerunitName:   "rpm-5",
				PowerunitOutlet: ".A5",
			}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation2 := mockLabstation("labstation-5", "machine-5")
			labstation2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = &chromeosLab.OSRPM{PowerunitOutlet: ".A6"}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.rpm.host", "labstation.rpm.outlet"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot update outlet")
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Reset the rpm outlet. Should fail, can only reset complete rpm.
			labstation2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = &chromeosLab.OSRPM{
				PowerunitOutlet: "",
			}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.rpm.outlet"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot remove rpm outlet")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			// Reset the rpm outlet and update rpm host. Should fail.
			labstation2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Rpm = &chromeosLab.OSRPM{
				PowerunitName:   "rpm-6",
				PowerunitOutlet: "",
			}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.rpm.outlet", "labstation.rpm.name"))
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Cannot remove rpm outlet")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 1)
			labstation3, err := GetMachineLSE(ctx, "labstation-5")
			So(err, ShouldBeNil)
			So(labstation3.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm(), ShouldResemble, &chromeosLab.OSRPM{
				PowerunitName:   "rpm-5",
				PowerunitOutlet: ".A5",
			})
			s, err := state.GetStateRecord(ctx, "hosts/labstation-5")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
		Convey("UpdateLabstation - Update/Delete pools", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-6",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-6", "machine-6")
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			labstation2 := mockLabstation("labstation-6", "machine-6")
			// Add a pool to the labstation.
			labstation2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"labstation_main"}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.pools"))
			So(err, ShouldBeNil)
			So(res.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), ShouldResemble, []string{"labstation_main"})
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].OldValue, ShouldEqual, "[]")
			So(changes[1].NewValue, ShouldEqual, "[labstation_main]")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-6")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-6")
			So(err, ShouldBeNil)
			So(labstation3.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), ShouldResemble, []string{"labstation_main"})
			// Reset pools assigned to labstation.
			labstation2.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = nil
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("labstation.pools"))
			So(err, ShouldBeNil)
			So(res.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), ShouldHaveLength, 0)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-6")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[2].OldValue, ShouldEqual, "[labstation_main]")
			So(changes[2].NewValue, ShouldEqual, "[]")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-6")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			labstation3, err = GetMachineLSE(ctx, "labstation-6")
			So(err, ShouldBeNil)
			So(labstation3.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "hosts/labstation-6")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
		Convey("UpdateLabstation - Update/Delete tags", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-7",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-7", "machine-7")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"labstation_main"}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation2 := mockLabstation("labstation-7", "machine-7")
			// Add a tag to the labstation.
			labstation2.Tags = []string{"decommission"}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("tags"))
			So(err, ShouldBeNil)
			So(res.GetTags(), ShouldResemble, []string{"decommission"})
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-7")
			So(err, ShouldBeNil)
			So(labstation3.GetTags(), ShouldResemble, []string{"decommission"})
			// Append another tag to the labstation.
			labstation2.Tags = []string{"needs_replacement"}
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("tags"))
			So(err, ShouldBeNil)
			So(res.GetTags(), ShouldResemble, []string{"decommission", "needs_replacement"})
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			labstation3, err = GetMachineLSE(ctx, "labstation-7")
			So(err, ShouldBeNil)
			So(labstation3.GetTags(), ShouldResemble, []string{"decommission", "needs_replacement"})
			// Clear all tags from the labstation.
			labstation2.Tags = nil
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("tags"))
			So(err, ShouldBeNil)
			So(res.GetTags(), ShouldBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-7")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 4)
			labstation3, err = GetMachineLSE(ctx, "labstation-7")
			So(err, ShouldBeNil)
			So(labstation3.GetTags(), ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "hosts/labstation-7")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
		Convey("UpdateLabstation - Update/Delete description", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-8",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-8", "machine-8")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"labstation_main"}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation2 := mockLabstation("labstation-8", "machine-8")
			// Add a description  to the labstation.
			labstation2.Description = "[12 Jan 2021] crbug.com/35007"
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("description"))
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-8")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].NewValue, ShouldEqual, "[12 Jan 2021] crbug.com/35007")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-8")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-8")
			So(err, ShouldBeNil)
			So(labstation3.GetDescription(), ShouldEqual, "[12 Jan 2021] crbug.com/35007")
			// Reset labstation description.
			labstation2.Description = ""
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("description"))
			So(err, ShouldBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-8")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[2].OldValue, ShouldEqual, "[12 Jan 2021] crbug.com/35007")
			So(changes[2].NewValue, ShouldEqual, "")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-8")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			labstation3, err = GetMachineLSE(ctx, "labstation-8")
			So(err, ShouldBeNil)
			So(labstation3.GetDescription(), ShouldEqual, "")
			s, err := state.GetStateRecord(ctx, "hosts/labstation-8")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
		Convey("UpdateLabstation - Update/Delete deploymentTicket", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-9",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-9", "machine-9")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"labstation_main"}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation2 := mockLabstation("labstation-9", "machine-9")
			// Add a deployment ticket to the labstation.
			labstation2.DeploymentTicket = "crbug.com/35007"
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("deploymentTicket"))
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-9")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].NewValue, ShouldEqual, "crbug.com/35007")
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-9")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-9")
			So(err, ShouldBeNil)
			So(labstation3.GetDeploymentTicket(), ShouldEqual, "crbug.com/35007")
			// Reset deployment ticket to the labstation.
			labstation2.DeploymentTicket = ""
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("deploymentTicket"))
			So(err, ShouldBeNil)
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-9")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[2].OldValue, ShouldEqual, "crbug.com/35007")
			So(changes[2].NewValue, ShouldEqual, "")
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-9")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 3)
			labstation3, err = GetMachineLSE(ctx, "labstation-9")
			So(err, ShouldBeNil)
			So(labstation3.GetDeploymentTicket(), ShouldEqual, "")
			s, err := state.GetStateRecord(ctx, "hosts/labstation-9")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
		Convey("UpdateLabstation - Update labstation state", func() {
			_, err := registration.CreateMachine(ctx, &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			})
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-10", "machine-10")
			labstation1.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Pools = []string{"labstation_main"}
			res, err := CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res, ShouldResembleProto, labstation1)
			labstation2 := mockLabstation("labstation-10", "machine-10")
			// Set labstation state to serving.
			labstation2.ResourceState = ufspb.State_STATE_SERVING
			res, err = UpdateLabstation(ctx, labstation2, mockFieldMask("resourceState"))
			So(err, ShouldBeNil)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-10")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 1)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-10")
			So(err, ShouldBeNil)
			So(msgs, ShouldHaveLength, 2)
			labstation3, err := GetMachineLSE(ctx, "labstation-10")
			So(err, ShouldBeNil)
			So(labstation3.ResourceState, ShouldEqual, ufspb.State_STATE_SERVING)
			// State record should not be needs_deploy
			s, err := state.GetStateRecord(ctx, "hosts/labstation-5")
			So(err, ShouldBeNil)
			// No update to machines of rpm. Should not be in needs_deploy.
			So(s.GetState(), ShouldNotEqual, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		})
	})
}

func TestRenameLabstation(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	ctx = external.WithTestingContext(ctx)
	Convey("renameLabstation", t, func() {
		Convey("renameLabstation - Missing Delete permission", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
				),
			})
			machine1 := &ufspb.Machine{
				Name:  "machine-1l",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name:  "machine-1d",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			labstation1 := mockLabstation("labstation-1", "machine-1l")
			_, err = CreateLabstation(ctx, labstation1)
			So(err, ShouldBeNil)
			dut1 := mockDUT("dut-1", "machine-1d", "labstation-1", "serial-1", "power-1", ".A1", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
			_, err = CreateDUT(ctx, dut1)
			So(err, ShouldBeNil)
			_, err = RenameMachineLSE(ctx, "labstation-1", "labstation-2")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.PermissionDenied.String())
		})
		Convey("renameLabstation - Missing Update permission", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
				),
			})
			machine1 := &ufspb.Machine{
				Name:  "machine-2l",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name:  "machine-2d",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			labstation2 := mockLabstation("labstation-2", "machine-2l")
			_, err = CreateLabstation(ctx, labstation2)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-2", "machine-2d", "labstation-2", "serial-2", "power-2", ".A2", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
			_, err = CreateDUT(ctx, dut2)
			So(err, ShouldBeNil)
			_, err = RenameMachineLSE(ctx, "labstation-2", "labstation-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.PermissionDenied.String())
		})
		Convey("renameLabstation - Missing Create permission", func() {
			createCtx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
				),
			})
			machine1 := &ufspb.Machine{
				Name:  "machine-3l",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name:  "machine-3d",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(createCtx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(createCtx, machine2)
			So(err, ShouldBeNil)
			labstation2 := mockLabstation("labstation-3", "machine-3l")
			_, err = CreateLabstation(createCtx, labstation2)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-3", "machine-3d", "labstation-3", "serial-3", "power-3", ".A3", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
			_, err = CreateDUT(createCtx, dut2)
			So(err, ShouldBeNil)
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
				),
			})
			_, err = RenameMachineLSE(ctx, "labstation-3", "labstation-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, codes.PermissionDenied.String())
		})
		Convey("renameLabstation - Happy path", func() {
			ctx := auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:user@example.com",
				FakeDB: authtest.NewFakeDB(
					authtest.MockMembership("user:user@example.com", "user"),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesCreate),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesDelete),
					authtest.MockPermission("user:user@example.com", util.AtlLabAdminRealm, util.InventoriesUpdate),
				),
			})
			machine1 := &ufspb.Machine{
				Name:  "machine-4l",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			machine2 := &ufspb.Machine{
				Name:  "machine-4d",
				Realm: util.AtlLabAdminRealm,
				Location: &ufspb.Location{
					Zone: ufspb.Zone_ZONE_CHROMEOS6,
				},
				Device: &ufspb.Machine_ChromeosMachine{
					ChromeosMachine: &ufspb.ChromeOSMachine{
						BuildTarget: "test",
						Model:       "test",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine1)
			So(err, ShouldBeNil)
			_, err = registration.CreateMachine(ctx, machine2)
			So(err, ShouldBeNil)
			labstation2 := mockLabstation("labstation-4", "machine-4l")
			_, err = CreateLabstation(ctx, labstation2)
			So(err, ShouldBeNil)
			dut2 := mockDUT("dut-4", "machine-4d", "labstation-4", "serial-4", "power-4", ".A4", int32(9999), []string{"DUT_POOL_QUOTA"}, "")
			_, err = CreateDUT(ctx, dut2)
			So(err, ShouldBeNil)
			_, err = RenameMachineLSE(ctx, "labstation-4", "labstation-5")
			So(err, ShouldBeNil)
			msgs, err := history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-4")
			So(err, ShouldBeNil)
			// One snapshot at registration and another at rename
			So(msgs, ShouldHaveLength, 2)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			// One snapshot at registration
			So(msgs, ShouldHaveLength, 1)
			msgs, err = history.QuerySnapshotMsgByPropertyName(ctx, "resource_name", "hosts/dut-4")
			So(err, ShouldBeNil)
			// One snapshot at registration and another at servo host change
			So(msgs, ShouldHaveLength, 2)
			// State record for new dut should be same.
			s, err := state.GetStateRecord(ctx, "hosts/dut-4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			// State record for old labstation should not exist.
			_, err = state.GetStateRecord(ctx, "hosts/labstation-4")
			So(err, ShouldNotBeNil)
			// State record for new labstation should be same as old one..
			s, err = state.GetStateRecord(ctx, "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_REGISTERED)
			changes, err := history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 3)
			// Verify all changes recorded by the history.
			So(changes[0].OldValue, ShouldEqual, "REGISTRATION")
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
			So(changes[1].OldValue, ShouldEqual, "RENAME")
			So(changes[1].NewValue, ShouldEqual, "RENAME")
			So(changes[2].OldValue, ShouldEqual, "labstation-4")
			So(changes[2].NewValue, ShouldEqual, "labstation-5")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/labstation-5")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[0].OldValue, ShouldEqual, "RENAME")
			So(changes[0].NewValue, ShouldEqual, "RENAME")
			So(changes[1].OldValue, ShouldEqual, "labstation-4")
			So(changes[1].NewValue, ShouldEqual, "labstation-5")
			changes, err = history.QueryChangesByPropertyName(ctx, "name", "hosts/dut-4")
			So(err, ShouldBeNil)
			So(changes, ShouldHaveLength, 2)
			So(changes[1].EventLabel, ShouldEqual, "machine_lse.chromeos_machine_lse.dut.servo.hostname")
			So(changes[1].OldValue, ShouldEqual, "labstation-4")
			So(changes[1].NewValue, ShouldEqual, "labstation-5")
			So(changes[0].OldValue, ShouldEqual, "REGISTRATION")
			So(changes[0].NewValue, ShouldEqual, "REGISTRATION")
		})
	})
}
