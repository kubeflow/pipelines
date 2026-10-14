// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package util

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
)

func TestScheduleHandoffErrorWireContract(t *testing.T) {
	err := Wrap(NewScheduleHandoffInProgressError(), "Create recurring run failed")
	status := ToGRPCStatus(err)
	require.Equal(t, codes.Unavailable, status.Code())
	found := false
	for _, detail := range status.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			require.Equal(t, "SCHEDULE_HANDOFF_IN_PROGRESS", info.Reason)
			require.Equal(t, "kubeflow.org", info.Domain)
			found = true
		}
	}
	require.True(t, found)
	response := httptest.NewRecorder()
	gateway.HTTPError(context.Background(), gateway.NewServeMux(), &gateway.JSONPb{}, response, httptest.NewRequest(http.MethodPost, "/", nil), status.Err())
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	var body struct {
		Code    int                      `json:"code"`
		Message string                   `json:"message"`
		Details []map[string]interface{} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, 14, body.Code)
	require.Contains(t, body.Message, "was not applied")
	require.Contains(t, body.Message, "after handoff completes")
	found = false
	for _, detail := range body.Details {
		if detail["@type"] == "type.googleapis.com/google.rpc.ErrorInfo" {
			require.Equal(t, "SCHEDULE_HANDOFF_IN_PROGRESS", detail["reason"])
			require.Equal(t, "kubeflow.org", detail["domain"])
			found = true
		}
	}
	require.True(t, found)
	// Ordinary outages do not promise that a mutation was rejected.
	for _, detail := range NewUnavailableError("temporary outage").GRPCStatus().Details() {
		_, marked := detail.(*errdetails.ErrorInfo)
		require.False(t, marked)
	}
}
