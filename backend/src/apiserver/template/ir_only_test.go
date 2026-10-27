// Copyright 2026 The Kubeflow Authors
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

package template

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestNew_RejectsArgoTemplatesIncludingLegacyMarkers(t *testing.T) {
	for _, manifest := range []string{
		`apiVersion: argoproj.io/v1alpha1
kind: Workflow
spec: {entrypoint: main}`,
		`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","metadata":{"annotations":{"pipelines.kubeflow.org/v2_pipeline":"true"}},"spec":{"entrypoint":"main"}}`,
		`{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","spec":{"podMetadata":{"labels":{"pipelines.kubeflow.org/v2_component":"true"}}}}`,
	} {
		_, err := New([]byte(manifest), TemplateOptions{})
		require.Error(t, err)
		require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument))
	}
}
