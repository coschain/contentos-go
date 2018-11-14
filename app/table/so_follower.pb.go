// Code generated by protoc-gen-go. DO NOT EDIT.
// source: app/table/so_follower.proto

package table

import (
	fmt "fmt"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type SoFollower struct {
	FollowerInfo         *prototype.FollowerRelation `protobuf:"bytes,1,opt,name=follower_info,json=followerInfo,proto3" json:"follower_info,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *SoFollower) Reset()         { *m = SoFollower{} }
func (m *SoFollower) String() string { return proto.CompactTextString(m) }
func (*SoFollower) ProtoMessage()    {}
func (*SoFollower) Descriptor() ([]byte, []int) {
	return fileDescriptor_d122501d0783c405, []int{0}
}

func (m *SoFollower) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoFollower.Unmarshal(m, b)
}
func (m *SoFollower) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoFollower.Marshal(b, m, deterministic)
}
func (m *SoFollower) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoFollower.Merge(m, src)
}
func (m *SoFollower) XXX_Size() int {
	return xxx_messageInfo_SoFollower.Size(m)
}
func (m *SoFollower) XXX_DiscardUnknown() {
	xxx_messageInfo_SoFollower.DiscardUnknown(m)
}

var xxx_messageInfo_SoFollower proto.InternalMessageInfo

func (m *SoFollower) GetFollowerInfo() *prototype.FollowerRelation {
	if m != nil {
		return m.FollowerInfo
	}
	return nil
}

type SoListFollowerByFollowerInfo struct {
	FollowerInfo         *prototype.FollowerRelation `protobuf:"bytes,1,opt,name=follower_info,json=followerInfo,proto3" json:"follower_info,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *SoListFollowerByFollowerInfo) Reset()         { *m = SoListFollowerByFollowerInfo{} }
func (m *SoListFollowerByFollowerInfo) String() string { return proto.CompactTextString(m) }
func (*SoListFollowerByFollowerInfo) ProtoMessage()    {}
func (*SoListFollowerByFollowerInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_d122501d0783c405, []int{1}
}

func (m *SoListFollowerByFollowerInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoListFollowerByFollowerInfo.Unmarshal(m, b)
}
func (m *SoListFollowerByFollowerInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoListFollowerByFollowerInfo.Marshal(b, m, deterministic)
}
func (m *SoListFollowerByFollowerInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoListFollowerByFollowerInfo.Merge(m, src)
}
func (m *SoListFollowerByFollowerInfo) XXX_Size() int {
	return xxx_messageInfo_SoListFollowerByFollowerInfo.Size(m)
}
func (m *SoListFollowerByFollowerInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_SoListFollowerByFollowerInfo.DiscardUnknown(m)
}

var xxx_messageInfo_SoListFollowerByFollowerInfo proto.InternalMessageInfo

func (m *SoListFollowerByFollowerInfo) GetFollowerInfo() *prototype.FollowerRelation {
	if m != nil {
		return m.FollowerInfo
	}
	return nil
}

type SoUniqueFollowerByFollowerInfo struct {
	FollowerInfo         *prototype.FollowerRelation `protobuf:"bytes,1,opt,name=follower_info,json=followerInfo,proto3" json:"follower_info,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *SoUniqueFollowerByFollowerInfo) Reset()         { *m = SoUniqueFollowerByFollowerInfo{} }
func (m *SoUniqueFollowerByFollowerInfo) String() string { return proto.CompactTextString(m) }
func (*SoUniqueFollowerByFollowerInfo) ProtoMessage()    {}
func (*SoUniqueFollowerByFollowerInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_d122501d0783c405, []int{2}
}

func (m *SoUniqueFollowerByFollowerInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoUniqueFollowerByFollowerInfo.Unmarshal(m, b)
}
func (m *SoUniqueFollowerByFollowerInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoUniqueFollowerByFollowerInfo.Marshal(b, m, deterministic)
}
func (m *SoUniqueFollowerByFollowerInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoUniqueFollowerByFollowerInfo.Merge(m, src)
}
func (m *SoUniqueFollowerByFollowerInfo) XXX_Size() int {
	return xxx_messageInfo_SoUniqueFollowerByFollowerInfo.Size(m)
}
func (m *SoUniqueFollowerByFollowerInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_SoUniqueFollowerByFollowerInfo.DiscardUnknown(m)
}

var xxx_messageInfo_SoUniqueFollowerByFollowerInfo proto.InternalMessageInfo

func (m *SoUniqueFollowerByFollowerInfo) GetFollowerInfo() *prototype.FollowerRelation {
	if m != nil {
		return m.FollowerInfo
	}
	return nil
}

func init() {
	proto.RegisterType((*SoFollower)(nil), "table.so_follower")
	proto.RegisterType((*SoListFollowerByFollowerInfo)(nil), "table.so_list_follower_by_follower_info")
	proto.RegisterType((*SoUniqueFollowerByFollowerInfo)(nil), "table.so_unique_follower_by_follower_info")
}

func init() { proto.RegisterFile("app/table/so_follower.proto", fileDescriptor_d122501d0783c405) }

var fileDescriptor_d122501d0783c405 = []byte{
	// 196 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4e, 0x2c, 0x28, 0xd0,
	0x2f, 0x49, 0x4c, 0xca, 0x49, 0xd5, 0x2f, 0xce, 0x8f, 0x4f, 0xcb, 0xcf, 0xc9, 0xc9, 0x2f, 0x4f,
	0x2d, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x4b, 0x48, 0x89, 0x80, 0x79, 0x25,
	0x95, 0x05, 0xa9, 0xfa, 0x20, 0x02, 0x22, 0xa9, 0x14, 0xc0, 0xc5, 0x8d, 0xa4, 0x43, 0xc8, 0x91,
	0x8b, 0x17, 0xc6, 0x8e, 0xcf, 0xcc, 0x4b, 0xcb, 0x97, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x36, 0x92,
	0xd1, 0x83, 0x6b, 0xd6, 0x83, 0xcb, 0x17, 0xa5, 0xe6, 0x24, 0x96, 0x64, 0xe6, 0xe7, 0x05, 0xf1,
	0xc0, 0x84, 0x3c, 0xf3, 0xd2, 0xf2, 0x95, 0xd2, 0xb8, 0x14, 0x8b, 0xf3, 0xe3, 0x73, 0x32, 0x8b,
	0x4b, 0xe0, 0xc6, 0xc6, 0x27, 0x55, 0xc6, 0xa3, 0x18, 0x4b, 0x0d, 0x7b, 0x32, 0xb8, 0x94, 0x8b,
	0xf3, 0xe3, 0x4b, 0xf3, 0x32, 0x0b, 0x4b, 0x53, 0x69, 0x6a, 0x93, 0x93, 0x46, 0x94, 0x5a, 0x7a,
	0x66, 0x49, 0x46, 0x69, 0x92, 0x5e, 0x72, 0x7e, 0xae, 0x7e, 0x72, 0x7e, 0x71, 0x72, 0x46, 0x62,
	0x66, 0x9e, 0x7e, 0x72, 0x7e, 0x5e, 0x49, 0x6a, 0x5e, 0x49, 0x7e, 0xb1, 0x6e, 0x7a, 0x3e, 0x24,
	0xf0, 0x93, 0xd8, 0xc0, 0x86, 0x1a, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xe0, 0x98, 0x99, 0x07,
	0x90, 0x01, 0x00, 0x00,
}
