// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This proto definition describes the deployment info of a machine LSE (host) in UFS.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.0
// source: infra/unifiedfleet/api/v1/models/machine_lse_deployment.proto

package ufspb

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DeploymentEnv int32

const (
	// Only add DeploymentEnv prefix to undefined as there're still discussions about whether to add
	// prefix to all enums to reduce the code readability.
	DeploymentEnv_DEPLOYMENTENV_UNDEFINED DeploymentEnv = 0
	DeploymentEnv_PROD                    DeploymentEnv = 1
	DeploymentEnv_AUTOPUSH                DeploymentEnv = 2
)

// Enum value maps for DeploymentEnv.
var (
	DeploymentEnv_name = map[int32]string{
		0: "DEPLOYMENTENV_UNDEFINED",
		1: "PROD",
		2: "AUTOPUSH",
	}
	DeploymentEnv_value = map[string]int32{
		"DEPLOYMENTENV_UNDEFINED": 0,
		"PROD":                    1,
		"AUTOPUSH":                2,
	}
)

func (x DeploymentEnv) Enum() *DeploymentEnv {
	p := new(DeploymentEnv)
	*p = x
	return p
}

func (x DeploymentEnv) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DeploymentEnv) Descriptor() protoreflect.EnumDescriptor {
	return file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_enumTypes[0].Descriptor()
}

func (DeploymentEnv) Type() protoreflect.EnumType {
	return &file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_enumTypes[0]
}

func (x DeploymentEnv) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DeploymentEnv.Descriptor instead.
func (DeploymentEnv) EnumDescriptor() ([]byte, []int) {
	return file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescGZIP(), []int{0}
}

// MachineLSEDeployment includes all info related to deployment of a machine LSE (host).
//
// This deployment record will be updated in 3 ways:
//
// 1. `shivas add machine`, a deployment record will be added to this machine even if it's
//    not deployed yet. It usually happens when users add DHCP records for this machine to
//    verify if it's physically set up well before adding the same hostname into UFS.
//          hostname: "no-host-yet-<serial_number>"
//          serial_number: from `shivas add machine`
//          deployment_identifier: ""
//          configs_to_push: nil
//
// 2. StartActivation phase in Chrome MDM service. When Chrome MDM gots a request from a mac
//    to activate itself, it will always update back this deployment record no matter whether
//    there's already a record existing or not. It usually happens when a mac automatically
//    connects to Google Guest WiFi network in the DC before anyone touches it yet. In this case,
//    the record here would be:
//          hostname: "no-host-yet-<serial_number>"
//          serial_number: from Chrome MDM
//          deployment_identifier: from Chrome MDM
//          configs_to_push: from Chrome MDM
//
// 3. `shivas add host`, a deployment record will be updated to reflect the real hostname given
//    by users.
//
// Next tag: 7
type MachineLSEDeployment struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the host which contains this deployment record.
	Hostname string `protobuf:"bytes,1,opt,name=hostname,proto3" json:"hostname,omitempty"`
	// Refer to Machine.serial_number in machine.proto
	SerialNumber string `protobuf:"bytes,2,opt,name=serial_number,json=serialNumber,proto3" json:"serial_number,omitempty"`
	// Usually it is empty by default.
	// If it's a mac host, the deployment_identifier here refers to the UUID generated by MegaMDM.
	DeploymentIdentifier string `protobuf:"bytes,3,opt,name=deployment_identifier,json=deploymentIdentifier,proto3" json:"deployment_identifier,omitempty"`
	// It refers to all configs which is gonna to be pushed to this host.
	ConfigsToPush []*Payload `protobuf:"bytes,4,rep,name=configs_to_push,json=configsToPush,proto3" json:"configs_to_push,omitempty"`
	// Record the last update timestamp of this host deployment (In UTC timezone)
	UpdateTime *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=update_time,json=updateTime,proto3" json:"update_time,omitempty"`
	// Specify the deployment environment of the MegaMDM service which enrolls this host.
	DeploymentEnv DeploymentEnv `protobuf:"varint,6,opt,name=deployment_env,json=deploymentEnv,proto3,enum=unifiedfleet.api.v1.models.DeploymentEnv" json:"deployment_env,omitempty"`
}

func (x *MachineLSEDeployment) Reset() {
	*x = MachineLSEDeployment{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MachineLSEDeployment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MachineLSEDeployment) ProtoMessage() {}

func (x *MachineLSEDeployment) ProtoReflect() protoreflect.Message {
	mi := &file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MachineLSEDeployment.ProtoReflect.Descriptor instead.
func (*MachineLSEDeployment) Descriptor() ([]byte, []int) {
	return file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescGZIP(), []int{0}
}

func (x *MachineLSEDeployment) GetHostname() string {
	if x != nil {
		return x.Hostname
	}
	return ""
}

func (x *MachineLSEDeployment) GetSerialNumber() string {
	if x != nil {
		return x.SerialNumber
	}
	return ""
}

func (x *MachineLSEDeployment) GetDeploymentIdentifier() string {
	if x != nil {
		return x.DeploymentIdentifier
	}
	return ""
}

func (x *MachineLSEDeployment) GetConfigsToPush() []*Payload {
	if x != nil {
		return x.ConfigsToPush
	}
	return nil
}

func (x *MachineLSEDeployment) GetUpdateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.UpdateTime
	}
	return nil
}

func (x *MachineLSEDeployment) GetDeploymentEnv() DeploymentEnv {
	if x != nil {
		return x.DeploymentEnv
	}
	return DeploymentEnv_DEPLOYMENTENV_UNDEFINED
}

var File_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto protoreflect.FileDescriptor

var file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDesc = []byte{
	0x0a, 0x3d, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f, 0x75, 0x6e, 0x69, 0x66, 0x69, 0x65, 0x64, 0x66,
	0x6c, 0x65, 0x65, 0x74, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x65,
	0x6c, 0x73, 0x2f, 0x6d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x5f, 0x6c, 0x73, 0x65, 0x5f, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x1a, 0x75, 0x6e, 0x69, 0x66, 0x69, 0x65, 0x64, 0x66, 0x6c, 0x65, 0x65, 0x74, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x1a, 0x1f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f, 0x62,
	0x65, 0x68, 0x61, 0x76, 0x69, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x31, 0x69,
	0x6e, 0x66, 0x72, 0x61, 0x2f, 0x75, 0x6e, 0x69, 0x66, 0x69, 0x65, 0x64, 0x66, 0x6c, 0x65, 0x65,
	0x74, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2f,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xed, 0x02, 0x0a, 0x14, 0x4d, 0x61, 0x63, 0x68, 0x69, 0x6e, 0x65, 0x4c, 0x53, 0x45, 0x44,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x68, 0x6f, 0x73,
	0x74, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x68, 0x6f, 0x73,
	0x74, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x73, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x5f,
	0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x73, 0x65,
	0x72, 0x69, 0x61, 0x6c, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x33, 0x0a, 0x15, 0x64, 0x65,
	0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66,
	0x69, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12,
	0x4b, 0x0a, 0x0f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x5f, 0x74, 0x6f, 0x5f, 0x70, 0x75,
	0x73, 0x68, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x75, 0x6e, 0x69, 0x66, 0x69,
	0x65, 0x64, 0x66, 0x6c, 0x65, 0x65, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x73, 0x2e, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x52, 0x0d, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x54, 0x6f, 0x50, 0x75, 0x73, 0x68, 0x12, 0x40, 0x0a, 0x0b,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x42, 0x03, 0xe0,
	0x41, 0x03, 0x52, 0x0a, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x50,
	0x0a, 0x0e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x65, 0x6e, 0x76,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x29, 0x2e, 0x75, 0x6e, 0x69, 0x66, 0x69, 0x65, 0x64,
	0x66, 0x6c, 0x65, 0x65, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x73, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x6e,
	0x76, 0x52, 0x0d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x6e, 0x76,
	0x2a, 0x44, 0x0a, 0x0d, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x6e,
	0x76, 0x12, 0x1b, 0x0a, 0x17, 0x44, 0x45, 0x50, 0x4c, 0x4f, 0x59, 0x4d, 0x45, 0x4e, 0x54, 0x45,
	0x4e, 0x56, 0x5f, 0x55, 0x4e, 0x44, 0x45, 0x46, 0x49, 0x4e, 0x45, 0x44, 0x10, 0x00, 0x12, 0x08,
	0x0a, 0x04, 0x50, 0x52, 0x4f, 0x44, 0x10, 0x01, 0x12, 0x0c, 0x0a, 0x08, 0x41, 0x55, 0x54, 0x4f,
	0x50, 0x55, 0x53, 0x48, 0x10, 0x02, 0x42, 0x28, 0x5a, 0x26, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f,
	0x75, 0x6e, 0x69, 0x66, 0x69, 0x65, 0x64, 0x66, 0x6c, 0x65, 0x65, 0x74, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x3b, 0x75, 0x66, 0x73, 0x70, 0x62,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescOnce sync.Once
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescData = file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDesc
)

func file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescGZIP() []byte {
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescOnce.Do(func() {
		file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescData = protoimpl.X.CompressGZIP(file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescData)
	})
	return file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDescData
}

var file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_goTypes = []interface{}{
	(DeploymentEnv)(0),            // 0: unifiedfleet.api.v1.models.DeploymentEnv
	(*MachineLSEDeployment)(nil),  // 1: unifiedfleet.api.v1.models.MachineLSEDeployment
	(*Payload)(nil),               // 2: unifiedfleet.api.v1.models.Payload
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_depIdxs = []int32{
	2, // 0: unifiedfleet.api.v1.models.MachineLSEDeployment.configs_to_push:type_name -> unifiedfleet.api.v1.models.Payload
	3, // 1: unifiedfleet.api.v1.models.MachineLSEDeployment.update_time:type_name -> google.protobuf.Timestamp
	0, // 2: unifiedfleet.api.v1.models.MachineLSEDeployment.deployment_env:type_name -> unifiedfleet.api.v1.models.DeploymentEnv
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_init() }
func file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_init() {
	if File_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto != nil {
		return
	}
	file_infra_unifiedfleet_api_v1_models_deployment_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MachineLSEDeployment); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_goTypes,
		DependencyIndexes: file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_depIdxs,
		EnumInfos:         file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_enumTypes,
		MessageInfos:      file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_msgTypes,
	}.Build()
	File_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto = out.File
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_rawDesc = nil
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_goTypes = nil
	file_infra_unifiedfleet_api_v1_models_machine_lse_deployment_proto_depIdxs = nil
}
