// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// source: backend/api/report.proto

package go_client

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/ptypes/empty"
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

type ReportWorkflowRequest struct {
	// Workflow is a workflow custom resource marshalled into a json string.
	Workflow             string   `protobuf:"bytes,1,opt,name=workflow,proto3" json:"workflow,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReportWorkflowRequest) Reset()         { *m = ReportWorkflowRequest{} }
func (m *ReportWorkflowRequest) String() string { return proto.CompactTextString(m) }
func (*ReportWorkflowRequest) ProtoMessage()    {}
func (*ReportWorkflowRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_cf464d903a9c793e, []int{0}
}

func (m *ReportWorkflowRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ReportWorkflowRequest.Unmarshal(m, b)
}
func (m *ReportWorkflowRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ReportWorkflowRequest.Marshal(b, m, deterministic)
}
func (m *ReportWorkflowRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReportWorkflowRequest.Merge(m, src)
}
func (m *ReportWorkflowRequest) XXX_Size() int {
	return xxx_messageInfo_ReportWorkflowRequest.Size(m)
}
func (m *ReportWorkflowRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ReportWorkflowRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ReportWorkflowRequest proto.InternalMessageInfo

func (m *ReportWorkflowRequest) GetWorkflow() string {
	if m != nil {
		return m.Workflow
	}
	return ""
}

type ReportScheduledWorkflowRequest struct {
	// ScheduledWorkflow a ScheduledWorkflow resource marshalled into a json string.
	ScheduledWorkflow    string   `protobuf:"bytes,1,opt,name=scheduled_workflow,json=scheduledWorkflow,proto3" json:"scheduled_workflow,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReportScheduledWorkflowRequest) Reset()         { *m = ReportScheduledWorkflowRequest{} }
func (m *ReportScheduledWorkflowRequest) String() string { return proto.CompactTextString(m) }
func (*ReportScheduledWorkflowRequest) ProtoMessage()    {}
func (*ReportScheduledWorkflowRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_cf464d903a9c793e, []int{1}
}

func (m *ReportScheduledWorkflowRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ReportScheduledWorkflowRequest.Unmarshal(m, b)
}
func (m *ReportScheduledWorkflowRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ReportScheduledWorkflowRequest.Marshal(b, m, deterministic)
}
func (m *ReportScheduledWorkflowRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReportScheduledWorkflowRequest.Merge(m, src)
}
func (m *ReportScheduledWorkflowRequest) XXX_Size() int {
	return xxx_messageInfo_ReportScheduledWorkflowRequest.Size(m)
}
func (m *ReportScheduledWorkflowRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ReportScheduledWorkflowRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ReportScheduledWorkflowRequest proto.InternalMessageInfo

func (m *ReportScheduledWorkflowRequest) GetScheduledWorkflow() string {
	if m != nil {
		return m.ScheduledWorkflow
	}
	return ""
}

func init() {
	proto.RegisterType((*ReportWorkflowRequest)(nil), "api.ReportWorkflowRequest")
	proto.RegisterType((*ReportScheduledWorkflowRequest)(nil), "api.ReportScheduledWorkflowRequest")
}

func init() { proto.RegisterFile("backend/api/report.proto", fileDescriptor_cf464d903a9c793e) }

var fileDescriptor_cf464d903a9c793e = []byte{
	// 310 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0xcf, 0x4a, 0xc3, 0x40,
	0x10, 0x87, 0x69, 0x05, 0xd1, 0x05, 0x05, 0x17, 0xb4, 0x25, 0x8a, 0x94, 0xf4, 0xa2, 0x07, 0x77,
	0xa9, 0x45, 0x0f, 0xe2, 0x49, 0xf0, 0x2c, 0xd4, 0x83, 0xe0, 0xa5, 0xec, 0xa6, 0xd3, 0x74, 0x69,
	0xba, 0xb3, 0x66, 0x37, 0x2d, 0x5e, 0x7d, 0x05, 0x05, 0x1f, 0xcc, 0x57, 0xf0, 0x41, 0x24, 0x7f,
	0x6d, 0x43, 0x73, 0x9c, 0xf9, 0x65, 0x26, 0xdf, 0x37, 0x2c, 0xe9, 0x4a, 0x11, 0xcc, 0x41, 0x4f,
	0xb8, 0x30, 0x8a, 0xc7, 0x60, 0x30, 0x76, 0xcc, 0xc4, 0xe8, 0x90, 0xee, 0x08, 0xa3, 0xbc, 0xb3,
	0x10, 0x31, 0x8c, 0x20, 0x4b, 0x85, 0xd6, 0xe8, 0x84, 0x53, 0xa8, 0x6d, 0xfe, 0x89, 0x77, 0x5a,
	0xa4, 0x59, 0x25, 0x93, 0x29, 0x87, 0x85, 0x71, 0xef, 0x79, 0xe8, 0x0f, 0xc9, 0xf1, 0x28, 0xdb,
	0xf7, 0x82, 0xf1, 0x7c, 0x1a, 0xe1, 0x6a, 0x04, 0x6f, 0x09, 0x58, 0x47, 0x3d, 0xb2, 0xb7, 0x2a,
	0x5a, 0xdd, 0x56, 0xaf, 0x75, 0xb1, 0x3f, 0xaa, 0x6a, 0xff, 0x89, 0x9c, 0xe7, 0x43, 0xcf, 0xc1,
	0x0c, 0x26, 0x49, 0x04, 0x93, 0xfa, 0xf4, 0x15, 0xa1, 0xb6, 0xcc, 0xc6, 0xb5, 0x3d, 0x47, 0xb6,
	0x3e, 0x75, 0xfd, 0xdd, 0x26, 0x07, 0xc5, 0x46, 0x88, 0x97, 0x2a, 0x00, 0x8a, 0xe4, 0x70, 0x93,
	0x8b, 0x7a, 0x4c, 0x18, 0xc5, 0xb6, 0xc2, 0x7a, 0x27, 0x2c, 0x77, 0x64, 0xa5, 0x23, 0x7b, 0x4c,
	0x1d, 0xfd, 0xcb, 0x8f, 0x9f, 0xdf, 0xcf, 0x76, 0xdf, 0xef, 0xa4, 0xa7, 0xb1, 0x7c, 0x39, 0x90,
	0xe0, 0xc4, 0x80, 0x97, 0x40, 0xf6, 0xae, 0x72, 0xa2, 0x5f, 0x2d, 0xd2, 0x69, 0x90, 0xa2, 0xfd,
	0xb5, 0x5f, 0x37, 0x29, 0x37, 0x32, 0xdc, 0x67, 0x0c, 0xb7, 0x7e, 0x6f, 0x93, 0xa1, 0x3a, 0xc2,
	0x3f, 0xcc, 0x96, 0x93, 0x3d, 0xdc, 0xbc, 0x0e, 0x43, 0xe5, 0x66, 0x89, 0x64, 0x01, 0x2e, 0xf8,
	0x3c, 0x91, 0x90, 0xb6, 0xb9, 0x51, 0x06, 0x22, 0xa5, 0xc1, 0xf2, 0xf5, 0x97, 0x11, 0xe2, 0x38,
	0x88, 0x14, 0x68, 0x27, 0x77, 0x33, 0x88, 0xe1, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x12, 0xe5,
	0x5d, 0x89, 0x39, 0x02, 0x00, 0x00,
}
