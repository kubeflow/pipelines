// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package api_server_v2 //nolint:staticcheck // Match the existing package name.

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	"github.com/stretchr/testify/require"
)

func TestTaskHistorySerializationCompatibility(t *testing.T) {
	for _, history := range [][]*run_model.PipelineTaskLogHistory{nil, {}} {
		body, err := json.Marshal(&run_model.V2PipelineTask{LogHistory: history})
		require.NoError(t, err)
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &fields))
		require.NotContains(t, fields, "log_history", "older servers reject this unknown field even when null")
	}
	body, err := json.Marshal(&run_model.V2PipelineTask{LogHistory: []*run_model.PipelineTaskLogHistory{{DriverLogsURI: "s3://logs/previous"}}})
	require.NoError(t, err)
	var decoded run_model.V2PipelineTask
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Len(t, decoded.LogHistory, 1)
	require.Equal(t, "s3://logs/previous", decoded.LogHistory[0].DriverLogsURI)
}
