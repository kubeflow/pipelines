// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"os"
	"path/filepath"
	"testing"
)

func fakeResourceManager() *resource.ResourceManager {
	clientManager := resource.NewFakeClientManagerOrFatalV2()
	resourceManager := resource.NewResourceManager(clientManager, &resource.ResourceManagerOptions{CollectMetrics: false})
	return resourceManager
}

func TestLoadSamplesConfigBackwardsCompatibility(t *testing.T) {
	rm := fakeResourceManager()
	pc := []deprecatedConfig{
		{
			Name:        "Pipeline 1",
			Description: "test description",
			File:        "testdata/sample_pipeline.yaml",
		},
		{
			Name:        "Pipeline 2",
			Description: "test description",
			File:        "testdata/sample_pipeline.yaml",
		},
	}

	path, err := writeSampleConfigDeprecated(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	_, err = rm.GetPipelineByNameAndNamespace(pc[0].Name, "")
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace(pc[1].Name, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName(pc[0].Name)
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName(pc[1].Name)
	require.NoError(t, err)

	// Update pipeline version for Pipeline 1
	pc[0].Name = "Pipeline 3"
	path, err = writeSampleConfigDeprecated(t, pc, "sample.json")
	require.NoError(t, err)

	// Loading samples should result in no pipeline uploaded
	err = LoadSamples(rm, path)
	require.NoError(t, err)
	_, err = rm.GetPipelineByNameAndNamespace(pc[0].Name, "")
	var userErr *util.UserError
	if assert.ErrorAs(t, err, &userErr) {
		require.Equal(t, codes.NotFound, userErr.ExternalStatusCode())
	}
}

func TestLoadSamples(t *testing.T) {
	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 1",
				VersionDescription: "Pipeline 1 - Ver 1 Description",
			},
			{
				Name:               "Pipeline 2",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 2 - Ver 1",
				VersionDescription: "Pipeline 2 - Ver 1 Description",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	var pipeline1 *model.Pipeline
	pipeline1, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[0].Name, "")
	require.NoError(t, err)
	var pipeline2 *model.Pipeline
	pipeline2, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[1].Name, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName(pc.Pipelines[0].VersionName)
	require.NoError(t, err)

	// Update pipeline version for Pipeline 1
	pc.Pipelines[0].VersionName = "Pipeline 1 - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 1
	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)
	_, totalSize, _, err := rm.ListPipelineVersions(pipeline1.UUID, opts)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 2"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect another Pipeline version added for Pipeline 2
	_, err = rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	require.NoError(t, err)
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts)
	require.Equal(t, totalSize, 2)

	// Confirm previous pipeline version count has not been affected
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline1.UUID, opts)
	require.Equal(t, totalSize, 2)

	// When LoadSamplesOnRestart is false, changes to config should
	// result in no new pipelines update upon restart
	pc.LoadSamplesOnRestart = false

	// Update pipeline version for Pipeline 2
	pc.Pipelines[1].VersionName = "Pipeline 2 - Ver 3"
	path, err = writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect no change
	_, totalSize, _, err = rm.ListPipelineVersions(pipeline2.UUID, opts)
	require.NoError(t, err)
	require.Equal(t, totalSize, 2)
}

func TestLoadSamplesMultiplePipelineVersionsInConfig(t *testing.T) {
	rm := fakeResourceManager()
	pc := config{
		LoadSamplesOnRestart: true,
		Pipelines: []configPipelines{
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 1",
				VersionDescription: "Pipeline 1 - Ver 1 Description",
			},
			{
				Name:               "Pipeline 1",
				Description:        "test description",
				File:               "testdata/sample_pipeline.yaml",
				VersionName:        "Pipeline 1 - Ver 2",
				VersionDescription: "Pipeline 1 - Ver 2 Description",
			},
		},
	}

	path, err := writeSampleConfig(t, pc, "sample.json")
	require.NoError(t, err)
	err = LoadSamples(rm, path)
	require.NoError(t, err)

	// Expect both versions to be added
	var pipeline *model.Pipeline
	pipeline, err = rm.GetPipelineByNameAndNamespace(pc.Pipelines[0].Name, "")
	require.NoError(t, err)

	_, err = rm.GetPipelineVersionByName(pc.Pipelines[0].VersionName)
	require.NoError(t, err)
	_, err = rm.GetPipelineVersionByName(pc.Pipelines[1].VersionName)
	require.NoError(t, err)

	opts, err := list.NewOptions(&model.PipelineVersion{}, 10, "id", nil)
	require.NoError(t, err)

	_, totalSize, _, err := rm.ListPipelineVersions(pipeline.UUID, opts)
	require.Equal(t, totalSize, 2)
}

func writeSampleConfig(t *testing.T, config config, path string) (string, error) {
	return writeContents(t, config, path)
}

func writeSampleConfigDeprecated(t *testing.T, config []deprecatedConfig, path string) (string, error) {
	return writeContents(t, config, path)
}

func writeContents(t *testing.T, content interface{}, path string) (string, error) {
	tempDir := t.TempDir()
	sampleFilePath := filepath.Join(tempDir, path)
	marshal, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(sampleFilePath, marshal, 0644); err != nil {
		t.Fatalf("Failed to create %v file: %v", path, err)
	}
	return sampleFilePath, nil
}
