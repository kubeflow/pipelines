// Copyright 2019 Google LLC
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
// source: backend/api/job.proto

package go_client

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import empty "github.com/golang/protobuf/ptypes/empty"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"
import _ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
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

type Job_Mode int32

const (
	Job_UNKNOWN_MODE Job_Mode = 0
	Job_ENABLED      Job_Mode = 1
	Job_DISABLED     Job_Mode = 2
)

var Job_Mode_name = map[int32]string{
	0: "UNKNOWN_MODE",
	1: "ENABLED",
	2: "DISABLED",
}
var Job_Mode_value = map[string]int32{
	"UNKNOWN_MODE": 0,
	"ENABLED":      1,
	"DISABLED":     2,
}

func (x Job_Mode) String() string {
	return proto.EnumName(Job_Mode_name, int32(x))
}
func (Job_Mode) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{10, 0}
}

type CreateJobRequest struct {
	Job                  *Job     `protobuf:"bytes,1,opt,name=job,proto3" json:"job,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateJobRequest) Reset()         { *m = CreateJobRequest{} }
func (m *CreateJobRequest) String() string { return proto.CompactTextString(m) }
func (*CreateJobRequest) ProtoMessage()    {}
func (*CreateJobRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{0}
}
func (m *CreateJobRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateJobRequest.Unmarshal(m, b)
}
func (m *CreateJobRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateJobRequest.Marshal(b, m, deterministic)
}
func (dst *CreateJobRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateJobRequest.Merge(dst, src)
}
func (m *CreateJobRequest) XXX_Size() int {
	return xxx_messageInfo_CreateJobRequest.Size(m)
}
func (m *CreateJobRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateJobRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateJobRequest proto.InternalMessageInfo

func (m *CreateJobRequest) GetJob() *Job {
	if m != nil {
		return m.Job
	}
	return nil
}

type GetJobRequest struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetJobRequest) Reset()         { *m = GetJobRequest{} }
func (m *GetJobRequest) String() string { return proto.CompactTextString(m) }
func (*GetJobRequest) ProtoMessage()    {}
func (*GetJobRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{1}
}
func (m *GetJobRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetJobRequest.Unmarshal(m, b)
}
func (m *GetJobRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetJobRequest.Marshal(b, m, deterministic)
}
func (dst *GetJobRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetJobRequest.Merge(dst, src)
}
func (m *GetJobRequest) XXX_Size() int {
	return xxx_messageInfo_GetJobRequest.Size(m)
}
func (m *GetJobRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetJobRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetJobRequest proto.InternalMessageInfo

func (m *GetJobRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type ListJobsRequest struct {
	PageToken            string       `protobuf:"bytes,1,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	PageSize             int32        `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	SortBy               string       `protobuf:"bytes,3,opt,name=sort_by,json=sortBy,proto3" json:"sort_by,omitempty"`
	ResourceReferenceKey *ResourceKey `protobuf:"bytes,4,opt,name=resource_reference_key,json=resourceReferenceKey,proto3" json:"resource_reference_key,omitempty"`
	Filter               string       `protobuf:"bytes,5,opt,name=filter,proto3" json:"filter,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *ListJobsRequest) Reset()         { *m = ListJobsRequest{} }
func (m *ListJobsRequest) String() string { return proto.CompactTextString(m) }
func (*ListJobsRequest) ProtoMessage()    {}
func (*ListJobsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{2}
}
func (m *ListJobsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListJobsRequest.Unmarshal(m, b)
}
func (m *ListJobsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListJobsRequest.Marshal(b, m, deterministic)
}
func (dst *ListJobsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListJobsRequest.Merge(dst, src)
}
func (m *ListJobsRequest) XXX_Size() int {
	return xxx_messageInfo_ListJobsRequest.Size(m)
}
func (m *ListJobsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListJobsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListJobsRequest proto.InternalMessageInfo

func (m *ListJobsRequest) GetPageToken() string {
	if m != nil {
		return m.PageToken
	}
	return ""
}

func (m *ListJobsRequest) GetPageSize() int32 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *ListJobsRequest) GetSortBy() string {
	if m != nil {
		return m.SortBy
	}
	return ""
}

func (m *ListJobsRequest) GetResourceReferenceKey() *ResourceKey {
	if m != nil {
		return m.ResourceReferenceKey
	}
	return nil
}

func (m *ListJobsRequest) GetFilter() string {
	if m != nil {
		return m.Filter
	}
	return ""
}

type ListJobsResponse struct {
	Jobs                 []*Job   `protobuf:"bytes,1,rep,name=jobs,proto3" json:"jobs,omitempty"`
	TotalSize            int32    `protobuf:"varint,3,opt,name=total_size,json=totalSize,proto3" json:"total_size,omitempty"`
	NextPageToken        string   `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListJobsResponse) Reset()         { *m = ListJobsResponse{} }
func (m *ListJobsResponse) String() string { return proto.CompactTextString(m) }
func (*ListJobsResponse) ProtoMessage()    {}
func (*ListJobsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{3}
}
func (m *ListJobsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListJobsResponse.Unmarshal(m, b)
}
func (m *ListJobsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListJobsResponse.Marshal(b, m, deterministic)
}
func (dst *ListJobsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListJobsResponse.Merge(dst, src)
}
func (m *ListJobsResponse) XXX_Size() int {
	return xxx_messageInfo_ListJobsResponse.Size(m)
}
func (m *ListJobsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListJobsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListJobsResponse proto.InternalMessageInfo

func (m *ListJobsResponse) GetJobs() []*Job {
	if m != nil {
		return m.Jobs
	}
	return nil
}

func (m *ListJobsResponse) GetTotalSize() int32 {
	if m != nil {
		return m.TotalSize
	}
	return 0
}

func (m *ListJobsResponse) GetNextPageToken() string {
	if m != nil {
		return m.NextPageToken
	}
	return ""
}

type DeleteJobRequest struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteJobRequest) Reset()         { *m = DeleteJobRequest{} }
func (m *DeleteJobRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteJobRequest) ProtoMessage()    {}
func (*DeleteJobRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{4}
}
func (m *DeleteJobRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteJobRequest.Unmarshal(m, b)
}
func (m *DeleteJobRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteJobRequest.Marshal(b, m, deterministic)
}
func (dst *DeleteJobRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteJobRequest.Merge(dst, src)
}
func (m *DeleteJobRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteJobRequest.Size(m)
}
func (m *DeleteJobRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteJobRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteJobRequest proto.InternalMessageInfo

func (m *DeleteJobRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type EnableJobRequest struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EnableJobRequest) Reset()         { *m = EnableJobRequest{} }
func (m *EnableJobRequest) String() string { return proto.CompactTextString(m) }
func (*EnableJobRequest) ProtoMessage()    {}
func (*EnableJobRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{5}
}
func (m *EnableJobRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EnableJobRequest.Unmarshal(m, b)
}
func (m *EnableJobRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EnableJobRequest.Marshal(b, m, deterministic)
}
func (dst *EnableJobRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EnableJobRequest.Merge(dst, src)
}
func (m *EnableJobRequest) XXX_Size() int {
	return xxx_messageInfo_EnableJobRequest.Size(m)
}
func (m *EnableJobRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_EnableJobRequest.DiscardUnknown(m)
}

var xxx_messageInfo_EnableJobRequest proto.InternalMessageInfo

func (m *EnableJobRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type DisableJobRequest struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DisableJobRequest) Reset()         { *m = DisableJobRequest{} }
func (m *DisableJobRequest) String() string { return proto.CompactTextString(m) }
func (*DisableJobRequest) ProtoMessage()    {}
func (*DisableJobRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{6}
}
func (m *DisableJobRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DisableJobRequest.Unmarshal(m, b)
}
func (m *DisableJobRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DisableJobRequest.Marshal(b, m, deterministic)
}
func (dst *DisableJobRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DisableJobRequest.Merge(dst, src)
}
func (m *DisableJobRequest) XXX_Size() int {
	return xxx_messageInfo_DisableJobRequest.Size(m)
}
func (m *DisableJobRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DisableJobRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DisableJobRequest proto.InternalMessageInfo

func (m *DisableJobRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type CronSchedule struct {
	StartTime            *timestamp.Timestamp `protobuf:"bytes,1,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime              *timestamp.Timestamp `protobuf:"bytes,2,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
	Cron                 string               `protobuf:"bytes,3,opt,name=cron,proto3" json:"cron,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *CronSchedule) Reset()         { *m = CronSchedule{} }
func (m *CronSchedule) String() string { return proto.CompactTextString(m) }
func (*CronSchedule) ProtoMessage()    {}
func (*CronSchedule) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{7}
}
func (m *CronSchedule) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CronSchedule.Unmarshal(m, b)
}
func (m *CronSchedule) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CronSchedule.Marshal(b, m, deterministic)
}
func (dst *CronSchedule) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CronSchedule.Merge(dst, src)
}
func (m *CronSchedule) XXX_Size() int {
	return xxx_messageInfo_CronSchedule.Size(m)
}
func (m *CronSchedule) XXX_DiscardUnknown() {
	xxx_messageInfo_CronSchedule.DiscardUnknown(m)
}

var xxx_messageInfo_CronSchedule proto.InternalMessageInfo

func (m *CronSchedule) GetStartTime() *timestamp.Timestamp {
	if m != nil {
		return m.StartTime
	}
	return nil
}

func (m *CronSchedule) GetEndTime() *timestamp.Timestamp {
	if m != nil {
		return m.EndTime
	}
	return nil
}

func (m *CronSchedule) GetCron() string {
	if m != nil {
		return m.Cron
	}
	return ""
}

type PeriodicSchedule struct {
	StartTime            *timestamp.Timestamp `protobuf:"bytes,1,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime              *timestamp.Timestamp `protobuf:"bytes,2,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
	IntervalSecond       int64                `protobuf:"varint,3,opt,name=interval_second,json=intervalSecond,proto3" json:"interval_second,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *PeriodicSchedule) Reset()         { *m = PeriodicSchedule{} }
func (m *PeriodicSchedule) String() string { return proto.CompactTextString(m) }
func (*PeriodicSchedule) ProtoMessage()    {}
func (*PeriodicSchedule) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{8}
}
func (m *PeriodicSchedule) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PeriodicSchedule.Unmarshal(m, b)
}
func (m *PeriodicSchedule) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PeriodicSchedule.Marshal(b, m, deterministic)
}
func (dst *PeriodicSchedule) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PeriodicSchedule.Merge(dst, src)
}
func (m *PeriodicSchedule) XXX_Size() int {
	return xxx_messageInfo_PeriodicSchedule.Size(m)
}
func (m *PeriodicSchedule) XXX_DiscardUnknown() {
	xxx_messageInfo_PeriodicSchedule.DiscardUnknown(m)
}

var xxx_messageInfo_PeriodicSchedule proto.InternalMessageInfo

func (m *PeriodicSchedule) GetStartTime() *timestamp.Timestamp {
	if m != nil {
		return m.StartTime
	}
	return nil
}

func (m *PeriodicSchedule) GetEndTime() *timestamp.Timestamp {
	if m != nil {
		return m.EndTime
	}
	return nil
}

func (m *PeriodicSchedule) GetIntervalSecond() int64 {
	if m != nil {
		return m.IntervalSecond
	}
	return 0
}

type Trigger struct {
	// Types that are valid to be assigned to Trigger:
	//	*Trigger_CronSchedule
	//	*Trigger_PeriodicSchedule
	Trigger              isTrigger_Trigger `protobuf_oneof:"trigger"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Trigger) Reset()         { *m = Trigger{} }
func (m *Trigger) String() string { return proto.CompactTextString(m) }
func (*Trigger) ProtoMessage()    {}
func (*Trigger) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{9}
}
func (m *Trigger) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Trigger.Unmarshal(m, b)
}
func (m *Trigger) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Trigger.Marshal(b, m, deterministic)
}
func (dst *Trigger) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Trigger.Merge(dst, src)
}
func (m *Trigger) XXX_Size() int {
	return xxx_messageInfo_Trigger.Size(m)
}
func (m *Trigger) XXX_DiscardUnknown() {
	xxx_messageInfo_Trigger.DiscardUnknown(m)
}

var xxx_messageInfo_Trigger proto.InternalMessageInfo

type isTrigger_Trigger interface {
	isTrigger_Trigger()
}

type Trigger_CronSchedule struct {
	CronSchedule *CronSchedule `protobuf:"bytes,1,opt,name=cron_schedule,json=cronSchedule,proto3,oneof"`
}

type Trigger_PeriodicSchedule struct {
	PeriodicSchedule *PeriodicSchedule `protobuf:"bytes,2,opt,name=periodic_schedule,json=periodicSchedule,proto3,oneof"`
}

func (*Trigger_CronSchedule) isTrigger_Trigger() {}

func (*Trigger_PeriodicSchedule) isTrigger_Trigger() {}

func (m *Trigger) GetTrigger() isTrigger_Trigger {
	if m != nil {
		return m.Trigger
	}
	return nil
}

func (m *Trigger) GetCronSchedule() *CronSchedule {
	if x, ok := m.GetTrigger().(*Trigger_CronSchedule); ok {
		return x.CronSchedule
	}
	return nil
}

func (m *Trigger) GetPeriodicSchedule() *PeriodicSchedule {
	if x, ok := m.GetTrigger().(*Trigger_PeriodicSchedule); ok {
		return x.PeriodicSchedule
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Trigger) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Trigger_OneofMarshaler, _Trigger_OneofUnmarshaler, _Trigger_OneofSizer, []interface{}{
		(*Trigger_CronSchedule)(nil),
		(*Trigger_PeriodicSchedule)(nil),
	}
}

func _Trigger_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Trigger)
	// trigger
	switch x := m.Trigger.(type) {
	case *Trigger_CronSchedule:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.CronSchedule); err != nil {
			return err
		}
	case *Trigger_PeriodicSchedule:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.PeriodicSchedule); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Trigger.Trigger has unexpected type %T", x)
	}
	return nil
}

func _Trigger_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Trigger)
	switch tag {
	case 1: // trigger.cron_schedule
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(CronSchedule)
		err := b.DecodeMessage(msg)
		m.Trigger = &Trigger_CronSchedule{msg}
		return true, err
	case 2: // trigger.periodic_schedule
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(PeriodicSchedule)
		err := b.DecodeMessage(msg)
		m.Trigger = &Trigger_PeriodicSchedule{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Trigger_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Trigger)
	// trigger
	switch x := m.Trigger.(type) {
	case *Trigger_CronSchedule:
		s := proto.Size(x.CronSchedule)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Trigger_PeriodicSchedule:
		s := proto.Size(x.PeriodicSchedule)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type Job struct {
	Id                   string               `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name                 string               `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Description          string               `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	PipelineSpec         *PipelineSpec        `protobuf:"bytes,4,opt,name=pipeline_spec,json=pipelineSpec,proto3" json:"pipeline_spec,omitempty"`
	ResourceReferences   []*ResourceReference `protobuf:"bytes,5,rep,name=resource_references,json=resourceReferences,proto3" json:"resource_references,omitempty"`
	MaxConcurrency       int64                `protobuf:"varint,6,opt,name=max_concurrency,json=maxConcurrency,proto3" json:"max_concurrency,omitempty"`
	Trigger              *Trigger             `protobuf:"bytes,7,opt,name=trigger,proto3" json:"trigger,omitempty"`
	Mode                 Job_Mode             `protobuf:"varint,8,opt,name=mode,proto3,enum=api.Job_Mode" json:"mode,omitempty"`
	CreatedAt            *timestamp.Timestamp `protobuf:"bytes,9,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	UpdatedAt            *timestamp.Timestamp `protobuf:"bytes,10,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	Status               string               `protobuf:"bytes,11,opt,name=status,proto3" json:"status,omitempty"`
	Error                string               `protobuf:"bytes,12,opt,name=error,proto3" json:"error,omitempty"`
	Enabled              bool                 `protobuf:"varint,16,opt,name=enabled,proto3" json:"enabled,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Job) Reset()         { *m = Job{} }
func (m *Job) String() string { return proto.CompactTextString(m) }
func (*Job) ProtoMessage()    {}
func (*Job) Descriptor() ([]byte, []int) {
	return fileDescriptor_job_073b88ec5135fd11, []int{10}
}
func (m *Job) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Job.Unmarshal(m, b)
}
func (m *Job) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Job.Marshal(b, m, deterministic)
}
func (dst *Job) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Job.Merge(dst, src)
}
func (m *Job) XXX_Size() int {
	return xxx_messageInfo_Job.Size(m)
}
func (m *Job) XXX_DiscardUnknown() {
	xxx_messageInfo_Job.DiscardUnknown(m)
}

var xxx_messageInfo_Job proto.InternalMessageInfo

func (m *Job) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Job) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Job) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *Job) GetPipelineSpec() *PipelineSpec {
	if m != nil {
		return m.PipelineSpec
	}
	return nil
}

func (m *Job) GetResourceReferences() []*ResourceReference {
	if m != nil {
		return m.ResourceReferences
	}
	return nil
}

func (m *Job) GetMaxConcurrency() int64 {
	if m != nil {
		return m.MaxConcurrency
	}
	return 0
}

func (m *Job) GetTrigger() *Trigger {
	if m != nil {
		return m.Trigger
	}
	return nil
}

func (m *Job) GetMode() Job_Mode {
	if m != nil {
		return m.Mode
	}
	return Job_UNKNOWN_MODE
}

func (m *Job) GetCreatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAt
	}
	return nil
}

func (m *Job) GetUpdatedAt() *timestamp.Timestamp {
	if m != nil {
		return m.UpdatedAt
	}
	return nil
}

func (m *Job) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func (m *Job) GetError() string {
	if m != nil {
		return m.Error
	}
	return ""
}

func (m *Job) GetEnabled() bool {
	if m != nil {
		return m.Enabled
	}
	return false
}

func init() {
	proto.RegisterType((*CreateJobRequest)(nil), "api.CreateJobRequest")
	proto.RegisterType((*GetJobRequest)(nil), "api.GetJobRequest")
	proto.RegisterType((*ListJobsRequest)(nil), "api.ListJobsRequest")
	proto.RegisterType((*ListJobsResponse)(nil), "api.ListJobsResponse")
	proto.RegisterType((*DeleteJobRequest)(nil), "api.DeleteJobRequest")
	proto.RegisterType((*EnableJobRequest)(nil), "api.EnableJobRequest")
	proto.RegisterType((*DisableJobRequest)(nil), "api.DisableJobRequest")
	proto.RegisterType((*CronSchedule)(nil), "api.CronSchedule")
	proto.RegisterType((*PeriodicSchedule)(nil), "api.PeriodicSchedule")
	proto.RegisterType((*Trigger)(nil), "api.Trigger")
	proto.RegisterType((*Job)(nil), "api.Job")
	proto.RegisterEnum("api.Job_Mode", Job_Mode_name, Job_Mode_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// JobServiceClient is the client API for JobService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type JobServiceClient interface {
	CreateJob(ctx context.Context, in *CreateJobRequest, opts ...grpc.CallOption) (*Job, error)
	GetJob(ctx context.Context, in *GetJobRequest, opts ...grpc.CallOption) (*Job, error)
	ListJobs(ctx context.Context, in *ListJobsRequest, opts ...grpc.CallOption) (*ListJobsResponse, error)
	EnableJob(ctx context.Context, in *EnableJobRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	DisableJob(ctx context.Context, in *DisableJobRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	DeleteJob(ctx context.Context, in *DeleteJobRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type jobServiceClient struct {
	cc *grpc.ClientConn
}

func NewJobServiceClient(cc *grpc.ClientConn) JobServiceClient {
	return &jobServiceClient{cc}
}

func (c *jobServiceClient) CreateJob(ctx context.Context, in *CreateJobRequest, opts ...grpc.CallOption) (*Job, error) {
	out := new(Job)
	err := c.cc.Invoke(ctx, "/api.JobService/CreateJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobServiceClient) GetJob(ctx context.Context, in *GetJobRequest, opts ...grpc.CallOption) (*Job, error) {
	out := new(Job)
	err := c.cc.Invoke(ctx, "/api.JobService/GetJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobServiceClient) ListJobs(ctx context.Context, in *ListJobsRequest, opts ...grpc.CallOption) (*ListJobsResponse, error) {
	out := new(ListJobsResponse)
	err := c.cc.Invoke(ctx, "/api.JobService/ListJobs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobServiceClient) EnableJob(ctx context.Context, in *EnableJobRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.JobService/EnableJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobServiceClient) DisableJob(ctx context.Context, in *DisableJobRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.JobService/DisableJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *jobServiceClient) DeleteJob(ctx context.Context, in *DeleteJobRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/api.JobService/DeleteJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// JobServiceServer is the server API for JobService service.
type JobServiceServer interface {
	CreateJob(context.Context, *CreateJobRequest) (*Job, error)
	GetJob(context.Context, *GetJobRequest) (*Job, error)
	ListJobs(context.Context, *ListJobsRequest) (*ListJobsResponse, error)
	EnableJob(context.Context, *EnableJobRequest) (*empty.Empty, error)
	DisableJob(context.Context, *DisableJobRequest) (*empty.Empty, error)
	DeleteJob(context.Context, *DeleteJobRequest) (*empty.Empty, error)
}

func RegisterJobServiceServer(s *grpc.Server, srv JobServiceServer) {
	s.RegisterService(&_JobService_serviceDesc, srv)
}

func _JobService_CreateJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).CreateJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/CreateJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).CreateJob(ctx, req.(*CreateJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _JobService_GetJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).GetJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/GetJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).GetJob(ctx, req.(*GetJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _JobService_ListJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).ListJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/ListJobs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).ListJobs(ctx, req.(*ListJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _JobService_EnableJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EnableJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).EnableJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/EnableJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).EnableJob(ctx, req.(*EnableJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _JobService_DisableJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DisableJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).DisableJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/DisableJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).DisableJob(ctx, req.(*DisableJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _JobService_DeleteJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(JobServiceServer).DeleteJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.JobService/DeleteJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(JobServiceServer).DeleteJob(ctx, req.(*DeleteJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _JobService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.JobService",
	HandlerType: (*JobServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateJob",
			Handler:    _JobService_CreateJob_Handler,
		},
		{
			MethodName: "GetJob",
			Handler:    _JobService_GetJob_Handler,
		},
		{
			MethodName: "ListJobs",
			Handler:    _JobService_ListJobs_Handler,
		},
		{
			MethodName: "EnableJob",
			Handler:    _JobService_EnableJob_Handler,
		},
		{
			MethodName: "DisableJob",
			Handler:    _JobService_DisableJob_Handler,
		},
		{
			MethodName: "DeleteJob",
			Handler:    _JobService_DeleteJob_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend/api/job.proto",
}

func init() { proto.RegisterFile("backend/api/job.proto", fileDescriptor_job_073b88ec5135fd11) }

var fileDescriptor_job_073b88ec5135fd11 = []byte{
	// 1090 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xc4, 0x56, 0xdd, 0x52, 0x23, 0xc5,
	0x17, 0x27, 0x1f, 0xe4, 0xe3, 0x90, 0x40, 0xe8, 0xe5, 0x63, 0xfe, 0x59, 0xf6, 0x4f, 0x76, 0xb4,
	0x58, 0xca, 0x72, 0x93, 0x5a, 0xb6, 0xb4, 0xd4, 0x3b, 0x20, 0xc8, 0x0a, 0x0b, 0x4b, 0x4d, 0xb0,
	0xac, 0xd2, 0x8b, 0xa9, 0x9e, 0x99, 0x43, 0x76, 0x20, 0x99, 0x1e, 0xbb, 0x3b, 0x2c, 0xc1, 0xf2,
	0xc6, 0x47, 0x50, 0x5f, 0xc0, 0x07, 0xf0, 0xca, 0x77, 0xf0, 0x05, 0x7c, 0x05, 0x1f, 0xc4, 0xea,
	0x9e, 0x9e, 0x90, 0x8f, 0x45, 0x2e, 0xbd, 0x4a, 0xce, 0xaf, 0x7f, 0xa7, 0xcf, 0x57, 0x9f, 0x73,
	0x06, 0x56, 0x3d, 0xea, 0x5f, 0x61, 0x14, 0xb4, 0x68, 0x1c, 0xb6, 0x2e, 0x99, 0xd7, 0x8c, 0x39,
	0x93, 0x8c, 0xe4, 0x68, 0x1c, 0xd6, 0x37, 0xba, 0x8c, 0x75, 0x7b, 0xa8, 0x8f, 0x68, 0x14, 0x31,
	0x49, 0x65, 0xc8, 0x22, 0x91, 0x50, 0xea, 0x9b, 0xe6, 0x54, 0x4b, 0xde, 0xe0, 0xa2, 0x25, 0xc3,
	0x3e, 0x0a, 0x49, 0xfb, 0xb1, 0x21, 0x3c, 0x9e, 0x26, 0x60, 0x3f, 0x96, 0xc3, 0xf4, 0x70, 0xdc,
	0x6e, 0x4c, 0x39, 0xed, 0xa3, 0x44, 0x9e, 0x5e, 0x3d, 0x71, 0x18, 0xc6, 0xd8, 0x0b, 0x23, 0x74,
	0x45, 0x8c, 0xbe, 0x21, 0x7c, 0x38, 0x4e, 0xe0, 0x28, 0xd8, 0x80, 0xfb, 0xe8, 0x72, 0xbc, 0x40,
	0x8e, 0x91, 0x8f, 0x86, 0x35, 0x11, 0x1b, 0x1f, 0x44, 0x06, 0xfe, 0x58, 0xff, 0xf8, 0xcf, 0xbb,
	0x18, 0x3d, 0x17, 0xef, 0x68, 0xb7, 0x8b, 0xbc, 0xc5, 0x62, 0x1d, 0xda, 0x7b, 0xc2, 0x5c, 0x1f,
	0xbf, 0x04, 0x39, 0x67, 0xc6, 0x49, 0xbb, 0x09, 0xb5, 0x7d, 0x8e, 0x54, 0xe2, 0x11, 0xf3, 0x1c,
	0xfc, 0x7e, 0x80, 0x42, 0x92, 0x3a, 0xe4, 0x2e, 0x99, 0x67, 0x65, 0x1a, 0x99, 0xed, 0x85, 0x9d,
	0x52, 0x93, 0xc6, 0x61, 0x53, 0x9d, 0x2a, 0xd0, 0xde, 0x84, 0xea, 0x21, 0xca, 0x31, 0xf2, 0x22,
	0x64, 0xc3, 0x40, 0x73, 0xcb, 0x4e, 0x36, 0x0c, 0xec, 0x3f, 0x33, 0xb0, 0xf4, 0x3a, 0x14, 0x8a,
	0x22, 0x52, 0xce, 0x13, 0x80, 0x98, 0x76, 0xd1, 0x95, 0xec, 0x0a, 0x23, 0xc3, 0x2d, 0x2b, 0xe4,
	0x5c, 0x01, 0xe4, 0x31, 0x68, 0xc1, 0x15, 0xe1, 0x2d, 0x5a, 0xd9, 0x46, 0x66, 0x7b, 0xde, 0x29,
	0x29, 0xa0, 0x13, 0xde, 0x22, 0x59, 0x87, 0xa2, 0x60, 0x5c, 0xba, 0xde, 0xd0, 0xca, 0x69, 0xc5,
	0x82, 0x12, 0xf7, 0x86, 0xe4, 0x4b, 0x58, 0x9b, 0xcd, 0x99, 0x7b, 0x85, 0x43, 0x2b, 0xaf, 0x1d,
	0xaf, 0x69, 0xc7, 0x1d, 0x43, 0x39, 0xc6, 0xa1, 0xb3, 0x92, 0xf2, 0x9d, 0x94, 0x7e, 0x8c, 0x43,
	0xb2, 0x06, 0x85, 0x8b, 0xb0, 0x27, 0x91, 0x5b, 0xf3, 0xc9, 0xfd, 0x89, 0x64, 0xbf, 0x83, 0xda,
	0x5d, 0x1c, 0x22, 0x66, 0x91, 0x40, 0xb2, 0x01, 0xf9, 0x4b, 0xe6, 0x09, 0x2b, 0xd3, 0xc8, 0x4d,
	0xa4, 0x46, 0xa3, 0x2a, 0x4c, 0xc9, 0x24, 0xed, 0x25, 0x81, 0xe4, 0x74, 0x20, 0x65, 0x8d, 0xe8,
	0x48, 0xb6, 0x60, 0x29, 0xc2, 0x1b, 0xe9, 0x8e, 0xa5, 0x22, 0xab, 0x2d, 0x56, 0x15, 0x7c, 0x96,
	0xa6, 0xc3, 0xb6, 0xa1, 0xd6, 0xc6, 0x1e, 0x4e, 0x94, 0x64, 0x3a, 0xcb, 0x36, 0xd4, 0x0e, 0x22,
	0xea, 0xf5, 0xfe, 0x8d, 0xf3, 0x01, 0x2c, 0xb7, 0x43, 0xf1, 0x00, 0xe9, 0xd7, 0x0c, 0x54, 0xf6,
	0x39, 0x8b, 0x3a, 0xfe, 0x5b, 0x0c, 0x06, 0x3d, 0x24, 0x9f, 0x03, 0x08, 0x49, 0xb9, 0x74, 0x55,
	0x23, 0x98, 0x37, 0x50, 0x6f, 0x26, 0x4d, 0xd0, 0x4c, 0x9b, 0xa0, 0x79, 0x9e, 0x76, 0x89, 0x53,
	0xd6, 0x6c, 0x25, 0x93, 0x4f, 0xa0, 0x84, 0x51, 0x90, 0x28, 0x66, 0x1f, 0x54, 0x2c, 0x62, 0x14,
	0x68, 0x35, 0x02, 0x79, 0x9f, 0xb3, 0xc8, 0x94, 0x57, 0xff, 0xb7, 0x7f, 0xcf, 0x40, 0xed, 0x0c,
	0x79, 0xc8, 0x82, 0xd0, 0xff, 0x0f, 0x5d, 0x7b, 0x06, 0x4b, 0x61, 0x24, 0x91, 0x5f, 0xab, 0xa2,
	0xa2, 0xcf, 0xa2, 0x40, 0x7b, 0x99, 0x73, 0x16, 0x53, 0xb8, 0xa3, 0x51, 0x95, 0xc6, 0xe2, 0x39,
	0x0f, 0x55, 0x17, 0x92, 0xcf, 0xa0, 0xaa, 0x62, 0x70, 0x85, 0xf1, 0xdb, 0x78, 0xba, 0xac, 0x5f,
	0xcb, 0x78, 0xae, 0x5f, 0xcd, 0x39, 0x15, 0x7f, 0x3c, 0xf7, 0x6d, 0x58, 0x8e, 0x4d, 0xd0, 0x77,
	0xda, 0x89, 0xbb, 0xab, 0x5a, 0x7b, 0x3a, 0x25, 0xaf, 0xe6, 0x9c, 0x5a, 0x3c, 0x85, 0xed, 0x95,
	0xa1, 0x28, 0x13, 0x57, 0xec, 0x3f, 0xf2, 0x90, 0x3b, 0x62, 0xde, 0x74, 0xd5, 0x55, 0xca, 0x23,
	0x6a, 0x52, 0x51, 0x76, 0xf4, 0x7f, 0xd2, 0x80, 0x85, 0x00, 0x85, 0xcf, 0x43, 0x3d, 0x44, 0x4c,
	0x35, 0xc6, 0x21, 0xf2, 0x29, 0x54, 0x27, 0xc6, 0x98, 0x69, 0xb4, 0x24, 0xb0, 0x33, 0x73, 0xd2,
	0x89, 0xd1, 0x77, 0x2a, 0xf1, 0x98, 0x44, 0x0e, 0xe1, 0xd1, 0x6c, 0xa7, 0x0a, 0x6b, 0x5e, 0x37,
	0xd1, 0xda, 0x44, 0x9b, 0x8e, 0x3a, 0xd3, 0x21, 0x33, 0xcd, 0x2a, 0x54, 0x39, 0xfa, 0xf4, 0xc6,
	0xf5, 0x59, 0xe4, 0x0f, 0xb8, 0xc2, 0x86, 0x56, 0x21, 0x29, 0x47, 0x9f, 0xde, 0xec, 0xdf, 0xa1,
	0x64, 0x6b, 0x94, 0x02, 0xab, 0xa8, 0x7d, 0xac, 0x68, 0x2b, 0xa6, 0x42, 0x4e, 0x7a, 0x48, 0x9e,
	0x42, 0xbe, 0xcf, 0x02, 0xb4, 0x4a, 0x8d, 0xcc, 0xf6, 0xe2, 0x4e, 0x35, 0xed, 0xe7, 0xe6, 0x09,
	0x0b, 0xd0, 0xd1, 0x47, 0xea, 0xd1, 0xf9, 0x7a, 0x40, 0x06, 0x2e, 0x95, 0x56, 0xf9, 0xe1, 0x47,
	0x67, 0xd8, 0xbb, 0x52, 0xa9, 0x0e, 0xe2, 0x20, 0x55, 0x85, 0x87, 0x55, 0x0d, 0x7b, 0x57, 0xaa,
	0xa1, 0x24, 0x24, 0x95, 0x03, 0x61, 0x2d, 0x98, 0xa1, 0xa7, 0x25, 0xb2, 0x02, 0xf3, 0x7a, 0x7a,
	0x5b, 0x15, 0x0d, 0x27, 0x02, 0xb1, 0xa0, 0x88, 0x7a, 0x1a, 0x04, 0x56, 0xad, 0x91, 0xd9, 0x2e,
	0x39, 0xa9, 0x68, 0xbf, 0x84, 0xbc, 0x8a, 0x85, 0xd4, 0xa0, 0xf2, 0xf5, 0xe9, 0xf1, 0xe9, 0x9b,
	0x6f, 0x4e, 0xdd, 0x93, 0x37, 0xed, 0x83, 0xda, 0x1c, 0x59, 0x80, 0xe2, 0xc1, 0xe9, 0xee, 0xde,
	0xeb, 0x83, 0x76, 0x2d, 0x43, 0x2a, 0x50, 0x6a, 0x7f, 0xd5, 0x49, 0xa4, 0xec, 0xce, 0x6f, 0x79,
	0x80, 0x23, 0xe6, 0x75, 0x90, 0x5f, 0x87, 0x3e, 0x92, 0x13, 0x28, 0x8f, 0x56, 0x04, 0x59, 0x35,
	0xaf, 0x78, 0x72, 0x65, 0xd4, 0x47, 0xa3, 0xd0, 0xde, 0xfc, 0xe9, 0xaf, 0xbf, 0x7f, 0xc9, 0xfe,
	0xcf, 0x26, 0x6a, 0xd5, 0x88, 0xd6, 0xf5, 0x0b, 0x0f, 0x25, 0x7d, 0xa1, 0x96, 0xb2, 0xf8, 0x42,
	0x6d, 0x10, 0x72, 0x08, 0x85, 0x64, 0x83, 0x10, 0xa2, 0x95, 0x26, 0xd6, 0xc9, 0xec, 0x45, 0x64,
	0x7d, 0xf6, 0xa2, 0xd6, 0x0f, 0x61, 0xf0, 0x23, 0xe9, 0x40, 0x29, 0x1d, 0xd0, 0x64, 0x45, 0xab,
	0x4d, 0xed, 0x9d, 0xfa, 0xea, 0x14, 0x9a, 0x4c, 0x71, 0xbb, 0xae, 0x6f, 0x5e, 0x21, 0xef, 0x71,
	0x91, 0x78, 0x50, 0x1e, 0x0d, 0x56, 0x13, 0xec, 0xf4, 0xa0, 0xad, 0xaf, 0xcd, 0xd4, 0xf0, 0x40,
	0x7d, 0x13, 0xd8, 0x5b, 0xfa, 0xde, 0x86, 0xfd, 0xff, 0x7b, 0x3c, 0x6e, 0x25, 0x55, 0x21, 0x08,
	0x70, 0x37, 0x98, 0x49, 0xd2, 0x00, 0x33, 0x93, 0xfa, 0x5e, 0x2b, 0xcf, 0xb4, 0x95, 0xa7, 0xf6,
	0xe6, 0x7d, 0x56, 0x82, 0xe4, 0x2a, 0xf2, 0x1d, 0x94, 0x47, 0x7b, 0xc4, 0x84, 0x32, 0xbd, 0x57,
	0xee, 0x35, 0x62, 0x92, 0xff, 0xd1, 0x7d, 0xc9, 0xdf, 0x3b, 0xfb, 0x79, 0xf7, 0xc4, 0xd9, 0x80,
	0x62, 0x80, 0x17, 0x74, 0xd0, 0x93, 0x64, 0x99, 0x2c, 0x41, 0xb5, 0xbe, 0xa0, 0xad, 0x74, 0xf4,
	0x5b, 0xfd, 0x76, 0x13, 0x9e, 0x40, 0x61, 0x0f, 0x29, 0x47, 0x4e, 0x1e, 0x95, 0xb2, 0xf5, 0x2a,
	0x1d, 0xc8, 0xb7, 0x8c, 0x87, 0xb7, 0xfa, 0xcb, 0xa4, 0x91, 0xf5, 0x2a, 0x00, 0x23, 0xc2, 0x9c,
	0x57, 0xd0, 0x2e, 0xbc, 0xfc, 0x27, 0x00, 0x00, 0xff, 0xff, 0x03, 0x21, 0xd3, 0x01, 0xcc, 0x09,
	0x00, 0x00,
}
