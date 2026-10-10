// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package list

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrderingVersionAffectedLegacySorts(t *testing.T) {
	cases := []struct {
		model  Listable
		fields []string
	}{
		{&model.Pipeline{UUID: "p"}, []string{"namespace"}},
		{&model.PipelineVersion{UUID: "v"}, []string{"description"}},
		{&model.Job{UUID: "j"}, []string{"updated_at"}},
		{&model.Run{UUID: "r"}, []string{"state", "recurring_run_id", "scheduled_at", "finished_at", "metric:score"}},
		{&model.Task{UUID: "t"}, []string{"display_name", "parent_task_id", "start_time", "end_time", "state", "state_history"}},
	}
	for _, tc := range cases {
		for _, field := range tc.fields {
			for _, direction := range []string{"asc", "desc"} {
				t.Run(tc.model.GetModelName()+field+direction, func(t *testing.T) {
					opts, err := NewOptions(tc.model, 2, field+" "+direction, nil)
					require.NoError(t, err)
					encoded, err := opts.NextPageToken(tc.model)
					require.NoError(t, err)
					require.True(t, strings.HasPrefix(encoded, orderingTokenPrefix))
					// This is the base64 decoder used by the actual 2.17.2 token implementation.
					_, oldDecodeError := base64.StdEncoding.DecodeString(encoded)
					require.Error(t, oldDecodeError)
					restored, err := NewOptionsFromToken(encoded, 2)
					require.NoError(t, err)
					require.NoError(t, restored.ValidateOrdering(tc.model))
					require.Equal(t, currentOrderingVersion, restored.OrderingVersion)
					// Remove metadata as a historical reader would when serializing its own token.
					restored.OrderingVersion = 0
					legacy, err := restored.marshal()
					require.NoError(t, err)
					old, err := NewOptionsFromToken(legacy, 2)
					require.NoError(t, err)
					err = old.ValidateOrdering(tc.model)
					require.Equal(t, codes.FailedPrecondition, status.Code(err))
				})
			}
		}
	}
}

func TestOrderingVersionCompatibleTokensAndUntrustedMetadata(t *testing.T) {
	opts, err := NewOptions(&model.Run{UUID: "r"}, 2, "created_at", nil)
	require.NoError(t, err)
	encoded, err := opts.NextPageToken(&model.Run{UUID: "r"})
	require.NoError(t, err)
	require.NotContains(t, encoded, ":")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var fields map[string]interface{}
	require.NoError(t, json.Unmarshal(decoded, &fields))
	encode := func() string {
		b, err := json.Marshal(fields)
		require.NoError(t, err)
		return base64.StdEncoding.EncodeToString(b)
	}
	delete(fields, "OrderingVersion")
	compatible, err := NewOptionsFromToken(encode(), 2)
	require.NoError(t, err)
	require.NoError(t, compatible.ValidateOrdering(&model.Run{}))
	for _, version := range []int{0, 2} {
		fields["OrderingVersion"] = version
		_, err = NewOptionsFromToken("kfp1:"+encode(), 2)
		require.Equal(t, codes.FailedPrecondition, status.Code(err))
	}
	fields["OrderingVersion"] = 2
	future, err := NewOptionsFromToken(encode(), 2)
	require.NoError(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(future.ValidateOrdering(&model.Run{})))
	fields["OrderingVersion"] = "malformed"
	_, err = NewOptionsFromToken(encode(), 2)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	fields["OrderingVersion"] = 1
	fields["KeyFieldName"] = "UUID; DROP TABLE runs"
	_, err = NewOptionsFromToken(encode(), 2)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	fields["KeyFieldName"] = "UUID"
	fields["OrderingVersion"] = 0
	fields["ModelName"] = "experiments" // A nonnullable model claim cannot override the endpoint.
	fields["SortByFieldName"] = "UUID"
	fields["SortBySQLColumn"] = "State"
	spoofed, err := NewOptionsFromToken(encode(), 2)
	require.NoError(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(spoofed.ValidateOrdering(&model.Run{})))
	_, err = NewOptionsFromToken("kfp2:"+encoded, 2)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	_, err = NewOptionsFromToken("kfp1:not-base64", 2)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// Run.StateHistory is exposed and nullable even though historical cursor minting
// for that field fails. An incoming token must still fail closed.
func TestOrderingVersionRunStateHistory(t *testing.T) {
	for _, desc := range []bool{false, true} {
		opts := &Options{token: &token{KeyFieldName: "UUID", SortByFieldName: "StateHistory", SortBySQLColumn: "StateHistory", IsDesc: desc}}
		require.Equal(t, codes.FailedPrecondition, status.Code(opts.ValidateOrdering(&model.Run{})))
	}
}
