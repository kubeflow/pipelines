// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// SetTransferDefaultVersion restores the source default after staging all
// versions. The caller only invokes it for a newly staged, owned pipeline.
func (k *PipelineStoreKubernetes) SetTransferDefaultVersion(pipelineID, versionID string) error {
	p, err := k.getK8sPipeline(pipelineID)
	if err != nil {
		return err
	}
	v, err := k.GetPipelineVersion(versionID)
	if err != nil {
		return err
	}
	if v.PipelineId != pipelineID {
		return util.NewInvalidInputError("Default version belongs to a different pipeline")
	}
	copy := p.DeepCopy()
	copy.Spec.DefaultVersionName = v.Name
	return k.client.Update(context.Background(), copy)
}
