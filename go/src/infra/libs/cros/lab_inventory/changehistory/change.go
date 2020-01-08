// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package changehistory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"

	"infra/libs/cros/lab_inventory/utils"
)

// DatastoreKind is the datastore kind.
const DatastoreKind = "ChangeHistory"

// Change tracks the change of a ChromeOSDevice.
type Change struct {
	_kind       string `gae:"$kind,ChangeHistory"`
	DeviceID    string
	Hostname    string
	Label       string
	OldValue    string
	NewValue    string
	Updated     time.Time
	ByWhomName  string
	ByWhomEmail string
	Comment     string
}

// Changes is a slice of Change.
type Changes []Change

type changeReasonKeyType struct{}

var changeReasonKey changeReasonKeyType

func (c *Changes) log(label string, oldValue interface{}, newValue interface{}) {
	oldValueStr := fmt.Sprintf("%v", oldValue)
	newValueStr := fmt.Sprintf("%v", newValue)
	if oldValueStr == newValueStr {
		return
	}
	change := Change{
		Label:    label,
		OldValue: oldValueStr,
		NewValue: newValueStr,
	}
	*c = append(*c, change)
}

// Use installs the value of change reason to context.
func Use(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, changeReasonKey, reason)
}

// SaveToDatastore saves the changes to datastore.
func (c Changes) SaveToDatastore(ctx context.Context) error {
	reason, _ := ctx.Value(changeReasonKey).(string)
	now := time.Now().UTC()
	user := auth.CurrentUser(ctx)

	for i := range c {
		c[i].ByWhomName = user.Name
		c[i].ByWhomEmail = user.Email
		c[i].Comment = reason
		c[i].Updated = now
	}
	if err := datastore.Put(ctx, c); err != nil {
		return errors.Annotate(err, "failed to save changes to datastore").Err()
	}
	return nil
}

// LogDutStateChanges logs the change of the given DutState.
func LogDutStateChanges(hostname string, old *lab.DutState, newData *lab.DutState) (changes Changes) {
	changes.log("DutState.Servo", old.GetServo(), newData.GetServo())
	changes.log("DutState.Chameleon", old.GetChameleon(), newData.GetChameleon())
	changes.log("DutState.AudioLoopbackDongle", old.GetAudioLoopbackDongle(), newData.GetAudioLoopbackDongle())

	// Set id and hostname for all changes.
	id := old.GetId().GetValue()
	for i := range changes {
		changes[i].DeviceID = id
		changes[i].Hostname = hostname
	}
	return
}

// LogChromeOSDeviceChanges logs the change of the given ChromeOSDevice.
func LogChromeOSDeviceChanges(old *lab.ChromeOSDevice, newData *lab.ChromeOSDevice) (changes Changes) {
	changes = append(changes, logDutChange(old.GetDut(), newData.GetDut())...)
	changes = append(changes, logLabstationChange(old.GetLabstation(), newData.GetLabstation())...)

	// Set id and hostname for all changes.
	id := old.GetId().GetValue()
	hostname := utils.GetHostname(newData)
	for i := range changes {
		changes[i].DeviceID = id
		changes[i].Hostname = hostname
	}
	return
}

func logLabstationChange(old *lab.Labstation, newData *lab.Labstation) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("Labstation", old, newData)
		return
	}
	changes.log("hostname", old.GetHostname(), newData.GetHostname())
	changes = append(changes, logServosChange(old.GetServos(), newData.GetServos())...)

	return
}

func logServosChange(old []*lab.Servo, newData []*lab.Servo) (changes Changes) {
	// Sort oldValue and newValue by serial number in alphabet order and then
	// compare.
	sort.Slice(old, func(i, j int) bool { return old[i].ServoSerial < old[i].ServoSerial })
	sort.Slice(newData, func(i, j int) bool { return newData[i].ServoSerial < newData[j].ServoSerial })
	i, j := 0, 0
	for i < len(old) && j < len(newData) {
		switch {
		case old[i].ServoSerial == newData[j].ServoSerial:
			// Servo attribute change, e.g. servo port, etc.
			changes = append(changes, logServoChange(old[i], newData[j])...)
			i++
			j++
		case old[i].ServoSerial < newData[j].ServoSerial:
			// removed an old servo.
			changes = append(changes, logServoChange(old[i], nil)...)
			i++
		case old[i].ServoSerial > newData[j].ServoSerial:
			// Added a new servo.
			changes = append(changes, logServoChange(nil, newData[j])...)
			j++
		}
	}
	for ; i < len(old); i++ {
		changes = append(changes, logServoChange(old[i], nil)...)
	}
	for ; j < len(newData); j++ {
		changes = append(changes, logServoChange(nil, newData[j])...)
	}
	return
}

func logServoChange(old *lab.Servo, newData *lab.Servo) (changes Changes) {
	servo := old
	if servo == nil {
		servo = newData
	}
	changes.log(fmt.Sprintf("servo.%v", servo.ServoSerial), old, newData)
	return
}

func logDutChange(old *lab.DeviceUnderTest, newData *lab.DeviceUnderTest) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("DeviceUnderTest", old, newData)
		return
	}
	changes.log("hostname", old.GetHostname(), newData.GetHostname())
	changes = append(changes, logPeripheralsChange(old.GetPeripherals(), newData.GetPeripherals())...)
	return
}

func logPeripheralsChange(old *lab.Peripherals, newData *lab.Peripherals) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("peripherals", old, newData)
		return
	}
	changes = append(changes, logServoChange(old.GetServo(), newData.GetServo())...)
	changes = append(changes, logChameleonChange(old.GetChameleon(), newData.GetChameleon())...)
	changes = append(changes, logRPMChange(old.GetRpm(), newData.GetRpm())...)
	changes = append(changes, logConnectedCameraChange(old.GetConnectedCamera(), newData.GetConnectedCamera())...)
	changes = append(changes, logAudioChange(old.GetAudio(), newData.GetAudio())...)
	changes = append(changes, logWifiChange(old.GetWifi(), newData.GetWifi())...)
	changes = append(changes, logTouchChange(old.GetTouch(), newData.GetTouch())...)

	changes.log("carrier", old.GetCarrier(), newData.GetCarrier())
	changes.log("camerabox", old.GetCamerabox(), newData.GetCamerabox())

	return
}

func logChameleonChange(old *lab.Chameleon, newData *lab.Chameleon) (changes Changes) {
	changes.log("chameleon", old, newData)
	return
}

func logRPMChange(old *lab.RPM, newData *lab.RPM) (changes Changes) {
	changes.log("powerunit_name", old.GetPowerunitName(), newData.GetPowerunitName())
	changes.log("powerunit_outlet", old.GetPowerunitOutlet(), newData.GetPowerunitOutlet())
	return
}

func logConnectedCameraChange(old []*lab.Camera, newData []*lab.Camera) (changes Changes) {
	changes.log("connected_camera", old, newData)
	return
}

func logAudioChange(old *lab.Audio, newData *lab.Audio) (changes Changes) {
	changes.log("audio_box", old.GetAudioBox(), newData.GetAudioBox())
	changes.log("atrus", old.GetAtrus(), newData.GetAtrus())
	return
}

func logWifiChange(old *lab.Wifi, newData *lab.Wifi) (changes Changes) {
	changes.log("wificell", old.GetWificell(), newData.GetWificell())
	changes.log("antenna_conn", old.GetAntennaConn(), newData.GetAntennaConn())
	return
}

func logTouchChange(old *lab.Touch, newData *lab.Touch) (changes Changes) {
	changes.log("mimo", old.GetMimo(), newData.GetMimo())
	return
}
