package resource

// Tests for ArchiveExperiment partial-failure scenarios.
// The existing TestArchiveExperiment_CustomResourceNotFound covers the NotFound
// (graceful) path.  These three tests cover the error paths where K8s and DB
// can fall out of sync.

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientv1beta1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// failingArchiveExperimentStore wraps a real ExperimentStore and returns an
// injected error only from ArchiveExperiment; all other methods are forwarded.
type failingArchiveExperimentStore struct {
	storage.ExperimentStoreInterface
}

func (f *failingArchiveExperimentStore) ArchiveExperiment(_ string) error {
	return errors.New("injected DB failure on ArchiveExperiment")
}

// patchCountingSwfClientFull wraps an existing SwfClientInterface and counts
// every Patch call.  The existing patchCountingSwfClient defined in
// resource_manager_test.go is identical — we define a new one here so this
// file can be compiled in isolation.
type patchCountingSwfClientFull struct {
	client.SwfClientInterface
	patchCalls int
}

func (c *patchCountingSwfClientFull) ScheduledWorkflow(namespace string) swfclientv1beta1.ScheduledWorkflowInterface {
	return &patchCountingSwfNsFull{
		ScheduledWorkflowInterface: c.SwfClientInterface.ScheduledWorkflow(namespace),
		counter:                    &c.patchCalls,
	}
}

type patchCountingSwfNsFull struct {
	swfclientv1beta1.ScheduledWorkflowInterface
	counter *int
}

func (c *patchCountingSwfNsFull) Patch(ctx context.Context, name string, patchType k8stypes.PatchType, data []byte, subresources ...string) (*swfapi.ScheduledWorkflow, error) {
	*c.counter++
	return c.ScheduledWorkflowInterface.Patch(ctx, name, patchType, data, subresources...)
}

// failOnNthPatchSwfClient returns an injected K8s error on the nth Patch call.
// Patches 1..(n-1) succeed; patch n fails; patches after n are unreachable.
type failOnNthPatchSwfClient struct {
	client.SwfClientInterface
	n   int
	cnt int
}

func (c *failOnNthPatchSwfClient) ScheduledWorkflow(namespace string) swfclientv1beta1.ScheduledWorkflowInterface {
	return &failOnNthPatchSwfNs{
		ScheduledWorkflowInterface: c.SwfClientInterface.ScheduledWorkflow(namespace),
		parent:                     c,
	}
}

type failOnNthPatchSwfNs struct {
	swfclientv1beta1.ScheduledWorkflowInterface
	parent *failOnNthPatchSwfClient
}

func (c *failOnNthPatchSwfNs) Patch(ctx context.Context, name string, patchType k8stypes.PatchType, data []byte, subresources ...string) (*swfapi.ScheduledWorkflow, error) {
	c.parent.cnt++
	if c.parent.cnt == c.parent.n {
		return nil, errors.New("injected K8s Patch failure")
	}
	return c.ScheduledWorkflowInterface.Patch(ctx, name, patchType, data, subresources...)
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

// TestArchiveExperiment_K8sFailBeforeAnyPatch_NoStateChange demonstrates that
// when ArchiveExperiment fails on its very first ScheduledWorkflow Patch call,
// the function returns a non-nil error and the experiment remains unarchived in
// the database.  This documents the early-return behaviour: the DB write at
// the end of ArchiveExperiment is never reached.
func TestArchiveExperiment_K8sFailBeforeAnyPatch_NoStateChange(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	// Replace the SWF client with one whose Patch always fails (non-NotFound).
	manager.swfClient = client.NewFakeSwfClientWithBadWorkflow()

	err := manager.ArchiveExperiment(context.Background(), job.ExperimentId)
	require.NotNil(t, err, "ArchiveExperiment must return an error when the first K8s Patch fails")
	require.Contains(t, err.Error(), "Failed to disable job",
		"error should name the failing step")

	// The DB write (experimentStore.ArchiveExperiment) was never reached, so
	// the experiment should still be active.
	exp, getErr := manager.GetExperiment(job.ExperimentId)
	require.Nil(t, getErr)
	require.NotEqual(t, model.StorageStateArchived, exp.StorageState,
		"experiment must NOT be archived in the DB when the K8s patch fails")
	t.Logf("experiment StorageState after K8s fail: %q (not archived — correct)", exp.StorageState)
}

// TestArchiveExperiment_PartialK8sPatch_K8sFailMidway_DBNotCalled demonstrates
// the atomicity gap: when there are N jobs, ArchiveExperiment patches them one
// by one.  If patch 1 succeeds but patch 2 fails, the function returns an
// error without calling experimentStore.ArchiveExperiment — the DB is left
// clean, but job 1's ScheduledWorkflow is already disabled in K8s.
//
// This is a correctness gap: there is no rollback of successful K8s patches.
// On retry the already-disabled SWF will be patched again (idempotent), so
// the net effect is acceptable, but the split state window exists.
//
// Note on fake infrastructure: the fake SWF client assigns every SWF the same
// UID (and therefore the same job UUID), so we cannot call manager.CreateJob
// twice.  Instead we seed the second job and SWF directly into the fake stores
// without going through the resource-manager layer.
func TestArchiveExperiment_PartialK8sPatch_K8sFailMidway_DBNotCalled(t *testing.T) {
	store, manager, job1 := initWithJob(t)
	defer store.Close()

	// Seed a second SWF directly into the fake K8s store.
	// FakeScheduledWorkflowClient.Update writes directly to the in-memory map.
	secondSwfName := "job-2"
	secondJobUUID := "aaa00000-0000-0000-0000-000000000002"
	secondSwf := &swfapi.ScheduledWorkflow{}
	secondSwf.Name = secondSwfName
	secondSwf.UID = "aaa00000-0000-0000-0000-000000000002"
	_, err := store.SwfClient().ScheduledWorkflow(job1.Namespace).Update(
		context.Background(), secondSwf)
	require.Nil(t, err, "inject second SWF")

	// Seed a matching job record directly into the DB.
	_, err = store.JobStore().CreateJob(&model.Job{
		UUID:           secondJobUUID,
		DisplayName:    "j2",
		K8SName:        secondSwfName,
		Namespace:      job1.Namespace,
		ServiceAccount: "pipeline-runner",
		Enabled:        true,
		ExperimentId:   job1.ExperimentId,
		PipelineSpec:   job1.PipelineSpec,
		Conditions:     "STATUS_UNSPECIFIED",
	})
	require.Nil(t, err, "inject second job into DB")

	// Inject a SWF client that fails only on the 2nd Patch call.
	manager.swfClient = &failOnNthPatchSwfClient{
		SwfClientInterface: manager.swfClient,
		n:                  2,
	}

	archiveErr := manager.ArchiveExperiment(context.Background(), job1.ExperimentId)
	require.NotNil(t, archiveErr, "ArchiveExperiment must fail when K8s fails mid-loop")
	require.Contains(t, archiveErr.Error(), "Failed to disable job",
		"error should name the failing step")

	// DB should NOT be updated.
	gotExp, getErr := manager.GetExperiment(job1.ExperimentId)
	require.Nil(t, getErr)
	require.NotEqual(t, model.StorageStateArchived, gotExp.StorageState,
		"experiment must NOT be archived in the DB when K8s fails mid-loop")
	t.Logf("experiment StorageState after mid-loop K8s fail: %q (not archived — correct)", gotExp.StorageState)
	t.Log("NOTE: the first job's ScheduledWorkflow WAS patched in K8s — split state window exists until retry")
}

// TestArchiveExperiment_AllPatchesSucceed_DBFails_SplitState demonstrates the
// opposite split: all ScheduledWorkflow Patch calls succeed, then
// experimentStore.ArchiveExperiment returns an error.  The result is a
// permanent split: K8s SWFs are disabled but the DB still marks the experiment
// as active.  Callers get an error, but a retry would succeed (idempotent K8s
// patches + DB write).
func TestArchiveExperiment_AllPatchesSucceed_DBFails_SplitState(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()

	// Count how many Patch calls K8s receives so we can assert they happened.
	patchCounter := &patchCountingSwfClientFull{SwfClientInterface: manager.swfClient}
	manager.swfClient = patchCounter

	// Save the real experiment store before wrapping it so we can query it after the test.
	realExperimentStore := manager.experimentStore

	// Replace the experiment store with a wrapper that fails only on ArchiveExperiment.
	manager.experimentStore = &failingArchiveExperimentStore{manager.experimentStore}

	archiveErr := manager.ArchiveExperiment(context.Background(), job.ExperimentId)
	require.NotNil(t, archiveErr, "ArchiveExperiment must return an error when the DB write fails")
	require.Contains(t, archiveErr.Error(), "injected DB failure",
		"error should originate from the injected DB failure")

	// At least one K8s Patch was issued before the DB call.
	require.Greater(t, patchCounter.patchCalls, 0,
		"K8s patches must have been issued before the DB failure")
	t.Logf("K8s Patch calls made: %d (SWF(s) disabled in K8s)", patchCounter.patchCalls)

	// The experiment is NOT archived in the DB — the wrapper's error prevented the write.
	// Read directly from the real store (bypasses the failing wrapper).
	gotExp, getErr := realExperimentStore.GetExperiment(job.ExperimentId)
	require.Nil(t, getErr)
	require.NotEqual(t, model.StorageStateArchived, gotExp.StorageState,
		"experiment must NOT be archived in the DB when experimentStore.ArchiveExperiment fails")
	t.Logf("experiment StorageState after DB fail: %q (not archived — split state documented)", gotExp.StorageState)
	t.Log("SPLIT STATE: K8s SWFs were disabled but DB still marks experiment active")
	t.Log("On retry: K8s patches are idempotent; the DB write will succeed — eventually consistent")
}
