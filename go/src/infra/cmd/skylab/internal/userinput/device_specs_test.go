// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package userinput

import (
	"infra/libs/skylab/inventory"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/kylelemons/godebug/pretty"
	"go.chromium.org/luci/common/errors"
)

func TestGetDeviceSpecs(t *testing.T) {
	s := &deviceSpecsGetter{
		inputFunc: newRegexpReplacer(regexp.MustCompile(`some\-hostname`), "this-other-hostname"),
	}

	initial := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("some-id"),
			Hostname: stringPtr("some-hostname"),
		},
	}
	got, err := s.Get(&initial, "")
	if err != nil {
		t.Errorf("error in GetDeviceSpecs(): %s", err)
	}

	want := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("some-id"),
			Hostname: stringPtr("this-other-hostname"),
		},
	}
	if !proto.Equal(&want, got) {
		t.Errorf("incorrect response from GetDeviceSpecs, -want, +got:\n%s", pretty.Compare(want, got))
	}
}

func TestGetDeviceSpecsAbortOnError(t *testing.T) {
	p := promptHandler{response: false}
	s := deviceSpecsGetter{
		inputFunc:  newRegexpReplacer(regexp.MustCompile(`hostname`), "not_a_field"),
		promptFunc: p.Handle,
	}
	initial := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("myid"),
			Hostname: stringPtr("myhost"),
		},
	}
	_, err := s.Get(&initial, "")
	if err == nil {
		t.Errorf("Get() succeeded with incorrect input")
	}
	if !p.Called {
		t.Errorf("user not prompted for retry on input error")
	}
}

func TestGetDeviceSpecsSpecsValidation(t *testing.T) {
	p := promptHandler{response: false}
	s := deviceSpecsGetter{
		inputFunc:    func(b []byte) ([]byte, error) { return b, nil },
		promptFunc:   p.Handle,
		validateFunc: func(*inventory.DeviceUnderTest) error { return errors.Reason("invalid").Err() },
	}
	initial := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("myid"),
			Hostname: stringPtr("myhost"),
		},
	}
	_, err := s.Get(&initial, "")
	if err == nil {
		t.Errorf("Get() succeeded despite validation error")
	}
	if !p.Called {
		t.Errorf("user not prompted for retry on validation error")
	}
}

func TestGetDeviceSpecsIterateOnError(t *testing.T) {
	p := promptHandler{response: true}
	s := deviceSpecsGetter{
		inputFunc: chainedInputFuncs([]inputFunc{
			newRegexpReplacer(regexp.MustCompile(`hostname`), "not_a_field"),
			newRegexpReplacer(regexp.MustCompile(`myhost`), "yourhost"),
			// Fix the error introduced earlier so that proto is valid again.
			newRegexpReplacer(regexp.MustCompile(`not_a_field`), "hostname"),
		}),
		promptFunc: p.Handle,
	}
	initial := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("myid"),
			Hostname: stringPtr("myhost"),
		},
	}

	got, err := s.Get(&initial, "")
	if err != nil {
		t.Errorf("error in GetDeviceSpecs(): %s", err)
	}
	if !p.Called {
		t.Errorf("user not prompted for retry on input error")
	}
	want := inventory.DeviceUnderTest{
		Common: &inventory.CommonDeviceSpecs{
			Id:       stringPtr("myid"),
			Hostname: stringPtr("yourhost"),
		},
	}
	if !proto.Equal(&want, got) {
		t.Errorf("incorrect response from GetDeviceSpecs, -want, +got:\n%s", pretty.Compare(want, got))
	}
}

func TestCommentLines(t *testing.T) {
	text := `// first line is already a comment
second line will be commented.
    third line has spaces at the start.
		fourth has tabs.`
	want := `// first line is already a comment
// second line will be commented.
//     third line has spaces at the start.
// 		fourth has tabs.`
	got := commentLines(text)
	if want != got {
		t.Errorf("Incorrect output from commentLines(), -want, +got:\n%s",
			pretty.Compare(strings.Split(want, "\n"), strings.Split(got, "\n")))
	}
}

func TestDropCommentLines(t *testing.T) {
	text := `// first line is a comment, will be dropped.
second line will survive.
  // third line has comment after spaces, will be dropped.
	// fourth line has comment after tabs, will be dropped.
//fifth line has no space before comment, will be dropped.`
	want := "second line will survive."
	got := DropCommentLines(text)
	if want != got {
		t.Errorf("Incorrect output from dropCommentLines(), -want, +got:\n%s",
			pretty.Compare(strings.Split(want, "\n"), strings.Split(got, "\n")))
	}
}

// newRegexpReplacer returns an inputFunc that replaces text matching re with
// repl.
func newRegexpReplacer(re *regexp.Regexp, repl string) inputFunc {
	r := regexpEditor{re: re, repl: []byte(repl)}
	return r.ReplaceAll
}

func chainedInputFuncs(is []inputFunc) inputFunc {
	return func(t []byte) ([]byte, error) {
		i := is[0]
		is = is[1:]
		return i(t)
	}
}

// regexpEditor has a method to replace occurrences of re with repl in given
// text.
type regexpEditor struct {
	re   *regexp.Regexp
	repl []byte
}

// ReplaceAll uses Regexp.ReplaceAll to modify incoming text.
//
// This function can be used as an inputFunc
func (e *regexpEditor) ReplaceAll(initial []byte) ([]byte, error) {
	return e.re.ReplaceAll(initial, e.repl), nil
}

func TestRegexpEditor(t *testing.T) {
	e := regexpEditor{
		re:   regexp.MustCompile(`some\-hostname`),
		repl: []byte("this-other-hostname"),
	}

	r, err := e.ReplaceAll([]byte(`Some multi-line.
text with some-hostname in the middle`))
	if err != nil {
		t.Errorf("error in regexpEditor.InteractiveInput(): %s", err)
	}

	got := string(r)
	want := `Some multi-line.
text with this-other-hostname in the middle`
	if want != got {
		t.Errorf("Incorrect output from regexpEditor.InteractiveInput(), -want, +got:\n%s",
			pretty.Compare(strings.Split(want, "\n"), strings.Split(got, "\n")))
	}
}

// TestMcsvFieldsConsistentWithMcsvPrompt checks whether the prompt string for the mcsv file
// and the field names themselves are consistent.
func TestMcsvFieldsConsistentWithMcsvPrompt(t *testing.T) {
	got := strings.Join(mcsvFields, ",")
	if got != mcsvFieldsPrompt {
		t.Errorf("mcsvFields not consistent with prompt -want +got:\n%s",
			pretty.Compare(strings.Split(mcsvFieldsPrompt, "\n"), strings.Split(got, "\n")))
	}
}

// TestMcsvFieldsNoDuplicates checks whether the mcsvfields contains any duplicates.
func TestMcsvFieldsNoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, string := range mcsvFields {
		if _, ok := seen[string]; ok {
			t.Errorf("mcsvFields contains duplicate key %s", string)
			break
		}
		seen[string] = true
	}
}

func getAttrMap(dut *inventory.DeviceUnderTest) map[string]string {
	attrs := dut.Common.Attributes
	out := make(map[string]string)
	for _, attr := range attrs {
		out[*attr.Key] = *attr.Value
	}
	return out
}

func TestDeviceUnderTestOfMcsvRecordEmptyServoSerial(t *testing.T) {
	input := &mcsvRecord{servoSerial: "", servoHost: "4"}
	dut, err := deviceUnderTestOfMcsvRecord(input)
	if err != nil {
		panic(err)
	}
	foundAttrs := getAttrMap(dut)
	expected := map[string]string{
		"servo_host": "4",
	}
	if diff := cmp.Diff(expected, foundAttrs); diff != "" {
		t.Errorf("attribute map differs: %s", diff)
	}
}

var testValidCSVTable = []struct {
	out  bool
	name string
	in   string
}{
	{true, "missing nothing", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,xxx-powerunit_hostname,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{false, "missing servo_port", "xxx-host,xxx-board,xxx-model,xxx-servo_host,,xxx-servo_serial,xxx-powerunit_hostname,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{false, "missing servo_host", "xxx-host,xxx-board,xxx-model,,xxx-servo_port,xxx-servo_serial,xxx-powerunit_hostname,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{true, "missing servo_host and servo_port", "xxx-host,xxx-board,xxx-model,,,xxx-servo_serial,xxx-powerunit_hostname,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{true, "missing servo_serial", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,,xxx-powerunit_hostname,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{false, "missing powerunit_hostname", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,xxx-powerunit_outlet,DUT_POOL_QUOTA"},
	{false, "missing powerunit_outlet", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,xxx-powerunit_hostname,,DUT_POOL_QUOTA"},
	{true, "missing powerunit_hostname and powerunit_outlet", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,DUT_POOL_QUOTA"},
	{true, "defaulted critical pool", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,"},
	{true, "invalid critical pool is self-serve pool", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,aaaaaaaaaa"},
	{true, "self-serve pool with internal dash", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,aaaaa-aaaa"},
	{false, "leading dash bad in pool name", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,-aaa"},
	{false, "invalid multiple critical pool is self-serve pool", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,DUT_POOL_QUOTA DUT_POOL_CQ"},
	{false, "pool with bad characters is bad", "xxx-host,xxx-board,xxx-model,xxx-servo_host,xxx-servo_port,xxx-servo_serial,,,pool@@@@@"},
}

func TestValidCSV(t *testing.T) {
	for i, item := range testValidCSVTable {
		t.Run(item.name, func(t *testing.T) {
			_, err := parseMCSV(item.in)
			res := (err == nil)
			if res != item.out {
				t.Errorf("misclassified csv row #%d expected: %t got: %t err: %v", i, item.out, res, err)
			}
		})
	}
}

var TestSplitPoolListTable = []struct {
	name string
	ok   bool
	in   []string
	out1 []string
	out2 []string
}{
	{
		"empty pool list",
		true,
		[]string{},
		[]string{"DUT_POOL_QUOTA"},
		[]string{},
	},
	{
		"single self-serve pool",
		true,
		[]string{"a"},
		[]string{},
		[]string{"a"},
	},
	{
		"multiple self-serve pools",
		true,
		[]string{"a", "b"},
		[]string{},
		[]string{"a", "b"},
	},
	{
		"non-default critical pool",
		true,
		[]string{"DUT_POOL_CQ"},
		[]string{"DUT_POOL_CQ"},
		[]string{},
	},
	{
		"empty pool bad",
		false,
		[]string{""},
		nil,
		nil,
	},
	{
		"multiple critical pools bad",
		false,
		[]string{"DUT_POOL_CQ", "DUT_POOL_QUOTA"},
		nil,
		nil,
	},
}

func TestSplitPoolList(t *testing.T) {
	for i, item := range TestSplitPoolListTable {
		t.Run(item.name, func(t *testing.T) {
			criticalPools, selfServePools, err := splitPoolList(item.in...)
			res := (err == nil)
			if res != item.ok {
				t.Errorf("misclassified TestSplitPoolList #%d expected: %t got: %t", i, item.ok, res)
			}
			if !reflect.DeepEqual(criticalPools, item.out1) {
				t.Errorf("bad critical pool TestSplitPoolList #%d expected: %v got: %v", i, item.out1, criticalPools)
			}
			if !reflect.DeepEqual(selfServePools, item.out2) {
				t.Errorf("bad self-serve pools TestSplitPoolList #%d expected: %v got: %v", i, item.out2, selfServePools)
			}
		})
	}
}

// promptHandler responds to user prompts using a canned response.
type promptHandler struct {
	// Canned response for prompts.
	response bool
	// Will be set by Handle()
	Called bool
}

// Handle implements promptFunc.
func (p *promptHandler) Handle(q string) bool {
	p.Called = true
	return p.response
}

func stringPtr(s string) *string {
	return &s
}
