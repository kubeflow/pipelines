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

// RunServiceClient is the client API for RunService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RunServiceClient interface {
	// Creates a new run.
	CreateRun(ctx context.Context, in *CreateRunRequest, opts ...grpc.CallOption) (*RunDetail, error)
	// Finds a specific run by ID.
	GetRun(ctx context.Context, in *GetRunRequest, opts ...grpc.CallOption) (*RunDetail, error)
	// Finds all runs.
	ListRuns(ctx context.Context, in *ListRunsRequest, opts ...grpc.CallOption) (*ListRunsResponse, error)
	// Archives a run.
	ArchiveRun(ctx context.Context, in *ArchiveRunRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Restores an archived run.
	UnarchiveRun(ctx context.Context, in *UnarchiveRunRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Deletes a run.
	DeleteRun(ctx context.Context, in *DeleteRunRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// ReportRunMetrics reports metrics of a run. Each metric is reported in its
	// own transaction, so this API accepts partial failures. Metric can be
	// uniquely identified by (run_id, node_id, name). Duplicate reporting will be
	// ignored by the API. First reporting wins.
	ReportRunMetrics(ctx context.Context, in *ReportRunMetricsRequest, opts ...grpc.CallOption) (*ReportRunMetricsResponse, error)
	// Finds a run's artifact data.
	ReadArtifact(ctx context.Context, in *ReadArtifactRequest, opts ...grpc.CallOption) (*ReadArtifactResponse, error)
	// Terminates an active run.
	TerminateRun(ctx context.Context, in *TerminateRunRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Re-initiates a failed or terminated run.
	RetryRun(ctx context.Context, in *RetryRunRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type runServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRunServiceClient(cc grpc.ClientConnInterface) RunServiceClient {
	return &runServiceClient{cc}
}

func (c *runServiceClient) CreateRun(ctx context.Context, in *CreateRunRequest, opts ...grpc.CallOption) (*RunDetail, error) {
	out := new(RunDetail)
	err := c.cc.Invoke(ctx, "/api.RunService/CreateRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) GetRun(ctx context.Context, in *GetRunRequest, opts ...grpc.CallOption) (*RunDetail, error) {
	out := new(RunDetail)
	err := c.cc.Invoke(ctx, "/api.RunService/GetRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) ListRuns(ctx context.Context, in *ListRunsRequest, opts ...grpc.CallOption) (*ListRunsResponse, error) {
	out := new(ListRunsResponse)
	err := c.cc.Invoke(ctx, "/api.RunService/ListRuns", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) ArchiveRun(ctx context.Context, in *ArchiveRunRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.RunService/ArchiveRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) UnarchiveRun(ctx context.Context, in *UnarchiveRunRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.RunService/UnarchiveRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) DeleteRun(ctx context.Context, in *DeleteRunRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.RunService/DeleteRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) ReportRunMetrics(ctx context.Context, in *ReportRunMetricsRequest, opts ...grpc.CallOption) (*ReportRunMetricsResponse, error) {
	out := new(ReportRunMetricsResponse)
	err := c.cc.Invoke(ctx, "/api.RunService/ReportRunMetrics", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) ReadArtifact(ctx context.Context, in *ReadArtifactRequest, opts ...grpc.CallOption) (*ReadArtifactResponse, error) {
	out := new(ReadArtifactResponse)
	err := c.cc.Invoke(ctx, "/api.RunService/ReadArtifact", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) TerminateRun(ctx context.Context, in *TerminateRunRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.RunService/TerminateRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *runServiceClient) RetryRun(ctx context.Context, in *RetryRunRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.RunService/RetryRun", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RunServiceServer is the server API for RunService service.
// All implementations must embed UnimplementedRunServiceServer
// for forward compatibility
type RunServiceServer interface {
	// Creates a new run.
	CreateRun(context.Context, *CreateRunRequest) (*RunDetail, error)
	// Finds a specific run by ID.
	GetRun(context.Context, *GetRunRequest) (*RunDetail, error)
	// Finds all runs.
	ListRuns(context.Context, *ListRunsRequest) (*ListRunsResponse, error)
	// Archives a run.
	ArchiveRun(context.Context, *ArchiveRunRequest) (*empty.Empty, error)
	// Restores an archived run.
	UnarchiveRun(context.Context, *UnarchiveRunRequest) (*empty.Empty, error)
	// Deletes a run.
	DeleteRun(context.Context, *DeleteRunRequest) (*empty.Empty, error)
	// ReportRunMetrics reports metrics of a run. Each metric is reported in its
	// own transaction, so this API accepts partial failures. Metric can be
	// uniquely identified by (run_id, node_id, name). Duplicate reporting will be
	// ignored by the API. First reporting wins.
	ReportRunMetrics(context.Context, *ReportRunMetricsRequest) (*ReportRunMetricsResponse, error)
	// Finds a run's artifact data.
	ReadArtifact(context.Context, *ReadArtifactRequest) (*ReadArtifactResponse, error)
	// Terminates an active run.
	TerminateRun(context.Context, *TerminateRunRequest) (*empty.Empty, error)
	// Re-initiates a failed or terminated run.
	RetryRun(context.Context, *RetryRunRequest) (*empty.Empty, error)
	mustEmbedUnimplementedRunServiceServer()
}

// UnimplementedRunServiceServer must be embedded to have forward compatible implementations.
type UnimplementedRunServiceServer struct {
}

func (UnimplementedRunServiceServer) CreateRun(context.Context, *CreateRunRequest) (*RunDetail, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRun not implemented")
}
func (UnimplementedRunServiceServer) GetRun(context.Context, *GetRunRequest) (*RunDetail, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRun not implemented")
}
func (UnimplementedRunServiceServer) ListRuns(context.Context, *ListRunsRequest) (*ListRunsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRuns not implemented")
}
func (UnimplementedRunServiceServer) ArchiveRun(context.Context, *ArchiveRunRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ArchiveRun not implemented")
}
func (UnimplementedRunServiceServer) UnarchiveRun(context.Context, *UnarchiveRunRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnarchiveRun not implemented")
}
func (UnimplementedRunServiceServer) DeleteRun(context.Context, *DeleteRunRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRun not implemented")
}
func (UnimplementedRunServiceServer) ReportRunMetrics(context.Context, *ReportRunMetricsRequest) (*ReportRunMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReportRunMetrics not implemented")
}
func (UnimplementedRunServiceServer) ReadArtifact(context.Context, *ReadArtifactRequest) (*ReadArtifactResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadArtifact not implemented")
}
func (UnimplementedRunServiceServer) TerminateRun(context.Context, *TerminateRunRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TerminateRun not implemented")
}
func (UnimplementedRunServiceServer) RetryRun(context.Context, *RetryRunRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetryRun not implemented")
}
func (UnimplementedRunServiceServer) mustEmbedUnimplementedRunServiceServer() {}

// UnsafeRunServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RunServiceServer will
// result in compilation errors.
type UnsafeRunServiceServer interface {
	mustEmbedUnimplementedRunServiceServer()
}

func RegisterRunServiceServer(s grpc.ServiceRegistrar, srv RunServiceServer) {
	s.RegisterService(&RunService_ServiceDesc, srv)
}

func _RunService_CreateRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).CreateRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/CreateRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).CreateRun(ctx, req.(*CreateRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_GetRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).GetRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/GetRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).GetRun(ctx, req.(*GetRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_ListRuns_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRunsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).ListRuns(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/ListRuns",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).ListRuns(ctx, req.(*ListRunsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_ArchiveRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ArchiveRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).ArchiveRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/ArchiveRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).ArchiveRun(ctx, req.(*ArchiveRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_UnarchiveRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnarchiveRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).UnarchiveRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/UnarchiveRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).UnarchiveRun(ctx, req.(*UnarchiveRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_DeleteRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).DeleteRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/DeleteRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).DeleteRun(ctx, req.(*DeleteRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_ReportRunMetrics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReportRunMetricsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).ReportRunMetrics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/ReportRunMetrics",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).ReportRunMetrics(ctx, req.(*ReportRunMetricsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_ReadArtifact_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadArtifactRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).ReadArtifact(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/ReadArtifact",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).ReadArtifact(ctx, req.(*ReadArtifactRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_TerminateRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TerminateRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).TerminateRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/TerminateRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).TerminateRun(ctx, req.(*TerminateRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RunService_RetryRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RetryRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RunServiceServer).RetryRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.RunService/RetryRun",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RunServiceServer).RetryRun(ctx, req.(*RetryRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RunService_ServiceDesc is the grpc.ServiceDesc for RunService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RunService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.RunService",
	HandlerType: (*RunServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateRun",
			Handler:    _RunService_CreateRun_Handler,
		},
		{
			MethodName: "GetRun",
			Handler:    _RunService_GetRun_Handler,
		},
		{
			MethodName: "ListRuns",
			Handler:    _RunService_ListRuns_Handler,
		},
		{
			MethodName: "ArchiveRun",
			Handler:    _RunService_ArchiveRun_Handler,
		},
		{
			MethodName: "UnarchiveRun",
			Handler:    _RunService_UnarchiveRun_Handler,
		},
		{
			MethodName: "DeleteRun",
			Handler:    _RunService_DeleteRun_Handler,
		},
		{
			MethodName: "ReportRunMetrics",
			Handler:    _RunService_ReportRunMetrics_Handler,
		},
		{
			MethodName: "ReadArtifact",
			Handler:    _RunService_ReadArtifact_Handler,
		},
		{
			MethodName: "TerminateRun",
			Handler:    _RunService_TerminateRun_Handler,
		},
		{
			MethodName: "RetryRun",
			Handler:    _RunService_RetryRun_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/run.proto",
}
