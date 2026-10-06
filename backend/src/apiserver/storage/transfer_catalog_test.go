// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	crd "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type transferUIDClient struct {
	ctrlclient.Client
	creates int
}

func (c *transferUIDClient) Create(ctx context.Context, obj ctrlclient.Object, opts ...ctrlclient.CreateOption) error {
	obj.SetUID(types.UID("local-" + obj.GetName()))
	c.creates++
	return c.Client.Create(ctx, obj, opts...)
}
func TestTransferCatalogPreviewApplyAndConflicts(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, crd.AddToScheme(scheme))
	client := &transferUIDClient{Client: fake.NewClientBuilder().WithScheme(scheme).Build()}
	adapter := TransferCatalog{Store: NewPipelineStoreKubernetes(client, client), RuntimeNamespace: "system"}
	pipelines := []model.Pipeline{{UUID: "source-id", Name: "example", Namespace: "team", Status: model.PipelineReady, Description: "original"}}
	pmap, _, _, err := adapter.Prepare(ctx, "source", "team", pipelines, nil, nil, true)
	require.NoError(t, err)
	require.Equal(t, "source-id", pmap["source-id"])
	require.Zero(t, client.creates)
	pmap, _, _, err = adapter.Prepare(ctx, "source", "team", pipelines, nil, nil, false)
	require.NoError(t, err)
	require.Equal(t, "local-example", pmap["source-id"])
	require.Equal(t, 1, client.creates)
	_, _, _, err = adapter.Prepare(ctx, "source", "team", pipelines, nil, nil, false)
	require.NoError(t, err)
	require.Equal(t, 1, client.creates)
	object := &crd.Pipeline{}
	require.NoError(t, client.Get(ctx, types.NamespacedName{Namespace: "team", Name: "example"}, object))
	object.Spec.Description = "changed by user"
	require.NoError(t, client.Update(ctx, object))
	_, _, _, err = adapter.Prepare(ctx, "source", "team", pipelines, nil, nil, true)
	require.ErrorContains(t, err, "modified")
	_, _, _, err = adapter.Prepare(ctx, "different-source", "team", pipelines, nil, nil, true)
	require.ErrorContains(t, err, "provenance")
}

func TestTransferCatalogSingleUserDoesNotListOtherNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, crd.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&crd.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: "mine", Namespace: "system", UID: "mine-id"}}, &crd.Pipeline{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "other", UID: "secret-id"}}).Build()
	adapter := TransferCatalog{Store: NewPipelineStoreKubernetes(client, client), RuntimeNamespace: "system"}
	pipelines, versions, _, err := adapter.Export(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, pipelines, 1)
	require.Equal(t, "mine", pipelines[0].Name)
	require.Empty(t, pipelines[0].Namespace)
	require.Empty(t, versions)
}

func TestTransferCatalogRetainsPinnedDefaultAcrossBatches(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	require.NoError(t, crd.AddToScheme(scheme))
	client := &transferUIDClient{Client: fake.NewClientBuilder().WithScheme(scheme).Build()}
	adapter := TransferCatalog{Store: NewPipelineStoreKubernetes(client, client)}
	pipelines := []model.Pipeline{{UUID: "source-id", Name: "pinned", Namespace: "team", Status: model.PipelineReady}}
	_, _, _, err := adapter.Prepare(ctx, "source", "team", pipelines, nil, map[string]string{"source-id": "v1"}, false)
	require.NoError(t, err)
	object := &crd.Pipeline{}
	key := types.NamespacedName{Namespace: "team", Name: "pinned"}
	require.NoError(t, client.Get(ctx, key, object))
	require.Equal(t, "v1", object.Spec.DefaultVersionName)
	_, _, effective, err := adapter.Prepare(ctx, "source", "team", pipelines, nil, map[string]string{"source-id": "v2"}, false)
	require.NoError(t, err)
	require.NoError(t, client.Get(ctx, key, object))
	require.Equal(t, "v1", object.Spec.DefaultVersionName)
	require.Equal(t, "v1", effective["source-id"])
	_, _, defaults, err := adapter.Export(ctx, "team")
	require.NoError(t, err)
	require.Equal(t, "v1", defaults[string(object.UID)])
	require.Equal(t, 1, client.creates)
}
