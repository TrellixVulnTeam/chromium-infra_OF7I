// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package migration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/protobuf/proto"
	"github.com/pmezard/go-difflib/difflib"
	"go.chromium.org/gae/service/info"
	authclient "go.chromium.org/luci/auth"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/auth"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/frontend"
	"infra/appengine/cros/lab_inventory/app/migration/internal/gitstore"
	"infra/libs/cros/lab_inventory/datastore"
	"infra/libs/skylab/inventory"
)

func newGitilesClient(c context.Context, host string) (gitiles.GitilesClient, error) {
	t, err := auth.GetRPCTransport(c, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, gitilesapi.OAuthScope))
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport").Err()
	}
	return gitilesapi.NewRESTClient(&http.Client{Transport: t}, host, true)
}

const (
	stagingEnv    = "ENVIRONMENT_STAGING"
	prodEnv       = "ENVIRONMENT_PROD"
	maxErrorToLog = 30
)

func getV1Duts(ctx context.Context) (stringset.Set, map[string]*inventory.DeviceUnderTest, error) {
	env := config.Get(ctx).Environment

	gitilesHost := config.Get(ctx).GetInventory().GetHost()
	client, err := newGitilesClient(ctx, gitilesHost)
	if err != nil {
		return nil, nil, errors.Annotate(err, "fail to create inventory v1 client").Err()
	}
	store := gitstore.NewInventoryStore(nil, client)
	if err := store.Refresh(ctx); err != nil {
		return nil, nil, errors.Annotate(err, "fail to refresh inventory v1 store").Err()
	}
	duts := store.Lab.GetDuts()
	hostnames := make([]string, 0, len(duts))
	dutMap := map[string]*inventory.DeviceUnderTest{}
	for _, dut := range duts {
		dutEnv := dut.GetCommon().GetEnvironment().String()
		// Ignore all DUTs of non-current environment.
		if env == prodEnv && dutEnv == stagingEnv {
			continue
		}
		if env == stagingEnv && dutEnv != stagingEnv {
			continue
		}
		name := dut.GetCommon().GetHostname()
		hostnames = append(hostnames, name)
		dutMap[name] = dut
	}
	return stringset.NewFromSlice(hostnames...), dutMap, nil
}

func getV2Duts(ctx context.Context) (stringset.Set, map[string]*inventory.DeviceUnderTest, error) {
	duts, err := datastore.GetAllDevices(ctx)
	if err != nil {
		return nil, nil, err
	}
	if l := len(duts.Failed()); l > 0 {
		logging.Warningf(ctx, "Failed to get %d devices from v2", l)
		for i, d := range duts.Failed() {
			if i > maxErrorToLog {
				logging.Warningf(ctx, "...")
				break
			}
			logging.Warningf(ctx, "%s: %s", d.Entity.Hostname, d.Err.Error())
		}
	}

	// Filter out all servo v3s.
	v2Duts := make([]datastore.DeviceOpResult, 0, len(duts.Passed()))
	for _, d := range duts.Passed() {
		if !strings.HasSuffix(d.Entity.Hostname, "-servo") {
			v2Duts = append(v2Duts, d)
		}
	}
	extendedData, failedDevices := frontend.GetExtendedDeviceData(ctx, v2Duts)
	if len(failedDevices) > 0 {
		logging.Warningf(ctx, "Failed to get extended data")
		for i, d := range failedDevices {
			if i > maxErrorToLog {
				logging.Warningf(ctx, "...")
				break
			}
			logging.Warningf(ctx, "%s: %s: %s", d.Id, d.Hostname, d.ErrorMsg)
		}
	}

	hostnames := make([]string, len(duts.Passed()))
	dutMap := map[string]*inventory.DeviceUnderTest{}
	for _, d := range extendedData {
		v1Dut, err := api.AdaptToV1DutSpec(d)
		if err != nil {
			logging.Warningf(ctx, "Adapter failure: %s", err.Error())
			continue
		}
		name := v1Dut.GetCommon().GetHostname()
		hostnames = append(hostnames, name)
		dutMap[name] = v1Dut
	}
	return stringset.NewFromSlice(hostnames...), dutMap, nil
}

func logDiffBetweenDutLists(ctx context.Context, lhs, rhs stringset.Set, msg string) {
	if d := lhs.Difference(rhs); d.Len() > 0 {
		logging.Warningf(ctx, msg)
		d.Iter(func(name string) bool {
			logging.Warningf(ctx, "%#v", name)
			return true
		})
	} else {
		logging.Infof(ctx, "No result of %#v", msg)
	}
}

func getCloudStorageWriter(ctx context.Context) *storage.Writer {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		logging.Warningf(ctx, "failed to create cloud storage client")
		return nil
	}
	bucketName := fmt.Sprintf("%s.appspot.com", info.AppID(ctx))
	bucket := storageClient.Bucket(bucketName)
	filename := fmt.Sprintf("inv_diff.%s.log", time.Now().UTC().Format("2006-01-02T03:04:05"))
	logging.Infof(ctx, "All inventory diff will be saved to https://storage.cloud.google.com/%s/%s", bucketName, filename)
	return bucket.Object(filename).NewWriter(ctx)
}

func logDiffOfOneDut(ctx context.Context, cw *storage.Writer, count *int, diffText string) {
	if *count == maxErrorToLog {
		logging.Warningf(ctx, "and more difference ...")
	}
	if *count < maxErrorToLog {
		logging.Warningf(ctx, diffText)
	}
	*count++
	if cw == nil {
		return
	}
	if _, err := fmt.Fprintf(cw, diffText); err != nil {
		logging.Warningf(ctx, "failed to write to cloud storage: %s", err.Error())
	}
}

// CompareInventory compares the inventory from v1 and v2 and log the
// difference.
func CompareInventory(ctx context.Context) error {
	v1Duts, v1DutMap, err := getV1Duts(ctx)
	if err != nil {
		return err
	}
	v2Duts, v2DutMap, err := getV2Duts(ctx)
	if err != nil {
		return err
	}
	logDiffBetweenDutLists(ctx, v1Duts, v2Duts, "Devices only in v1")
	logDiffBetweenDutLists(ctx, v2Duts, v1Duts, "Devices only in v2")

	cw := getCloudStorageWriter(ctx)
	defer func() {
		if cw != nil {
			fmt.Fprintf(cw, "This is the end.")
			if err := cw.Close(); err != nil {
				logging.Warningf(ctx, "failed to close cloud storage writer: %s", err)
			}
		}
	}()

	count := 0
	v1Duts.Intersect(v2Duts).Iter(func(name string) bool {
		d1 := v1DutMap[name]
		d2 := v2DutMap[name]
		filterOutKnownDifference(d1, d2)
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(proto.MarshalTextString(d1)),
			B:        difflib.SplitLines(proto.MarshalTextString(d2)),
			FromFile: "v1",
			ToFile:   "v2",
			Context:  0,
		}
		diffText, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			logging.Errorf(ctx, "failed to compare %#v: %s", name, err.Error())
			return true
		}
		if diffText != "" {
			logDiffOfOneDut(ctx, cw, &count, fmt.Sprintf("%#v is different: \n%s", name, diffText))
		}
		return true
	})
	logging.Infof(ctx, "That's all what I have.")
	return nil
}
func filterAttrs(attrs []*inventory.KeyValue, toRemove ...string) []*inventory.KeyValue {
	toRemoveSet := stringset.NewFromSlice(toRemove...)
	var newAttrs []*inventory.KeyValue
	for _, attr := range attrs {
		if toRemoveSet.Has(attr.GetKey()) {
			continue
		}
		newAttrs = append(newAttrs, attr)
	}
	return newAttrs
}

func filterOutKnownDifference(d1, d2 *inventory.DeviceUnderTest) {
	alignBooleans(d1, d2)
	// Add other know difference here.
	cmn1 := d1.GetCommon()
	cmn2 := d2.GetCommon()
	l1 := cmn1.GetLabels()
	l2 := cmn2.GetLabels()
	c1 := l1.GetCapabilities()
	c2 := l2.GetCapabilities()

	cmn1.Environment = cmn2.Environment
	cmn1.SerialNumber = cmn2.SerialNumber

	c1.Modem = c2.Modem
	c1.Telephony = c2.Telephony

	l1.ReferenceDesign = l2.ReferenceDesign
	l1.Brand = l2.Brand
	l1.Platform = l2.Platform
	l1.Cr50RwKeyid = l2.Cr50RwKeyid
	l1.Cr50RwVersion = l2.Cr50RwVersion
	l1.Cr50RoVersion = l2.Cr50RoVersion
	l1.CtsAbi = l2.CtsAbi
	l1.CtsCpu = l2.CtsCpu

	c1.VideoAcceleration = c2.VideoAcceleration

	cmn1.Attributes = filterAttrs(cmn1.GetAttributes(), "stashed_labels", "job_repo_url", "outlet_changed", "serial_number", "HWID")
	cmn2.Attributes = filterAttrs(cmn2.GetAttributes(), "serial_number", "HWID")
}

func alignBooleans(d1, d2 *inventory.DeviceUnderTest) {
	l1 := d1.GetCommon().Labels
	l2 := d2.GetCommon().Labels
	if l1 == nil || l2 == nil {
		return
	}
	if l1.Capabilities == nil {
		l1.Capabilities = &inventory.HardwareCapabilities{}
	}
	if l2.Capabilities == nil {
		l2.Capabilities = &inventory.HardwareCapabilities{}
	}

	alignBooleansInCapabilities(l1.Capabilities, l2.Capabilities)

	if l1.Peripherals == nil {
		l1.Peripherals = &inventory.Peripherals{}
	}
	if l2.Peripherals == nil {
		l2.Peripherals = &inventory.Peripherals{}
	}
	alignBooleansInPeripherals(l1.Peripherals, l2.Peripherals)

	if l1.TestCoverageHints == nil {
		l1.TestCoverageHints = &inventory.TestCoverageHints{}
	}
	if l2.TestCoverageHints == nil {
		l2.TestCoverageHints = &inventory.TestCoverageHints{}
	}
	alignBooleansInTestCoverageHints(l1.TestCoverageHints, l2.TestCoverageHints)
}

func alignBooleansInPeripherals(p1, p2 *inventory.Peripherals) {
	if p1.GetAudioBoard() == p2.GetAudioBoard() {
		p1.AudioBoard = p2.AudioBoard
	}
	if p1.GetAudioBox() == p2.GetAudioBox() {
		p1.AudioBox = p2.AudioBox
	}
	if p1.GetAudioLoopbackDongle() == p2.GetAudioLoopbackDongle() {
		p1.AudioLoopbackDongle = p2.AudioLoopbackDongle
	}
	if p1.GetChameleon() == p2.GetChameleon() {
		p1.Chameleon = p2.Chameleon
	}
	if p1.GetConductive() == p2.GetConductive() {
		p1.Conductive = p2.Conductive
	}
	if p1.GetHuddly() == p2.GetHuddly() {
		p1.Huddly = p2.Huddly
	}
	if p1.GetMimo() == p2.GetMimo() {
		p1.Mimo = p2.Mimo
	}
	if p1.GetServo() == p2.GetServo() {
		p1.Servo = p2.Servo
	}
	if p1.GetStylus() == p2.GetStylus() {
		p1.Stylus = p2.Stylus
	}
	if p1.GetCamerabox() == p2.GetCamerabox() {
		p1.Camerabox = p2.Camerabox
	}
	if p1.GetWificell() == p2.GetWificell() {
		p1.Wificell = p2.Wificell
	}
	if p1.GetRouter_802_11Ax() == p2.GetRouter_802_11Ax() {
		p1.Router_802_11Ax = p2.Router_802_11Ax
	}
}

func alignBooleansInCapabilities(c1, c2 *inventory.HardwareCapabilities) {
	if c1.GetAtrus() == c2.GetAtrus() {
		c1.Atrus = c2.Atrus
	}
	if c1.GetBluetooth() == c2.GetBluetooth() {
		c1.Bluetooth = c2.Bluetooth
	}
	if c1.GetDetachablebase() == c2.GetDetachablebase() {
		c1.Detachablebase = c2.Detachablebase
	}
	if c1.GetFingerprint() == c2.GetFingerprint() {
		c1.Fingerprint = c2.Fingerprint
	}
	if c1.GetFlashrom() == c2.GetFlashrom() {
		c1.Flashrom = c2.Flashrom
	}
	if c1.GetHotwording() == c2.GetHotwording() {
		c1.Hotwording = c2.Hotwording
	}
	if c1.GetInternalDisplay() == c2.GetInternalDisplay() {
		c1.InternalDisplay = c2.InternalDisplay
	}
	if c1.GetLucidsleep() == c2.GetLucidsleep() {
		c1.Lucidsleep = c2.Lucidsleep
	}
	if c1.GetWebcam() == c2.GetWebcam() {
		c1.Webcam = c2.Webcam
	}
	if c1.GetTouchpad() == c2.GetTouchpad() {
		c1.Touchpad = c2.Touchpad
	}
	if c1.GetTouchscreen() == c2.GetTouchscreen() {
		c1.Touchscreen = c2.Touchscreen
	}
}

func alignBooleansInTestCoverageHints(h1, h2 *inventory.TestCoverageHints) {
	if h1.GetChaosDut() == h2.GetChaosDut() {
		h1.ChaosDut = h2.ChaosDut
	}
	if h1.GetChaosNightly() == h2.GetChaosNightly() {
		h1.ChaosNightly = h2.ChaosNightly
	}
	if h1.GetChromesign() == h2.GetChromesign() {
		h1.Chromesign = h2.Chromesign
	}
	if h1.GetHangoutApp() == h2.GetHangoutApp() {
		h1.HangoutApp = h2.HangoutApp
	}
	if h1.GetMeetApp() == h2.GetMeetApp() {
		h1.MeetApp = h2.MeetApp
	}
	if h1.GetRecoveryTest() == h2.GetRecoveryTest() {
		h1.RecoveryTest = h2.RecoveryTest
	}
	if h1.GetTestAudiojack() == h2.GetTestAudiojack() {
		h1.TestAudiojack = h2.TestAudiojack
	}
	if h1.GetTestHdmiaudio() == h2.GetTestHdmiaudio() {
		h1.TestHdmiaudio = h2.TestHdmiaudio
	}
	if h1.GetTestUsbaudio() == h2.GetTestUsbaudio() {
		h1.TestUsbaudio = h2.TestUsbaudio
	}
	if h1.GetTestUsbprinting() == h2.GetTestUsbprinting() {
		h1.TestUsbprinting = h2.TestUsbprinting
	}
	if h1.GetUsbDetect() == h2.GetUsbDetect() {
		h1.UsbDetect = h2.UsbDetect
	}
	if h1.GetUseLid() == h2.GetUseLid() {
		h1.UseLid = h2.UseLid
	}
}
