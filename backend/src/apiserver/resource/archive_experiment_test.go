// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

// Partial-failure coverage for ArchiveExperiment, which disables every
// ScheduledWorkflow of the experiment's jobs in Kubernetes before writing the
// archived state to the database. The two steps are not atomic, so these
// tests pin the intermediate state after each kind of failure and verify that
// retrying the archive converges to a consistent result.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientv1beta1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// failingArchiveExperimentStore forwards every call to the wrapped store
// except ArchiveExperiment, which fails.
type failingArchiveExperimentStore struct {
	storage.ExperimentStoreInterface
}

func (f *failingArchiveExperimentStore) ArchiveExperiment(_ string) error {
	return errors.New("injected DB failure on ArchiveExperiment")
}

// statefulPatchSwfClient applies merge patches to the wrapped fake. The fake's
// own Patch only checks that the object exists, so without this a test cannot
// tell whether ArchiveExperiment actually disabled a ScheduledWorkflow.
// Patches of the ScheduledWorkflows named in failOn fail before any change.
type statefulPatchSwfClient struct {
	client.SwfClientInterface
	failOn  map[string]bool
	patched []string
}

func (c *statefulPatchSwfClient) ScheduledWorkflow(namespace string) swfclientv1beta1.ScheduledWorkflowInterface {
	return &statefulPatchSwfNamespace{
		ScheduledWorkflowInterface: c.SwfClientInterface.ScheduledWorkflow(namespace),
		parent:                     c,
	}
}

type statefulPatchSwfNamespace struct {
	swfclientv1beta1.ScheduledWorkflowInterface
	parent *statefulPatchSwfClient
}

func (c *statefulPatchSwfNamespace) Patch(ctx context.Context, name string, patchType k8stypes.PatchType, data []byte, subresources ...string) (*swfapi.ScheduledWorkflow, error) {
	if c.parent.failOn[name] {
		return nil, fmt.Errorf("injected K8s Patch failure for %s", name)
	}
	if patchType != k8stypes.MergePatchType {
		return nil, fmt.Errorf("unexpected patch type %q", patchType)
	}
	current, err := c.ScheduledWorkflowInterface.Patch(ctx, name, patchType, data, subresources...)
	if err != nil {
		return nil, err
	}
	// A JSON merge patch of scalar fields is equivalent to decoding the patch
	// over the current object.
	patched := current.DeepCopy()
	if err := json.Unmarshal(data, patched); err != nil {
		return nil, err
	}
	c.parent.patched = append(c.parent.patched, name)
	return c.Update(ctx, patched)
}

// useStatefulSwfClient routes the manager's ScheduledWorkflow patches through a
// statefulPatchSwfClient that fails for the given ScheduledWorkflow names.
func useStatefulSwfClient(manager *ResourceManager, failOn ...string) *statefulPatchSwfClient {
	swf := &statefulPatchSwfClient{SwfClientInterface: manager.swfClient, failOn: map[string]bool{}}
	for _, name := range failOn {
		swf.failOn[name] = true
	}
	manager.swfClient = swf
	return swf
}

// addSecondJob seeds a second enabled job, sorted after "j1", and its enabled
// ScheduledWorkflow. They are written to the stores directly because the fake
// ScheduledWorkflow client gives every created object the same UID, which
// CreateJob uses as the job UUID.
func addSecondJob(t *testing.T, store *FakeClientManager, first *model.Job) *model.Job {
	t.Helper()
	swf := &swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{Name: "job-2", Namespace: first.Namespace, UID: "aaa00000-0000-0000-0000-000000000002"},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: true},
	}
	_, err := store.SwfClient().ScheduledWorkflow(first.Namespace).Update(context.Background(), swf)
	require.NoError(t, err)
	second, err := store.JobStore().CreateJob(&model.Job{
		UUID:           string(swf.UID),
		DisplayName:    "j2",
		K8SName:        swf.Name,
		Namespace:      first.Namespace,
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		ExperimentId:   first.ExperimentId,
		PipelineSpec:   first.PipelineSpec,
		Conditions:     "STATUS_UNSPECIFIED",
	})
	require.NoError(t, err)
	return second
}

func swfEnabled(t *testing.T, store *FakeClientManager, job *model.Job) bool {
	t.Helper()
	swf, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(context.Background(), job.K8SName, v1.GetOptions{})
	require.NoError(t, err)
	return swf.Spec.Enabled
}

func requireExperimentState(t *testing.T, manager *ResourceManager, experimentID string, want model.StorageState) {
	t.Helper()
	experiment, err := manager.GetExperiment(experimentID)
	require.NoError(t, err)
	require.Equal(t, want, experiment.StorageState)
}

func requireJobEnabledInDB(t *testing.T, manager *ResourceManager, job *model.Job, want bool) {
	t.Helper()
	got, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, want, got.Enabled, "database Enabled of job %s", job.DisplayName)
}

// requireArchivedConsistently asserts the consistent end state of a successful
// archive: the experiment is archived and every job is disabled both in the
// database and in its ScheduledWorkflow.
func requireArchivedConsistently(t *testing.T, store *FakeClientManager, manager *ResourceManager, experimentID string, jobs ...*model.Job) {
	t.Helper()
	requireExperimentState(t, manager, experimentID, model.StorageStateArchived)
	for _, job := range jobs {
		requireJobEnabledInDB(t, manager, job, false)
		require.False(t, swfEnabled(t, store, job), "ScheduledWorkflow of job %s must be disabled", job.DisplayName)
	}
}

// The first ScheduledWorkflow patch fails: nothing changes, and a retry
// archives the experiment.
func TestArchiveExperiment_FirstPatchFails_NoStateChangeAndRetryConverges(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	require.True(t, swfEnabled(t, store, job), "precondition: ScheduledWorkflow starts enabled")
	swf := useStatefulSwfClient(manager, job.K8SName)

	err := manager.ArchiveExperiment(context.Background(), job.ExperimentId)
	require.ErrorContains(t, err, "Failed to disable job")
	require.Empty(t, swf.patched)
	require.True(t, swfEnabled(t, store, job))
	requireJobEnabledInDB(t, manager, job, true)
	requireExperimentState(t, manager, job.ExperimentId, model.StorageStateAvailable)

	delete(swf.failOn, job.K8SName)
	require.NoError(t, manager.ArchiveExperiment(context.Background(), job.ExperimentId))
	requireArchivedConsistently(t, store, manager, job.ExperimentId, job)
}

// A later ScheduledWorkflow patch fails: earlier ScheduledWorkflows stay
// disabled (there is no rollback), the database is untouched, and a retry
// re-applies the idempotent patches and archives the experiment.
func TestArchiveExperiment_LaterPatchFails_SplitStateAndRetryConverges(t *testing.T) {
	store, manager, first := initWithJob(t)
	defer store.Close()
	second := addSecondJob(t, store, first)
	require.True(t, swfEnabled(t, store, first), "precondition: ScheduledWorkflow starts enabled")
	require.True(t, swfEnabled(t, store, second), "precondition: ScheduledWorkflow starts enabled")
	swf := useStatefulSwfClient(manager, second.K8SName)

	err := manager.ArchiveExperiment(context.Background(), first.ExperimentId)
	require.ErrorContains(t, err, "Failed to disable job")
	require.Equal(t, []string{first.K8SName}, swf.patched, "jobs are processed in display-name order")
	require.False(t, swfEnabled(t, store, first), "the successful patch is not rolled back")
	require.True(t, swfEnabled(t, store, second))
	requireJobEnabledInDB(t, manager, first, true)
	requireJobEnabledInDB(t, manager, second, true)
	requireExperimentState(t, manager, first.ExperimentId, model.StorageStateAvailable)

	delete(swf.failOn, second.K8SName)
	require.NoError(t, manager.ArchiveExperiment(context.Background(), first.ExperimentId))
	requireArchivedConsistently(t, store, manager, first.ExperimentId, first, second)
}

// Every ScheduledWorkflow is disabled but the database write fails: the
// experiment and its jobs stay active in the database, and a retry after the
// database recovers archives the experiment.
func TestArchiveExperiment_DBWriteFails_SplitStateAndRetryConverges(t *testing.T) {
	store, manager, first := initWithJob(t)
	defer store.Close()
	second := addSecondJob(t, store, first)
	swf := useStatefulSwfClient(manager)
	realExperimentStore := manager.experimentStore
	manager.experimentStore = &failingArchiveExperimentStore{realExperimentStore}

	err := manager.ArchiveExperiment(context.Background(), first.ExperimentId)
	require.ErrorContains(t, err, "injected DB failure")
	require.Equal(t, []string{first.K8SName, second.K8SName}, swf.patched)
	require.False(t, swfEnabled(t, store, first))
	require.False(t, swfEnabled(t, store, second))
	requireJobEnabledInDB(t, manager, first, true)
	requireJobEnabledInDB(t, manager, second, true)
	requireExperimentState(t, manager, first.ExperimentId, model.StorageStateAvailable)

	manager.experimentStore = realExperimentStore
	require.NoError(t, manager.ArchiveExperiment(context.Background(), first.ExperimentId))
	requireArchivedConsistently(t, store, manager, first.ExperimentId, first, second)
}
