// Copyright 2018 The Kubeflow Authors
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

package resource

import (
	"context"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deletePods(ctx context.Context, k8sCoreClient client.KubernetesCoreInterface, podsToDelete []string, namespace string) error {
	for _, podId := range podsToDelete {
		err := k8sCoreClient.PodClient(namespace).Delete(ctx, podId, metav1.DeleteOptions{})
		if err != nil && !apierr.IsNotFound(err) {
			return util.NewInternalServerError(err, "Failed to delete pods.")
		}
	}
	return nil
}

// Convert PipelineId in PipelineSpec to the pipeline's default pipeline version.
// This is for legacy usage of pipeline id to create run. The standard way to
// create run is by specifying the pipeline version.
func convertPipelineIdToDefaultPipelineVersion(pipelineSpec *api.PipelineSpec, resourceReferences *[]*api.ResourceReference, r *ResourceManager) error {
	if pipelineSpec == nil || pipelineSpec.GetPipelineId() == "" {
		return nil
	}
	// If there is already a pipeline version in resource references, don't convert pipeline id.
	for _, reference := range *resourceReferences {
		if reference.Key.Type == api.ResourceType_PIPELINE_VERSION && reference.Relationship == api.Relationship_CREATOR {
			return nil
		}
	}
	pipeline, err := r.pipelineStore.GetPipelineWithStatus(pipelineSpec.GetPipelineId(), model.PipelineReady)
	if err != nil {
		return util.Wrap(err, "Failed to find the specified pipeline")
	}
	// Add default pipeline version to resource references
	*resourceReferences = append(*resourceReferences, &api.ResourceReference{
		Key:          &api.ResourceKey{Type: api.ResourceType_PIPELINE_VERSION, Id: pipeline.DefaultVersionId},
		Relationship: api.Relationship_CREATOR,
	})
	return nil
}
