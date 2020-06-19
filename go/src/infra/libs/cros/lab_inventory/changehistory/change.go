// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package changehistory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/server/auth"

	"infra/libs/cros/lab_inventory/utils"
)

// LifeCycleEvent is the type for all life cycle events, e.g. deployment,
// decommission, etc.
type LifeCycleEvent string

const (
	// DatastoreKind is the datastore kind.
	DatastoreKind = "ChangeHistory"

	lifeCycleEventLabel = "LIFE_CYCLE_EVENT"

	// LifeCycleDeployment indicates the deployment of a device.
	LifeCycleDeployment LifeCycleEvent = "DEPLOYMENT"

	// LifeCycleDecomm indicates the decommission of a device.
	LifeCycleDecomm LifeCycleEvent = "DECOMMISSION"
)

// Change tracks the change of a ChromeOSDevice.
type Change struct {
	_kind       string    `gae:"$kind,ChangeHistory"`
	ID          int64     `gae:"$id"`
	DeviceID    string    `gae:",noindex"`
	Hostname    string    `gae:",noindex"`
	Label       string    `gae:",noindex"`
	OldValue    string    `gae:",noindex"`
	NewValue    string    `gae:",noindex"`
	Updated     time.Time `gae:",noindex"`
	ByWhomName  string    `gae:",noindex"`
	ByWhomEmail string    `gae:",noindex"`
	Comment     string    `gae:",noindex"`
}

// Changes is a slice of Change.
type Changes []Change

type changeReasonKeyType struct{}

var changeReasonKey changeReasonKeyType

// LogDeployment logs the deployment of a ChromeOSDevice.
func (c *Changes) LogDeployment(id, hostname string) {
	deployment := Change{
		DeviceID: id,
		Hostname: hostname,
		Label:    lifeCycleEventLabel,
		// Set the event to both old and new value so queries might be more
		// convenient.
		OldValue: string(LifeCycleDeployment),
		NewValue: string(LifeCycleDeployment),
	}
	*c = append(*c, deployment)
}

// LogDecommission logs the decommission of a ChromeOSDevice.
func (c *Changes) LogDecommission(id, hostname string) {
	decomm := Change{
		DeviceID: id,
		Hostname: hostname,
		Label:    lifeCycleEventLabel,
		// Set the event to both old and new value so queries might be more
		// convenient.
		OldValue: string(LifeCycleDecomm),
		NewValue: string(LifeCycleDecomm),
	}
	*c = append(*c, decomm)
}

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
		if c[i].Updated.IsZero() {
			c[i].Updated = now
		}
	}
	if err := datastore.Put(ctx, c); err != nil {
		return errors.Annotate(err, "failed to save changes to datastore").Err()
	}
	return nil
}

// LoadFromDatastore loads all Changes entities from datastore.
func LoadFromDatastore(ctx context.Context) (Changes, error) {
	q := datastore.NewQuery(DatastoreKind)
	var changes Changes
	if err := datastore.GetAll(ctx, q, &changes); err != nil {
		return nil, err
	}
	return changes, nil
}

// FlushDatastore deletes changes dumped to bigquery from datastore.
func FlushDatastore(ctx context.Context, changes Changes) error {
	return datastore.Delete(ctx, changes)
}

// LogDutStateChanges logs the change of the given DutState.
func LogDutStateChanges(hostname string, old *lab.DutState, newData *lab.DutState) (changes Changes) {
	changes.log("DutState.Servo", old.GetServo(), newData.GetServo())
	changes.log("DutState.Chameleon", old.GetChameleon(), newData.GetChameleon())
	changes.log("DutState.AudioLoopbackDongle", old.GetAudioLoopbackDongle(), newData.GetAudioLoopbackDongle())
	changes.log("DutState.WorkingBluetoothBtpeer", old.GetWorkingBluetoothBtpeer(), newData.GetWorkingBluetoothBtpeer())
	changes.log("DutState.Cr50Phase", old.GetCr50Phase(), newData.GetCr50Phase())
	changes.log("DutState.Cr50KeyEnv", old.GetCr50KeyEnv(), newData.GetCr50KeyEnv())

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
	changes.log("serial_number", old.GetSerialNumber(), newData.GetSerialNumber())
	changes.log("manufacturing.config_id", old.GetManufacturingId().GetValue(), newData.GetManufacturingId().GetValue())
	changes = append(changes, logDeviceConfigID(old.GetDeviceConfigId(), newData.GetDeviceConfigId())...)
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

// LogLabstationChange logs the labstation changes
func logLabstationChange(old *lab.Labstation, newData *lab.Labstation) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("Labstation", old, newData)
		return
	}
	changes.log("hostname", old.GetHostname(), newData.GetHostname())
	changes = append(changes, logServosChange(old.GetServos(), newData.GetServos())...)
	changes = append(changes, logRPMChange(old.GetRpm(), newData.GetRpm())...)
	return
}

// LogChromeOSLabstationChange logs the change of the given labstation
func LogChromeOSLabstationChange(old *lab.ChromeOSDevice, newData *lab.ChromeOSDevice) (changes Changes) {
	oldL := old.GetLabstation()
	newL := newData.GetLabstation()
	if oldL == nil || newL == nil {
		return
	}
	changes.log("hostname", oldL.GetHostname(), newL.GetHostname())
	changes = append(changes, logServosChange(oldL.GetServos(), newL.GetServos())...)
	changes = append(changes, logRPMChange(oldL.GetRpm(), newL.GetRpm())...)
	// Set id and hostname for all changes.
	id := old.GetId().GetValue()
	hostname := utils.GetHostname(newData)
	for i := range changes {
		changes[i].DeviceID = id
		changes[i].Hostname = hostname
	}
	return
}

func logServosChange(old []*lab.Servo, newData []*lab.Servo) (changes Changes) {
	// Sort oldValue and newValue by serial number in alphabet order and then
	// compare.
	sort.Slice(old, func(i, j int) bool { return old[i].ServoSerial < old[j].ServoSerial })
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
	if old == nil && newData == nil {
		changes.log("servos", old, newData)
		return
	}
	servo := old
	if servo == nil {
		servo = newData
	}
	changes.log(fmt.Sprintf("servo.%v", servo.ServoSerial), old, newData)
	return
}

func logDeviceConfigID(old *device.ConfigId, newData *device.ConfigId) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("DeviceConfigID", old, newData)
		return
	}
	changes.log("platform_id", old.GetPlatformId().GetValue(), newData.GetPlatformId().GetValue())
	changes.log("model_id", old.GetModelId().GetValue(), newData.GetModelId().GetValue())
	changes.log("variant_id", old.GetVariantId().GetValue(), newData.GetVariantId().GetValue())
	changes.log("brand_id", old.GetBrandId().GetValue(), newData.GetBrandId().GetValue())
	return
}

func logDutChange(old *lab.DeviceUnderTest, newData *lab.DeviceUnderTest) (changes Changes) {
	if old == nil || newData == nil {
		changes.log("DeviceUnderTest", old, newData)
		return
	}
	changes.log("hostname", old.GetHostname(), newData.GetHostname())
	changes.log("critical_pools", old.GetCriticalPools(), newData.GetCriticalPools())
	changes.log("pools", old.GetPools(), newData.GetPools())
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
	changes.log("chaos", old.GetChaos(), newData.GetChaos())
	changes.log("cable", old.GetCable(), newData.GetCable())

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
	changes.log("router", old.GetRouter(), newData.GetRouter())
	return
}

func logTouchChange(old *lab.Touch, newData *lab.Touch) (changes Changes) {
	changes.log("mimo", old.GetMimo(), newData.GetMimo())
	return
}
