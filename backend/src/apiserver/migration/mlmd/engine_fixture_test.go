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

package mlmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

type sourceFixture struct {
	LegacyTask struct {
		UUID              string `json:"UUID"`
		RunID             string `json:"RunID"`
		Namespace         string `json:"Namespace"`
		PodName           string `json:"PodName"`
		State             string `json:"State"`
		CreatedTimestamp  int64  `json:"CreatedTimestamp"`
		StartedTimestamp  int64  `json:"StartedTimestamp"`
		FinishedTimestamp int64  `json:"FinishedTimestamp"`
	} `json:"legacy_task"`
	ExecutionIDs map[string]int64 `json:"execution_ids"`
	ArtifactIDs  map[string]int64 `json:"artifact_ids"`
	Artifacts    struct {
		Artifacts []struct {
			ID               string                     `json:"id"`
			Type             string                     `json:"type"`
			CustomProperties map[string]json.RawMessage `json:"customProperties"`
		} `json:"artifacts"`
	} `json:"artifacts"`
}

type nativeFixture struct {
	Tasks     []model.Task         `json:"tasks"`
	Artifacts []model.Artifact     `json:"artifacts"`
	Links     []model.ArtifactTask `json:"links"`
}

func readFixture(t *testing.T, path string, value any) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "engine", path))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, value))
}

type releaseGraph struct {
	Runs []struct {
		ID        string `json:"id"`
		State     string `json:"state"`
		Namespace string `json:"namespace"`
	} `json:"runs"`
	Tasks []struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		RunID             string `json:"run_id"`
		Namespace         string `json:"namespace"`
		State             string `json:"state"`
		ExecutionID       string `json:"execution_id"`
		PodName           string `json:"pod_name"`
		Fingerprint       string `json:"fingerprint"`
		CreatedTimestamp  int64  `json:"created_timestamp"`
		StartedTimestamp  int64  `json:"started_timestamp"`
		FinishedTimestamp int64  `json:"finished_timestamp"`
		StateHistory      string `json:"state_history"`
	} `json:"tasks"`
	Types []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"types"`
	Contexts []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"contexts"`
	Executions []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		TypeID    int    `json:"type_id"`
		State     int    `json:"state"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"executions"`
	Associations []struct {
		ContextID   int `json:"context_id"`
		ExecutionID int `json:"execution_id"`
	} `json:"associations"`
	Properties []struct {
		ExecutionID int    `json:"execution_id"`
		Name        string `json:"name"`
		IntValue    *int64 `json:"int_value"`
		StringValue string `json:"string_value"`
	} `json:"execution_properties"`
	Artifacts []struct {
		ID        int    `json:"id"`
		TypeID    int    `json:"type_id"`
		URI       string `json:"uri"`
		CreatedMS int64  `json:"created_ms"`
		UpdatedMS int64  `json:"updated_ms"`
	} `json:"artifacts"`
	ArtifactProperties []struct {
		ArtifactID  int      `json:"artifact_id"`
		Name        string   `json:"name"`
		DoubleValue *float64 `json:"double_value"`
		StringValue string   `json:"string_value"`
	} `json:"artifact_properties"`
	Events []struct {
		ID          int `json:"id"`
		ExecutionID int `json:"execution_id"`
		ArtifactID  int `json:"artifact_id"`
		Type        int `json:"type"`
	} `json:"events"`
	EventPaths []struct {
		EventID int    `json:"event_id"`
		StepKey string `json:"step_key"`
	} `json:"event_paths"`
}
