// Code generated by protoc-gen-go. DO NOT EDIT.
// source: package.proto

package api

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import empty "github.com/golang/protobuf/ptypes/empty"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"
import _ "google.golang.org/genproto/googleapis/api/annotations"

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

type GetPackageRequest struct {
	Id                   uint32   `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetPackageRequest) Reset()         { *m = GetPackageRequest{} }
func (m *GetPackageRequest) String() string { return proto.CompactTextString(m) }
func (*GetPackageRequest) ProtoMessage()    {}
func (*GetPackageRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{0}
}
func (m *GetPackageRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetPackageRequest.Unmarshal(m, b)
}
func (m *GetPackageRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetPackageRequest.Marshal(b, m, deterministic)
}
func (dst *GetPackageRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetPackageRequest.Merge(dst, src)
}
func (m *GetPackageRequest) XXX_Size() int {
	return xxx_messageInfo_GetPackageRequest.Size(m)
}
func (m *GetPackageRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetPackageRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetPackageRequest proto.InternalMessageInfo

func (m *GetPackageRequest) GetId() uint32 {
	if m != nil {
		return m.Id
	}
	return 0
}

type ListPackagesRequest struct {
	PageToken            string   `protobuf:"bytes,1,opt,name=page_token,json=pageToken" json:"page_token,omitempty"`
	PageSize             int32    `protobuf:"varint,2,opt,name=page_size,json=pageSize" json:"page_size,omitempty"`
	SortBy               string   `protobuf:"bytes,3,opt,name=sort_by,json=sortBy" json:"sort_by,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListPackagesRequest) Reset()         { *m = ListPackagesRequest{} }
func (m *ListPackagesRequest) String() string { return proto.CompactTextString(m) }
func (*ListPackagesRequest) ProtoMessage()    {}
func (*ListPackagesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{1}
}
func (m *ListPackagesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListPackagesRequest.Unmarshal(m, b)
}
func (m *ListPackagesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListPackagesRequest.Marshal(b, m, deterministic)
}
func (dst *ListPackagesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListPackagesRequest.Merge(dst, src)
}
func (m *ListPackagesRequest) XXX_Size() int {
	return xxx_messageInfo_ListPackagesRequest.Size(m)
}
func (m *ListPackagesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListPackagesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListPackagesRequest proto.InternalMessageInfo

func (m *ListPackagesRequest) GetPageToken() string {
	if m != nil {
		return m.PageToken
	}
	return ""
}

func (m *ListPackagesRequest) GetPageSize() int32 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *ListPackagesRequest) GetSortBy() string {
	if m != nil {
		return m.SortBy
	}
	return ""
}

type ListPackagesResponse struct {
	Packages             []*Package `protobuf:"bytes,1,rep,name=packages" json:"packages,omitempty"`
	NextPageToken        string     `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken" json:"next_page_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *ListPackagesResponse) Reset()         { *m = ListPackagesResponse{} }
func (m *ListPackagesResponse) String() string { return proto.CompactTextString(m) }
func (*ListPackagesResponse) ProtoMessage()    {}
func (*ListPackagesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{2}
}
func (m *ListPackagesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListPackagesResponse.Unmarshal(m, b)
}
func (m *ListPackagesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListPackagesResponse.Marshal(b, m, deterministic)
}
func (dst *ListPackagesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListPackagesResponse.Merge(dst, src)
}
func (m *ListPackagesResponse) XXX_Size() int {
	return xxx_messageInfo_ListPackagesResponse.Size(m)
}
func (m *ListPackagesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListPackagesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListPackagesResponse proto.InternalMessageInfo

func (m *ListPackagesResponse) GetPackages() []*Package {
	if m != nil {
		return m.Packages
	}
	return nil
}

func (m *ListPackagesResponse) GetNextPageToken() string {
	if m != nil {
		return m.NextPageToken
	}
	return ""
}

type DeletePackageRequest struct {
	Id                   uint32   `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeletePackageRequest) Reset()         { *m = DeletePackageRequest{} }
func (m *DeletePackageRequest) String() string { return proto.CompactTextString(m) }
func (*DeletePackageRequest) ProtoMessage()    {}
func (*DeletePackageRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{3}
}
func (m *DeletePackageRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeletePackageRequest.Unmarshal(m, b)
}
func (m *DeletePackageRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeletePackageRequest.Marshal(b, m, deterministic)
}
func (dst *DeletePackageRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeletePackageRequest.Merge(dst, src)
}
func (m *DeletePackageRequest) XXX_Size() int {
	return xxx_messageInfo_DeletePackageRequest.Size(m)
}
func (m *DeletePackageRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeletePackageRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeletePackageRequest proto.InternalMessageInfo

func (m *DeletePackageRequest) GetId() uint32 {
	if m != nil {
		return m.Id
	}
	return 0
}

type GetTemplateRequest struct {
	Id                   uint32   `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetTemplateRequest) Reset()         { *m = GetTemplateRequest{} }
func (m *GetTemplateRequest) String() string { return proto.CompactTextString(m) }
func (*GetTemplateRequest) ProtoMessage()    {}
func (*GetTemplateRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{4}
}
func (m *GetTemplateRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetTemplateRequest.Unmarshal(m, b)
}
func (m *GetTemplateRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetTemplateRequest.Marshal(b, m, deterministic)
}
func (dst *GetTemplateRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetTemplateRequest.Merge(dst, src)
}
func (m *GetTemplateRequest) XXX_Size() int {
	return xxx_messageInfo_GetTemplateRequest.Size(m)
}
func (m *GetTemplateRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetTemplateRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetTemplateRequest proto.InternalMessageInfo

func (m *GetTemplateRequest) GetId() uint32 {
	if m != nil {
		return m.Id
	}
	return 0
}

type GetTemplateResponse struct {
	Template             string   `protobuf:"bytes,1,opt,name=template" json:"template,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetTemplateResponse) Reset()         { *m = GetTemplateResponse{} }
func (m *GetTemplateResponse) String() string { return proto.CompactTextString(m) }
func (*GetTemplateResponse) ProtoMessage()    {}
func (*GetTemplateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{5}
}
func (m *GetTemplateResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetTemplateResponse.Unmarshal(m, b)
}
func (m *GetTemplateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetTemplateResponse.Marshal(b, m, deterministic)
}
func (dst *GetTemplateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetTemplateResponse.Merge(dst, src)
}
func (m *GetTemplateResponse) XXX_Size() int {
	return xxx_messageInfo_GetTemplateResponse.Size(m)
}
func (m *GetTemplateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetTemplateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetTemplateResponse proto.InternalMessageInfo

func (m *GetTemplateResponse) GetTemplate() string {
	if m != nil {
		return m.Template
	}
	return ""
}

type Package struct {
	Id                   uint32               `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	CreatedAt            *timestamp.Timestamp `protobuf:"bytes,2,opt,name=created_at,json=createdAt" json:"created_at,omitempty"`
	Name                 string               `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
	Description          string               `protobuf:"bytes,4,opt,name=description" json:"description,omitempty"`
	Parameters           []*Parameter         `protobuf:"bytes,5,rep,name=parameters" json:"parameters,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Package) Reset()         { *m = Package{} }
func (m *Package) String() string { return proto.CompactTextString(m) }
func (*Package) ProtoMessage()    {}
func (*Package) Descriptor() ([]byte, []int) {
	return fileDescriptor_package_cce06a887bdfde92, []int{6}
}
func (m *Package) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Package.Unmarshal(m, b)
}
func (m *Package) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Package.Marshal(b, m, deterministic)
}
func (dst *Package) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Package.Merge(dst, src)
}
func (m *Package) XXX_Size() int {
	return xxx_messageInfo_Package.Size(m)
}
func (m *Package) XXX_DiscardUnknown() {
	xxx_messageInfo_Package.DiscardUnknown(m)
}

var xxx_messageInfo_Package proto.InternalMessageInfo

func (m *Package) GetId() uint32 {
	if m != nil {
		return m.Id
	}
	return 0
}

func (m *Package) GetCreatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAt
	}
	return nil
}

func (m *Package) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Package) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *Package) GetParameters() []*Parameter {
	if m != nil {
		return m.Parameters
	}
	return nil
}

func init() {
	proto.RegisterType((*GetPackageRequest)(nil), "api.GetPackageRequest")
	proto.RegisterType((*ListPackagesRequest)(nil), "api.ListPackagesRequest")
	proto.RegisterType((*ListPackagesResponse)(nil), "api.ListPackagesResponse")
	proto.RegisterType((*DeletePackageRequest)(nil), "api.DeletePackageRequest")
	proto.RegisterType((*GetTemplateRequest)(nil), "api.GetTemplateRequest")
	proto.RegisterType((*GetTemplateResponse)(nil), "api.GetTemplateResponse")
	proto.RegisterType((*Package)(nil), "api.Package")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for PackageService service

type PackageServiceClient interface {
	GetPackage(ctx context.Context, in *GetPackageRequest, opts ...grpc.CallOption) (*Package, error)
	ListPackages(ctx context.Context, in *ListPackagesRequest, opts ...grpc.CallOption) (*ListPackagesResponse, error)
	DeletePackage(ctx context.Context, in *DeletePackageRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	GetTemplate(ctx context.Context, in *GetTemplateRequest, opts ...grpc.CallOption) (*GetTemplateResponse, error)
}

type packageServiceClient struct {
	cc *grpc.ClientConn
}

func NewPackageServiceClient(cc *grpc.ClientConn) PackageServiceClient {
	return &packageServiceClient{cc}
}

func (c *packageServiceClient) GetPackage(ctx context.Context, in *GetPackageRequest, opts ...grpc.CallOption) (*Package, error) {
	out := new(Package)
	err := grpc.Invoke(ctx, "/api.PackageService/GetPackage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *packageServiceClient) ListPackages(ctx context.Context, in *ListPackagesRequest, opts ...grpc.CallOption) (*ListPackagesResponse, error) {
	out := new(ListPackagesResponse)
	err := grpc.Invoke(ctx, "/api.PackageService/ListPackages", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *packageServiceClient) DeletePackage(ctx context.Context, in *DeletePackageRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := grpc.Invoke(ctx, "/api.PackageService/DeletePackage", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *packageServiceClient) GetTemplate(ctx context.Context, in *GetTemplateRequest, opts ...grpc.CallOption) (*GetTemplateResponse, error) {
	out := new(GetTemplateResponse)
	err := grpc.Invoke(ctx, "/api.PackageService/GetTemplate", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for PackageService service

type PackageServiceServer interface {
	GetPackage(context.Context, *GetPackageRequest) (*Package, error)
	ListPackages(context.Context, *ListPackagesRequest) (*ListPackagesResponse, error)
	DeletePackage(context.Context, *DeletePackageRequest) (*empty.Empty, error)
	GetTemplate(context.Context, *GetTemplateRequest) (*GetTemplateResponse, error)
}

func RegisterPackageServiceServer(s *grpc.Server, srv PackageServiceServer) {
	s.RegisterService(&_PackageService_serviceDesc, srv)
}

func _PackageService_GetPackage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPackageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PackageServiceServer).GetPackage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.PackageService/GetPackage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PackageServiceServer).GetPackage(ctx, req.(*GetPackageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PackageService_ListPackages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListPackagesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PackageServiceServer).ListPackages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.PackageService/ListPackages",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PackageServiceServer).ListPackages(ctx, req.(*ListPackagesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PackageService_DeletePackage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeletePackageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PackageServiceServer).DeletePackage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.PackageService/DeletePackage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PackageServiceServer).DeletePackage(ctx, req.(*DeletePackageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PackageService_GetTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PackageServiceServer).GetTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.PackageService/GetTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PackageServiceServer).GetTemplate(ctx, req.(*GetTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _PackageService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.PackageService",
	HandlerType: (*PackageServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPackage",
			Handler:    _PackageService_GetPackage_Handler,
		},
		{
			MethodName: "ListPackages",
			Handler:    _PackageService_ListPackages_Handler,
		},
		{
			MethodName: "DeletePackage",
			Handler:    _PackageService_DeletePackage_Handler,
		},
		{
			MethodName: "GetTemplate",
			Handler:    _PackageService_GetTemplate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "package.proto",
}

func init() { proto.RegisterFile("package.proto", fileDescriptor_package_cce06a887bdfde92) }

var fileDescriptor_package_cce06a887bdfde92 = []byte{
	// 535 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x53, 0xcd, 0x6e, 0xd3, 0x40,
	0x10, 0x56, 0x92, 0xfe, 0x24, 0x93, 0x26, 0x15, 0xd3, 0xaa, 0x71, 0xdd, 0x42, 0x23, 0x53, 0x45,
	0x11, 0x07, 0x5b, 0x09, 0x27, 0x8e, 0x20, 0x50, 0x2f, 0x1c, 0x2a, 0x37, 0x07, 0x6e, 0xd1, 0x26,
	0x1e, 0x92, 0xa5, 0xb1, 0xbd, 0x78, 0xb7, 0x15, 0x29, 0xe2, 0xc2, 0x2b, 0xf0, 0x1a, 0x3c, 0x01,
	0xaf, 0xc1, 0x2b, 0xf0, 0x20, 0xc8, 0xeb, 0xdd, 0x92, 0xbf, 0x96, 0x5b, 0xe6, 0x9b, 0xcf, 0x3b,
	0xdf, 0xcc, 0xf7, 0x05, 0x1a, 0x82, 0x8d, 0xaf, 0xd9, 0x84, 0x7c, 0x91, 0xa5, 0x2a, 0xc5, 0x0a,
	0x13, 0xdc, 0x3d, 0x9d, 0xa4, 0xe9, 0x64, 0x46, 0x01, 0x13, 0x3c, 0x60, 0x49, 0x92, 0x2a, 0xa6,
	0x78, 0x9a, 0xc8, 0x82, 0xe2, 0x9e, 0x99, 0xae, 0xae, 0x46, 0x37, 0x1f, 0x03, 0xc5, 0x63, 0x92,
	0x8a, 0xc5, 0xc2, 0x10, 0x4e, 0x56, 0x09, 0x14, 0x0b, 0x35, 0x37, 0xcd, 0x7d, 0xc1, 0x32, 0x16,
	0x93, 0xa2, 0xac, 0x00, 0xbc, 0xe7, 0xf0, 0xe4, 0x82, 0xd4, 0x65, 0xa1, 0x22, 0xa4, 0xcf, 0x37,
	0x24, 0x15, 0x36, 0xa1, 0xcc, 0x23, 0xa7, 0xd4, 0x2e, 0x75, 0x1b, 0x61, 0x99, 0x47, 0xde, 0x27,
	0x38, 0x78, 0xcf, 0xa5, 0x65, 0x49, 0x4b, 0x7b, 0x0a, 0x20, 0xd8, 0x84, 0x86, 0x2a, 0xbd, 0xa6,
	0x44, 0xd3, 0x6b, 0x61, 0x2d, 0x47, 0x06, 0x39, 0x80, 0x27, 0xa0, 0x8b, 0xa1, 0xe4, 0x77, 0xe4,
	0x94, 0xdb, 0xa5, 0xee, 0x76, 0x58, 0xcd, 0x81, 0x2b, 0x7e, 0x47, 0xd8, 0x82, 0x5d, 0x99, 0x66,
	0x6a, 0x38, 0x9a, 0x3b, 0x15, 0xfd, 0xe1, 0x4e, 0x5e, 0xbe, 0x99, 0x7b, 0x53, 0x38, 0x5c, 0x9e,
	0x25, 0x45, 0x9a, 0x48, 0xc2, 0x2e, 0x54, 0xcd, 0xad, 0xa4, 0x53, 0x6a, 0x57, 0xba, 0xf5, 0xfe,
	0x9e, 0xcf, 0x04, 0xf7, 0xad, 0xf4, 0xfb, 0x2e, 0x76, 0x60, 0x3f, 0xa1, 0x2f, 0x6a, 0xb8, 0xa0,
	0xad, 0xac, 0x47, 0x34, 0x72, 0xf8, 0xd2, 0xea, 0xf3, 0x3a, 0x70, 0xf8, 0x96, 0x66, 0xa4, 0xe8,
	0x3f, 0xdb, 0x9f, 0x03, 0x5e, 0x90, 0x1a, 0x50, 0x2c, 0x66, 0x4c, 0x3d, 0xc8, 0xea, 0xc1, 0xc1,
	0x12, 0xcb, 0xc8, 0x76, 0xa1, 0xaa, 0x0c, 0x66, 0x2e, 0x74, 0x5f, 0x7b, 0xbf, 0x4a, 0xb0, 0x6b,
	0x66, 0xaf, 0x3e, 0x87, 0xaf, 0x00, 0xc6, 0x19, 0x31, 0x45, 0xd1, 0x90, 0x29, 0xad, 0xbf, 0xde,
	0x77, 0xfd, 0xc2, 0x5a, 0xdf, 0x5a, 0xeb, 0x0f, 0xac, 0xf7, 0x61, 0xcd, 0xb0, 0x5f, 0x2b, 0x44,
	0xd8, 0x4a, 0x58, 0x4c, 0xe6, 0xae, 0xfa, 0x37, 0xb6, 0xa1, 0x1e, 0x91, 0x1c, 0x67, 0x5c, 0xe4,
	0x59, 0x72, 0xb6, 0x74, 0x6b, 0x11, 0x42, 0x3f, 0x37, 0xd3, 0x64, 0x43, 0x3a, 0xdb, 0xfa, 0xc2,
	0x4d, 0x73, 0x61, 0x03, 0x87, 0x0b, 0x8c, 0xfe, 0xcf, 0x0a, 0x34, 0x8d, 0xf8, 0x2b, 0xca, 0x6e,
	0xf9, 0x98, 0xf0, 0x03, 0xc0, 0xbf, 0x2c, 0xe1, 0x91, 0xfe, 0x78, 0x2d, 0x5c, 0xee, 0x92, 0x6d,
	0xde, 0xf9, 0xf7, 0xdf, 0x7f, 0x7e, 0x94, 0x9f, 0xe1, 0x69, 0x1e, 0x77, 0x19, 0xdc, 0xf6, 0xd8,
	0x4c, 0x4c, 0x59, 0x2f, 0xb0, 0x6e, 0x06, 0x5f, 0x79, 0xf4, 0x0d, 0x23, 0xd8, 0x5b, 0x0c, 0x05,
	0x3a, 0xfa, 0x8d, 0x0d, 0x99, 0x74, 0x8f, 0x37, 0x74, 0x0a, 0x2b, 0xbc, 0x33, 0x3d, 0xea, 0x18,
	0x5b, 0x0f, 0x8c, 0xc2, 0x29, 0x34, 0x96, 0x02, 0x81, 0xc5, 0x63, 0x9b, 0x42, 0xe2, 0x1e, 0xad,
	0x79, 0xf1, 0x2e, 0xff, 0x9b, 0xd9, 0x7d, 0x5e, 0x3c, 0xbe, 0x8f, 0x80, 0xfa, 0x42, 0x58, 0xb0,
	0x65, 0x4f, 0xb5, 0x12, 0x32, 0xd7, 0x59, 0x6f, 0x98, 0x65, 0x7c, 0x3d, 0xa7, 0x8b, 0x9d, 0xc7,
	0xe6, 0x04, 0x36, 0x6a, 0x72, 0xb4, 0xa3, 0x75, 0xbe, 0xfc, 0x1b, 0x00, 0x00, 0xff, 0xff, 0xa7,
	0xb8, 0x7a, 0xea, 0x71, 0x04, 0x00, 0x00,
}
