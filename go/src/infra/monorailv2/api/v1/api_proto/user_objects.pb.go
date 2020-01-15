// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/v1/api_proto/user_objects.proto

package monorail_v1

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// The global site role for a User.
// Next available tag: 3
type User_SiteRole int32

const (
	// Default value. This value is unused.
	User_SITE_ROLE_UNSPECIFIED User_SiteRole = 0
	// Normal site user with no special site-wide extra permissions.
	User_NORMAL User_SiteRole = 1
	// Site-wide admin role.
	User_ADMIN User_SiteRole = 2
)

var User_SiteRole_name = map[int32]string{
	0: "SITE_ROLE_UNSPECIFIED",
	1: "NORMAL",
	2: "ADMIN",
}

var User_SiteRole_value = map[string]int32{
	"SITE_ROLE_UNSPECIFIED": 0,
	"NORMAL":                1,
	"ADMIN":                 2,
}

func (x User_SiteRole) String() string {
	return proto.EnumName(User_SiteRole_name, int32(x))
}

func (User_SiteRole) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_b8c6722b190bb817, []int{0, 0}
}

// User represents a user of the Monorail site.
// Next available tag: 6
type User struct {
	// Resource name of the user.
	// Format: users/{user_id}
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Obscured or un-obscured user email or name to show other users using the site.
	DisplayName string        `protobuf:"bytes,2,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	SiteRole    User_SiteRole `protobuf:"varint,3,opt,name=site_role,json=siteRole,proto3,enum=monorail.v1.User_SiteRole" json:"site_role,omitempty"`
	// User-written indication of their availability or working hours.
	AvailabilityMessage string `protobuf:"bytes,4,opt,name=availability_message,json=availabilityMessage,proto3" json:"availability_message,omitempty"`
	// Resource name of a linked primary User.
	LinkedPrimaryUser    string   `protobuf:"bytes,5,opt,name=linked_primary_user,json=linkedPrimaryUser,proto3" json:"linked_primary_user,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *User) Reset()         { *m = User{} }
func (m *User) String() string { return proto.CompactTextString(m) }
func (*User) ProtoMessage()    {}
func (*User) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8c6722b190bb817, []int{0}
}

func (m *User) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_User.Unmarshal(m, b)
}
func (m *User) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_User.Marshal(b, m, deterministic)
}
func (m *User) XXX_Merge(src proto.Message) {
	xxx_messageInfo_User.Merge(m, src)
}
func (m *User) XXX_Size() int {
	return xxx_messageInfo_User.Size(m)
}
func (m *User) XXX_DiscardUnknown() {
	xxx_messageInfo_User.DiscardUnknown(m)
}

var xxx_messageInfo_User proto.InternalMessageInfo

func (m *User) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *User) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *User) GetSiteRole() User_SiteRole {
	if m != nil {
		return m.SiteRole
	}
	return User_SITE_ROLE_UNSPECIFIED
}

func (m *User) GetAvailabilityMessage() string {
	if m != nil {
		return m.AvailabilityMessage
	}
	return ""
}

func (m *User) GetLinkedPrimaryUser() string {
	if m != nil {
		return m.LinkedPrimaryUser
	}
	return ""
}

func init() {
	proto.RegisterEnum("monorail.v1.User_SiteRole", User_SiteRole_name, User_SiteRole_value)
	proto.RegisterType((*User)(nil), "monorail.v1.User")
}

func init() {
	proto.RegisterFile("api/v1/api_proto/user_objects.proto", fileDescriptor_b8c6722b190bb817)
}

var fileDescriptor_b8c6722b190bb817 = []byte{
	// 344 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0x41, 0x4f, 0xc2, 0x40,
	0x10, 0x85, 0x2d, 0x05, 0x02, 0x8b, 0x31, 0xb8, 0x60, 0x52, 0x7b, 0x22, 0x98, 0x10, 0x0e, 0xa6,
	0x0d, 0x7a, 0xf0, 0xe2, 0xa5, 0x4a, 0x4d, 0x6a, 0xa0, 0x90, 0x22, 0xe7, 0xcd, 0xb6, 0x8c, 0x75,
	0x75, 0xcb, 0x36, 0xbb, 0xa5, 0x09, 0xff, 0xcc, 0x9f, 0xc3, 0xef, 0xf0, 0x64, 0xda, 0x62, 0xc2,
	0xc5, 0xdb, 0xcc, 0x7c, 0xef, 0x65, 0x5e, 0x1e, 0xba, 0xa1, 0x29, 0xb3, 0xf3, 0x89, 0x4d, 0x53,
	0x46, 0x52, 0x29, 0x32, 0x61, 0xef, 0x14, 0x48, 0x22, 0xc2, 0x4f, 0x88, 0x32, 0x65, 0x95, 0x27,
	0xdc, 0x49, 0xc4, 0x56, 0x48, 0xca, 0xb8, 0x95, 0x4f, 0xcc, 0x51, 0x2c, 0x44, 0xcc, 0xe1, 0xa8,
	0xae, 0x96, 0xc2, 0x6e, 0x4b, 0x50, 0x62, 0x27, 0x23, 0xa8, 0x4c, 0xe6, 0xed, 0x7f, 0xba, 0x77,
	0x06, 0x7c, 0x43, 0x42, 0xf8, 0xa0, 0x39, 0x13, 0xb2, 0x52, 0x0f, 0xbf, 0x6b, 0xa8, 0xbe, 0x56,
	0x20, 0x31, 0x46, 0xf5, 0x2d, 0x4d, 0xc0, 0xd0, 0x06, 0xda, 0xb8, 0x1d, 0x94, 0x33, 0x1e, 0xa1,
	0xf3, 0x0d, 0x53, 0x29, 0xa7, 0x7b, 0x52, 0xb2, 0x5a, 0xc1, 0x9e, 0xf4, 0x83, 0xa3, 0x07, 0x9d,
	0x23, 0xf0, 0x0b, 0xdd, 0x03, 0x6a, 0x2b, 0x96, 0x01, 0x91, 0x82, 0x83, 0xa1, 0x0f, 0xb4, 0xf1,
	0xc5, 0x9d, 0x69, 0x9d, 0x64, 0xb7, 0x8a, 0x0f, 0xd6, 0x8a, 0x65, 0x10, 0x08, 0x0e, 0x41, 0x4b,
	0x1d, 0x27, 0x3c, 0x41, 0x7d, 0x9a, 0x53, 0xc6, 0x69, 0xc8, 0x38, 0xcb, 0xf6, 0x24, 0x01, 0xa5,
	0x68, 0x0c, 0x46, 0xbd, 0x0c, 0xd1, 0x3b, 0x65, 0xf3, 0x0a, 0xe1, 0x57, 0xd4, 0xe3, 0x6c, 0xfb,
	0x05, 0x1b, 0x92, 0x4a, 0x96, 0x50, 0xb9, 0x27, 0x45, 0x71, 0x46, 0xa3, 0x8c, 0x66, 0x1e, 0x1c,
	0xfd, 0xc7, 0xe9, 0x23, 0x4c, 0x53, 0x66, 0x45, 0x32, 0xdc, 0xc5, 0x56, 0x24, 0x12, 0xbb, 0x78,
	0x1f, 0x5c, 0x56, 0xb6, 0x65, 0xe5, 0x2a, 0x4e, 0xc3, 0x47, 0xd4, 0xfa, 0x0b, 0x85, 0xaf, 0xd1,
	0xd5, 0xca, 0x7b, 0x73, 0x49, 0xb0, 0x98, 0xb9, 0x64, 0xed, 0xaf, 0x96, 0xee, 0xb3, 0xf7, 0xe2,
	0xb9, 0xd3, 0xee, 0x19, 0x46, 0xa8, 0xe9, 0x2f, 0x82, 0xb9, 0x33, 0xeb, 0x6a, 0xb8, 0x8d, 0x1a,
	0xce, 0x74, 0xee, 0xf9, 0xdd, 0x5a, 0xd8, 0x2c, 0x1b, 0xbc, 0xff, 0x0d, 0x00, 0x00, 0xff, 0xff,
	0xae, 0xc9, 0xea, 0x49, 0xcb, 0x01, 0x00, 0x00,
}
