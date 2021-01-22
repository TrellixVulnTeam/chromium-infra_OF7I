// Copyright 2020 The Chromium Authors. All Rights Reserved.
// Use of this source code is governed by the Apache v2.0 license that can be
// found in the LICENSE file.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.12.1
// source: infra/appengine/rubber-stamper/config/config.proto

package config

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Config is the service-wide configuration data for rubber-stamper.
type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A map stores configs for all the Gerrit hosts, where keys are names of
	// hosts (e.g. "chromium" or "chrome-internal"), values are corresponding
	// configs.
	HostConfigs map[string]*HostConfig `protobuf:"bytes,1,rep,name=host_configs,json=hostConfigs,proto3" json:"host_configs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// A global default time window for clean reverts and cherry picks. The
	// format is the same as that of CleanRevertPattern.time_window.
	DefaultTimeWindow string `protobuf:"bytes,2,opt,name=default_time_window,json=defaultTimeWindow,proto3" json:"default_time_window,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetHostConfigs() map[string]*HostConfig {
	if x != nil {
		return x.HostConfigs
	}
	return nil
}

func (x *Config) GetDefaultTimeWindow() string {
	if x != nil {
		return x.DefaultTimeWindow
	}
	return ""
}

// HostConfig describes the config to be used for a Gerrit host.
type HostConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A map stores config for repositories, where keys are names of repos (e.g.
	// "chromium/src", "infra/infra") and values are corresponding configs.
	RepoConfigs map[string]*RepoConfig `protobuf:"bytes,1,rep,name=repo_configs,json=repoConfigs,proto3" json:"repo_configs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// The default valid time window for clean reverts. This time window is
	// applied at a host-level and the time window configured in repo-level
	// configs will override this one. The format is the same as that of
	// CleanRevertPattern.time_window.
	CleanRevertTimeWindow string `protobuf:"bytes,2,opt,name=clean_revert_time_window,json=cleanRevertTimeWindow,proto3" json:"clean_revert_time_window,omitempty"`
	// The default valid time window for clean cherry-picks. This time window is
	// applied at a host-level and the time window configured in repo-level
	// configs will override this one. The format is the same as that of
	// CleanCherryPickPattern.time_window.
	CleanCherryPickTimeWindow string `protobuf:"bytes,3,opt,name=clean_cherry_pick_time_window,json=cleanCherryPickTimeWindow,proto3" json:"clean_cherry_pick_time_window,omitempty"`
}

func (x *HostConfig) Reset() {
	*x = HostConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HostConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HostConfig) ProtoMessage() {}

func (x *HostConfig) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HostConfig.ProtoReflect.Descriptor instead.
func (*HostConfig) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{1}
}

func (x *HostConfig) GetRepoConfigs() map[string]*RepoConfig {
	if x != nil {
		return x.RepoConfigs
	}
	return nil
}

func (x *HostConfig) GetCleanRevertTimeWindow() string {
	if x != nil {
		return x.CleanRevertTimeWindow
	}
	return ""
}

func (x *HostConfig) GetCleanCherryPickTimeWindow() string {
	if x != nil {
		return x.CleanCherryPickTimeWindow
	}
	return ""
}

// RepoConfig describes the config to be used for a Gerrit repository.
type RepoConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BenignFilePattern      *BenignFilePattern      `protobuf:"bytes,1,opt,name=benign_file_pattern,json=benignFilePattern,proto3" json:"benign_file_pattern,omitempty"`
	CleanRevertPattern     *CleanRevertPattern     `protobuf:"bytes,2,opt,name=clean_revert_pattern,json=cleanRevertPattern,proto3" json:"clean_revert_pattern,omitempty"`
	CleanCherryPickPattern *CleanCherryPickPattern `protobuf:"bytes,3,opt,name=clean_cherry_pick_pattern,json=cleanCherryPickPattern,proto3" json:"clean_cherry_pick_pattern,omitempty"`
}

func (x *RepoConfig) Reset() {
	*x = RepoConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RepoConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepoConfig) ProtoMessage() {}

func (x *RepoConfig) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepoConfig.ProtoReflect.Descriptor instead.
func (*RepoConfig) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{2}
}

func (x *RepoConfig) GetBenignFilePattern() *BenignFilePattern {
	if x != nil {
		return x.BenignFilePattern
	}
	return nil
}

func (x *RepoConfig) GetCleanRevertPattern() *CleanRevertPattern {
	if x != nil {
		return x.CleanRevertPattern
	}
	return nil
}

func (x *RepoConfig) GetCleanCherryPickPattern() *CleanCherryPickPattern {
	if x != nil {
		return x.CleanCherryPickPattern
	}
	return nil
}

// BenignFilePattern describes pattern of changes to benign files.
type BenignFilePattern struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Paths contains the information that which files are allowed and which are
	// not. The paths is parsed as lines in a .gitignore document, and therefore
	// should follows rules listed in https://git-scm.com/docs/gitignore.
	Paths []string `protobuf:"bytes,2,rep,name=paths,proto3" json:"paths,omitempty"`
}

func (x *BenignFilePattern) Reset() {
	*x = BenignFilePattern{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BenignFilePattern) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BenignFilePattern) ProtoMessage() {}

func (x *BenignFilePattern) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BenignFilePattern.ProtoReflect.Descriptor instead.
func (*BenignFilePattern) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{3}
}

func (x *BenignFilePattern) GetPaths() []string {
	if x != nil {
		return x.Paths
	}
	return nil
}

// CleanRevertPattern describes pattern of clean reverts.
type CleanRevertPattern struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The length of time in <int><unit> form. Reverts need to be within this
	// time_window to be valid.
	// Valid units are "s", "m", "h", "d", meaning "seconds", "minutes",
	// "hours", "days" respectively.
	TimeWindow string `protobuf:"bytes,1,opt,name=time_window,json=timeWindow,proto3" json:"time_window,omitempty"`
	// Paths that must have a human reviewer.
	ExcludedPaths []string `protobuf:"bytes,2,rep,name=excluded_paths,json=excludedPaths,proto3" json:"excluded_paths,omitempty"`
}

func (x *CleanRevertPattern) Reset() {
	*x = CleanRevertPattern{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CleanRevertPattern) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CleanRevertPattern) ProtoMessage() {}

func (x *CleanRevertPattern) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CleanRevertPattern.ProtoReflect.Descriptor instead.
func (*CleanRevertPattern) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{4}
}

func (x *CleanRevertPattern) GetTimeWindow() string {
	if x != nil {
		return x.TimeWindow
	}
	return ""
}

func (x *CleanRevertPattern) GetExcludedPaths() []string {
	if x != nil {
		return x.ExcludedPaths
	}
	return nil
}

type CleanCherryPickPattern struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The length of time in <int><unit> form. Has the same format as the
	// `time_window` in CleanRevertPattern.
	TimeWindow string `protobuf:"bytes,1,opt,name=time_window,json=timeWindow,proto3" json:"time_window,omitempty"`
	// Paths that must have a human reviewer.
	ExcludedPaths []string `protobuf:"bytes,2,rep,name=excluded_paths,json=excludedPaths,proto3" json:"excluded_paths,omitempty"`
}

func (x *CleanCherryPickPattern) Reset() {
	*x = CleanCherryPickPattern{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CleanCherryPickPattern) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CleanCherryPickPattern) ProtoMessage() {}

func (x *CleanCherryPickPattern) ProtoReflect() protoreflect.Message {
	mi := &file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CleanCherryPickPattern.ProtoReflect.Descriptor instead.
func (*CleanCherryPickPattern) Descriptor() ([]byte, []int) {
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP(), []int{5}
}

func (x *CleanCherryPickPattern) GetTimeWindow() string {
	if x != nil {
		return x.TimeWindow
	}
	return ""
}

func (x *CleanCherryPickPattern) GetExcludedPaths() []string {
	if x != nil {
		return x.ExcludedPaths
	}
	return nil
}

var File_infra_appengine_rubber_stamper_config_config_proto protoreflect.FileDescriptor

var file_infra_appengine_rubber_stamper_config_config_proto_rawDesc = []byte{
	0x0a, 0x32, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x67, 0x69, 0x6e,
	0x65, 0x2f, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x2d, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0xee, 0x01, 0x0a, 0x06,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x51, 0x0a, 0x0c, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x72,
	0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x48, 0x6f, 0x73, 0x74,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x68, 0x6f,
	0x73, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x12, 0x2e, 0x0a, 0x13, 0x64, 0x65, 0x66,
	0x61, 0x75, 0x6c, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x64, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x54,
	0x69, 0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x1a, 0x61, 0x0a, 0x10, 0x48, 0x6f, 0x73,
	0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x37, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x21,
	0x2e, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72, 0x2e,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x48, 0x6f, 0x73, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xc1, 0x02, 0x0a,
	0x0a, 0x48, 0x6f, 0x73, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x55, 0x0a, 0x0c, 0x72,
	0x65, 0x70, 0x6f, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x32, 0x2e, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x65, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x48, 0x6f, 0x73, 0x74, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x0b, 0x72, 0x65, 0x70, 0x6f, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x73, 0x12, 0x37, 0x0a, 0x18, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x5f, 0x72, 0x65, 0x76, 0x65,
	0x72, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x15, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x52, 0x65, 0x76, 0x65, 0x72,
	0x74, 0x54, 0x69, 0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x12, 0x40, 0x0a, 0x1d, 0x63,
	0x6c, 0x65, 0x61, 0x6e, 0x5f, 0x63, 0x68, 0x65, 0x72, 0x72, 0x79, 0x5f, 0x70, 0x69, 0x63, 0x6b,
	0x5f, 0x74, 0x69, 0x6d, 0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x19, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x43, 0x68, 0x65, 0x72, 0x72, 0x79, 0x50,
	0x69, 0x63, 0x6b, 0x54, 0x69, 0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x1a, 0x61, 0x0a,
	0x10, 0x52, 0x65, 0x70, 0x6f, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x37, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x21, 0x2e, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0xad, 0x02, 0x0a, 0x0a, 0x52, 0x65, 0x70, 0x6f, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12,
	0x58, 0x0a, 0x13, 0x62, 0x65, 0x6e, 0x69, 0x67, 0x6e, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x70,
	0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x72,
	0x75, 0x62, 0x62, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x42, 0x65, 0x6e, 0x69, 0x67, 0x6e, 0x46, 0x69, 0x6c, 0x65, 0x50,
	0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x52, 0x11, 0x62, 0x65, 0x6e, 0x69, 0x67, 0x6e, 0x46, 0x69,
	0x6c, 0x65, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x12, 0x5b, 0x0a, 0x14, 0x63, 0x6c, 0x65,
	0x61, 0x6e, 0x5f, 0x72, 0x65, 0x76, 0x65, 0x72, 0x74, 0x5f, 0x70, 0x61, 0x74, 0x74, 0x65, 0x72,
	0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72,
	0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x43, 0x6c, 0x65, 0x61, 0x6e, 0x52, 0x65, 0x76, 0x65, 0x72, 0x74, 0x50, 0x61, 0x74, 0x74, 0x65,
	0x72, 0x6e, 0x52, 0x12, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x52, 0x65, 0x76, 0x65, 0x72, 0x74, 0x50,
	0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x12, 0x68, 0x0a, 0x19, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x5f,
	0x63, 0x68, 0x65, 0x72, 0x72, 0x79, 0x5f, 0x70, 0x69, 0x63, 0x6b, 0x5f, 0x70, 0x61, 0x74, 0x74,
	0x65, 0x72, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x72, 0x75, 0x62, 0x62,
	0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x65, 0x72, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x43, 0x68, 0x65, 0x72, 0x72, 0x79, 0x50, 0x69, 0x63,
	0x6b, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x52, 0x16, 0x63, 0x6c, 0x65, 0x61, 0x6e, 0x43,
	0x68, 0x65, 0x72, 0x72, 0x79, 0x50, 0x69, 0x63, 0x6b, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e,
	0x22, 0x2f, 0x0a, 0x11, 0x42, 0x65, 0x6e, 0x69, 0x67, 0x6e, 0x46, 0x69, 0x6c, 0x65, 0x50, 0x61,
	0x74, 0x74, 0x65, 0x72, 0x6e, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x61, 0x74, 0x68, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x70, 0x61, 0x74, 0x68, 0x73, 0x4a, 0x04, 0x08, 0x01, 0x10,
	0x02, 0x22, 0x5c, 0x0a, 0x12, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x52, 0x65, 0x76, 0x65, 0x72, 0x74,
	0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x74, 0x69, 0x6d, 0x65, 0x5f,
	0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x74, 0x69,
	0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x78, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x64, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x0d, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x50, 0x61, 0x74, 0x68, 0x73, 0x22,
	0x60, 0x0a, 0x16, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x43, 0x68, 0x65, 0x72, 0x72, 0x79, 0x50, 0x69,
	0x63, 0x6b, 0x50, 0x61, 0x74, 0x74, 0x65, 0x72, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x74, 0x69, 0x6d,
	0x65, 0x5f, 0x77, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a,
	0x74, 0x69, 0x6d, 0x65, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x12, 0x25, 0x0a, 0x0e, 0x65, 0x78,
	0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x73, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x0d, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x64, 0x50, 0x61, 0x74, 0x68,
	0x73, 0x42, 0x27, 0x5a, 0x25, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f, 0x61, 0x70, 0x70, 0x65, 0x6e,
	0x67, 0x69, 0x6e, 0x65, 0x2f, 0x72, 0x75, 0x62, 0x62, 0x65, 0x72, 0x2d, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_infra_appengine_rubber_stamper_config_config_proto_rawDescOnce sync.Once
	file_infra_appengine_rubber_stamper_config_config_proto_rawDescData = file_infra_appengine_rubber_stamper_config_config_proto_rawDesc
)

func file_infra_appengine_rubber_stamper_config_config_proto_rawDescGZIP() []byte {
	file_infra_appengine_rubber_stamper_config_config_proto_rawDescOnce.Do(func() {
		file_infra_appengine_rubber_stamper_config_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_infra_appengine_rubber_stamper_config_config_proto_rawDescData)
	})
	return file_infra_appengine_rubber_stamper_config_config_proto_rawDescData
}

var file_infra_appengine_rubber_stamper_config_config_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_infra_appengine_rubber_stamper_config_config_proto_goTypes = []interface{}{
	(*Config)(nil),                 // 0: rubber_stamper.config.Config
	(*HostConfig)(nil),             // 1: rubber_stamper.config.HostConfig
	(*RepoConfig)(nil),             // 2: rubber_stamper.config.RepoConfig
	(*BenignFilePattern)(nil),      // 3: rubber_stamper.config.BenignFilePattern
	(*CleanRevertPattern)(nil),     // 4: rubber_stamper.config.CleanRevertPattern
	(*CleanCherryPickPattern)(nil), // 5: rubber_stamper.config.CleanCherryPickPattern
	nil,                            // 6: rubber_stamper.config.Config.HostConfigsEntry
	nil,                            // 7: rubber_stamper.config.HostConfig.RepoConfigsEntry
}
var file_infra_appengine_rubber_stamper_config_config_proto_depIdxs = []int32{
	6, // 0: rubber_stamper.config.Config.host_configs:type_name -> rubber_stamper.config.Config.HostConfigsEntry
	7, // 1: rubber_stamper.config.HostConfig.repo_configs:type_name -> rubber_stamper.config.HostConfig.RepoConfigsEntry
	3, // 2: rubber_stamper.config.RepoConfig.benign_file_pattern:type_name -> rubber_stamper.config.BenignFilePattern
	4, // 3: rubber_stamper.config.RepoConfig.clean_revert_pattern:type_name -> rubber_stamper.config.CleanRevertPattern
	5, // 4: rubber_stamper.config.RepoConfig.clean_cherry_pick_pattern:type_name -> rubber_stamper.config.CleanCherryPickPattern
	1, // 5: rubber_stamper.config.Config.HostConfigsEntry.value:type_name -> rubber_stamper.config.HostConfig
	2, // 6: rubber_stamper.config.HostConfig.RepoConfigsEntry.value:type_name -> rubber_stamper.config.RepoConfig
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_infra_appengine_rubber_stamper_config_config_proto_init() }
func file_infra_appengine_rubber_stamper_config_config_proto_init() {
	if File_infra_appengine_rubber_stamper_config_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config); i {
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
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HostConfig); i {
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
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RepoConfig); i {
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
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BenignFilePattern); i {
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
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CleanRevertPattern); i {
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
		file_infra_appengine_rubber_stamper_config_config_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CleanCherryPickPattern); i {
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
			RawDescriptor: file_infra_appengine_rubber_stamper_config_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_infra_appengine_rubber_stamper_config_config_proto_goTypes,
		DependencyIndexes: file_infra_appengine_rubber_stamper_config_config_proto_depIdxs,
		MessageInfos:      file_infra_appengine_rubber_stamper_config_config_proto_msgTypes,
	}.Build()
	File_infra_appengine_rubber_stamper_config_config_proto = out.File
	file_infra_appengine_rubber_stamper_config_config_proto_rawDesc = nil
	file_infra_appengine_rubber_stamper_config_config_proto_goTypes = nil
	file_infra_appengine_rubber_stamper_config_config_proto_depIdxs = nil
}
