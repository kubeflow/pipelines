// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package resource

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swf "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swffake "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/fake"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
)

// Use the generated client so deterministic names and enable patches exercise
// Kubernetes object persistence, which the legacy resource test fake omits.
type transferSwfClient struct{ client *swffake.Clientset }

func (c transferSwfClient) ScheduledWorkflow(namespace string) swfclient.ScheduledWorkflowInterface {
	return c.client.ScheduledworkflowV1beta1().ScheduledWorkflows(namespace)
}

func newTransferSwfClient() transferSwfClient {
	c := swffake.NewSimpleClientset()
	nextID := 0
	c.PrependReactor("create", "scheduledworkflows", func(action ktesting.Action) (bool, runtime.Object, error) {
		object := action.(ktesting.CreateAction).GetObject().(*swf.ScheduledWorkflow)
		nextID++
		object.UID = types.UID(fmt.Sprintf("destination-schedule-%d", nextID))
		return false, nil, nil
	})
	return transferSwfClient{client: c}
}

func TestTransferSchedulePreviewDisabledStageAndEnable(t *testing.T) {
	ctx := context.Background()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{})
	manager.swfClient = newTransferSwfClient()
	adapter := transferSchedules{r: manager}
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "Transfer", Namespace: "ns1"})
	require.NoError(t, err)
	job := model.Job{UUID: "source-job", K8SName: "source-name", Namespace: "ns1", DisplayName: "Schedule", ExperimentId: experiment.UUID, Enabled: true, PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())}}
	preview, err := adapter.Prepare(ctx, "source", &job, "digest", []byte(job.WorkflowSpecManifest), true)
	require.NoError(t, err)
	require.Equal(t, "source-job", preview.UUID)
	require.False(t, preview.Enabled)
	require.True(t, preview.NoCatchup)
	staged, err := adapter.Prepare(ctx, "source", &job, "digest", []byte(job.WorkflowSpecManifest), false)
	require.NoError(t, err)
	require.NotEqual(t, job.UUID, staged.UUID)
	require.NotEqual(t, job.K8SName, staged.K8SName)
	require.Equal(t, "pipeline-runner", staged.ServiceAccount)
	swf, err := manager.getScheduledWorkflowClient(staged.Namespace).Get(ctx, staged.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.False(t, swf.Spec.Enabled)
	require.NotNil(t, swf.Spec.NoCatchup)
	require.True(t, *swf.Spec.NoCatchup)
	require.Equal(t, staged.UUID, string(swf.UID))
	again, err := adapter.Prepare(ctx, "source", &job, "digest", []byte(job.WorkflowSpecManifest), false)
	require.NoError(t, err)
	require.Equal(t, staged.UUID, again.UUID)
	_, err = manager.jobStore.CreateJob(staged)
	require.NoError(t, err)
	require.NoError(t, manager.ChangeJobMode(ctx, staged.UUID, true))
	enabled, err := manager.GetJob(staged.UUID)
	require.NoError(t, err)
	require.True(t, enabled.Enabled)
	_, err = adapter.Prepare(ctx, "source", &job, "digest", []byte(job.WorkflowSpecManifest), false)
	require.ErrorContains(t, err, "changed")
	_, err = adapter.Prepare(ctx, "source", &job, "wrong-digest", []byte(job.WorkflowSpecManifest), true)
	require.ErrorContains(t, err, "provenance")
}

type transferPluginDispatcher struct {
	apiserverPlugins.RunPluginDispatcher
}

func (transferPluginDispatcher) PluginsRegistered() bool { return true }
func TestTransferScheduleUsesPluginAwarePreparation(t *testing.T) {
	ctx := context.Background()
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{})
	manager.swfClient = newTransferSwfClient()
	manager.pluginDispatcher = transferPluginDispatcher{}
	adapter := transferSchedules{r: manager}
	job := model.Job{UUID: "source-plugin-job", K8SName: "source", Namespace: "ns1", PipelineSpec: model.PipelineSpec{PipelineId: "local-pipeline", PipelineVersionId: "local-version", WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())}}
	staged, err := adapter.Prepare(ctx, "source", &job, "digest", []byte(job.WorkflowSpecManifest), false)
	require.NoError(t, err)
	swf, err := manager.getScheduledWorkflowClient(staged.Namespace).Get(ctx, staged.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "local-pipeline", swf.Spec.PipelineId)
	require.Equal(t, "local-version", swf.Spec.PipelineVersionId)
	require.Empty(t, swf.Spec.Workflow.Spec)
	require.False(t, swf.Spec.Enabled)
	require.Equal(t, "pipeline-runner", staged.ServiceAccount)
}
