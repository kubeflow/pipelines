// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model_test

import (
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Transfer archives and their receipt digests encode runs and recurring runs as
// JSON, and imports reject unknown fields. The transient name reference must
// stay out of that encoding, so archives match across server versions.
func TestPipelineSpecNameRefIsNotEncoded(t *testing.T) {
	nameRef := &model.PipelineNameRef{PipelineName: "training", PipelineVersionName: "v1"}
	for name, value := range map[string]any{
		"run":           model.Run{PipelineSpec: model.PipelineSpec{NameRef: nameRef}},
		"recurring run": model.Job{PipelineSpec: model.PipelineSpec{NameRef: nameRef}},
	} {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(value)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, json.Unmarshal(data, &fields))
			assert.NotContains(t, fields, "NameRef")
		})
	}
}
