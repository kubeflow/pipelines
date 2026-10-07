// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestRuntimeStatusJSONRetainsRPCDetails(t *testing.T) {
	opaque := &anypb.Any{TypeUrl: "type.example/NotLinked", Value: []byte{8, 42}}
	rpc := &statuspb.Status{Code: int32(codes.Internal), Message: "source error", Details: []*anypb.Any{opaque}}
	original := RuntimeStatus{State: RuntimeStateFailed, UpdateTimeInSec: 1700000000, Error: status.ErrorProto(rpc)}
	encoded, err := json.Marshal(original)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "source error")
	var restored RuntimeStatus
	require.NoError(t, json.Unmarshal(encoded, &restored))
	require.Equal(t, original.State, restored.State)
	require.Equal(t, original.UpdateTimeInSec, restored.UpdateTimeInSec)
	actual, ok := status.FromError(restored.Error)
	require.True(t, ok)
	require.True(t, proto.Equal(rpc, actual.Proto()))
	again, err := json.Marshal(restored)
	require.NoError(t, err)
	require.JSONEq(t, string(encoded), string(again))
}
func TestRuntimeStatusJSONReadsProtoJSONDetails(t *testing.T) {
	detail, err := anypb.New(&errdetails.ErrorInfo{Reason: "LEGACY", Domain: "kfp"})
	require.NoError(t, err)
	rpc := &statuspb.Status{Code: int32(codes.InvalidArgument), Message: "bad input", Details: []*anypb.Any{detail}}
	raw, err := protojson.Marshal(rpc)
	require.NoError(t, err)
	encoded, err := json.Marshal(runtimeStatusJSON{State: RuntimeStateFailed, Error: raw})
	require.NoError(t, err)
	var restored RuntimeStatus
	require.NoError(t, json.Unmarshal(encoded, &restored))
	require.True(t, proto.Equal(rpc, status.Convert(restored.Error).Proto()))
}
func TestRuntimeStatusJSONLegacyEmptyAndInvalid(t *testing.T) {
	for _, data := range []string{`{}`, `{"State":"FAILED"}`, `{"Error":null}`, `{"Error":{}}`} {
		var restored RuntimeStatus
		require.NoError(t, json.Unmarshal([]byte(data), &restored))
		require.Nil(t, restored.Error)
	}
	for _, data := range []string{`{"Error":"bad"}`, `{"Error":{"code":"bad"}}`, `{"Error":{"unknown":true}}`, `{"Error":[]}`} {
		var restored RuntimeStatus
		require.Error(t, json.Unmarshal([]byte(data), &restored))
	}
	var unusual RuntimeStatus
	require.NoError(t, json.Unmarshal([]byte(`{"Error":{"code":0,"message":"retained"}}`), &unusual))
	require.NotNil(t, unusual.Error)
	require.Equal(t, "retained", status.Convert(unusual.Error).Message())
	ordinary := RuntimeStatus{Error: errors.New("plain error")}
	encoded, err := json.Marshal(ordinary)
	require.NoError(t, err)
	var restored RuntimeStatus
	require.NoError(t, json.Unmarshal(encoded, &restored))
	require.Equal(t, codes.Unknown, status.Code(restored.Error))
	require.Equal(t, "plain error", status.Convert(restored.Error).Message())
}
