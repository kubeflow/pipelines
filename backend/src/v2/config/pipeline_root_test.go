// Copyright 2026 The Kubeflow Authors
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

package config_test

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineRootForRun(t *testing.T) {
	loads := 0
	load := func(context.Context, string) (*config.Config, error) {
		loads++
		return nil, nil // absent config falls back to the default root
	}
	withRoot := &go_client.Run{RunId: "run", RuntimeConfig: &go_client.RuntimeConfig{PipelineRoot: "s3://bucket/custom"}}
	root, err := config.PipelineRootForRun(context.Background(), "pipeline", "ns", withRoot, load)
	require.NoError(t, err)
	assert.Equal(t, "s3://bucket/custom/pipeline/run", root)
	assert.Zero(t, loads, "a run with its own root never consults the launcher config")

	root, err = config.PipelineRootForRun(context.Background(), "pipeline", "ns", &go_client.Run{RunId: "run"}, load)
	require.NoError(t, err)
	assert.Equal(t, (*config.Config)(nil).DefaultPipelineRoot()+"/pipeline/run", root)
	assert.Equal(t, 1, loads)

	_, err = config.PipelineRootForRun(context.Background(), "pipeline", "ns", &go_client.Run{RunId: "run"},
		func(context.Context, string) (*config.Config, error) { return nil, assert.AnError })
	require.ErrorIs(t, err, assert.AnError)
}
