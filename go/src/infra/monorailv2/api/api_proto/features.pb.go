// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/api_proto/features.proto

package monorail

import prpc "go.chromium.org/luci/grpc/prpc"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Next available tag: 3
type ListHotlistsByUserRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	User                 *UserRef      `protobuf:"bytes,2,opt,name=user,proto3" json:"user,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *ListHotlistsByUserRequest) Reset()         { *m = ListHotlistsByUserRequest{} }
func (m *ListHotlistsByUserRequest) String() string { return proto.CompactTextString(m) }
func (*ListHotlistsByUserRequest) ProtoMessage()    {}
func (*ListHotlistsByUserRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{0}
}
func (m *ListHotlistsByUserRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHotlistsByUserRequest.Unmarshal(m, b)
}
func (m *ListHotlistsByUserRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHotlistsByUserRequest.Marshal(b, m, deterministic)
}
func (dst *ListHotlistsByUserRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHotlistsByUserRequest.Merge(dst, src)
}
func (m *ListHotlistsByUserRequest) XXX_Size() int {
	return xxx_messageInfo_ListHotlistsByUserRequest.Size(m)
}
func (m *ListHotlistsByUserRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHotlistsByUserRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListHotlistsByUserRequest proto.InternalMessageInfo

func (m *ListHotlistsByUserRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *ListHotlistsByUserRequest) GetUser() *UserRef {
	if m != nil {
		return m.User
	}
	return nil
}

// Next available tag: 2
type ListHotlistsByUserResponse struct {
	Hotlists             []*Hotlist `protobuf:"bytes,1,rep,name=hotlists,proto3" json:"hotlists,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *ListHotlistsByUserResponse) Reset()         { *m = ListHotlistsByUserResponse{} }
func (m *ListHotlistsByUserResponse) String() string { return proto.CompactTextString(m) }
func (*ListHotlistsByUserResponse) ProtoMessage()    {}
func (*ListHotlistsByUserResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{1}
}
func (m *ListHotlistsByUserResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHotlistsByUserResponse.Unmarshal(m, b)
}
func (m *ListHotlistsByUserResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHotlistsByUserResponse.Marshal(b, m, deterministic)
}
func (dst *ListHotlistsByUserResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHotlistsByUserResponse.Merge(dst, src)
}
func (m *ListHotlistsByUserResponse) XXX_Size() int {
	return xxx_messageInfo_ListHotlistsByUserResponse.Size(m)
}
func (m *ListHotlistsByUserResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHotlistsByUserResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListHotlistsByUserResponse proto.InternalMessageInfo

func (m *ListHotlistsByUserResponse) GetHotlists() []*Hotlist {
	if m != nil {
		return m.Hotlists
	}
	return nil
}

// Next available tag: 3
type GetHotlistStarCountRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	HotlistRef           *HotlistRef   `protobuf:"bytes,2,opt,name=hotlist_ref,json=hotlistRef,proto3" json:"hotlist_ref,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *GetHotlistStarCountRequest) Reset()         { *m = GetHotlistStarCountRequest{} }
func (m *GetHotlistStarCountRequest) String() string { return proto.CompactTextString(m) }
func (*GetHotlistStarCountRequest) ProtoMessage()    {}
func (*GetHotlistStarCountRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{2}
}
func (m *GetHotlistStarCountRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetHotlistStarCountRequest.Unmarshal(m, b)
}
func (m *GetHotlistStarCountRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetHotlistStarCountRequest.Marshal(b, m, deterministic)
}
func (dst *GetHotlistStarCountRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetHotlistStarCountRequest.Merge(dst, src)
}
func (m *GetHotlistStarCountRequest) XXX_Size() int {
	return xxx_messageInfo_GetHotlistStarCountRequest.Size(m)
}
func (m *GetHotlistStarCountRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetHotlistStarCountRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetHotlistStarCountRequest proto.InternalMessageInfo

func (m *GetHotlistStarCountRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *GetHotlistStarCountRequest) GetHotlistRef() *HotlistRef {
	if m != nil {
		return m.HotlistRef
	}
	return nil
}

// Next available tag: 2
type GetHotlistStarCountResponse struct {
	StarCount            uint32   `protobuf:"varint,1,opt,name=star_count,json=starCount,proto3" json:"star_count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetHotlistStarCountResponse) Reset()         { *m = GetHotlistStarCountResponse{} }
func (m *GetHotlistStarCountResponse) String() string { return proto.CompactTextString(m) }
func (*GetHotlistStarCountResponse) ProtoMessage()    {}
func (*GetHotlistStarCountResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{3}
}
func (m *GetHotlistStarCountResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetHotlistStarCountResponse.Unmarshal(m, b)
}
func (m *GetHotlistStarCountResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetHotlistStarCountResponse.Marshal(b, m, deterministic)
}
func (dst *GetHotlistStarCountResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetHotlistStarCountResponse.Merge(dst, src)
}
func (m *GetHotlistStarCountResponse) XXX_Size() int {
	return xxx_messageInfo_GetHotlistStarCountResponse.Size(m)
}
func (m *GetHotlistStarCountResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetHotlistStarCountResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetHotlistStarCountResponse proto.InternalMessageInfo

func (m *GetHotlistStarCountResponse) GetStarCount() uint32 {
	if m != nil {
		return m.StarCount
	}
	return 0
}

// Next available tag: 4
type StarHotlistRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	HotlistRef           *HotlistRef   `protobuf:"bytes,2,opt,name=hotlist_ref,json=hotlistRef,proto3" json:"hotlist_ref,omitempty"`
	Starred              bool          `protobuf:"varint,3,opt,name=starred,proto3" json:"starred,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *StarHotlistRequest) Reset()         { *m = StarHotlistRequest{} }
func (m *StarHotlistRequest) String() string { return proto.CompactTextString(m) }
func (*StarHotlistRequest) ProtoMessage()    {}
func (*StarHotlistRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{4}
}
func (m *StarHotlistRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StarHotlistRequest.Unmarshal(m, b)
}
func (m *StarHotlistRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StarHotlistRequest.Marshal(b, m, deterministic)
}
func (dst *StarHotlistRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StarHotlistRequest.Merge(dst, src)
}
func (m *StarHotlistRequest) XXX_Size() int {
	return xxx_messageInfo_StarHotlistRequest.Size(m)
}
func (m *StarHotlistRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_StarHotlistRequest.DiscardUnknown(m)
}

var xxx_messageInfo_StarHotlistRequest proto.InternalMessageInfo

func (m *StarHotlistRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *StarHotlistRequest) GetHotlistRef() *HotlistRef {
	if m != nil {
		return m.HotlistRef
	}
	return nil
}

func (m *StarHotlistRequest) GetStarred() bool {
	if m != nil {
		return m.Starred
	}
	return false
}

// Next available tag: 2
type StarHotlistResponse struct {
	StarCount            uint32   `protobuf:"varint,1,opt,name=star_count,json=starCount,proto3" json:"star_count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StarHotlistResponse) Reset()         { *m = StarHotlistResponse{} }
func (m *StarHotlistResponse) String() string { return proto.CompactTextString(m) }
func (*StarHotlistResponse) ProtoMessage()    {}
func (*StarHotlistResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{5}
}
func (m *StarHotlistResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StarHotlistResponse.Unmarshal(m, b)
}
func (m *StarHotlistResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StarHotlistResponse.Marshal(b, m, deterministic)
}
func (dst *StarHotlistResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StarHotlistResponse.Merge(dst, src)
}
func (m *StarHotlistResponse) XXX_Size() int {
	return xxx_messageInfo_StarHotlistResponse.Size(m)
}
func (m *StarHotlistResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_StarHotlistResponse.DiscardUnknown(m)
}

var xxx_messageInfo_StarHotlistResponse proto.InternalMessageInfo

func (m *StarHotlistResponse) GetStarCount() uint32 {
	if m != nil {
		return m.StarCount
	}
	return 0
}

// Next available tag: 4
type ListHotlistIssuesRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	HotlistRef           *HotlistRef   `protobuf:"bytes,2,opt,name=hotlist_ref,json=hotlistRef,proto3" json:"hotlist_ref,omitempty"`
	Pagination           *Pagination   `protobuf:"bytes,3,opt,name=pagination,proto3" json:"pagination,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *ListHotlistIssuesRequest) Reset()         { *m = ListHotlistIssuesRequest{} }
func (m *ListHotlistIssuesRequest) String() string { return proto.CompactTextString(m) }
func (*ListHotlistIssuesRequest) ProtoMessage()    {}
func (*ListHotlistIssuesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{6}
}
func (m *ListHotlistIssuesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHotlistIssuesRequest.Unmarshal(m, b)
}
func (m *ListHotlistIssuesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHotlistIssuesRequest.Marshal(b, m, deterministic)
}
func (dst *ListHotlistIssuesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHotlistIssuesRequest.Merge(dst, src)
}
func (m *ListHotlistIssuesRequest) XXX_Size() int {
	return xxx_messageInfo_ListHotlistIssuesRequest.Size(m)
}
func (m *ListHotlistIssuesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHotlistIssuesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListHotlistIssuesRequest proto.InternalMessageInfo

func (m *ListHotlistIssuesRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *ListHotlistIssuesRequest) GetHotlistRef() *HotlistRef {
	if m != nil {
		return m.HotlistRef
	}
	return nil
}

func (m *ListHotlistIssuesRequest) GetPagination() *Pagination {
	if m != nil {
		return m.Pagination
	}
	return nil
}

// Next available tag: 2
type ListHotlistIssuesResponse struct {
	Items                []*HotlistItem `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *ListHotlistIssuesResponse) Reset()         { *m = ListHotlistIssuesResponse{} }
func (m *ListHotlistIssuesResponse) String() string { return proto.CompactTextString(m) }
func (*ListHotlistIssuesResponse) ProtoMessage()    {}
func (*ListHotlistIssuesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{7}
}
func (m *ListHotlistIssuesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListHotlistIssuesResponse.Unmarshal(m, b)
}
func (m *ListHotlistIssuesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListHotlistIssuesResponse.Marshal(b, m, deterministic)
}
func (dst *ListHotlistIssuesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListHotlistIssuesResponse.Merge(dst, src)
}
func (m *ListHotlistIssuesResponse) XXX_Size() int {
	return xxx_messageInfo_ListHotlistIssuesResponse.Size(m)
}
func (m *ListHotlistIssuesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListHotlistIssuesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListHotlistIssuesResponse proto.InternalMessageInfo

func (m *ListHotlistIssuesResponse) GetItems() []*HotlistItem {
	if m != nil {
		return m.Items
	}
	return nil
}

// Next available tag: 3
type DismissCueRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	CueId                string        `protobuf:"bytes,2,opt,name=cue_id,json=cueId,proto3" json:"cue_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *DismissCueRequest) Reset()         { *m = DismissCueRequest{} }
func (m *DismissCueRequest) String() string { return proto.CompactTextString(m) }
func (*DismissCueRequest) ProtoMessage()    {}
func (*DismissCueRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{8}
}
func (m *DismissCueRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DismissCueRequest.Unmarshal(m, b)
}
func (m *DismissCueRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DismissCueRequest.Marshal(b, m, deterministic)
}
func (dst *DismissCueRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DismissCueRequest.Merge(dst, src)
}
func (m *DismissCueRequest) XXX_Size() int {
	return xxx_messageInfo_DismissCueRequest.Size(m)
}
func (m *DismissCueRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DismissCueRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DismissCueRequest proto.InternalMessageInfo

func (m *DismissCueRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *DismissCueRequest) GetCueId() string {
	if m != nil {
		return m.CueId
	}
	return ""
}

// Next available tag: 1
type DismissCueResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DismissCueResponse) Reset()         { *m = DismissCueResponse{} }
func (m *DismissCueResponse) String() string { return proto.CompactTextString(m) }
func (*DismissCueResponse) ProtoMessage()    {}
func (*DismissCueResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{9}
}
func (m *DismissCueResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DismissCueResponse.Unmarshal(m, b)
}
func (m *DismissCueResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DismissCueResponse.Marshal(b, m, deterministic)
}
func (dst *DismissCueResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DismissCueResponse.Merge(dst, src)
}
func (m *DismissCueResponse) XXX_Size() int {
	return xxx_messageInfo_DismissCueResponse.Size(m)
}
func (m *DismissCueResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DismissCueResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DismissCueResponse proto.InternalMessageInfo

// Next available tag: 7
type CreateHotlistRequest struct {
	Trace                *RequestTrace `protobuf:"bytes,1,opt,name=trace,proto3" json:"trace,omitempty"`
	Name                 string        `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Summary              string        `protobuf:"bytes,3,opt,name=summary,proto3" json:"summary,omitempty"`
	Description          string        `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	EditorRefs           []*UserRef    `protobuf:"bytes,5,rep,name=editor_refs,json=editorRefs,proto3" json:"editor_refs,omitempty"`
	IssueRefs            []*IssueRef   `protobuf:"bytes,6,rep,name=issue_refs,json=issueRefs,proto3" json:"issue_refs,omitempty"`
	IsPrivate            bool          `protobuf:"varint,7,opt,name=is_private,json=isPrivate,proto3" json:"is_private,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *CreateHotlistRequest) Reset()         { *m = CreateHotlistRequest{} }
func (m *CreateHotlistRequest) String() string { return proto.CompactTextString(m) }
func (*CreateHotlistRequest) ProtoMessage()    {}
func (*CreateHotlistRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{10}
}
func (m *CreateHotlistRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateHotlistRequest.Unmarshal(m, b)
}
func (m *CreateHotlistRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateHotlistRequest.Marshal(b, m, deterministic)
}
func (dst *CreateHotlistRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateHotlistRequest.Merge(dst, src)
}
func (m *CreateHotlistRequest) XXX_Size() int {
	return xxx_messageInfo_CreateHotlistRequest.Size(m)
}
func (m *CreateHotlistRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateHotlistRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateHotlistRequest proto.InternalMessageInfo

func (m *CreateHotlistRequest) GetTrace() *RequestTrace {
	if m != nil {
		return m.Trace
	}
	return nil
}

func (m *CreateHotlistRequest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *CreateHotlistRequest) GetSummary() string {
	if m != nil {
		return m.Summary
	}
	return ""
}

func (m *CreateHotlistRequest) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *CreateHotlistRequest) GetEditorRefs() []*UserRef {
	if m != nil {
		return m.EditorRefs
	}
	return nil
}

func (m *CreateHotlistRequest) GetIssueRefs() []*IssueRef {
	if m != nil {
		return m.IssueRefs
	}
	return nil
}

func (m *CreateHotlistRequest) GetIsPrivate() bool {
	if m != nil {
		return m.IsPrivate
	}
	return false
}

// Next available tag: 1
type CreateHotlistResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateHotlistResponse) Reset()         { *m = CreateHotlistResponse{} }
func (m *CreateHotlistResponse) String() string { return proto.CompactTextString(m) }
func (*CreateHotlistResponse) ProtoMessage()    {}
func (*CreateHotlistResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_features_3933b38ea524f3fb, []int{11}
}
func (m *CreateHotlistResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateHotlistResponse.Unmarshal(m, b)
}
func (m *CreateHotlistResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateHotlistResponse.Marshal(b, m, deterministic)
}
func (dst *CreateHotlistResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateHotlistResponse.Merge(dst, src)
}
func (m *CreateHotlistResponse) XXX_Size() int {
	return xxx_messageInfo_CreateHotlistResponse.Size(m)
}
func (m *CreateHotlistResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateHotlistResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CreateHotlistResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*ListHotlistsByUserRequest)(nil), "monorail.ListHotlistsByUserRequest")
	proto.RegisterType((*ListHotlistsByUserResponse)(nil), "monorail.ListHotlistsByUserResponse")
	proto.RegisterType((*GetHotlistStarCountRequest)(nil), "monorail.GetHotlistStarCountRequest")
	proto.RegisterType((*GetHotlistStarCountResponse)(nil), "monorail.GetHotlistStarCountResponse")
	proto.RegisterType((*StarHotlistRequest)(nil), "monorail.StarHotlistRequest")
	proto.RegisterType((*StarHotlistResponse)(nil), "monorail.StarHotlistResponse")
	proto.RegisterType((*ListHotlistIssuesRequest)(nil), "monorail.ListHotlistIssuesRequest")
	proto.RegisterType((*ListHotlistIssuesResponse)(nil), "monorail.ListHotlistIssuesResponse")
	proto.RegisterType((*DismissCueRequest)(nil), "monorail.DismissCueRequest")
	proto.RegisterType((*DismissCueResponse)(nil), "monorail.DismissCueResponse")
	proto.RegisterType((*CreateHotlistRequest)(nil), "monorail.CreateHotlistRequest")
	proto.RegisterType((*CreateHotlistResponse)(nil), "monorail.CreateHotlistResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// FeaturesClient is the client API for Features service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FeaturesClient interface {
	ListHotlistsByUser(ctx context.Context, in *ListHotlistsByUserRequest, opts ...grpc.CallOption) (*ListHotlistsByUserResponse, error)
	GetHotlistStarCount(ctx context.Context, in *GetHotlistStarCountRequest, opts ...grpc.CallOption) (*GetHotlistStarCountResponse, error)
	StarHotlist(ctx context.Context, in *StarHotlistRequest, opts ...grpc.CallOption) (*StarHotlistResponse, error)
	ListHotlistIssues(ctx context.Context, in *ListHotlistIssuesRequest, opts ...grpc.CallOption) (*ListHotlistIssuesResponse, error)
	DismissCue(ctx context.Context, in *DismissCueRequest, opts ...grpc.CallOption) (*DismissCueResponse, error)
	CreateHotlist(ctx context.Context, in *CreateHotlistRequest, opts ...grpc.CallOption) (*CreateHotlistResponse, error)
}
type featuresPRPCClient struct {
	client *prpc.Client
}

func NewFeaturesPRPCClient(client *prpc.Client) FeaturesClient {
	return &featuresPRPCClient{client}
}

func (c *featuresPRPCClient) ListHotlistsByUser(ctx context.Context, in *ListHotlistsByUserRequest, opts ...grpc.CallOption) (*ListHotlistsByUserResponse, error) {
	out := new(ListHotlistsByUserResponse)
	err := c.client.Call(ctx, "monorail.Features", "ListHotlistsByUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresPRPCClient) GetHotlistStarCount(ctx context.Context, in *GetHotlistStarCountRequest, opts ...grpc.CallOption) (*GetHotlistStarCountResponse, error) {
	out := new(GetHotlistStarCountResponse)
	err := c.client.Call(ctx, "monorail.Features", "GetHotlistStarCount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresPRPCClient) StarHotlist(ctx context.Context, in *StarHotlistRequest, opts ...grpc.CallOption) (*StarHotlistResponse, error) {
	out := new(StarHotlistResponse)
	err := c.client.Call(ctx, "monorail.Features", "StarHotlist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresPRPCClient) ListHotlistIssues(ctx context.Context, in *ListHotlistIssuesRequest, opts ...grpc.CallOption) (*ListHotlistIssuesResponse, error) {
	out := new(ListHotlistIssuesResponse)
	err := c.client.Call(ctx, "monorail.Features", "ListHotlistIssues", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresPRPCClient) DismissCue(ctx context.Context, in *DismissCueRequest, opts ...grpc.CallOption) (*DismissCueResponse, error) {
	out := new(DismissCueResponse)
	err := c.client.Call(ctx, "monorail.Features", "DismissCue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresPRPCClient) CreateHotlist(ctx context.Context, in *CreateHotlistRequest, opts ...grpc.CallOption) (*CreateHotlistResponse, error) {
	out := new(CreateHotlistResponse)
	err := c.client.Call(ctx, "monorail.Features", "CreateHotlist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type featuresClient struct {
	cc *grpc.ClientConn
}

func NewFeaturesClient(cc *grpc.ClientConn) FeaturesClient {
	return &featuresClient{cc}
}

func (c *featuresClient) ListHotlistsByUser(ctx context.Context, in *ListHotlistsByUserRequest, opts ...grpc.CallOption) (*ListHotlistsByUserResponse, error) {
	out := new(ListHotlistsByUserResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/ListHotlistsByUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresClient) GetHotlistStarCount(ctx context.Context, in *GetHotlistStarCountRequest, opts ...grpc.CallOption) (*GetHotlistStarCountResponse, error) {
	out := new(GetHotlistStarCountResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/GetHotlistStarCount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresClient) StarHotlist(ctx context.Context, in *StarHotlistRequest, opts ...grpc.CallOption) (*StarHotlistResponse, error) {
	out := new(StarHotlistResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/StarHotlist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresClient) ListHotlistIssues(ctx context.Context, in *ListHotlistIssuesRequest, opts ...grpc.CallOption) (*ListHotlistIssuesResponse, error) {
	out := new(ListHotlistIssuesResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/ListHotlistIssues", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresClient) DismissCue(ctx context.Context, in *DismissCueRequest, opts ...grpc.CallOption) (*DismissCueResponse, error) {
	out := new(DismissCueResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/DismissCue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *featuresClient) CreateHotlist(ctx context.Context, in *CreateHotlistRequest, opts ...grpc.CallOption) (*CreateHotlistResponse, error) {
	out := new(CreateHotlistResponse)
	err := c.cc.Invoke(ctx, "/monorail.Features/CreateHotlist", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FeaturesServer is the server API for Features service.
type FeaturesServer interface {
	ListHotlistsByUser(context.Context, *ListHotlistsByUserRequest) (*ListHotlistsByUserResponse, error)
	GetHotlistStarCount(context.Context, *GetHotlistStarCountRequest) (*GetHotlistStarCountResponse, error)
	StarHotlist(context.Context, *StarHotlistRequest) (*StarHotlistResponse, error)
	ListHotlistIssues(context.Context, *ListHotlistIssuesRequest) (*ListHotlistIssuesResponse, error)
	DismissCue(context.Context, *DismissCueRequest) (*DismissCueResponse, error)
	CreateHotlist(context.Context, *CreateHotlistRequest) (*CreateHotlistResponse, error)
}

func RegisterFeaturesServer(s prpc.Registrar, srv FeaturesServer) {
	s.RegisterService(&_Features_serviceDesc, srv)
}

func _Features_ListHotlistsByUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListHotlistsByUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).ListHotlistsByUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/ListHotlistsByUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).ListHotlistsByUser(ctx, req.(*ListHotlistsByUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Features_GetHotlistStarCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetHotlistStarCountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).GetHotlistStarCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/GetHotlistStarCount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).GetHotlistStarCount(ctx, req.(*GetHotlistStarCountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Features_StarHotlist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StarHotlistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).StarHotlist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/StarHotlist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).StarHotlist(ctx, req.(*StarHotlistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Features_ListHotlistIssues_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListHotlistIssuesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).ListHotlistIssues(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/ListHotlistIssues",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).ListHotlistIssues(ctx, req.(*ListHotlistIssuesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Features_DismissCue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DismissCueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).DismissCue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/DismissCue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).DismissCue(ctx, req.(*DismissCueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Features_CreateHotlist_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateHotlistRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FeaturesServer).CreateHotlist(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/monorail.Features/CreateHotlist",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FeaturesServer).CreateHotlist(ctx, req.(*CreateHotlistRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Features_serviceDesc = grpc.ServiceDesc{
	ServiceName: "monorail.Features",
	HandlerType: (*FeaturesServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListHotlistsByUser",
			Handler:    _Features_ListHotlistsByUser_Handler,
		},
		{
			MethodName: "GetHotlistStarCount",
			Handler:    _Features_GetHotlistStarCount_Handler,
		},
		{
			MethodName: "StarHotlist",
			Handler:    _Features_StarHotlist_Handler,
		},
		{
			MethodName: "ListHotlistIssues",
			Handler:    _Features_ListHotlistIssues_Handler,
		},
		{
			MethodName: "DismissCue",
			Handler:    _Features_DismissCue_Handler,
		},
		{
			MethodName: "CreateHotlist",
			Handler:    _Features_CreateHotlist_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/api_proto/features.proto",
}

func init() {
	proto.RegisterFile("api/api_proto/features.proto", fileDescriptor_features_3933b38ea524f3fb)
}

var fileDescriptor_features_3933b38ea524f3fb = []byte{
	// 628 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x55, 0x4f, 0x4f, 0xd4, 0x40,
	0x14, 0xb7, 0xb0, 0x0b, 0xbb, 0x6f, 0xc3, 0x81, 0x01, 0xb4, 0x16, 0xd0, 0xcd, 0x00, 0x09, 0x89,
	0x0a, 0x11, 0xf1, 0xe6, 0x49, 0x8c, 0xb2, 0x91, 0x03, 0x19, 0x35, 0xf1, 0xe4, 0x66, 0xe8, 0xbe,
	0x95, 0x31, 0xb4, 0x53, 0x67, 0xa6, 0x26, 0x1c, 0xfd, 0x0c, 0xde, 0x3c, 0xf9, 0x25, 0xfc, 0x7e,
	0xa6, 0xd3, 0x69, 0xbb, 0x65, 0xbb, 0x68, 0x30, 0xe1, 0xd6, 0x99, 0xf7, 0xfb, 0xfd, 0xde, 0x7b,
	0xf3, 0xfe, 0x14, 0x36, 0x78, 0x22, 0xf6, 0x79, 0x22, 0x86, 0x89, 0x92, 0x46, 0xee, 0x8f, 0x91,
	0x9b, 0x54, 0xa1, 0xde, 0xb3, 0x47, 0xd2, 0x89, 0x64, 0x2c, 0x15, 0x17, 0x17, 0x41, 0x50, 0xc7,
	0x85, 0x32, 0x8a, 0x64, 0x9c, 0xa3, 0x82, 0xed, 0x66, 0x8d, 0xa1, 0x3c, 0xfb, 0x82, 0xa1, 0x71,
	0x5a, 0x34, 0x81, 0xfb, 0x27, 0x42, 0x9b, 0x63, 0x69, 0x2e, 0x84, 0x36, 0xfa, 0xe5, 0xe5, 0x07,
	0x8d, 0x8a, 0xe1, 0xd7, 0x14, 0xb5, 0x21, 0x8f, 0xa1, 0x6d, 0x14, 0x0f, 0xd1, 0xf7, 0xfa, 0xde,
	0x6e, 0xef, 0xe0, 0xee, 0x5e, 0xe1, 0x78, 0xcf, 0x21, 0xde, 0x67, 0x56, 0x96, 0x83, 0xc8, 0x0e,
	0xb4, 0x52, 0x8d, 0xca, 0x9f, 0xb3, 0xe0, 0xe5, 0x0a, 0x9c, 0x4b, 0x8e, 0x99, 0x35, 0xd3, 0xb7,
	0x10, 0x34, 0x79, 0xd4, 0x89, 0x8c, 0x35, 0x92, 0x27, 0xd0, 0x39, 0x77, 0x16, 0xdf, 0xeb, 0xcf,
	0xd7, 0x85, 0x1c, 0x87, 0x95, 0x10, 0xfa, 0xdd, 0x83, 0xe0, 0x0d, 0x16, 0x62, 0xef, 0x0c, 0x57,
	0x47, 0x32, 0x8d, 0xcd, 0xcd, 0x12, 0x78, 0x0e, 0x3d, 0x27, 0x3c, 0x54, 0x38, 0x76, 0x79, 0xac,
	0x4e, 0xbb, 0xc7, 0x31, 0x83, 0xf3, 0xf2, 0x9b, 0xbe, 0x80, 0xf5, 0xc6, 0x10, 0x5c, 0x46, 0x9b,
	0x00, 0xda, 0x70, 0x35, 0x0c, 0xb3, 0x5b, 0x1b, 0xc8, 0x12, 0xeb, 0xea, 0x02, 0x46, 0x7f, 0x78,
	0x40, 0x32, 0x52, 0x29, 0x7e, 0x7b, 0x91, 0x13, 0x1f, 0x16, 0xb3, 0x40, 0x14, 0x8e, 0xfc, 0xf9,
	0xbe, 0xb7, 0xdb, 0x61, 0xc5, 0x91, 0x1e, 0xc2, 0x4a, 0x2d, 0xa8, 0x7f, 0xcb, 0xe5, 0xb7, 0x07,
	0xfe, 0x44, 0x6d, 0x07, 0x5a, 0xa7, 0xa8, 0x6f, 0x35, 0xa3, 0x43, 0x80, 0x84, 0x7f, 0x16, 0x31,
	0x37, 0x42, 0xc6, 0x36, 0xa9, 0x1a, 0xeb, 0xb4, 0xb4, 0xb1, 0x09, 0x1c, 0x3d, 0xae, 0x0d, 0x41,
	0x11, 0xb6, 0xcb, 0xf9, 0x11, 0xb4, 0x85, 0xc1, 0xa8, 0x68, 0xc7, 0xb5, 0xa9, 0x18, 0x06, 0x06,
	0x23, 0x96, 0x63, 0xe8, 0x47, 0x58, 0x7e, 0x25, 0x74, 0x24, 0xb4, 0x3e, 0x4a, 0xf1, 0x66, 0x99,
	0xaf, 0xc1, 0x42, 0x98, 0xe2, 0x50, 0x8c, 0x6c, 0xd2, 0x5d, 0xd6, 0x0e, 0x53, 0x1c, 0x8c, 0xe8,
	0x2a, 0x90, 0x49, 0xe5, 0x3c, 0x38, 0xfa, 0x73, 0x0e, 0x56, 0x8f, 0x14, 0x72, 0x83, 0xff, 0xd5,
	0x3f, 0x04, 0x5a, 0x31, 0x8f, 0xd0, 0x79, 0xb4, 0xdf, 0xb6, 0x39, 0xd2, 0x28, 0xe2, 0xea, 0xd2,
	0xbe, 0x63, 0x97, 0x15, 0x47, 0xd2, 0x87, 0xde, 0x08, 0x75, 0xa8, 0x44, 0x62, 0x5f, 0xb9, 0x65,
	0xad, 0x93, 0x57, 0xe4, 0x00, 0x7a, 0x38, 0x12, 0x46, 0xaa, 0xac, 0x78, 0xda, 0x6f, 0x5f, 0x1d,
	0xe4, 0x62, 0x23, 0x40, 0x8e, 0x62, 0x38, 0xd6, 0xe4, 0x29, 0x80, 0xc8, 0x5e, 0x3e, 0xa7, 0x2c,
	0x58, 0x0a, 0xa9, 0x28, 0xb6, 0x2a, 0x19, 0xa7, 0x2b, 0xdc, 0x97, 0xce, 0xda, 0x51, 0xe8, 0x61,
	0xa2, 0xc4, 0x37, 0x6e, 0xd0, 0x5f, 0xb4, 0x2d, 0xdc, 0x15, 0xfa, 0x34, 0xbf, 0xa0, 0xf7, 0x60,
	0xed, 0xca, 0xdb, 0xe4, 0xaf, 0x76, 0xf0, 0xab, 0x05, 0x9d, 0xd7, 0x6e, 0x1f, 0x12, 0x0e, 0x64,
	0x7a, 0x1f, 0x91, 0xad, 0xca, 0xf3, 0xcc, 0xfd, 0x18, 0x6c, 0x5f, 0x0f, 0x72, 0x35, 0xba, 0x43,
	0x46, 0xb0, 0xd2, 0xb0, 0x21, 0xc8, 0x04, 0x7d, 0xf6, 0x0e, 0x0b, 0x76, 0xfe, 0x82, 0x2a, 0xbd,
	0x9c, 0x40, 0x6f, 0x62, 0x66, 0xc9, 0x46, 0xc5, 0x9b, 0xde, 0x2f, 0xc1, 0xe6, 0x0c, 0x6b, 0xa9,
	0xf6, 0x09, 0x96, 0xa7, 0x66, 0x82, 0xd0, 0xc6, 0x84, 0x6b, 0x73, 0x1e, 0x6c, 0x5d, 0x8b, 0x29,
	0xf5, 0x07, 0x00, 0x55, 0x3f, 0x93, 0xf5, 0x8a, 0x34, 0x35, 0x3f, 0xc1, 0x46, 0xb3, 0xb1, 0x94,
	0x62, 0xb0, 0x54, 0xab, 0x33, 0x79, 0x50, 0x11, 0x9a, 0x86, 0x23, 0x78, 0x38, 0xd3, 0x5e, 0x68,
	0x9e, 0x2d, 0xd8, 0xdf, 0xe3, 0xb3, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x7d, 0x12, 0x9a, 0x4b,
	0x8a, 0x07, 0x00, 0x00,
}
