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

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package go_client

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// ReportServiceClient is the client API for ReportService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ReportServiceClient interface {
	ReportWorkflow(ctx context.Context, in *ReportWorkflowRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	ReportScheduledWorkflow(ctx context.Context, in *ReportScheduledWorkflowRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type reportServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewReportServiceClient(cc grpc.ClientConnInterface) ReportServiceClient {
	return &reportServiceClient{cc}
}

func (c *reportServiceClient) ReportWorkflow(ctx context.Context, in *ReportWorkflowRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.ReportService/ReportWorkflow", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *reportServiceClient) ReportScheduledWorkflow(ctx context.Context, in *ReportScheduledWorkflowRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.ReportService/ReportScheduledWorkflow", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReportServiceServer is the server API for ReportService service.
// All implementations must embed UnimplementedReportServiceServer
// for forward compatibility
type ReportServiceServer interface {
	ReportWorkflow(context.Context, *ReportWorkflowRequest) (*empty.Empty, error)
	ReportScheduledWorkflow(context.Context, *ReportScheduledWorkflowRequest) (*empty.Empty, error)
	mustEmbedUnimplementedReportServiceServer()
}

// UnimplementedReportServiceServer must be embedded to have forward compatible implementations.
type UnimplementedReportServiceServer struct {
}

func (UnimplementedReportServiceServer) ReportWorkflow(context.Context, *ReportWorkflowRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportWorkflow not implemented")
}
func (UnimplementedReportServiceServer) ReportScheduledWorkflow(context.Context, *ReportScheduledWorkflowRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportScheduledWorkflow not implemented")
}
func (UnimplementedReportServiceServer) mustEmbedUnimplementedReportServiceServer() {}

// UnsafeReportServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ReportServiceServer will
// result in compilation errors.
type UnsafeReportServiceServer interface {
	mustEmbedUnimplementedReportServiceServer()
}

func RegisterReportServiceServer(s grpc.ServiceRegistrar, srv ReportServiceServer) {
	s.RegisterService(&ReportService_ServiceDesc, srv)
}

func _ReportService_ReportWorkflow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportWorkflowRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReportServiceServer).ReportWorkflow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ReportService/ReportWorkflow",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReportServiceServer).ReportWorkflow(ctx, req.(*ReportWorkflowRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ReportService_ReportScheduledWorkflow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportScheduledWorkflowRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReportServiceServer).ReportScheduledWorkflow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.ReportService/ReportScheduledWorkflow",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReportServiceServer).ReportScheduledWorkflow(ctx, req.(*ReportScheduledWorkflowRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ReportService_ServiceDesc is the grpc.ServiceDesc for ReportService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ReportService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.ReportService",
	HandlerType: (*ReportServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ReportWorkflow",
			Handler:    _ReportService_ReportWorkflow_Handler,
		},
		{
			MethodName: "ReportScheduledWorkflow",
			Handler:    _ReportService_ReportScheduledWorkflow_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/report.proto",
}
