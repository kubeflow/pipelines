// Copyright 2026 The Kubeflow Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	v2beta1 "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

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

// WalkTransferCatalog reads one CR per API page rather than using the normal
// catalog listing, whose pagination happens after every spec has been decoded.
// Returning an error from either visitor stops before fetching the next page.
func (k *PipelineStoreKubernetes) WalkTransferCatalog(ctx context.Context, namespace string, pipeline func(*model.Pipeline, string) error, version func(*model.PipelineVersion) error) error {
	if namespace == "" {
		return util.NewInvalidInputError("A runtime namespace is required to export the catalog")
	}
	parents := map[string]bool{}
	token := ""
	for {
		var page v2beta1.PipelineList
		if err := k.clientNoCache.List(ctx, &page, ctrlclient.InNamespace(namespace), ctrlclient.Limit(1), ctrlclient.Continue(token)); err != nil {
			return err
		}
		for i := range page.Items {
			p := &page.Items[i]
			if err := pipeline(p.ToModel(), p.Spec.DefaultVersionName); err != nil {
				return err
			}
			parents[string(p.UID)] = true
		}
		token = page.Continue
		if token == "" {
			break
		}
	}
	token = ""
	for {
		var page v2beta1.PipelineVersionList
		if err := k.clientNoCache.List(ctx, &page, ctrlclient.InNamespace(namespace), ctrlclient.Limit(1), ctrlclient.Continue(token)); err != nil {
			return err
		}
		for i := range page.Items {
			v := &page.Items[i]
			owned := false
			for _, ref := range v.OwnerReferences {
				if ref.Kind == "Pipeline" && ref.APIVersion == v2beta1.GroupVersion.String() {
					owned = parents[string(ref.UID)]
					break
				}
			}
			if !owned {
				continue
			}
			row, err := v.ToModel()
			if err != nil {
				return err
			}
			if err := version(row); err != nil {
				return err
			}
		}
		token = page.Continue
		if token == "" {
			break
		}
	}
	return nil
}
