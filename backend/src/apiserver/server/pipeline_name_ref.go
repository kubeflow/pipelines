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

package server

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// resolvePipelineNameRef replaces a name-based pipeline reference with the IDs it
// names, looking the pipeline up in the destination namespace. It is a no-op for
// ID-based references.
func resolvePipelineNameRef(ctx context.Context, resourceManager *resource.ResourceManager, pipelineSpec *model.PipelineSpec, namespace string) error {
	ref := pipelineSpec.NameRef
	if ref == nil {
		return nil
	}
	pipelineSpec.NameRef = nil
	if ref.PipelineName == "" {
		return util.NewInvalidInputError("pipeline_name is required when pipeline_version_name is set")
	}
	if pipelineSpec.PipelineId != "" || pipelineSpec.PipelineVersionId != "" {
		return util.NewInvalidInputError("Specify the pipeline either by name or by ID, not both")
	}
	namespace = resourceManager.ReplaceNamespace(namespace)
	// Authorize before the lookup, as GetPipelineByName does, so a name cannot
	// reveal whether a pipeline exists in a namespace the caller cannot read.
	if err := authorizePipelineAccess(ctx, resourceManager, "", &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Name:      ref.PipelineName,
		Verb:      common.RbacResourceVerbGet,
	}); err != nil {
		return util.Wrapf(err, "Failed to authorize access to pipeline %q in namespace %q", ref.PipelineName, namespace)
	}
	pipeline, err := resourceManager.GetPipelineByNameAndNamespace(ref.PipelineName, namespace)
	if err != nil {
		return util.Wrapf(err, "Failed to find pipeline %q in namespace %q", ref.PipelineName, namespace)
	}
	pipelineSpec.PipelineId = pipeline.UUID
	if ref.PipelineVersionName == "" {
		return nil
	}
	pipelineVersion, err := resourceManager.GetPipelineVersionByName(pipeline.UUID, ref.PipelineVersionName)
	if err != nil {
		return util.Wrapf(err, "Failed to find version %q of pipeline %q", ref.PipelineVersionName, ref.PipelineName)
	}
	pipelineSpec.PipelineVersionId = pipelineVersion.UUID
	return nil
}
