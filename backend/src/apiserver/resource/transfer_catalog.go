// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package resource

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/history"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/transfer"
)

func (r *ResourceManager) exportTransferKubernetesCatalog(ctx context.Context, store *storage.PipelineStoreKubernetes, namespace string, budget *transfer.ExportBudget) ([]model.Pipeline, []model.PipelineVersion, error) {
	var pipelines []model.Pipeline
	var versions []model.PipelineVersion
	indices := map[string]int{}
	defaults := map[string]string{}
	latest := map[string]*model.PipelineVersion{}
	checkCount := func() error {
		if len(pipelines)+len(versions) >= history.MaxTransferObjects {
			return transferPrecondition("Catalog exceeds the transfer object limit")
		}
		return nil
	}
	err := store.WalkTransferCatalog(ctx, transferRuntimeNamespace(namespace), func(row *model.Pipeline, defaultName string) error {
		if err := checkCount(); err != nil {
			return err
		}
		p := *row
		if p.Namespace != transferRuntimeNamespace(namespace) {
			return transferPrecondition("Catalog contains a pipeline outside the selected namespace")
		}
		p.Namespace = namespace
		p.Status = model.PipelineReady
		delete(p.Tags, transferReceiptTag)
		delete(p.Tags, transferDigestTag)
		if err := budget.Add(p); err != nil {
			return err
		}
		indices[p.UUID] = len(pipelines)
		defaults[p.UUID] = defaultName
		pipelines = append(pipelines, p)
		return nil
	}, func(row *model.PipelineVersion) error {
		if err := checkCount(); err != nil {
			return err
		}
		v := *row
		v.Status = model.PipelineVersionReady
		// Kubernetes ToModel supplies the inline compiled definition.
		v.PipelineSpecURI = ""
		v.Pipeline = model.Pipeline{}
		delete(v.Tags, transferReceiptTag)
		delete(v.Tags, transferDigestTag)
		if err := budget.Add(v); err != nil {
			return err
		}
		current := latest[v.PipelineId]
		pinned := defaults[v.PipelineId]
		if pinned != "" {
			if v.Name == pinned {
				if current != nil {
					return transferPrecondition("Default pipeline version is ambiguous")
				}
				selected := v
				latest[v.PipelineId] = &selected
			}
		} else if current == nil || v.CreatedAtInSec > current.CreatedAtInSec || (v.CreatedAtInSec == current.CreatedAtInSec && v.UUID > current.UUID) {
			selected := v
			latest[v.PipelineId] = &selected
		}
		versions = append(versions, v)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	for id, index := range indices {
		if selected := latest[id]; selected != nil {
			pipelines[index].DefaultVersionId = selected.UUID
			if err := budget.Reserve(len(selected.UUID)); err != nil {
				return nil, nil, err
			}
		} else if defaults[id] != "" {
			return nil, nil, transferPrecondition("Default pipeline version is unresolved")
		}
	}
	return pipelines, versions, nil
}
