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

package argocompiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStablyMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		m    proto.Message
		want string
	}{
		{
			"valid - timestamp",
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   1,
			},
			`"2022-04-05T16:13:20.000000001Z"`,
		},
		{
			"valid - timestamp with omitted values",
			&timestamppb.Timestamp{
				Seconds: 1649175200,
				Nanos:   0,
			},
			`"2022-04-05T16:13:20Z"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stablyMarshalJSON(tt.m)
			assert.Nil(t, err)
			assert.NotEmpty(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}
