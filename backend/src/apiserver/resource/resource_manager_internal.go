// Copyright 2018-2023 The Kubeflow Authors
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
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflowclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
)

func (r *ResourceManager) getWorkflowClient(namespace string) util.ExecutionInterface {
	return r.execClient.Execution(namespace)
}

func (r *ResourceManager) getScheduledWorkflowClient(namespace string) scheduledworkflowclient.ScheduledWorkflowInterface {
	return r.swfClient.ScheduledWorkflow(namespace)
}

// Fetches PipelineSpec as []byte array and a new URI of PipelineSpec.
// Returns empty string if PipelineSpec is found via PipelineSpecURI.
// It attempts to fetch PipelineSpec in the following order:
//  1. Directly read from pipeline versions's PipelineSpec field.
//  2. Fetch a yaml file from object store based on pipeline versions's PipelineSpecURI field.
//  3. Fetch a yaml file from object store based on pipeline versions's id.
//  4. Fetch a yaml file from object store based on pipeline's id.
func (r *ResourceManager) fetchTemplateFromPipelineVersion(pipelineVersion *model.PipelineVersion) ([]byte, string, error) {
	if len(pipelineVersion.PipelineSpec) != 0 {
		// Check pipeline spec string first
		bytes := []byte(pipelineVersion.PipelineSpec)
		return bytes, pipelineVersion.PipelineSpecURI, nil
	} else {
		// Try reading object store from pipeline_spec_uri
		template, errUri := r.objectStore.GetFile(pipelineVersion.PipelineSpecURI)
		if errUri != nil {
			// Try reading object store from pipeline_version_id
			template, errUUID := r.objectStore.GetFile(r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.UUID)))
			if errUUID != nil {
				// Try reading object store from pipeline_id
				template, errPipelineId := r.objectStore.GetFile(r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.PipelineId)))
				if errPipelineId != nil {
					return nil, "", util.Wrap(
						util.Wrap(
							util.Wrap(errUri, "ResourceManager: Failed to read a file from pipeline_spec_uri."),
							util.Wrap(errUUID, "ResourceManager: Failed to read a file from OS with pipeline_version_id.").Error(),
						),
						util.Wrap(errPipelineId, "ResourceManager: Failed to read a file from OS with pipeline_id.").Error(),
					)
				}
				return template, r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.PipelineId)), nil
			}
			return template, r.objectStore.GetPipelineKey(fmt.Sprint(pipelineVersion.UUID)), nil
		}
		return template, "", nil
	}
}

// Fetches PipelineSpec's manifest as []byte array.
// It attempts to fetch PipelineSpec manifest in the following order:
//  1. Directly read from PipelineSpec's PipelineSpecManifest field.
//  2. Directly read from PipelineSpec's WorkflowSpecManifest field.
//  3. Fetch pipeline spec manifest from the pipeline version for PipelineSpec's PipelineVersionId field.
//  4. Fetch pipeline spec manifest from the latest pipeline version for PipelineSpec's PipelineId field.
func (r *ResourceManager) fetchTemplateFromPipelineSpec(p *model.PipelineSpec) ([]byte, error) {
	if p == nil {
		return nil, util.NewInvalidInputError("Failed to read pipeline spec manifest from nil.")
	}
	if len(p.PipelineSpecManifest) != 0 {
		return []byte(p.PipelineSpecManifest), nil
	}
	if len(p.WorkflowSpecManifest) != 0 {
		return []byte(p.WorkflowSpecManifest), nil
	}
	var errPv, errP error
	if p.PipelineVersionId != "" {
		pv, errPv1 := r.GetPipelineVersion(p.PipelineVersionId)
		if errPv1 == nil {
			bytes, _, errPv2 := r.fetchTemplateFromPipelineVersion(pv)
			if errPv2 == nil {
				return bytes, nil
			} else {
				errPv = errPv2
			}
		} else {
			errPv = errPv1
		}
	}
	if p.PipelineId != "" {
		pv, errP1 := r.GetLatestPipelineVersion(p.PipelineId)
		if errP1 == nil {
			bytes, _, errP2 := r.fetchTemplateFromPipelineVersion(pv)
			if errP2 == nil {
				return bytes, nil
			} else {
				errP = errP2
			}
		} else {
			errP = errP1
		}
	}
	return nil, util.Wrap(
		util.Wrap(errPv, fmt.Sprintf("ResourceManager: Failed to read pipeline spec for pipeline version id %v.", p.PipelineVersionId)),
		util.Wrap(errP, fmt.Sprintf("ResourceManager: Failed to read pipeline spec for pipeline id %v.", p.PipelineId)).Error(),
	)
}

// Fetches PipelineSpec's manifest as []byte array from pipeline version's id.
func (r *ResourceManager) fetchTemplateFromPipelineVersionId(pipelineVersionId string) ([]byte, error) {
	if len(pipelineVersionId) == 0 {
		return nil, util.NewInvalidInputError("ResourceManager: Failed to get manifest as pipeline version id is empty.")
	}
	pv, err := r.GetPipelineVersion(pipelineVersionId)
	if err == nil {
		bytes, _, err := r.fetchTemplateFromPipelineVersion(pv)
		if err == nil {
			return bytes, nil
		}
	}
	return nil, util.Wrap(err, fmt.Sprintf("ResourceManager: Failed to read pipeline spec for pipeline version id %v.", pipelineVersionId))
}

func (r *ResourceManager) getDefaultExperimentId() (string, error) {
	return r.defaultExperimentStore.GetDefaultExperimentId()
}

func (r *ResourceManager) setDefaultExperimentId(id string) error {
	return r.defaultExperimentStore.SetDefaultExperimentId(id)
}

func (r *ResourceManager) getNamespaceFromExperimentId(experimentID string) (string, error) {
	experiment, err := r.GetExperiment(experimentID)
	if err != nil {
		return "", util.Wrap(err, "Failed to get namespace from experiment ID.")
	}
	namespace := experiment.Namespace

	if len(namespace) == 0 {
		if common.IsMultiUserMode() {
			return "", util.NewInternalServerError(errors.New("Missing namespace"), "Experiment %v doesn't have a namespace.", experiment.Name)
		} else {
			namespace = common.GetPodNamespace()
		}
	}
	return namespace, nil
}

func (r *ResourceManager) getNamespaceFromRunId(runId string) (string, error) {
	runDetail, err := r.GetRun(runId)
	if err != nil {
		return "", util.Wrap(err, "Failed to get namespace from run id.")
	}
	return runDetail.Namespace, nil
}
