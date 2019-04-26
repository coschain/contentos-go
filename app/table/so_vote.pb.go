// Code generated by protoc-gen-go. DO NOT EDIT.
// source: app/table/so_vote.proto

package table // import "github.com/coschain/contentos-go/app/table"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import prototype "github.com/coschain/contentos-go/prototype"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type SoVote struct {
	Voter                *prototype.VoterId      `protobuf:"bytes,1,opt,name=voter,proto3" json:"voter,omitempty"`
	VoteTime             *prototype.TimePointSec `protobuf:"bytes,2,opt,name=vote_time,json=voteTime,proto3" json:"vote_time,omitempty"`
	PostId               uint64                  `protobuf:"varint,3,opt,name=post_id,json=postId,proto3" json:"post_id,omitempty"`
	WeightedVp           uint64                  `protobuf:"varint,4,opt,name=weighted_vp,json=weightedVp,proto3" json:"weighted_vp,omitempty"`
	Upvote               bool                    `protobuf:"varint,5,opt,name=upvote,proto3" json:"upvote,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *SoVote) Reset()         { *m = SoVote{} }
func (m *SoVote) String() string { return proto.CompactTextString(m) }
func (*SoVote) ProtoMessage()    {}
func (*SoVote) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{0}
}
func (m *SoVote) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoVote.Unmarshal(m, b)
}
func (m *SoVote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoVote.Marshal(b, m, deterministic)
}
func (dst *SoVote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoVote.Merge(dst, src)
}
func (m *SoVote) XXX_Size() int {
	return xxx_messageInfo_SoVote.Size(m)
}
func (m *SoVote) XXX_DiscardUnknown() {
	xxx_messageInfo_SoVote.DiscardUnknown(m)
}

var xxx_messageInfo_SoVote proto.InternalMessageInfo

func (m *SoVote) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

func (m *SoVote) GetVoteTime() *prototype.TimePointSec {
	if m != nil {
		return m.VoteTime
	}
	return nil
}

func (m *SoVote) GetPostId() uint64 {
	if m != nil {
		return m.PostId
	}
	return 0
}

func (m *SoVote) GetWeightedVp() uint64 {
	if m != nil {
		return m.WeightedVp
	}
	return 0
}

func (m *SoVote) GetUpvote() bool {
	if m != nil {
		return m.Upvote
	}
	return false
}

type SoMemVoteByVoter struct {
	Voter                *prototype.VoterId `protobuf:"bytes,1,opt,name=voter,proto3" json:"voter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *SoMemVoteByVoter) Reset()         { *m = SoMemVoteByVoter{} }
func (m *SoMemVoteByVoter) String() string { return proto.CompactTextString(m) }
func (*SoMemVoteByVoter) ProtoMessage()    {}
func (*SoMemVoteByVoter) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{1}
}
func (m *SoMemVoteByVoter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoMemVoteByVoter.Unmarshal(m, b)
}
func (m *SoMemVoteByVoter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoMemVoteByVoter.Marshal(b, m, deterministic)
}
func (dst *SoMemVoteByVoter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoMemVoteByVoter.Merge(dst, src)
}
func (m *SoMemVoteByVoter) XXX_Size() int {
	return xxx_messageInfo_SoMemVoteByVoter.Size(m)
}
func (m *SoMemVoteByVoter) XXX_DiscardUnknown() {
	xxx_messageInfo_SoMemVoteByVoter.DiscardUnknown(m)
}

var xxx_messageInfo_SoMemVoteByVoter proto.InternalMessageInfo

func (m *SoMemVoteByVoter) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

type SoMemVoteByVoteTime struct {
	VoteTime             *prototype.TimePointSec `protobuf:"bytes,1,opt,name=vote_time,json=voteTime,proto3" json:"vote_time,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *SoMemVoteByVoteTime) Reset()         { *m = SoMemVoteByVoteTime{} }
func (m *SoMemVoteByVoteTime) String() string { return proto.CompactTextString(m) }
func (*SoMemVoteByVoteTime) ProtoMessage()    {}
func (*SoMemVoteByVoteTime) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{2}
}
func (m *SoMemVoteByVoteTime) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoMemVoteByVoteTime.Unmarshal(m, b)
}
func (m *SoMemVoteByVoteTime) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoMemVoteByVoteTime.Marshal(b, m, deterministic)
}
func (dst *SoMemVoteByVoteTime) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoMemVoteByVoteTime.Merge(dst, src)
}
func (m *SoMemVoteByVoteTime) XXX_Size() int {
	return xxx_messageInfo_SoMemVoteByVoteTime.Size(m)
}
func (m *SoMemVoteByVoteTime) XXX_DiscardUnknown() {
	xxx_messageInfo_SoMemVoteByVoteTime.DiscardUnknown(m)
}

var xxx_messageInfo_SoMemVoteByVoteTime proto.InternalMessageInfo

func (m *SoMemVoteByVoteTime) GetVoteTime() *prototype.TimePointSec {
	if m != nil {
		return m.VoteTime
	}
	return nil
}

type SoMemVoteByPostId struct {
	PostId               uint64   `protobuf:"varint,1,opt,name=post_id,json=postId,proto3" json:"post_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SoMemVoteByPostId) Reset()         { *m = SoMemVoteByPostId{} }
func (m *SoMemVoteByPostId) String() string { return proto.CompactTextString(m) }
func (*SoMemVoteByPostId) ProtoMessage()    {}
func (*SoMemVoteByPostId) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{3}
}
func (m *SoMemVoteByPostId) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoMemVoteByPostId.Unmarshal(m, b)
}
func (m *SoMemVoteByPostId) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoMemVoteByPostId.Marshal(b, m, deterministic)
}
func (dst *SoMemVoteByPostId) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoMemVoteByPostId.Merge(dst, src)
}
func (m *SoMemVoteByPostId) XXX_Size() int {
	return xxx_messageInfo_SoMemVoteByPostId.Size(m)
}
func (m *SoMemVoteByPostId) XXX_DiscardUnknown() {
	xxx_messageInfo_SoMemVoteByPostId.DiscardUnknown(m)
}

var xxx_messageInfo_SoMemVoteByPostId proto.InternalMessageInfo

func (m *SoMemVoteByPostId) GetPostId() uint64 {
	if m != nil {
		return m.PostId
	}
	return 0
}

type SoMemVoteByWeightedVp struct {
	WeightedVp           uint64   `protobuf:"varint,1,opt,name=weighted_vp,json=weightedVp,proto3" json:"weighted_vp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SoMemVoteByWeightedVp) Reset()         { *m = SoMemVoteByWeightedVp{} }
func (m *SoMemVoteByWeightedVp) String() string { return proto.CompactTextString(m) }
func (*SoMemVoteByWeightedVp) ProtoMessage()    {}
func (*SoMemVoteByWeightedVp) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{4}
}
func (m *SoMemVoteByWeightedVp) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoMemVoteByWeightedVp.Unmarshal(m, b)
}
func (m *SoMemVoteByWeightedVp) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoMemVoteByWeightedVp.Marshal(b, m, deterministic)
}
func (dst *SoMemVoteByWeightedVp) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoMemVoteByWeightedVp.Merge(dst, src)
}
func (m *SoMemVoteByWeightedVp) XXX_Size() int {
	return xxx_messageInfo_SoMemVoteByWeightedVp.Size(m)
}
func (m *SoMemVoteByWeightedVp) XXX_DiscardUnknown() {
	xxx_messageInfo_SoMemVoteByWeightedVp.DiscardUnknown(m)
}

var xxx_messageInfo_SoMemVoteByWeightedVp proto.InternalMessageInfo

func (m *SoMemVoteByWeightedVp) GetWeightedVp() uint64 {
	if m != nil {
		return m.WeightedVp
	}
	return 0
}

type SoMemVoteByUpvote struct {
	Upvote               bool     `protobuf:"varint,1,opt,name=upvote,proto3" json:"upvote,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SoMemVoteByUpvote) Reset()         { *m = SoMemVoteByUpvote{} }
func (m *SoMemVoteByUpvote) String() string { return proto.CompactTextString(m) }
func (*SoMemVoteByUpvote) ProtoMessage()    {}
func (*SoMemVoteByUpvote) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{5}
}
func (m *SoMemVoteByUpvote) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoMemVoteByUpvote.Unmarshal(m, b)
}
func (m *SoMemVoteByUpvote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoMemVoteByUpvote.Marshal(b, m, deterministic)
}
func (dst *SoMemVoteByUpvote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoMemVoteByUpvote.Merge(dst, src)
}
func (m *SoMemVoteByUpvote) XXX_Size() int {
	return xxx_messageInfo_SoMemVoteByUpvote.Size(m)
}
func (m *SoMemVoteByUpvote) XXX_DiscardUnknown() {
	xxx_messageInfo_SoMemVoteByUpvote.DiscardUnknown(m)
}

var xxx_messageInfo_SoMemVoteByUpvote proto.InternalMessageInfo

func (m *SoMemVoteByUpvote) GetUpvote() bool {
	if m != nil {
		return m.Upvote
	}
	return false
}

type SoListVoteByVoter struct {
	Voter                *prototype.VoterId `protobuf:"bytes,1,opt,name=voter,proto3" json:"voter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *SoListVoteByVoter) Reset()         { *m = SoListVoteByVoter{} }
func (m *SoListVoteByVoter) String() string { return proto.CompactTextString(m) }
func (*SoListVoteByVoter) ProtoMessage()    {}
func (*SoListVoteByVoter) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{6}
}
func (m *SoListVoteByVoter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoListVoteByVoter.Unmarshal(m, b)
}
func (m *SoListVoteByVoter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoListVoteByVoter.Marshal(b, m, deterministic)
}
func (dst *SoListVoteByVoter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoListVoteByVoter.Merge(dst, src)
}
func (m *SoListVoteByVoter) XXX_Size() int {
	return xxx_messageInfo_SoListVoteByVoter.Size(m)
}
func (m *SoListVoteByVoter) XXX_DiscardUnknown() {
	xxx_messageInfo_SoListVoteByVoter.DiscardUnknown(m)
}

var xxx_messageInfo_SoListVoteByVoter proto.InternalMessageInfo

func (m *SoListVoteByVoter) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

type SoListVoteByVoteTime struct {
	VoteTime             *prototype.TimePointSec `protobuf:"bytes,1,opt,name=vote_time,json=voteTime,proto3" json:"vote_time,omitempty"`
	Voter                *prototype.VoterId      `protobuf:"bytes,2,opt,name=voter,proto3" json:"voter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *SoListVoteByVoteTime) Reset()         { *m = SoListVoteByVoteTime{} }
func (m *SoListVoteByVoteTime) String() string { return proto.CompactTextString(m) }
func (*SoListVoteByVoteTime) ProtoMessage()    {}
func (*SoListVoteByVoteTime) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{7}
}
func (m *SoListVoteByVoteTime) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoListVoteByVoteTime.Unmarshal(m, b)
}
func (m *SoListVoteByVoteTime) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoListVoteByVoteTime.Marshal(b, m, deterministic)
}
func (dst *SoListVoteByVoteTime) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoListVoteByVoteTime.Merge(dst, src)
}
func (m *SoListVoteByVoteTime) XXX_Size() int {
	return xxx_messageInfo_SoListVoteByVoteTime.Size(m)
}
func (m *SoListVoteByVoteTime) XXX_DiscardUnknown() {
	xxx_messageInfo_SoListVoteByVoteTime.DiscardUnknown(m)
}

var xxx_messageInfo_SoListVoteByVoteTime proto.InternalMessageInfo

func (m *SoListVoteByVoteTime) GetVoteTime() *prototype.TimePointSec {
	if m != nil {
		return m.VoteTime
	}
	return nil
}

func (m *SoListVoteByVoteTime) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

type SoListVoteByPostId struct {
	PostId               uint64             `protobuf:"varint,1,opt,name=post_id,json=postId,proto3" json:"post_id,omitempty"`
	Voter                *prototype.VoterId `protobuf:"bytes,2,opt,name=voter,proto3" json:"voter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *SoListVoteByPostId) Reset()         { *m = SoListVoteByPostId{} }
func (m *SoListVoteByPostId) String() string { return proto.CompactTextString(m) }
func (*SoListVoteByPostId) ProtoMessage()    {}
func (*SoListVoteByPostId) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{8}
}
func (m *SoListVoteByPostId) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoListVoteByPostId.Unmarshal(m, b)
}
func (m *SoListVoteByPostId) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoListVoteByPostId.Marshal(b, m, deterministic)
}
func (dst *SoListVoteByPostId) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoListVoteByPostId.Merge(dst, src)
}
func (m *SoListVoteByPostId) XXX_Size() int {
	return xxx_messageInfo_SoListVoteByPostId.Size(m)
}
func (m *SoListVoteByPostId) XXX_DiscardUnknown() {
	xxx_messageInfo_SoListVoteByPostId.DiscardUnknown(m)
}

var xxx_messageInfo_SoListVoteByPostId proto.InternalMessageInfo

func (m *SoListVoteByPostId) GetPostId() uint64 {
	if m != nil {
		return m.PostId
	}
	return 0
}

func (m *SoListVoteByPostId) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

type SoUniqueVoteByVoter struct {
	Voter                *prototype.VoterId `protobuf:"bytes,1,opt,name=voter,proto3" json:"voter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *SoUniqueVoteByVoter) Reset()         { *m = SoUniqueVoteByVoter{} }
func (m *SoUniqueVoteByVoter) String() string { return proto.CompactTextString(m) }
func (*SoUniqueVoteByVoter) ProtoMessage()    {}
func (*SoUniqueVoteByVoter) Descriptor() ([]byte, []int) {
	return fileDescriptor_so_vote_e2689b83ca56c18a, []int{9}
}
func (m *SoUniqueVoteByVoter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SoUniqueVoteByVoter.Unmarshal(m, b)
}
func (m *SoUniqueVoteByVoter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SoUniqueVoteByVoter.Marshal(b, m, deterministic)
}
func (dst *SoUniqueVoteByVoter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SoUniqueVoteByVoter.Merge(dst, src)
}
func (m *SoUniqueVoteByVoter) XXX_Size() int {
	return xxx_messageInfo_SoUniqueVoteByVoter.Size(m)
}
func (m *SoUniqueVoteByVoter) XXX_DiscardUnknown() {
	xxx_messageInfo_SoUniqueVoteByVoter.DiscardUnknown(m)
}

var xxx_messageInfo_SoUniqueVoteByVoter proto.InternalMessageInfo

func (m *SoUniqueVoteByVoter) GetVoter() *prototype.VoterId {
	if m != nil {
		return m.Voter
	}
	return nil
}

func init() {
	proto.RegisterType((*SoVote)(nil), "table.so_vote")
	proto.RegisterType((*SoMemVoteByVoter)(nil), "table.so_mem_vote_by_voter")
	proto.RegisterType((*SoMemVoteByVoteTime)(nil), "table.so_mem_vote_by_vote_time")
	proto.RegisterType((*SoMemVoteByPostId)(nil), "table.so_mem_vote_by_post_id")
	proto.RegisterType((*SoMemVoteByWeightedVp)(nil), "table.so_mem_vote_by_weighted_vp")
	proto.RegisterType((*SoMemVoteByUpvote)(nil), "table.so_mem_vote_by_upvote")
	proto.RegisterType((*SoListVoteByVoter)(nil), "table.so_list_vote_by_voter")
	proto.RegisterType((*SoListVoteByVoteTime)(nil), "table.so_list_vote_by_vote_time")
	proto.RegisterType((*SoListVoteByPostId)(nil), "table.so_list_vote_by_post_id")
	proto.RegisterType((*SoUniqueVoteByVoter)(nil), "table.so_unique_vote_by_voter")
}

func init() { proto.RegisterFile("app/table/so_vote.proto", fileDescriptor_so_vote_e2689b83ca56c18a) }

var fileDescriptor_so_vote_e2689b83ca56c18a = []byte{
	// 366 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x93, 0x4f, 0x4f, 0xf2, 0x40,
	0x10, 0xc6, 0xb3, 0xbc, 0xfc, 0x7b, 0x87, 0x5b, 0x45, 0x28, 0x5c, 0x24, 0x3d, 0xa1, 0xd1, 0x6e,
	0xd4, 0xc4, 0x9b, 0x07, 0x89, 0x17, 0xaf, 0x8d, 0xf1, 0x60, 0x62, 0x36, 0xb4, 0xdd, 0xc0, 0x26,
	0xb4, 0xbb, 0xb2, 0x53, 0x0c, 0x17, 0x3f, 0x9a, 0x9f, 0xcd, 0xec, 0x16, 0xa1, 0x34, 0x24, 0x20,
	0x5e, 0xda, 0xce, 0xec, 0x33, 0xcf, 0xf4, 0xf9, 0x25, 0x0b, 0xdd, 0xb1, 0x52, 0x14, 0xc7, 0xe1,
	0x8c, 0x53, 0x2d, 0xd9, 0x42, 0x22, 0xf7, 0xd5, 0x5c, 0xa2, 0x74, 0x6a, 0xb6, 0xd9, 0x77, 0x6d,
	0x85, 0x4b, 0xc5, 0x69, 0x92, 0xcd, 0x50, 0x30, 0x11, 0xe7, 0x82, 0x7e, 0x7b, 0x73, 0x62, 0x1e,
	0x79, 0xd7, 0xfb, 0x22, 0xd0, 0x58, 0x19, 0x39, 0xe7, 0x50, 0x33, 0xef, 0xb9, 0x4b, 0x06, 0x64,
	0xd8, 0xba, 0x39, 0xf1, 0xd7, 0x13, 0xbe, 0xed, 0x33, 0x11, 0x07, 0xb9, 0xc2, 0xb9, 0x83, 0xff,
	0xe6, 0x83, 0xa1, 0x48, 0xb8, 0x5b, 0xb1, 0xf2, 0x5e, 0x41, 0x6e, 0xda, 0x4c, 0x49, 0x91, 0x22,
	0xd3, 0x3c, 0x0a, 0x9a, 0x46, 0xfb, 0x2c, 0x12, 0xee, 0x74, 0xa1, 0xa1, 0xa4, 0x46, 0x26, 0x62,
	0xf7, 0xdf, 0x80, 0x0c, 0xab, 0x41, 0xdd, 0x94, 0x4f, 0xb1, 0x73, 0x06, 0xad, 0x0f, 0x2e, 0x26,
	0x53, 0xe4, 0x31, 0x5b, 0x28, 0xb7, 0x6a, 0x0f, 0xe1, 0xa7, 0xf5, 0xa2, 0x9c, 0x0e, 0xd4, 0x33,
	0x65, 0x7c, 0xdc, 0xda, 0x80, 0x0c, 0x9b, 0xc1, 0xaa, 0xf2, 0x1e, 0xa0, 0xad, 0x25, 0x4b, 0x78,
	0x62, 0x33, 0xb0, 0x70, 0xc9, 0xf2, 0x3f, 0x3c, 0x3c, 0x8c, 0x17, 0x80, 0xbb, 0xc3, 0xc2, 0x66,
	0xdb, 0x0e, 0x4a, 0x0e, 0x0e, 0xea, 0x5d, 0x43, 0xa7, 0xe4, 0xb9, 0xca, 0x5d, 0x44, 0x40, 0x8a,
	0x08, 0xbc, 0x7b, 0xe8, 0x97, 0x46, 0x0a, 0x44, 0xca, 0x80, 0x48, 0x19, 0x90, 0x47, 0xe1, 0xb4,
	0x34, 0x9e, 0x13, 0x2a, 0x90, 0x23, 0x5b, 0xe4, 0x46, 0x76, 0x60, 0x26, 0x34, 0x1e, 0x8f, 0xee,
	0x13, 0x7a, 0xbb, 0x3c, 0xfe, 0xc4, 0x6e, 0xb3, 0xbf, 0xb2, 0x77, 0xff, 0x1b, 0x74, 0xcb, 0xfb,
	0xf7, 0x71, 0xfe, 0x8d, 0xfd, 0xa3, 0xb5, 0xcf, 0x52, 0xf1, 0x9e, 0xf1, 0xa3, 0x21, 0x8d, 0x2e,
	0x5f, 0x2f, 0x26, 0x02, 0xa7, 0x59, 0xe8, 0x47, 0x32, 0xa1, 0x91, 0xd4, 0xd1, 0x74, 0x2c, 0x52,
	0x1a, 0xc9, 0x14, 0x79, 0x8a, 0x52, 0x5f, 0x4d, 0x24, 0x5d, 0x5f, 0xeb, 0xb0, 0x6e, 0x8d, 0x6e,
	0xbf, 0x03, 0x00, 0x00, 0xff, 0xff, 0xd1, 0xc0, 0xfe, 0x0b, 0xea, 0x03, 0x00, 0x00,
}
