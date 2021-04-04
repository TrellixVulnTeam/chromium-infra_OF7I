// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/protobuf/encoding/protojson"

	ufspb "infra/unifiedfleet/api/v1/models"
	chromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

const (
	// SummaryResultsLegendPass is the symbol displayed in summary results table for pass.
	SummaryResultsLegendPass = "✔"
	// SummaryResultsLegendFail is the symbol displayed in summary results table for fail.
	SummaryResultsLegendFail = "✗"
	// SummaryResultsLegendSkip is the symbol displayed in summary results table for skip.
	SummaryResultsLegendSkip = "-"
	// SummaryResultsMinCommentSize is the minimum length of the comment string to be displayed.
	SummaryResultsMinCommentSize = 15
	// SummaryResultsTableMinWidth is the min width of a cell including padding in the summary results table.
	SummaryResultsTableMinWidth = 2
	// SummaryResultsTableTabWidth is the tab width to be used in the summary results table.
	SummaryResultsTableTabWidth = 2
	// SummaryResultsTablePadding is the padding added to the cell before computing its width.
	SummaryResultsTablePadding = 4
	// SummaryResultsTablePadChar is the char used for padding a cell in the table.
	SummaryResultsTablePadChar = ' '
)

// Titles for printing table format list
var (
	SwitchTitle               = []string{"Switch Name", "CapacityPort", "Zone", "Rack", "State", "UpdateTime"}
	KvmTitle                  = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "Zone", "Rack", "State", "UpdateTime"}
	KvmFullTitle              = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "IP", "Vlan", "State", "Zone", "Rack", "UpdateTime"}
	RpmTitle                  = []string{"RPM Name", "MAC Address", "CapacityPort", "Zone", "Rack", "State", "UpdateTime"}
	DracTitle                 = []string{"Drac Name", "Display name", "MAC Address", "Switch", "Switch Port", "Password", "Zone", "Rack", "Machine", "UpdateTime"}
	DracFullTitle             = []string{"Drac Name", "MAC Address", "Switch", "Switch Port", "Attached Host", "IP", "Vlan", "Zone", "Rack", "Machine", "UpdateTime"}
	NicTitle                  = []string{"Nic Name", "MAC Address", "Switch", "Switch Port", "Zone", "Rack", "Machine", "UpdateTime"}
	BrowserMachineTitle       = []string{"Machine Name", "Serial Number", "Zone", "Rack", "KVM", "KVM Port", "ChromePlatform", "DeploymentTicket", "Description", "State", "Realm", "UpdateTime"}
	OSMachineTitle            = []string{"Machine Name", "Zone", "Rack", "Barcode", "Hardware ID", "Model", "DeviceType", "MacAddress", "SKU", "Phase", "Build Target", "State", "Realm", "UpdateTime"}
	MachinelseprototypeTitle  = []string{"Machine Prototype Name", "Occupied Capacity", "PeripheralTypes", "VirtualTypes", "Tags", "UpdateTime"}
	RacklseprototypeTitle     = []string{"Rack Prototype Name", "PeripheralTypes", "Tags", "UpdateTime"}
	ChromePlatformTitle       = []string{"Platform Name", "Manufacturer", "Description", "UpdateTime"}
	VlanTitle                 = []string{"Vlan Name", "CIDR Block", "IP Capacity", "DHCP range", "Description", "State", "Zones", "Reserved IPs", "UpdateTime"}
	VMTitle                   = []string{"VM Name", "OS Version", "MAC Address", "Zone", "Host", "Vlan", "IP", "State", "DeploymentTicket", "Description", "UpdateTime"}
	RackTitle                 = []string{"Rack Name", "Zone", "Capacity", "State", "Realm", "UpdateTime"}
	MachineLSETitle           = []string{"Host", "OS Version", "Zone", "Virtual Datacenter", "Rack", "Machine(s)", "Nic", "Vlan", "IP", "State", "VM capacity", "DeploymentTicket", "Description", "UpdateTime"}
	MachineLSEFullTitle       = []string{"Host", "OS Version", "Manufacturer", "Machine", "Zone", "Virtual Datacenter", "Rack", "Nic", "IP", "Vlan", "MAC Address", "State", "VM capacity", "Description", "UpdateTime"}
	MachineLSEDeploymentTitle = []string{"Serial Number", "Hostname", "Deployment Identifier", "UpdateTime"}
	VMFreeSlotTitle           = []string{"Host", "OS Version", "Zone", "Virtual Datacenter", "Rack", "Machine(s)", "Nic", "Vlan", "IP", "State", "Free slots", "DeploymentTicket", "Description", "UpdateTime"}
	VMFreeSlotFullTitle       = []string{"Host", "OS Version", "Manufacturer", "Machine", "Zone", "Virtual Datacenter", "Rack", "Nic", "IP", "Vlan", "MAC Address", "State", "Free slots", "Description", "UpdateTime"}
	ZoneTitle                 = []string{"Name", "EnumName", "Department"}
	StateTitle                = []string{"Name", "EnumName", "Description"}
	AssetTitle                = []string{"Asset Name", "Zone", "Rack", "Barcode", "Serial Number", "Hardware ID", "Model", "AssetType", "MacAddress", "SKU", "Phase", "Build Target", "Realm", "UpdateTime"}
	CachingServiceTitle       = []string{"CachingService Name", "Port", "Subnet", "Primary", "Secondary", "State", "Description", "UpdateTime"}
)

// TimeFormat for all timestamps handled by shivas
var timeFormat = "2006-01-02_15:04:05_MST"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

type listAll func(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error)
type printJSONFunc func(res []proto.Message, emit bool)
type printFullFunc func(ctx context.Context, ic ufsAPI.FleetClient, res []proto.Message, tsv bool) error
type printNormalFunc func(res []proto.Message, tsv, keysOnly bool) error
type printAll func(context.Context, ufsAPI.FleetClient, bool, int32, string, string, bool, bool, bool) (string, error)
type getSingleFunc func(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error)
type deleteSingleFunc func(ctx context.Context, ic ufsAPI.FleetClient, name string) error

// Enum to represent states of SummaryResults
const (
	SummaryResultsSkipped = iota // To represent that the operation was skipped.
	SummaryResultsPass           // To represent that the operation was a success.
	SummaryResultsFail           // To represent that the operation failed.
)

// SummaryResultsTable is used to print the summary of operations.
type SummaryResultsTable struct {
	columns  []string          // List of column names.
	results  map[string][]int  // Map of results for each operation on entity.
	comments map[string]string // Optional comment to be displayed on the row
}

// PrintEntities a batch of entities based on user parameters
func PrintEntities(ctx context.Context, ic ufsAPI.FleetClient, res []proto.Message, printJSON printJSONFunc, printFull printFullFunc, printNormal printNormalFunc, json, emit, full, tsv, keysOnly bool) error {
	if json {
		printJSON(res, emit)
		return nil
	}
	if full {
		return printFull(ctx, ic, res, tsv)
	}
	printNormal(res, tsv, keysOnly)
	return nil
}

// BatchList returns the all listed entities by filters
func BatchList(ctx context.Context, ic ufsAPI.FleetClient, listFunc listAll, filters []string, pageSize int, keysOnly, full bool) ([]proto.Message, error) {
	errs := make(map[string]error)
	res := make([]proto.Message, 0)
	if len(filters) == 0 {
		// No filters, single DoList call
		protos, err := DoList(ctx, ic, listFunc, int32(pageSize), "", keysOnly, full)
		if err != nil {
			errs["emptyFilter"] = err
		}
		res = append(res, protos...)
		if pageSize > 0 && len(res) >= pageSize {
			res = res[0:pageSize]
		}
	} else if pageSize > 0 {
		// Filters with a pagesize limit
		// If user specifies a limit, calling DoList without concrrency avoids non-required list calls to UFS
		for _, filter := range filters {
			protos, err := DoList(ctx, ic, listFunc, int32(pageSize), filter, keysOnly, full)
			if err != nil {
				errs[filter] = err
			} else {
				res = append(res, protos...)
				if len(res) >= pageSize {
					res = res[0:pageSize]
					break
				}
			}
		}
	} else {
		// Filters without pagesize limit
		// If user doesnt specify any limit, call DoList for each filter concurrently to improve latency
		res, errs = concurrentList(ctx, ic, listFunc, filters, pageSize, keysOnly, full)
	}

	if len(errs) > 0 {
		fmt.Println("Fail to do some queries:")
		resErr := make([]error, 0, len(errs))
		for f, err := range errs {
			fmt.Printf("Filter %s: %s\n", f, err.Error())
			resErr = append(resErr, err)
		}
		return nil, errors.MultiError(resErr)
	}
	return res, nil
}

// concurrentList calls Dolist concurrently for each filter
func concurrentList(ctx context.Context, ic ufsAPI.FleetClient, listFunc listAll, filters []string, pageSize int, keysOnly, full bool) ([]proto.Message, map[string]error) {
	// buffered channel to append data to a slice in a thread safe env
	queue := make(chan []proto.Message, 1)
	// waitgroup for multiple goroutines
	var wg sync.WaitGroup
	// number of goroutines/threads in the wait group to run concurrently
	wg.Add(len(filters))
	// sync map to store the errors
	var merr sync.Map
	errs := make(map[string]error)
	res := make([]proto.Message, 0)
	for i := 0; i < len(filters); i++ {
		// goroutine for each filter
		go func(i int) {
			protos, err := DoList(ctx, ic, listFunc, int32(pageSize), filters[i], keysOnly, full)
			if err != nil {
				// store the err in sync map
				merr.Store(filters[i], err)
				// inform waitgroup that thread is completed
				wg.Done()
			} else {
				// send the protos to the buffered channel
				queue <- protos
			}
		}(i)
	}

	// goroutine to append data to slice
	go func() {
		// receive protos on queue channel
		for pm := range queue {
			// append proto messages to slice
			res = append(res, pm...)
			// inform waitgroup that one more goroutine/thread is completed.
			wg.Done()
		}
	}()

	// defer closing the channel
	defer close(queue)
	// wait for all goroutines in the waitgroup to complete
	wg.Wait()

	// iterate over sync map to copy data to a normal map for filter->errors
	merr.Range(func(key, value interface{}) bool {
		errs[fmt.Sprint(key)] = value.(error)
		return true
	})
	return res, errs
}

// DoList lists the outputs
func DoList(ctx context.Context, ic ufsAPI.FleetClient, listFunc listAll, pageSize int32, filter string, keysOnly, full bool) ([]proto.Message, error) {
	var pageToken string
	res := make([]proto.Message, 0)
	if pageSize == 0 {
		for {
			protos, token, err := listFunc(ctx, ic, ufsUtil.MaxPageSize, pageToken, filter, keysOnly, full)
			if err != nil {
				return nil, err
			}
			res = append(res, protos...)
			if token == "" {
				break
			}
			pageToken = token
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			protos, token, err := listFunc(ctx, ic, size, pageToken, filter, keysOnly, full)
			if err != nil {
				return nil, err
			}
			res = append(res, protos...)
			if token == "" {
				break
			}
			pageToken = token
		}
	}
	return res, nil
}

// ConcurrentGet runs multiple goroutines making Get calls to UFS
func ConcurrentGet(ctx context.Context, ic ufsAPI.FleetClient, names []string, getSingle getSingleFunc) []proto.Message {
	var res []proto.Message
	// buffered channel to append data to a slice in a thread safe env
	queue := make(chan proto.Message, 1)
	// waitgroup for multiple goroutines
	var wg sync.WaitGroup
	// number of goroutines/threads in the wait group to run concurrently
	wg.Add(len(names))
	for i := 0; i < len(names); i++ {
		// goroutine for each id/name
		go func(i int) {
			// single Get request call to UFS
			m, err := getSingle(ctx, ic, names[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error()+" => "+names[i])
				// inform waitgroup that thread is completed
				wg.Done()
			} else {
				// send the proto to the buffered channel
				queue <- m
			}
		}(i)
	}

	// goroutine to append data to slice
	go func() {
		// receive proto on queue channel
		for pm := range queue {
			// append proto message to slice
			res = append(res, pm)
			// inform waitgroup that one more goroutine/thread is completed.
			wg.Done()
		}
	}()

	// defer closing the channel
	defer close(queue)
	// wait for all goroutines in the waitgroup to complete
	wg.Wait()
	return res
}

// ConcurrentDelete runs multiple goroutines making Delete calls to UFS
func ConcurrentDelete(ctx context.Context, ic ufsAPI.FleetClient, names []string, deleteSingle deleteSingleFunc) ([]string, []string) {
	var success, failure []string
	// buffered channel to append data to a slice in a thread safe env
	successQueue := make(chan string, 1)
	// buffered channel to append data to a slice in a thread safe env
	failureQueue := make(chan string, 1)
	// waitgroup for multiple goroutines
	var wg sync.WaitGroup
	// number of goroutines/threads in the wait group to run concurrently
	wg.Add(len(names))
	for i := 0; i < len(names); i++ {
		// goroutine for each id/name
		go func(i int) {
			// single Delete request call to UFS
			err := deleteSingle(ctx, ic, names[i])
			if err != nil {
				// log error message
				fmt.Fprintln(os.Stderr, err.Error()+" => "+names[i])
				// send failure deletion message to the buffered channel
				failureQueue <- fmt.Sprintf("%s", names[i])
			} else {
				// send successful deletion message to the buffered channel
				successQueue <- fmt.Sprintf("%s", names[i])
			}
		}(i)
	}

	// goroutine to append data to successful queue slice
	go func() {
		// receive proto on queue channel
		for pm := range successQueue {
			// append proto message to slice
			success = append(success, pm)
			// inform waitgroup that one more goroutine/thread is completed.
			wg.Done()
		}
	}()

	// goroutine to append data to failure queue slice
	go func() {
		// receive proto on queue channel
		for pm := range failureQueue {
			// append proto message to slice
			failure = append(failure, pm)
			// inform waitgroup that one more goroutine/thread is completed.
			wg.Done()
		}
	}()

	// defer closing the channel
	defer close(successQueue)
	defer close(failureQueue)
	// wait for all goroutines in the waitgroup to complete
	wg.Wait()
	return success, failure
}

// PrintListJSONFormat prints the list output in JSON format
func PrintListJSONFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, keysOnly, emit bool) error {
	var pageToken string
	fmt.Print("[")
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly, false, emit)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			fmt.Print(",")
			pageToken = token
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly, false, emit)
			if err != nil {
				return err
			}
			if token == "" {
				break
			} else if i+ufsUtil.MaxPageSize < pageSize {
				fmt.Print(",")
			}
			pageToken = token
		}
	}
	fmt.Println("]")
	return nil
}

// PrintTableTitle prints the table title with parameters
func PrintTableTitle(title []string, tsv, keysOnly bool) {
	if !tsv && !keysOnly {
		PrintTitle(title)
	}
}

// PrintListTableFormat prints list output in Table format
func PrintListTableFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, keysOnly bool, title []string, tsv bool) error {
	if !tsv {
		if keysOnly {
			PrintTitle(title[0:1])
		} else {
			PrintTitle(title)
		}
	}
	var pageToken string
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly, tsv, false)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			pageToken = token
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly, tsv, false)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			pageToken = token
		}
	}
	return nil
}

// PrintJSON prints the interface output as json
func PrintJSON(t interface{}) error {
	switch reflect.TypeOf(t).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(t)
		fmt.Print("[")
		for i := 0; i < s.Len(); i++ {
			e, err := json.MarshalIndent(s.Index(i).Interface(), "", "\t")
			if err != nil {
				return err
			}
			fmt.Println(string(e))
			if i != s.Len()-1 {
				fmt.Println(",")
			}
		}
		fmt.Println("]")
	}
	return nil
}

// PrintProtoJSON prints the output as json
func PrintProtoJSON(pm proto.Message, emit bool) {
	defer bw.Flush()
	m := protojson.MarshalOptions{
		EmitUnpopulated: emit,
		Indent:          "\t",
	}
	json, err := m.Marshal(proto.MessageV2(pm))
	if err != nil {
		fmt.Println("Failed to marshal JSON")
	} else {
		bw.Write(json)
		fmt.Fprintf(bw, "\n")
	}
}

// PrintTitle prints the title fields in table form.
func PrintTitle(title []string) {
	for _, s := range title {
		fmt.Fprint(tw, fmt.Sprintf("%s\t", s))
	}
	fmt.Fprintln(tw)
}

// PrintSwitches prints the all switches in table form.
func PrintSwitches(res []proto.Message, keysOnly bool) {
	switches := make([]*ufspb.Switch, len(res))
	for i, r := range res {
		switches[i] = r.(*ufspb.Switch)
	}
	defer tw.Flush()
	for _, s := range switches {
		printSwitch(s, keysOnly)
	}
}

func switchOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Switch)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetResourceState().String(),
		ts,
	}
}

func printSwitch(sw *ufspb.Switch, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(sw.Name))
		return
	}
	var out string
	for _, s := range switchOutputStrs(sw) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(res []proto.Message, emit bool) {
	switches := make([]*ufspb.Switch, len(res))
	for i, r := range res {
		switches[i] = r.(*ufspb.Switch)
	}
	fmt.Print("[")
	for i, s := range switches {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(switches)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func kvmFullOutputStrs(kvm *ufspb.KVM, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(kvm.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(kvm.Name),
		kvm.GetMacAddress(),
		kvm.GetChromePlatform(),
		fmt.Sprintf("%d", kvm.GetCapacityPort()),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		kvm.GetResourceState().String(),
		kvm.GetZone(),
		kvm.GetRack(),
		ts,
	}
}

// PrintKVMFull prints the full info for kvm
func PrintKVMFull(kvms []*ufspb.KVM, dhcps map[string]*ufspb.DHCPConfig) {
	defer tw.Flush()
	for i := range kvms {
		var out string
		for _, s := range kvmFullOutputStrs(kvms[i], dhcps[kvms[i].GetName()]) {
			out += fmt.Sprintf("%s\t", s)
		}
		fmt.Fprintln(tw, out)
	}
}

// PrintKVMs prints the all kvms in table form.
func PrintKVMs(res []proto.Message, keysOnly bool) {
	kvms := make([]*ufspb.KVM, len(res))
	for i, r := range res {
		kvms[i] = r.(*ufspb.KVM)
	}
	defer tw.Flush()
	for _, kvm := range kvms {
		printKVM(kvm, keysOnly)
	}
}

func kvmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.KVM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		m.GetChromePlatform(),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetResourceState().String(),
		ts,
	}
}

func printKVM(kvm *ufspb.KVM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(kvm.Name))
		return
	}
	var out string
	for _, s := range kvmOutputStrs(kvm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintKVMsJSON prints the kvm details in json format.
func PrintKVMsJSON(res []proto.Message, emit bool) {
	kvms := make([]*ufspb.KVM, len(res))
	for i, r := range res {
		kvms[i] = r.(*ufspb.KVM)
	}
	fmt.Print("[")
	for i, s := range kvms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(kvms)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintRPMs prints the all rpms in table form.
func PrintRPMs(res []proto.Message, keysOnly bool) {
	rpms := make([]*ufspb.RPM, len(res))
	for i, r := range res {
		rpms[i] = r.(*ufspb.RPM)
	}
	defer tw.Flush()
	for _, rpm := range rpms {
		printRPM(rpm, keysOnly)
	}
}

func rpmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.RPM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetResourceState().String(),
		ts,
	}
}

func printRPM(rpm *ufspb.RPM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(rpm.Name))
		return
	}
	var out string
	for _, s := range rpmOutputStrs(rpm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRPMsJSON prints the rpm details in json format.
func PrintRPMsJSON(res []proto.Message, emit bool) {
	rpms := make([]*ufspb.RPM, len(res))
	for i, r := range res {
		rpms[i] = r.(*ufspb.RPM)
	}
	fmt.Print("[")
	for i, s := range rpms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(rpms)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func dracFullOutputStrs(m *ufspb.Drac, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.Name),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		dhcp.GetHostname(),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

// PrintDracFull prints the full related msg for drac
func PrintDracFull(entities []*ufspb.Drac, dhcps map[string]*ufspb.DHCPConfig) {
	defer tw.Flush()
	for i := range entities {
		var out string
		for _, s := range dracFullOutputStrs(entities[i], dhcps[entities[i].GetName()]) {
			out += fmt.Sprintf("%s\t", s)
		}
		fmt.Fprintln(tw, out)
	}
}

// PrintDracs prints the all dracs in table form.
func PrintDracs(res []proto.Message, keysOnly bool) {
	dracs := make([]*ufspb.Drac, len(res))
	for i, r := range res {
		dracs[i] = r.(*ufspb.Drac)
	}
	defer tw.Flush()
	for _, drac := range dracs {
		printDrac(drac, keysOnly)
	}
}

func dracOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Drac)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.Name),
		m.GetDisplayName(),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		m.GetPassword(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

func printDrac(drac *ufspb.Drac, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(drac.Name))
		return
	}
	var out string
	for _, s := range dracOutputStrs(drac) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintDracsJSON prints the drac details in json format.
func PrintDracsJSON(res []proto.Message, emit bool) {
	dracs := make([]*ufspb.Drac, len(res))
	for i, r := range res {
		dracs[i] = r.(*ufspb.Drac)
	}
	fmt.Print("[")
	for i, s := range dracs {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(dracs)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintNics prints the all nics in table form.
func PrintNics(res []proto.Message, keysOnly bool) {
	nics := make([]*ufspb.Nic, len(res))
	for i, r := range res {
		nics[i] = r.(*ufspb.Nic)
	}
	defer tw.Flush()
	for _, nic := range nics {
		printNic(nic, keysOnly)
	}
}

func nicOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Nic)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

func printNic(nic *ufspb.Nic, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(nic.Name))
		return
	}
	var out string
	for _, s := range nicOutputStrs(nic) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintNicsJSON prints the nic details in json format.
func PrintNicsJSON(res []proto.Message, emit bool) {
	nics := make([]*ufspb.Nic, len(res))
	for i, r := range res {
		nics[i] = r.(*ufspb.Nic)
	}
	fmt.Print("[")
	for i, s := range nics {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(nics)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func assetOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Asset)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetLocation().GetZone().String(),
		m.GetLocation().GetRack(),
		m.GetLocation().GetBarcodeName(),
		m.GetInfo().GetSerialNumber(),
		m.GetInfo().GetHwid(),
		m.GetModel(),
		m.GetType().String(),
		m.GetInfo().GetEthernetMacAddress(),
		m.GetInfo().GetSku(),
		m.GetInfo().GetPhase(),
		m.GetInfo().GetBuildTarget(),
		m.GetRealm(),
		ts,
	}
}

// PrintAssets prints the all assets in table form.
func PrintAssets(res []proto.Message, keysOnly bool) {
	assets := make([]*ufspb.Asset, len(res))
	for i, r := range res {
		assets[i] = r.(*ufspb.Asset)
	}
	defer tw.Flush()
	for _, m := range assets {
		printAsset(m, keysOnly)
	}
}

func printAsset(m *ufspb.Asset, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range assetOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintAssetsJSON prints the asset details in json format.
func PrintAssetsJSON(res []proto.Message, emit bool) {
	assets := make([]*ufspb.Asset, len(res))
	for i, r := range res {
		assets[i] = r.(*ufspb.Asset)
	}
	fmt.Print("[")
	for i, m := range assets {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(assets)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func machineOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Machine)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	if m.GetChromeBrowserMachine() != nil {
		return []string{
			ufsUtil.RemovePrefix(m.GetName()),
			m.GetSerialNumber(),
			m.GetLocation().GetZone().String(),
			m.GetLocation().GetRack(),
			m.GetChromeBrowserMachine().GetKvmInterface().GetKvm(),
			m.GetChromeBrowserMachine().GetKvmInterface().GetPortName(),
			m.GetChromeBrowserMachine().GetChromePlatform(),
			m.GetChromeBrowserMachine().GetDeploymentTicket(),
			m.GetChromeBrowserMachine().GetDescription(),
			m.GetResourceState().String(),
			m.GetRealm(),
			ts,
		}
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetLocation().GetZone().String(),
		m.GetLocation().GetRack(),
		m.GetLocation().GetBarcodeName(),
		m.GetChromeosMachine().GetHwid(),
		m.GetChromeosMachine().GetModel(),
		m.GetChromeosMachine().GetDeviceType().String(),
		m.GetChromeosMachine().GetMacAddress(),
		m.GetChromeosMachine().GetSku(),
		m.GetChromeosMachine().GetPhase(),
		m.GetChromeosMachine().GetBuildTarget(),
		m.GetResourceState().String(),
		m.GetRealm(),
		ts,
	}
}

// PrintMachines prints the all machines in table form.
func PrintMachines(res []proto.Message, keysOnly bool) {
	machines := make([]*ufspb.Machine, len(res))
	for i, r := range res {
		machines[i] = r.(*ufspb.Machine)
	}
	defer tw.Flush()
	for _, m := range machines {
		printMachine(m, keysOnly)
	}
}

func printMachine(m *ufspb.Machine, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintMachinesJSON prints the machine details in json format.
func PrintMachinesJSON(res []proto.Message, emit bool) {
	machines := make([]*ufspb.Machine, len(res))
	for i, r := range res {
		machines[i] = r.(*ufspb.Machine)
	}
	fmt.Print("[")
	for i, m := range machines {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(machines)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintMachineLSEPrototypes prints the all msleps in table form.
func PrintMachineLSEPrototypes(res []proto.Message, keysOnly bool) {
	entities := make([]*ufspb.MachineLSEPrototype, len(res))
	for i, r := range res {
		entities[i] = r.(*ufspb.MachineLSEPrototype)
	}
	defer tw.Flush()
	for _, m := range entities {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		printMachineLSEPrototype(m, keysOnly)
	}
}

func machineLSEPrototypeOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.MachineLSEPrototype)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	res := []string{
		ufsUtil.RemovePrefix(m.GetName()),
		fmt.Sprintf("%d", m.GetOccupiedCapacityRu()),
	}
	prs := m.GetPeripheralRequirements()
	var peripheralTypes string
	for _, pr := range prs {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	res = append(res, strings.TrimSuffix(peripheralTypes, ","))
	var virtualTypes string
	for _, vm := range m.GetVirtualRequirements() {
		virtualTypes += fmt.Sprintf("%s,", vm.GetVirtualType())
	}
	res = append(res, strings.TrimSuffix(virtualTypes, ","))
	res = append(res, fmt.Sprintf("%s", m.GetTags()))
	res = append(res, ts)
	return res
}

func printMachineLSEPrototype(m *ufspb.MachineLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineLSEPrototypeOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintMachineLSEPrototypesJSON prints the mslep details in json format.
func PrintMachineLSEPrototypesJSON(res []proto.Message, emit bool) {
	entities := make([]*ufspb.MachineLSEPrototype, len(res))
	for i, r := range res {
		entities[i] = r.(*ufspb.MachineLSEPrototype)
	}
	fmt.Print("[")
	for i, m := range entities {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(entities)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintRackLSEPrototypes prints the all msleps in table form.
func PrintRackLSEPrototypes(res []proto.Message, keysOnly bool) {
	rlseps := make([]*ufspb.RackLSEPrototype, len(res))
	for i, r := range res {
		rlseps[i] = r.(*ufspb.RackLSEPrototype)
	}
	defer tw.Flush()
	for _, m := range rlseps {
		printRackLSEPrototype(m, keysOnly)
	}
}

func rackLSEPrototypeOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.RackLSEPrototype)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	res := []string{ufsUtil.RemovePrefix(m.GetName())}
	var peripheralTypes string
	for _, pr := range m.GetPeripheralRequirements() {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	res = append(res, strings.TrimSuffix(peripheralTypes, ","))
	res = append(res, fmt.Sprintf("%s", m.GetTags()))
	res = append(res, ts)
	return res
}

func printRackLSEPrototype(m *ufspb.RackLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range rackLSEPrototypeOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRackLSEPrototypesJSON prints the mslep details in json format.
func PrintRackLSEPrototypesJSON(res []proto.Message, emit bool) {
	rlseps := make([]*ufspb.RackLSEPrototype, len(res))
	for i, r := range res {
		rlseps[i] = r.(*ufspb.RackLSEPrototype)
	}
	fmt.Print("[")
	for i, m := range rlseps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(rlseps)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintVlansJSON prints the vlan details in json format.
func PrintVlansJSON(res []proto.Message, emit bool) {
	vlans := make([]*ufspb.Vlan, len(res))
	for i, r := range res {
		vlans[i] = r.(*ufspb.Vlan)
	}
	fmt.Print("[")
	for i, m := range vlans {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(vlans)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintVlans prints the all vlans in table form.
func PrintVlans(res []proto.Message, keysOnly bool) {
	vlans := make([]*ufspb.Vlan, len(res))
	for i, r := range res {
		vlans[i] = r.(*ufspb.Vlan)
	}
	defer tw.Flush()
	for _, v := range vlans {
		printVlan(v, keysOnly)
	}
}

func vlanOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Vlan)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	zones := make([]string, len(m.GetZones()))
	for i, z := range m.GetZones() {
		zones[i] = z.String()
	}
	var dhcpRange string
	if m.GetFreeStartIpv4Str() != "" {
		dhcpRange = fmt.Sprintf("%s-%s", m.GetFreeStartIpv4Str(), m.GetFreeEndIpv4Str())
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetVlanAddress(),
		fmt.Sprintf("%d", m.GetCapacityIp()),
		dhcpRange,
		m.GetDescription(),
		m.GetResourceState().String(),
		strSlicesToStr(zones),
		strSlicesToStr(m.GetReservedIps()),
		ts,
	}
}

func printVlan(m *ufspb.Vlan, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range vlanOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintChromePlatforms prints the all msleps in table form.
func PrintChromePlatforms(res []proto.Message, keysOnly bool) {
	platforms := make([]*ufspb.ChromePlatform, len(res))
	for i, r := range res {
		platforms[i] = r.(*ufspb.ChromePlatform)
	}
	defer tw.Flush()
	for _, m := range platforms {
		printChromePlatform(m, keysOnly)
	}
}

func platformOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.ChromePlatform)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetManufacturer(),
		m.GetDescription(),
		ts,
	}
}

func printChromePlatform(m *ufspb.ChromePlatform, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range platformOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintChromePlatformsJSON prints the mslep details in json format.
func PrintChromePlatformsJSON(res []proto.Message, emit bool) {
	platforms := make([]*ufspb.ChromePlatform, len(res))
	for i, r := range res {
		platforms[i] = r.(*ufspb.ChromePlatform)
	}
	fmt.Print("[")
	for i, m := range platforms {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(platforms)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintDutsFull prints the most commonly used dut information in a
// human-readable format along with Machine information.
//
// TODO(xixuan): remove the similar function for skylab tooling package
func PrintDutsFull(duts []*ufspb.MachineLSE, machineMap map[string]*ufspb.Machine) {
	defer tw.Flush()
	for _, dut := range duts {
		if dut.GetChromeosMachineLse() == nil {
			continue
		}
		m, ok := machineMap[dut.GetName()]
		if !ok {
			continue
		}
		fmt.Fprintf(tw, "\nHostname:\t%s\n", dut.GetName())
		fmt.Fprintf(tw, "Inventory Id:\t%s\n", dut.GetMachines()[0])
		fmt.Fprintf(tw, "Serial number:\t%s\n", m.GetSerialNumber())
		fmt.Fprintf(tw, "Model:\t%s\n", m.GetChromeosMachine().GetModel())
		fmt.Fprintf(tw, "Board:\t%s\n", m.GetChromeosMachine().GetBuildTarget())
		fmt.Fprintf(tw, "ReferenceDesign:\t%s\n", m.GetChromeosMachine().GetReferenceBoard())
		fmt.Fprintf(tw, "Variant:\t%s\n", m.GetChromeosMachine().GetSku())
		fmt.Fprintf(tw, "HWID:\t%s\n", m.GetChromeosMachine().GetHwid())
		fmt.Fprintf(tw, "State:\t%s\n", m.GetResourceState().String())

		servo := dut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if servo != nil {
			fmt.Fprintf(tw, "Servo:\n")
			fmt.Fprintf(tw, "\thostname\t%s\n", servo.GetServoHostname())
			fmt.Fprintf(tw, "\tport\t%d\n", servo.GetServoPort())
			fmt.Fprintf(tw, "\tserial number\t%s\n", servo.GetServoSerial())
			fmt.Fprintf(tw, "\ttype\t%s\n", servo.GetServoType())
			fmt.Fprintf(tw, "\tsetup\t%s\n", servo.GetServoSetup())
		} else {
			fmt.Fprintf(tw, "Servo: None\n")
		}

		var rpm *chromeoslab.OSRPM
		if dut.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			rpm = dut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm()
		} else {
			rpm = dut.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm()
		}
		if rpm != nil {
			fmt.Fprintf(tw, "RPM:\n")
			fmt.Fprintf(tw, "\thostname\t%s\n", rpm.GetPowerunitName())
			fmt.Fprintf(tw, "\toutlet\t%s\n", rpm.GetPowerunitOutlet())
		} else {
			fmt.Fprintf(tw, "RPM: None\n")
		}
	}
}

// PrintDutsShort prints only the dut info from MachineLSE
func PrintDutsShort(res []proto.Message, keysOnly bool) {
	defer tw.Flush()
	for _, r := range res {
		dut := r.(*ufspb.MachineLSE)
		dut.Name = ufsUtil.RemovePrefix(dut.Name)
		if keysOnly {
			fmt.Fprintf(tw, "%s\n", dut.GetName())
			continue
		}

		fmt.Fprintf(tw, "\nHostname:\t%s\n", dut.GetName())
		fmt.Fprintf(tw, "Inventory Id:\t%s\n", dut.GetMachines()[0])

		servo := dut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
		if servo != nil {
			fmt.Fprintf(tw, "Servo:\n")
			fmt.Fprintf(tw, "\thostname\t%s\n", servo.GetServoHostname())
			fmt.Fprintf(tw, "\tport\t%d\n", servo.GetServoPort())
			fmt.Fprintf(tw, "\tserial number\t%s\n", servo.GetServoSerial())
			fmt.Fprintf(tw, "\ttype\t%s\n", servo.GetServoType())
			fmt.Fprintf(tw, "\tsetup\t%s\n", servo.GetServoSetup())
		} else {
			fmt.Fprintf(tw, "Servo: None\n")
		}

		var rpm *chromeoslab.OSRPM
		if dut.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			rpm = dut.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetRpm()
		} else {
			rpm = dut.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm()
		}
		if rpm != nil {
			fmt.Fprintf(tw, "RPM:\n")
			fmt.Fprintf(tw, "\thostname\t%s\n", rpm.GetPowerunitName())
			fmt.Fprintf(tw, "\toutlet\t%s\n", rpm.GetPowerunitOutlet())
		} else {
			fmt.Fprintf(tw, "RPM: None\n")
		}
	}
}

// PrintMachineLSEsJSON prints the machinelse details in json format.
func PrintMachineLSEsJSON(res []proto.Message, emit bool) {
	machinelses := make([]*ufspb.MachineLSE, len(res))
	for i, r := range res {
		machinelses[i] = r.(*ufspb.MachineLSE)
	}
	fmt.Print("[")
	for i, m := range machinelses {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(machinelses)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

func machineLSEFullOutputStrs(lse *ufspb.MachineLSE, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(lse.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(lse.GetName()),
		lse.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
		lse.GetManufacturer(),
		strSlicesToStr(lse.GetMachines()),
		lse.GetZone(),
		lse.GetChromeBrowserMachineLse().GetVirtualDatacenter(),
		lse.GetRack(),
		lse.GetNic(),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		dhcp.GetMacAddress(),
		lse.GetResourceState().String(),
		fmt.Sprintf("%d", lse.GetChromeBrowserMachineLse().GetVmCapacity()),
		lse.GetDescription(),
		ts,
	}
}

// PrintMachineLSEFull prints the full info for a host
func PrintMachineLSEFull(entities []*ufspb.MachineLSE, dhcps map[string]*ufspb.DHCPConfig) {
	defer tw.Flush()
	for i := range entities {
		var out string
		for _, s := range machineLSEFullOutputStrs(entities[i], dhcps[entities[i].GetName()]) {
			out += fmt.Sprintf("%s\t", s)
		}
		fmt.Fprintln(tw, out)
	}
}

// PrintMachineLSEs prints the all machinelses in table form.
func PrintMachineLSEs(res []proto.Message, keysOnly bool) {
	entities := make([]*ufspb.MachineLSE, len(res))
	for i, r := range res {
		entities[i] = r.(*ufspb.MachineLSE)
	}
	defer tw.Flush()
	for _, m := range entities {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		printMachineLSE(m, keysOnly)
	}
}

func machineLSEOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.MachineLSE)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	machine := ""
	if len(m.GetMachines()) == 1 {
		machine = m.GetMachines()[0]
	}
	if len(m.GetMachines()) > 1 {
		machine = strSlicesToStr(m.GetMachines())
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
		m.GetZone(),
		m.GetChromeBrowserMachineLse().GetVirtualDatacenter(),
		m.GetRack(),
		machine,
		m.GetNic(),
		m.GetVlan(),
		m.GetIp(),
		m.GetResourceState().String(),
		fmt.Sprintf("%d", m.GetChromeBrowserMachineLse().GetVmCapacity()),
		m.GetDeploymentTicket(),
		m.GetDescription(),
		ts,
	}
}

func printMachineLSE(m *ufspb.MachineLSE, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineLSEOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintVMs prints the all vms in table form.
func PrintVMs(res []proto.Message, keysOnly bool) {
	vms := make([]*ufspb.VM, len(res))
	for i, r := range res {
		vms[i] = r.(*ufspb.VM)
	}
	defer tw.Flush()
	for _, vm := range vms {
		printVM(vm, keysOnly)
	}
}

func vmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.VM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetOsVersion().GetValue(),
		m.GetMacAddress(),
		m.GetZone(),
		m.GetMachineLseId(),
		m.GetVlan(),
		m.GetIp(),
		m.GetResourceState().String(),
		m.GetDeploymentTicket(),
		m.GetDescription(),
		ts,
	}
}

func printVM(vm *ufspb.VM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(vm.Name))
		return
	}
	var out string
	for _, s := range vmOutputStrs(vm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintVMsJSON prints the vm details in json format.
func PrintVMsJSON(res []proto.Message, emit bool) {
	vms := make([]*ufspb.VM, len(res))
	for i, r := range res {
		vms[i] = r.(*ufspb.VM)
	}
	fmt.Print("[")
	for i, m := range vms {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(vms)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintRacks prints the all racks in table form.
func PrintRacks(res []proto.Message, keysOnly bool) {
	racks := make([]*ufspb.Rack, len(res))
	for i, r := range res {
		racks[i] = r.(*ufspb.Rack)
	}
	defer tw.Flush()
	for _, m := range racks {
		printRack(m, keysOnly)
	}
}

func rackOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Rack)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetLocation().GetZone().String(),
		fmt.Sprintf("%d", m.GetCapacityRu()),
		m.GetResourceState().String(),
		m.GetRealm(),
		ts,
	}
}

func printRack(m *ufspb.Rack, keysOnly bool) {
	m.Name = ufsUtil.RemovePrefix(m.Name)
	if keysOnly {
		fmt.Fprintln(tw, m.GetName())
		return
	}
	var out string
	for _, s := range rackOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRacksJSON prints the rack details in json format.
func PrintRacksJSON(res []proto.Message, emit bool) {
	racks := make([]*ufspb.Rack, len(res))
	for i, r := range res {
		racks[i] = r.(*ufspb.Rack)
	}
	fmt.Print("[")
	for i, m := range racks {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len(racks)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintCachingServices prints the all CachingServices in table form.
func PrintCachingServices(res []proto.Message, keysOnly bool) {
	cs := make([]*ufspb.CachingService, len(res))
	for i, r := range res {
		cs[i] = r.(*ufspb.CachingService)
	}
	defer tw.Flush()
	for _, c := range cs {
		printCachingService(c, keysOnly)
	}
}

func cachingServiceOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.CachingService)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.Name),
		fmt.Sprintf("%d", m.GetPort()),
		m.GetServingSubnet(),
		m.GetPrimaryNode(),
		m.GetSecondaryNode(),
		m.GetState().String(),
		m.GetDescription(),
		ts,
	}
}

func printCachingService(cs *ufspb.CachingService, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(cs.Name))
		return
	}
	var out string
	for _, s := range cachingServiceOutputStrs(cs) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintCachingServicesJSON prints the CachingService details in json format.
func PrintCachingServicesJSON(res []proto.Message, emit bool) {
	cs := make([]*ufspb.CachingService, len(res))
	for i, r := range res {
		cs[i] = r.(*ufspb.CachingService)
	}
	fmt.Print("[")
	for i, s := range cs {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len(cs)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintMachineLSEDeploymentsJSON prints the machine lse deployment records details in json format.
func PrintMachineLSEDeploymentsJSON(res []proto.Message, emit bool) {
	drs := make([]*ufspb.MachineLSEDeployment, len(res))
	for i, r := range res {
		drs[i] = r.(*ufspb.MachineLSEDeployment)
	}
	fmt.Print("[")
	for i, m := range drs {
		m.SerialNumber = ufsUtil.RemovePrefix(m.SerialNumber)
		PrintProtoJSON(m, emit)
		if i < len(drs)-1 {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Println("]")
}

// PrintMachineLSEDeployments prints a list of machine lse deployment records
func PrintMachineLSEDeployments(res []proto.Message, keysOnly bool) {
	entities := make([]*ufspb.MachineLSEDeployment, len(res))
	for i, r := range res {
		entities[i] = r.(*ufspb.MachineLSEDeployment)
	}
	defer tw.Flush()
	for _, e := range entities {
		printMachineLSEDeployment(e, keysOnly)
	}
}

func printMachineLSEDeployment(m *ufspb.MachineLSEDeployment, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.SerialNumber))
		return
	}
	var out string
	for _, s := range machineLSEDeploymentFullOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

func machineLSEDeploymentFullOutputStrs(res proto.Message) []string {
	dr := res.(*ufspb.MachineLSEDeployment)
	var ts string
	if t, err := ptypes.Timestamp(dr.GetUpdateTime()); err == nil {
		ts = t.Local().Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(dr.GetSerialNumber()),
		dr.GetHostname(),
		dr.GetDeploymentIdentifier(),
		ts,
	}
}

func strSlicesToStr(slices []string) string {
	return strings.Join(slices, ",")
}

// PrintAllNormal prints a 2D slice with tabwriter
func PrintAllNormal(title []string, res [][]string, keysOnly bool) {
	defer tw.Flush()
	PrintTableTitle(title, false, keysOnly)
	for i := 0; i < len(res); i++ {
		var out string
		for _, s := range res[i] {
			out += fmt.Sprintf("%s\t", s)
		}
		fmt.Fprintln(tw, out)
	}
}

// NewSummaryResultsTable constructs a new SummaryResultsTable.
func NewSummaryResultsTable(cols []string) *SummaryResultsTable {
	if len(cols) == 0 {
		return nil
	}
	res := make(map[string][]int)
	comms := make(map[string]string)
	return &SummaryResultsTable{
		columns:  cols,
		results:  res,
		comments: comms,
	}
}

// RecordResult records the res for operation on entity.
func (sRes *SummaryResultsTable) RecordResult(operation, entity string, res error) {
	if _, ok := sRes.results[entity]; !ok {
		// Create a new entry for entity if one doesn't exist.
		sRes.results[entity] = make([]int, len(sRes.columns))
	}
	for idx, col := range sRes.columns {
		if col == operation {
			if res == nil {
				// If there is no error. Record SummaryResultsPass.
				sRes.results[entity][idx] = SummaryResultsPass
			} else {
				// If there is an error. Record the result and error string as comment.
				sRes.results[entity][idx] = SummaryResultsFail
				sRes.comments[entity] = res.Error()
			}
		}
	}
}

// IsSuccessForAny returns true if the operation succeeded for any of the entities.
func (sRes *SummaryResultsTable) IsSuccessForAny(operation string) bool {
	for idx, col := range sRes.columns {
		// Find the idx of the operation.
		if col == operation {
			for _, res := range sRes.results {
				// Return true if sucess was recorded for any of the entites.
				if res[idx] == SummaryResultsPass {
					return true
				}
			}
			break
		}
	}
	return false
}

// RecordSkip records an operation that was skipped.
func (sRes *SummaryResultsTable) RecordSkip(operation, entity, reason string) {
	if _, ok := sRes.results[entity]; !ok {
		// Create a new entry for entity if one doesn't exist.
		sRes.results[entity] = make([]int, len(sRes.columns))
	}
	for idx, col := range sRes.columns {
		if col == operation {
			sRes.results[entity][idx] = SummaryResultsSkipped
			if reason != "" {
				sRes.comments[entity] = reason
			}
		}
	}
}

// PrintResultsTable prints a summary table of the results of operations. Comments are truncated to
// fit terminal width if fitWidth is true. Doesn't truncate anything, if it results in comment
// becoming smaller than SummaryResultsMinCommentSize.
func (sRes *SummaryResultsTable) PrintResultsTable(out *os.File, fitWidth bool) {
	// Use a buffer to create a table. This allows us to truncate long strings in comment section.
	var buf bytes.Buffer
	// Construct the rows
	printRows := [][]string{}

	// Generate a tabwriter out of the bytes buffer.
	bufTw := tabwriter.NewWriter(&buf, SummaryResultsTableMinWidth, SummaryResultsTableTabWidth, SummaryResultsTablePadding, SummaryResultsTablePadChar, 0)

	// Construct the header
	header := append(sRes.columns, "Comments")
	fmt.Fprintln(bufTw, strings.Join(header, "\t"))

	// Add header to the printRows
	printRows = append(printRows, header)
	for name, row := range sRes.results {
		// Include entity in the first column.
		rowElements := []string{name}
		// Skip the first row as the first column is entity.
		for _, r := range row[1:] {
			switch r {
			case SummaryResultsPass:
				rowElements = append(rowElements, SummaryResultsLegendPass)
			case SummaryResultsFail:
				rowElements = append(rowElements, SummaryResultsLegendFail)
			case SummaryResultsSkipped:
				rowElements = append(rowElements, SummaryResultsLegendSkip)
			}
		}
		rowElements = append(rowElements, sRes.comments[name])
		printRows = append(printRows, rowElements)
		fmt.Fprintln(bufTw, strings.Join(rowElements, "\t"))
	}
	bufTw.Flush()
	output := buf.String()
	// If fitWidth is true, Check and truncate the comment row.
	if width, _, err := terminal.GetSize(int(out.Fd())); fitWidth && err == nil && width > 3 {
		for idx, line := range strings.Split(output, "\n") {
			if line == "" {
				// Skip empty line. Sometimes added by strings.Split
				continue
			}
			// Last string is the comment.
			comment := printRows[idx][len(printRows[idx])-1]
			// Check if the line is larger than width and if the comment can be truncated. Don't truncate line if it requires truncating any other column.
			if len(line) > width && len(comment) > (len(line)-width+SummaryResultsMinCommentSize) {
				line = line[:(width-3)] + "..."
			}
			fmt.Fprintln(out, line)
		}
	} else {
		// Print table as is, if truncating comments is not possible.
		fmt.Fprintln(out, output)
	}

	// Print legend for easy reference.
	fmt.Fprintf(out, "\n\nLegend:\n\tSuccess\t%s\n\tFail\t%s\n\tSkip\t%s\n", SummaryResultsLegendPass, SummaryResultsLegendFail, SummaryResultsLegendSkip)
}
