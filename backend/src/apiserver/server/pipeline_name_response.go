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
	"github.com/golang/glog"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
)

// pipelineRefHolder is a run or recurring run, either of which may reference a
// pipeline version instead of carrying an inline pipeline spec.
type pipelineRefHolder interface {
	GetPipelineVersionReference() *apiv2.PipelineVersionReference
}

// fillPipelineNames resolves the names behind each reference's stored IDs, so
// responses report them whether the run was created by ID or by name. It costs
// two queries no matter how many items are passed, since a list page usually
// shares a handful of distinct pipelines. Items with an inline pipeline spec
// have no reference and are skipped.
//
// A reference whose pipeline or version no longer resolves keeps empty names,
// matching how recurring runs already record the names of deleted pipelines.
// The names are informational, so a failed lookup is logged and leaves them
// empty rather than failing the read, as setScheduledWorkflowPipelineNames
// does when a recurring run is created.
func fillPipelineNames[T pipelineRefHolder](resourceManager *resource.ResourceManager, items ...T) {
	refs := make([]*apiv2.PipelineVersionReference, 0, len(items))
	var pipelineIds, pipelineVersionIds []string
	for _, item := range items {
		ref := item.GetPipelineVersionReference()
		if ref == nil {
			continue
		}
		refs = append(refs, ref)
		if id := ref.GetPipelineId(); id != "" {
			pipelineIds = append(pipelineIds, id)
		}
		if id := ref.GetPipelineVersionId(); id != "" {
			pipelineVersionIds = append(pipelineVersionIds, id)
		}
	}
	if len(refs) == 0 {
		return
	}
	// A nil map reads as empty, so a failed lookup leaves its names empty.
	pipelineNames, err := resourceManager.GetPipelineNamesForIds(pipelineIds)
	if err != nil {
		glog.Warningf("Reporting %d references without pipeline names: %v", len(refs), err)
	}
	pipelineVersionNames, err := resourceManager.GetPipelineVersionNamesForIds(pipelineVersionIds)
	if err != nil {
		glog.Warningf("Reporting %d references without pipeline version names: %v", len(refs), err)
	}
	for _, ref := range refs {
		// Overwrite rather than fill: a create request may have carried the
		// client's own spelling of a name, and the stored ID is authoritative.
		ref.PipelineName = pipelineNames[ref.GetPipelineId()]
		ref.PipelineVersionName = pipelineVersionNames[ref.GetPipelineVersionId()]
	}
}
