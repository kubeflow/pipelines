// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/codes"
)

// transferScheduleVersion resolves the definition the destination controller
// would select, without pinning an unpinned schedule to the archive default.
func (r *ResourceManager) transferScheduleVersion(plan *history.TransferPlan, job *model.Job) (*model.PipelineVersion, error) {
	if job.PipelineVersionId != "" {
		for i := range plan.Bundle.Versions {
			v := &plan.Bundle.Versions[i]
			if v.UUID == job.PipelineVersionId && (job.PipelineId == "" || v.PipelineId == job.PipelineId) {
				return v, nil
			}
		}
		return nil, transferPrecondition("Schedule pipeline version must resolve to its archived pipeline")
	}
	if job.PipelineId == "" {
		return nil, nil
	}
	receipt := plan.Receipt("pipeline", job.PipelineId)
	if receipt == nil {
		return nil, transferPrecondition("Schedule pipeline has no transfer receipt")
	}
	if plan.ExternalCatalog && plan.Existing[receipt.Key] {
		pinStore, ok := r.pipelineStore.(interface{ TransferDefaultPinned(string) (bool, error) })
		if !ok {
			return nil, transferPrecondition("Cannot inspect the destination pipeline default")
		}
		pinned, err := pinStore.TransferDefaultPinned(job.PipelineId)
		if err != nil {
			return nil, err
		}
		if !pinned {
			for _, v := range plan.Bundle.Versions {
				versionReceipt := plan.Receipt("version", v.UUID)
				if v.PipelineId == job.PipelineId && versionReceipt != nil && !plan.Existing[versionReceipt.Key] {
					return nil, transferPrecondition("Adding versions can change an unpinned schedule's destination default; pin the destination pipeline default and retry")
				}
			}
		}
		// Catalog staging retains an existing Kubernetes default, which may
		// select a destination version that is absent from the source archive.
		v, err := r.pipelineStore.GetDefaultPipelineVersion(job.PipelineId)
		if err != nil {
			return nil, err
		}
		if v == nil || v.PipelineId != job.PipelineId {
			return nil, transferPrecondition("Destination default version belongs to a different pipeline")
		}
		return v, nil
	}
	target := ""
	if plan.ExternalCatalog {
		for _, p := range plan.Bundle.Pipelines {
			if p.UUID == job.PipelineId {
				target = p.DefaultVersionId
				break
			}
		}
	}
	var selected *model.PipelineVersion
	for i := range plan.Bundle.Versions {
		v := &plan.Bundle.Versions[i]
		if v.PipelineId != job.PipelineId {
			continue
		}
		if target != "" {
			if v.UUID == target {
				selected = v
				break
			}
		} else if selected == nil || newerTransferVersion(v, selected) {
			selected = v
		}
	}
	if !plan.ExternalCatalog && plan.Existing[receipt.Key] {
		// SQL defaults use the newest row after the merge, including native
		// destination versions. The deprecated pipeline default field is ignored.
		existing, err := r.pipelineStore.GetDefaultPipelineVersion(job.PipelineId)
		if err != nil && !util.IsUserErrorCodeMatch(err, codes.NotFound) {
			return nil, err
		}
		if err == nil {
			if existing == nil || existing.PipelineId != job.PipelineId {
				return nil, transferPrecondition("Destination default version belongs to a different pipeline")
			}
			if selected == nil || existing.UUID == selected.UUID || newerTransferVersion(existing, selected) {
				selected = existing
			}
		}
	}
	if selected == nil {
		return nil, transferPrecondition("Schedule pipeline default cannot be resolved at the destination")
	}
	return selected, nil
}

func newerTransferVersion(a, b *model.PipelineVersion) bool {
	return a.CreatedAtInSec > b.CreatedAtInSec || (a.CreatedAtInSec == b.CreatedAtInSec && a.UUID > b.UUID)
}
