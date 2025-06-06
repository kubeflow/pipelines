// Copyright 2023 The Kubeflow Authors
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
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.20.3
// source: backend/api/v2beta1/healthz.proto

package go_client

import (
	context "context"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/genproto/googleapis/rpc/status"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetHealthzResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// TODO(gkcalat): redesign this service to return status
	// and move server configuration into a separate service
	// TODO(gkcalat): rename or deprecate v1beta1 HealthzService
	//
	// Returns if KFP in multi-user mode
	MultiUser bool `protobuf:"varint,3,opt,name=multi_user,json=multiUser,proto3" json:"multi_user,omitempty"`
	// Returns the pipeline storage type (database or kubernetes)
	PipelineStore string `protobuf:"bytes,4,opt,name=pipeline_store,json=pipelineStore,proto3" json:"pipeline_store,omitempty"`
}

func (x *GetHealthzResponse) Reset() {
	*x = GetHealthzResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_backend_api_v2beta1_healthz_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHealthzResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHealthzResponse) ProtoMessage() {}

func (x *GetHealthzResponse) ProtoReflect() protoreflect.Message {
	mi := &file_backend_api_v2beta1_healthz_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHealthzResponse.ProtoReflect.Descriptor instead.
func (*GetHealthzResponse) Descriptor() ([]byte, []int) {
	return file_backend_api_v2beta1_healthz_proto_rawDescGZIP(), []int{0}
}

func (x *GetHealthzResponse) GetMultiUser() bool {
	if x != nil {
		return x.MultiUser
	}
	return false
}

func (x *GetHealthzResponse) GetPipelineStore() string {
	if x != nil {
		return x.PipelineStore
	}
	return ""
}

var File_backend_api_v2beta1_healthz_proto protoreflect.FileDescriptor

var file_backend_api_v2beta1_healthz_proto_rawDesc = []byte{
	0x0a, 0x21, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x32,
	0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x7a, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x26, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69,
	0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x1a, 0x1c, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x2d, 0x67,
	0x65, 0x6e, 0x2d, 0x73, 0x77, 0x61, 0x67, 0x67, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x72, 0x70, 0x63,
	0x2f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5a, 0x0a,
	0x12, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x7a, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x5f, 0x75, 0x73, 0x65,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x55, 0x73,
	0x65, 0x72, 0x12, 0x25, 0x0a, 0x0e, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x5f, 0x73,
	0x74, 0x6f, 0x72, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x70, 0x69, 0x70, 0x65,
	0x6c, 0x69, 0x6e, 0x65, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x32, 0x91, 0x01, 0x0a, 0x0e, 0x48, 0x65,
	0x61, 0x6c, 0x74, 0x68, 0x7a, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x7f, 0x0a, 0x0a,
	0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x7a, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x1a, 0x3a, 0x2e, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x69,
	0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2e, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x48,
	0x65, 0x61, 0x6c, 0x74, 0x68, 0x7a, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1d,
	0x82, 0xd3, 0xe4, 0x93, 0x02, 0x17, 0x12, 0x15, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x76, 0x32,
	0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x68, 0x65, 0x61, 0x6c, 0x74, 0x68, 0x7a, 0x42, 0x98, 0x01,
	0x92, 0x41, 0x58, 0x2a, 0x02, 0x01, 0x02, 0x52, 0x23, 0x0a, 0x07, 0x64, 0x65, 0x66, 0x61, 0x75,
	0x6c, 0x74, 0x12, 0x18, 0x12, 0x16, 0x0a, 0x14, 0x1a, 0x12, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x72, 0x70, 0x63, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x5a, 0x1f, 0x0a, 0x1d,
	0x0a, 0x06, 0x42, 0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x13, 0x08, 0x02, 0x1a, 0x0d, 0x61, 0x75,
	0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x02, 0x62, 0x0c, 0x0a,
	0x0a, 0x0a, 0x06, 0x42, 0x65, 0x61, 0x72, 0x65, 0x72, 0x12, 0x00, 0x5a, 0x3b, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6b, 0x75, 0x62, 0x65, 0x66, 0x6c, 0x6f, 0x77,
	0x2f, 0x70, 0x69, 0x70, 0x65, 0x6c, 0x69, 0x6e, 0x65, 0x73, 0x2f, 0x62, 0x61, 0x63, 0x6b, 0x65,
	0x6e, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x32, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x67,
	0x6f, 0x5f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_backend_api_v2beta1_healthz_proto_rawDescOnce sync.Once
	file_backend_api_v2beta1_healthz_proto_rawDescData = file_backend_api_v2beta1_healthz_proto_rawDesc
)

func file_backend_api_v2beta1_healthz_proto_rawDescGZIP() []byte {
	file_backend_api_v2beta1_healthz_proto_rawDescOnce.Do(func() {
		file_backend_api_v2beta1_healthz_proto_rawDescData = protoimpl.X.CompressGZIP(file_backend_api_v2beta1_healthz_proto_rawDescData)
	})
	return file_backend_api_v2beta1_healthz_proto_rawDescData
}

var file_backend_api_v2beta1_healthz_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_backend_api_v2beta1_healthz_proto_goTypes = []interface{}{
	(*GetHealthzResponse)(nil), // 0: kubeflow.pipelines.backend.api.v2beta1.GetHealthzResponse
	(*emptypb.Empty)(nil),      // 1: google.protobuf.Empty
}
var file_backend_api_v2beta1_healthz_proto_depIdxs = []int32{
	1, // 0: kubeflow.pipelines.backend.api.v2beta1.HealthzService.GetHealthz:input_type -> google.protobuf.Empty
	0, // 1: kubeflow.pipelines.backend.api.v2beta1.HealthzService.GetHealthz:output_type -> kubeflow.pipelines.backend.api.v2beta1.GetHealthzResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_backend_api_v2beta1_healthz_proto_init() }
func file_backend_api_v2beta1_healthz_proto_init() {
	if File_backend_api_v2beta1_healthz_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_backend_api_v2beta1_healthz_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHealthzResponse); i {
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
			RawDescriptor: file_backend_api_v2beta1_healthz_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_backend_api_v2beta1_healthz_proto_goTypes,
		DependencyIndexes: file_backend_api_v2beta1_healthz_proto_depIdxs,
		MessageInfos:      file_backend_api_v2beta1_healthz_proto_msgTypes,
	}.Build()
	File_backend_api_v2beta1_healthz_proto = out.File
	file_backend_api_v2beta1_healthz_proto_rawDesc = nil
	file_backend_api_v2beta1_healthz_proto_goTypes = nil
	file_backend_api_v2beta1_healthz_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// HealthzServiceClient is the client API for HealthzService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type HealthzServiceClient interface {
	// Get healthz data.
	GetHealthz(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetHealthzResponse, error)
}

type healthzServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewHealthzServiceClient(cc grpc.ClientConnInterface) HealthzServiceClient {
	return &healthzServiceClient{cc}
}

func (c *healthzServiceClient) GetHealthz(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*GetHealthzResponse, error) {
	out := new(GetHealthzResponse)
	err := c.cc.Invoke(ctx, "/kubeflow.pipelines.backend.api.v2beta1.HealthzService/GetHealthz", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HealthzServiceServer is the server API for HealthzService service.
type HealthzServiceServer interface {
	// Get healthz data.
	GetHealthz(context.Context, *emptypb.Empty) (*GetHealthzResponse, error)
}

// UnimplementedHealthzServiceServer can be embedded to have forward compatible implementations.
type UnimplementedHealthzServiceServer struct {
}

func (*UnimplementedHealthzServiceServer) GetHealthz(context.Context, *emptypb.Empty) (*GetHealthzResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetHealthz not implemented")
}

func RegisterHealthzServiceServer(s *grpc.Server, srv HealthzServiceServer) {
	s.RegisterService(&_HealthzService_serviceDesc, srv)
}

func _HealthzService_GetHealthz_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HealthzServiceServer).GetHealthz(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kubeflow.pipelines.backend.api.v2beta1.HealthzService/GetHealthz",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HealthzServiceServer).GetHealthz(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

var _HealthzService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "kubeflow.pipelines.backend.api.v2beta1.HealthzService",
	HandlerType: (*HealthzServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetHealthz",
			Handler:    _HealthzService_GetHealthz_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/v2beta1/healthz.proto",
}
