// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	crd "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const transferSourceAnnotation = "pipelines.kubeflow.org/transfer-source"
const transferIDAnnotation = "pipelines.kubeflow.org/transfer-id"
const transferDigestAnnotation = "pipelines.kubeflow.org/transfer-digest"

// TransferCatalog adapts the Kubernetes pipeline store to restartable transfer staging.
type TransferCatalog struct {
	Store            *PipelineStoreKubernetes
	RuntimeNamespace string
}

func (a TransferCatalog) Export(ctx context.Context, namespace string) ([]model.Pipeline, []model.PipelineVersion, map[string]string, error) {
	scope := namespace
	if scope == "" {
		scope = a.RuntimeNamespace
	}
	if scope == "" {
		return nil, nil, nil, util.NewInvalidInputError("Runtime namespace is required for catalog transfer")
	}
	filter := &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: scope}}
	rows, _, _, err := a.Store.ListPipelines(filter, list.EmptyOptions(), nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defaults := map[string]string{}
	var pipelines []model.Pipeline
	var versions []model.PipelineVersion
	for _, p := range rows {
		object := &crd.Pipeline{}
		if err := a.Store.clientNoCache.Get(ctx, types.NamespacedName{Namespace: scope, Name: p.Name}, object); err != nil {
			return nil, nil, nil, err
		}
		if object.Spec.DefaultVersionName != "" {
			defaults[p.UUID] = object.Spec.DefaultVersionName
		}

		p.Status = model.PipelineReady
		p.Namespace = namespace
		pipelines = append(pipelines, *p)
		vs, _, _, err := a.Store.ListPipelineVersions(p.UUID, list.EmptyOptions(), nil)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, v := range vs {
			v.Status = model.PipelineVersionReady
			versions = append(versions, *v)
		}
	}
	return pipelines, versions, defaults, ctx.Err()
}

func catalogDigest(row any) string {
	b, _ := json.Marshal(row)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (a TransferCatalog) Prepare(ctx context.Context, source, namespace string, pipelines []model.Pipeline, versions []model.PipelineVersion, defaults map[string]string, dry bool) (map[string]string, map[string]string, map[string]string, error) {
	pmap, vmap := map[string]string{}, map[string]string{}
	effectiveDefaults := map[string]string{}
	for id, name := range defaults {
		effectiveDefaults[id] = name
	}
	parents := map[string]model.Pipeline{}
	stage := func(object ctrlclient.Object, original any, id string) error {
		want := map[string]string{transferSourceAnnotation: source, transferIDAnnotation: id, transferDigestAnnotation: catalogDigest(original)}
		object.SetAnnotations(want)
		object.SetUID("")
		current := object.DeepCopyObject().(ctrlclient.Object)
		err := a.Store.clientNoCache.Get(ctx, types.NamespacedName{Namespace: object.GetNamespace(), Name: object.GetName()}, current)
		if err == nil {
			annotations := current.GetAnnotations()
			for key, value := range want {
				if annotations[key] != value {
					return util.NewAlreadyExistError("Catalog name %s already exists with different transfer provenance; choose a name prefix", object.GetName())
				}
			}
			switch desired := object.(type) {
			case *crd.Pipeline:
				// Later batches retain the destination's explicit default.
				desired.Spec.DefaultVersionName = current.(*crd.Pipeline).Spec.DefaultVersionName
				if desired.Spec.DefaultVersionName == "" {
					delete(effectiveDefaults, id)
				} else {
					effectiveDefaults[id] = desired.Spec.DefaultVersionName
				}
				if !reflect.DeepEqual(desired.Spec, current.(*crd.Pipeline).Spec) {
					return util.NewAlreadyExistError("Previously staged pipeline was modified; restore it before retrying")
				}
			case *crd.PipelineVersion:
				if !reflect.DeepEqual(desired.Spec, current.(*crd.PipelineVersion).Spec) {
					return util.NewAlreadyExistError("Previously staged pipeline version was modified; restore it before retrying")
				}
			}
			object.SetUID(current.GetUID())
			return nil
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
		if dry {
			object.SetUID(types.UID(id))
			return nil
		}
		if err := a.Store.clientNoCache.Create(ctx, object); err != nil {
			return err
		}
		if object.GetUID() == "" {
			return fmt.Errorf("kubernetes did not assign an ID to %s", object.GetName())
		}
		return nil
	}
	for _, p := range pipelines {
		if problems := k8svalidation.IsDNS1123Subdomain(p.Name); len(problems) > 0 {
			return nil, nil, nil, util.NewInvalidInputError("Pipeline name with prefix is not a valid Kubernetes name")
		}
		if p.Namespace != namespace {
			return nil, nil, nil, util.NewInvalidInputError("Catalog namespace differs from transfer namespace")
		}
		if err := model.ValidateTags(p.Tags); err != nil {
			return nil, nil, nil, err
		}
		owner := p
		if owner.Namespace == "" {
			owner.Namespace = a.RuntimeNamespace
		}
		object := crd.FromPipelineModel(owner)
		object.Spec.DefaultVersionName = defaults[p.UUID]
		if err := stage(&object, p, p.UUID); err != nil {
			return nil, nil, nil, err
		}
		pmap[p.UUID] = string(object.UID)
		parent := owner
		parent.UUID = string(object.UID)
		parents[p.UUID] = parent
	}
	for _, v := range versions {
		parent, ok := parents[v.PipelineId]
		if !ok {
			return nil, nil, nil, util.NewInvalidInputError("Pipeline version parent is missing")
		}
		if err := model.ValidateTags(v.Tags); err != nil {
			return nil, nil, nil, err
		}
		object, err := crd.FromPipelineVersionModel(parent, v)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := stage(object, v, v.UUID); err != nil {
			return nil, nil, nil, err
		}
		vmap[v.UUID] = string(object.UID)
	}
	return pmap, vmap, effectiveDefaults, nil
}
