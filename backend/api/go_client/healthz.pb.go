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
// source: backend/api/healthz.proto

package go_client

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	_ "github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type GetHealthzResponse struct {
	// Returns if KFP in multi-user mode
	MultiUser            bool     `protobuf:"varint,3,opt,name=multi_user,json=multiUser,proto3" json:"multi_user,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetHealthzResponse) Reset()         { *m = GetHealthzResponse{} }
func (m *GetHealthzResponse) String() string { return proto.CompactTextString(m) }
func (*GetHealthzResponse) ProtoMessage()    {}
func (*GetHealthzResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_499453e9dc64832b, []int{0}
}

func (m *GetHealthzResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetHealthzResponse.Unmarshal(m, b)
}
func (m *GetHealthzResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetHealthzResponse.Marshal(b, m, deterministic)
}
func (m *GetHealthzResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetHealthzResponse.Merge(m, src)
}
func (m *GetHealthzResponse) XXX_Size() int {
	return xxx_messageInfo_GetHealthzResponse.Size(m)
}
func (m *GetHealthzResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetHealthzResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetHealthzResponse proto.InternalMessageInfo

func (m *GetHealthzResponse) GetMultiUser() bool {
	if m != nil {
		return m.MultiUser
	}
	return false
}

func init() {
	proto.RegisterType((*GetHealthzResponse)(nil), "api.GetHealthzResponse")
}

func init() { proto.RegisterFile("backend/api/healthz.proto", fileDescriptor_499453e9dc64832b) }

var fileDescriptor_499453e9dc64832b = []byte{
	// 377 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x91, 0xcd, 0xae, 0x12, 0x31,
	0x14, 0x80, 0x05, 0x12, 0xc4, 0x2a, 0x1a, 0x6b, 0x14, 0x1d, 0x21, 0x10, 0xe2, 0xc2, 0x85, 0xcc,
	0x04, 0x79, 0x02, 0x49, 0x8c, 0x6e, 0xdc, 0x40, 0xdc, 0x10, 0x93, 0x49, 0xa7, 0x9c, 0x99, 0x69,
	0x98, 0x69, 0x9b, 0xf6, 0x14, 0x22, 0x4b, 0x13, 0x5f, 0x40, 0x1f, 0xed, 0xbe, 0xc2, 0x7d, 0x90,
	0x1b, 0x3a, 0x03, 0x77, 0x72, 0xef, 0x5d, 0x35, 0xed, 0xf9, 0xce, 0x4f, 0xbf, 0x43, 0xde, 0x25,
	0x8c, 0xef, 0x40, 0x6e, 0x23, 0xa6, 0x45, 0x94, 0x03, 0x2b, 0x30, 0x3f, 0x86, 0xda, 0x28, 0x54,
	0xb4, 0xc3, 0xb4, 0x08, 0x86, 0x99, 0x52, 0x59, 0x01, 0x3e, 0xcc, 0xa4, 0x54, 0xc8, 0x50, 0x28,
	0x69, 0x2b, 0x24, 0x18, 0xd7, 0x51, 0x7f, 0x4b, 0x5c, 0x1a, 0xa1, 0x28, 0xc1, 0x22, 0x2b, 0x75,
	0x0d, 0xbc, 0xbf, 0x0b, 0x40, 0xa9, 0xf1, 0xf7, 0x39, 0xbb, 0xd9, 0x5b, 0x0b, 0x0d, 0x85, 0x90,
	0x10, 0x5b, 0x0d, 0xbc, 0x06, 0x3e, 0x34, 0x01, 0x03, 0x56, 0x39, 0xc3, 0x21, 0x36, 0x90, 0x82,
	0x01, 0xc9, 0xa1, 0xa6, 0x3e, 0xf9, 0x83, 0xcf, 0x32, 0x90, 0x33, 0x7b, 0x60, 0x59, 0x06, 0x26,
	0x52, 0xda, 0x8f, 0xf9, 0xc0, 0xc8, 0x83, 0x66, 0x4d, 0x30, 0x46, 0x99, 0x2a, 0x30, 0x5d, 0x10,
	0xfa, 0x0d, 0xf0, 0x7b, 0xa5, 0x60, 0x05, 0x56, 0x2b, 0x69, 0x81, 0x8e, 0x08, 0x29, 0x5d, 0x81,
	0x22, 0x76, 0x16, 0xcc, 0xdb, 0xce, 0xa4, 0xf5, 0xb1, 0xb7, 0x7a, 0xe2, 0x5f, 0x7e, 0x5a, 0x30,
	0x9f, 0x25, 0x79, 0x5e, 0x67, 0xac, 0xc1, 0xec, 0x05, 0x07, 0xfa, 0x8b, 0x90, 0xdb, 0x32, 0xf4,
	0x4d, 0x58, 0x09, 0x08, 0xcf, 0x02, 0xc2, 0xaf, 0x27, 0x01, 0xc1, 0x20, 0x64, 0x5a, 0x84, 0xf7,
	0xfb, 0x4d, 0x47, 0x7f, 0xae, 0xae, 0xff, 0xb7, 0x07, 0xf4, 0xf5, 0x69, 0x3e, 0x1b, 0xed, 0xe7,
	0x09, 0x20, 0x9b, 0x9f, 0x37, 0xb3, 0xfc, 0xdb, 0xfa, 0xf7, 0xe5, 0xc7, 0x6a, 0x48, 0x1e, 0x6f,
	0x21, 0x65, 0xae, 0x40, 0xfa, 0x92, 0xbe, 0x20, 0xfd, 0xe0, 0xa9, 0x2f, 0xb7, 0x46, 0x86, 0xce,
	0x6e, 0xc6, 0x64, 0x44, 0xba, 0x4b, 0x60, 0x06, 0x0c, 0x7d, 0xd5, 0x6b, 0x07, 0x7d, 0xe6, 0x30,
	0x57, 0x46, 0x1c, 0xbd, 0x87, 0x49, 0x3b, 0x79, 0x46, 0xc8, 0x05, 0x78, 0xb4, 0x59, 0x64, 0x02,
	0x73, 0x97, 0x84, 0x5c, 0x95, 0xd1, 0xce, 0x25, 0x90, 0x16, 0xea, 0x70, 0xd9, 0x86, 0x8d, 0x9a,
	0xba, 0x32, 0x15, 0xf3, 0x42, 0x80, 0xc4, 0xa4, 0xeb, 0xff, 0xb3, 0xb8, 0x09, 0x00, 0x00, 0xff,
	0xff, 0xd1, 0xc1, 0x9c, 0x02, 0x3f, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// HealthzServiceClient is the client API for HealthzService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type HealthzServiceClient interface {
	// Get healthz data.
	GetHealthz(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetHealthzResponse, error)
}

type healthzServiceClient struct {
	cc *grpc.ClientConn
}

func NewHealthzServiceClient(cc *grpc.ClientConn) HealthzServiceClient {
	return &healthzServiceClient{cc}
}

func (c *healthzServiceClient) GetHealthz(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*GetHealthzResponse, error) {
	out := new(GetHealthzResponse)
	err := c.cc.Invoke(ctx, "/api.HealthzService/GetHealthz", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HealthzServiceServer is the server API for HealthzService service.
type HealthzServiceServer interface {
	// Get healthz data.
	GetHealthz(context.Context, *empty.Empty) (*GetHealthzResponse, error)
}

// UnimplementedHealthzServiceServer can be embedded to have forward compatible implementations.
type UnimplementedHealthzServiceServer struct {
}

func (*UnimplementedHealthzServiceServer) GetHealthz(ctx context.Context, req *empty.Empty) (*GetHealthzResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHealthz not implemented")
}

func RegisterHealthzServiceServer(s *grpc.Server, srv HealthzServiceServer) {
	s.RegisterService(&_HealthzService_serviceDesc, srv)
}

func _HealthzService_GetHealthz_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HealthzServiceServer).GetHealthz(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.HealthzService/GetHealthz",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HealthzServiceServer).GetHealthz(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _HealthzService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.HealthzService",
	HandlerType: (*HealthzServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetHealthz",
			Handler:    _HealthzService_GetHealthz_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/healthz.proto",
}
