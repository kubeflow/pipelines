// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	structpb "google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestProtoMessageToProtoStruct(t *testing.T) {
	tests := []struct {
		name    string
		m       proto.Message
		want    *structpb.Struct
		wantErr bool
		errMsg  string
	}{
		{
			"valid - timestamp",
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   1,
			},
			&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds": structpb.NewNumberValue(1649175200),
					"nanos":   structpb.NewNumberValue(1),
				},
			},
			false,
			"",
		},
		{
			"valid - timestamp with omitted values",
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   0,
			},
			&structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds": structpb.NewNumberValue(1649175200),
				},
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProtoMessageToProtoStruct(tt.m)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Nil(t, got)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, got)
				assert.True(t, proto.Equal(tt.want, got), fmt.Sprintf("Wanted: %v \nGot: %v", tt.want, got))
			}
		})
	}
}

func TestProtoStructToProtoMessage(t *testing.T) {
	tests := []struct {
		name    string
		s       structpb.Struct
		got     proto.Message
		want    proto.Message
		wantErr bool
		errMsg  string
	}{
		{
			"valid - timestamp with omitted value",
			structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds": structpb.NewNumberValue(1649175200),
				},
			},
			&timestamppb.Timestamp{},
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   0,
			},
			false,
			"",
		},
		{
			"valid - timestamp with zero value",
			structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds": structpb.NewNumberValue(1649175200),
					"nanos":   structpb.NewNumberValue(0),
				},
			},
			&timestamppb.Timestamp{},
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   0,
			},
			false,
			"",
		},
		{
			"valid - timestamp",
			structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds": structpb.NewNumberValue(1649175200),
					"nanos":   structpb.NewNumberValue(1),
				},
			},
			&timestamppb.Timestamp{},
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   1,
			},
			false,
			"",
		},
		{
			"valid - timestamp with additional field",
			structpb.Struct{
				Fields: map[string]*structpb.Value{
					"seconds":    structpb.NewNumberValue(1649175200),
					"nanos":      structpb.NewNumberValue(1),
					"irrelevant": structpb.NewNumberValue(1),
				},
			},
			&timestamppb.Timestamp{},
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   1,
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProtoStructToProtoMessage(&tt.s, &tt.got)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.Nil(t, err)
				assert.True(t, proto.Equal(tt.want, tt.got), fmt.Sprintf("Wanted: %v \nGot: %v", tt.want, tt.got))
			}
		})
	}
}
