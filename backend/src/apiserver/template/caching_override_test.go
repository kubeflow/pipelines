// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const cachingOverrideTestdata = "testdata/caching_override/"

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return raw
}

// parseSpec parses a manifest the way the server parses uploads (yaml.v3). A
// YAML 1.1 parser would read the input parameter named "n" as false.
func parseSpec(t *testing.T, manifest []byte) *V2Spec {
	t.Helper()
	tmpl, err := NewV2SpecTemplate(manifest, TemplateOptions{})
	require.NoError(t, err)
	return tmpl
}

func loadSpecJSON(t *testing.T, path string) *pipelinespec.PipelineSpec {
	t.Helper()
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal(readTestFile(t, path), spec))
	return spec
}

// The JSON fixtures were produced by the SDK's _override_caching_options on
// input.yaml, so these tests pin server/SDK parity, including tasks nested
// under ExitHandler, dsl.If, ParallelFor and sub-pipelines.
func TestWithCachingOverride_MatchesSDK(t *testing.T) {
	for _, tc := range []struct {
		enableCaching bool
		want          string
	}{
		{false, "disabled.json"},
		{true, "enabled.json"},
	} {
		input := parseSpec(t, readTestFile(t, cachingOverrideTestdata+"input.yaml")).spec
		original := proto.Clone(input)
		got := withCachingOverride(input, tc.enableCaching)
		require.Empty(t, cmp.Diff(loadSpecJSON(t, cachingOverrideTestdata+tc.want), got, protocmp.Transform()), tc.want)
		require.True(t, proto.Equal(original, input), "the input spec must not be modified")
	}
}

// The rewritten manifest is what the driver reads, so it must parse back to
// the SDK result.
func TestApplyCachingOverride_MatchesSDK(t *testing.T) {
	for _, tc := range []struct {
		enableCaching bool
		want          string
	}{
		{false, "disabled.json"},
		{true, "enabled.json"},
	} {
		got, err := ApplyCachingOverride(readTestFile(t, cachingOverrideTestdata+"input.yaml"), &tc.enableCaching)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(loadSpecJSON(t, cachingOverrideTestdata+tc.want), parseSpec(t, got).spec, protocmp.Transform()), tc.want)
	}
}

func TestApplyCachingOverride_NilKeepsManifest(t *testing.T) {
	raw := readTestFile(t, cachingOverrideTestdata+"input.yaml")
	got, err := ApplyCachingOverride(raw, nil)
	require.NoError(t, err)
	require.Equal(t, raw, got)
}

func TestApplyCachingOverride_RejectsManifestWithoutPipelineSpec(t *testing.T) {
	disabled := false
	_, err := ApplyCachingOverride([]byte("kind: ConfigMap\n"), &disabled)
	require.Error(t, err)
}

// The platform spec (volumes, node selectors, ...) is a second YAML document.
// It must survive the rewrite, executors must not change, and the result
// must still compile.
func TestApplyCachingOverride_KeepsPlatformSpecAndExecutors(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	for _, path := range []string{cachingOverrideTestdata + "input.yaml", "testdata/pipeline_with_volume.yaml"} {
		raw := readTestFile(t, path)
		before := parseSpec(t, raw)
		disabled := false
		got, err := ApplyCachingOverride(raw, &disabled)
		require.NoError(t, err)
		after := parseSpec(t, got)
		require.Empty(t, cmp.Diff(before.platformSpec, after.platformSpec, protocmp.Transform()), path)
		require.Empty(t, cmp.Diff(before.spec.GetDeploymentSpec(), after.spec.GetDeploymentSpec(), protocmp.Transform()), path)
		for _, task := range after.spec.GetRoot().GetDag().GetTasks() {
			require.False(t, task.GetCachingOptions().GetEnableCache(), path)
		}
		_, err = after.RunWorkflow(&model.Run{DisplayName: "run"}, RunWorkflowOptions{})
		require.NoError(t, err, path)
	}
}
