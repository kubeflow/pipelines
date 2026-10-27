package poc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

type leaseLifecycleExecutor struct{ fakeExecutorLifecycle }

func (leaseLifecycleExecutor) Type() string { return ExecutorKubernetes }

func (leaseLifecycleExecutor) Capabilities() ExecutorCapabilities {
	return ExecutorCapabilities{
		RecoverLostTasks: true, SupportsContainers: true, SupportsKubernetes: true,
		SupportedWorkspaceProvisioners: []string{WorkspaceTypePVC},
	}
}

type replicaBackend struct {
	leaseLifecycleExecutor
	mu        sync.Mutex
	requests  map[string]ExecutionReference
	complete  chan struct{}
	submitted chan struct{}
	waits     atomic.Int32
	cancels   atomic.Int32
}

func (e *replicaBackend) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if existing, ok := e.requests[request.Identity.AttemptID]; ok {
		return existing, nil
	}
	reference := ExecutionReference{
		Executor: ExecutorKubernetes, Version: 1, Identity: request.Identity,
		Data: model.JSONData{
			"namespace": request.Identity.Namespace, "podName": attemptWorkloadName(request.Identity),
			"podUID": "uid-" + request.Identity.AttemptID,
		},
	}
	e.requests[request.Identity.AttemptID] = reference
	if len(e.requests) == 1 {
		close(e.submitted)
	}
	return reference, nil
}

func (e *replicaBackend) Wait(ctx context.Context, _ ExecutionReference) (TaskExecutionResult, error) {
	e.waits.Add(1)
	select {
	case <-ctx.Done():
		return TaskExecutionResult{}, ctx.Err()
	case <-e.complete:
		return TaskExecutionResult{Outcome: TaskOutcomeSuccess}, nil
	}
}

func (e *replicaBackend) Recover(_ context.Context, identity TaskAttemptIdentity, _ *ExecutionReference) (RecoveryResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	reference, ok := e.requests[identity.AttemptID]
	if !ok {
		return RecoveryResult{State: RecoveryUnknown}, nil
	}
	return RecoveryResult{State: RecoveryExisting, Reference: &reference}, nil
}

func (e *replicaBackend) Cancel(context.Context, ExecutionReference) (CancellationResult, error) {
	e.cancels.Add(1)
	return CancellationResult{State: CancellationConfirmed}, nil
}

func (e *replicaBackend) Logs(_ context.Context, reference ExecutionReference) (ExecutionLogSource, error) {
	return ExecutionLogSource{Identity: reference.Identity, stream: func(ctx context.Context, follow bool, destination io.Writer) error {
		return e.ReadLogs(ctx, reference, follow, destination)
	}}, nil
}

func (e *replicaBackend) ReadLogs(_ context.Context, _ ExecutionReference, _ bool, destination io.Writer) error {
	_, err := io.WriteString(destination, "replica workload logs\n")
	return err
}

func replicaFixture(t *testing.T) (*Coordinator, *Coordinator, *replicaBackend, *model.Run) {
	t.Helper()
	proxy.InitializeConfigWithEmptyForTests()
	t.Cleanup(proxy.InitializeConfigWithEmptyForTests)
	root, bucket := t.TempDir(), t.TempDir()
	for key, value := range map[string]any{
		common.V2RuntimeExecutor: ExecutorKubernetes, common.V2RuntimeAutoExecute: true,
		common.V2RuntimeWorkspaceRoot: root, "ObjectStoreConfig.BucketURL": "file://" + bucket,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	t.Setenv("KFP_DEFAULT_PIPELINE_ROOT", "file://"+bucket)
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	clock, generator := util.NewFakeTimeForEpoch(), util.NewUUIDGenerator()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, generator, dialect)
	artifacts := storage.NewArtifactStore(db, clock, generator, dialect)
	links := storage.NewArtifactTaskStore(db, generator, dialect)
	manifest, err := NewRuntimeManifest(ExecutorKubernetes).ToJSON()
	require.NoError(t, err)
	pending := &model.Run{
		UUID: "replica-run", Namespace: "ns", K8SName: "replica-run", ServiceAccount: "pipeline-runner",
		RunDetails: model.RunDetails{
			State: model.RuntimeStatePending, ExecutorType: util.StringPointer(ExecutorKubernetes),
			PipelineRuntimeManifest: model.LargeText(manifest), CreatedAtInSec: 100,
		},
	}
	pending.PipelineRoot = model.LargeText("file://" + bucket)
	pending.PipelineSpecManifest = `{"pipelineInfo":{"name":"replica-test"},"schemaVersion":"2.1.0","components":{"task":{"executorLabel":"task"}},"deploymentSpec":{"executors":{"task":{"container":{"image":"busybox:1.36","command":["true"]}}}},"root":{"dag":{"tasks":{"task":{"taskInfo":{"name":"task"},"componentRef":{"name":"task"}}}}}}`
	run, err := runs.CreateRun(pending)
	require.NoError(t, err)
	backend := &replicaBackend{
		requests: map[string]ExecutionReference{}, submitted: make(chan struct{}), complete: make(chan struct{}),
	}
	workers := make([]*Coordinator, 2)
	for index := range workers {
		workers[index] = NewCoordinator(runs, tasks, artifacts, links, clock, &fakeKubernetesCoreWithClientset{clientSet: kubernetesfake.NewSimpleClientset()}, nil, false)
		workers[index].RegisterExecutor(backend)
		workers[index].leaseDuration = 500 * time.Millisecond
		workers[index].leaseRenewInterval = 50 * time.Millisecond
		workers[index].leasePollInterval = 20 * time.Millisecond
		worker := workers[index]
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, worker.Shutdown(ctx))
		})
	}
	return workers[0], workers[1], backend, run
}

func workerOwnsRun(worker *Coordinator, runID string) bool {
	worker.activeRunMu.Lock()
	defer worker.activeRunMu.Unlock()
	return worker.ownedRuns[runID] != nil
}

func TestReplicasRenewOwnershipWhileAWorkloadWaits(t *testing.T) {
	first, second, backend, run := replicaFixture(t)
	require.NoError(t, first.Start(context.Background()))
	require.NoError(t, second.Start(context.Background()))
	select {
	case <-backend.submitted:
	case <-time.After(5 * time.Second):
		t.Fatal("workers did not submit the managed run")
	}
	require.Eventually(t, func() bool { return backend.waits.Load() == 1 }, 5*time.Second, time.Millisecond)
	require.Never(t, func() bool { return backend.waits.Load() != 1 }, 2*first.leaseDuration, 20*time.Millisecond,
		"renewal must run independently of the blocked observer")
	close(backend.complete)
	require.Eventually(t, func() bool {
		current, err := first.runStore.GetRun(run.UUID, false)
		return err == nil && current.State == model.RuntimeStateSucceeded
	}, 5*time.Second, 10*time.Millisecond)
	backend.mu.Lock()
	require.Len(t, backend.requests, 1)
	backend.mu.Unlock()
	require.Zero(t, backend.cancels.Load())
}

type pausedRenewalStore struct {
	storage.RunLeaseStoreInterface
	paused chan struct{}
	resume chan struct{}
	once   sync.Once
}

func (s *pausedRenewalStore) Renew(token storage.RunLeaseToken, ttl time.Duration) error {
	s.once.Do(func() {
		close(s.paused)
		<-s.resume
	})
	return s.RunLeaseStoreInterface.Renew(token, ttl)
}

func TestReplicaTakesOverAnExpiredLeaseWithoutCancelingOrReplayingWork(t *testing.T) {
	first, second, backend, run := replicaFixture(t)
	paused := &pausedRenewalStore{
		RunLeaseStoreInterface: first.leaseStore, paused: make(chan struct{}), resume: make(chan struct{}),
	}
	var resume sync.Once
	t.Cleanup(func() { resume.Do(func() { close(paused.resume) }) })
	first.leaseStore = paused
	require.NoError(t, first.Start(context.Background()))
	select {
	case <-backend.submitted:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker never submitted its attempt")
	}
	select {
	case <-paused.paused:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker never began renewal")
	}
	oldToken := storage.RunLeaseToken{RunUUID: run.UUID, OwnerID: first.ownerID, Generation: 1}
	staleStores, err := first.leaseProvider.LeaseScopedStores(oldToken)
	require.NoError(t, err)
	require.NoError(t, second.Start(context.Background()))
	require.Eventually(t, func() bool { return workerOwnsRun(second, run.UUID) && backend.waits.Load() == 2 },
		5*time.Second, 10*time.Millisecond, "a second replica must recover the same attempt after database-time expiry")
	require.ErrorIs(t, first.leaseStore.Check(oldToken), storage.ErrRunLeaseConflict)
	err = staleStores.Runtime.FinalizeRun(run.UUID, model.RuntimeStateFailed, 1)
	require.ErrorIs(t, err, storage.ErrRunLeaseConflict, "a paused old coordinator must not mutate the replacement owner's run")
	resume.Do(func() { close(paused.resume) })
	require.Eventually(t, func() bool { return !workerOwnsRun(first, run.UUID) }, 5*time.Second, time.Millisecond)
	require.Zero(t, backend.cancels.Load())
	close(backend.complete)
	require.Eventually(t, func() bool {
		current, err := second.runStore.GetRun(run.UUID, false)
		return err == nil && current.State == model.RuntimeStateSucceeded
	}, 5*time.Second, 10*time.Millisecond)
	backend.mu.Lock()
	require.Len(t, backend.requests, 1)
	backend.mu.Unlock()
}

func TestAPIOnlyReplicaDoesNotClaimOrMaterializeWork(t *testing.T) {
	first, _, backend, run := replicaFixture(t)
	previous := viper.Get(common.V2RuntimeAutoExecute)
	viper.Set(common.V2RuntimeAutoExecute, false)
	t.Cleanup(func() { viper.Set(common.V2RuntimeAutoExecute, previous) })
	require.NoError(t, first.Start(context.Background()))
	require.NoError(t, first.ReconcileManagedRuns(context.Background()))
	require.Empty(t, backend.requests)
	require.Nil(t, first.workerContext)
	token, err := first.leaseStore.Acquire(run.UUID, "independent-worker", time.Second, false)
	require.NoError(t, err, "an API-only replica must leave pending work available to an independent worker")
	require.NoError(t, first.leaseStore.Release(*token))
}

func TestWorkerAcquiresCancellationAfterThePreviousOwnerExpires(t *testing.T) {
	first, second, backend, run := replicaFixture(t)
	first.leasePollInterval = time.Hour
	paused := &pausedRenewalStore{
		RunLeaseStoreInterface: first.leaseStore, paused: make(chan struct{}), resume: make(chan struct{}),
	}
	var resume sync.Once
	t.Cleanup(func() { resume.Do(func() { close(paused.resume) }) })
	first.leaseStore = paused
	require.NoError(t, first.Start(context.Background()))
	select {
	case <-backend.submitted:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker did not submit the attempt")
	}
	select {
	case <-paused.paused:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker did not reach the paused renewal")
	}
	for index := 0; index < leaseClaimBatchSize; index++ {
		older := *run
		older.UUID = fmt.Sprintf("older-recovery-%d", index)
		older.K8SName = older.UUID
		older.State = model.RuntimeStateRunning
		older.CreatedAtInSec = int64(index + 1)
		_, err := first.runStore.CreateRun(&older)
		require.NoError(t, err)
		task, err := first.createTask(&older, nil, "recovery", "recovery", apiv2.PipelineTask_RUNTIME, "root.recovery", nil)
		require.NoError(t, err)
		require.NoError(t, first.runtimeStore.Update(&model.RuntimeTask{
			TaskUUID: task.UUID, RunUUID: older.UUID, State: model.RuntimeTaskRunning,
			AttemptUUID: util.StringPointer(fmt.Sprintf("older-attempt-%d", index)),
		}, ""))
	}
	require.NoError(t, first.runStore.TerminateRun(run.UUID))
	dispatcher := second.dispatcher(backend)
	dispatcher.limit, dispatcher.active = 1, 1
	require.NoError(t, second.Start(context.Background()))
	require.Eventually(t, func() bool {
		current, err := second.runStore.GetRun(run.UUID, false)
		return err == nil && current.State == model.RuntimeStateCanceled
	}, 5*time.Second, 10*time.Millisecond, "a persisted cancellation must not depend on the original owner or API handler surviving")
	resume.Do(func() { close(paused.resume) })
	require.Equal(t, int32(1), backend.cancels.Load())
	dispatcher.mu.Lock()
	require.Equal(t, 1, dispatcher.active, "recovered cancellation must not consume or release another run's occupied task slot")
	dispatcher.mu.Unlock()
	backend.mu.Lock()
	require.Len(t, backend.requests, 1)
	backend.mu.Unlock()
}

func TestLeaseClaimBudgetUsesTaskCapacityAndBoundsUnlimitedPrefetch(t *testing.T) {
	first, _, backend, _ := replicaFixture(t)
	dispatcher := first.dispatcher(backend)
	require.Equal(t, leaseClaimBatchSize, first.availableLeaseClaims())
	dispatcher.limit = 1
	dispatcher.active = 1
	require.Zero(t, first.availableLeaseClaims(), "a worker with no compatible task capacity must not acquire fresh runs")
	dispatcher.active = 0
	require.Equal(t, 1, first.availableLeaseClaims())
	dispatcher.waiting = []*dispatchTicket{{}}
	require.Zero(t, first.availableLeaseClaims(), "already admitted dispatch backlog must count against prefetch")
	dispatcher.limit = 0
	require.Equal(t, leaseClaimBatchSize, first.availableLeaseClaims(), "unlimited executors still yield after a bounded claim batch")
}

func (leaseLifecycleExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}

func (leaseLifecycleExecutor) Wait(ctx context.Context, _ ExecutionReference) (TaskExecutionResult, error) {
	<-ctx.Done()
	return TaskExecutionResult{}, ctx.Err()
}

func (leaseLifecycleExecutor) Recover(context.Context, TaskAttemptIdentity, *ExecutionReference) (RecoveryResult, error) {
	return RecoveryResult{State: RecoveryUnknown}, nil
}

func TestKubernetesWorkersDoNotRequireExclusiveInstallationLock(t *testing.T) {
	for key, value := range map[string]any{
		common.V2RuntimeExecutor:      ExecutorKubernetes,
		common.V2RuntimeAutoExecute:   true,
		common.V2RuntimeWorkspaceRoot: t.TempDir(),
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	first, _, _ := attemptFixture(t)
	second := NewCoordinator(first.runStore, first.taskStore, nil, nil, first.time, nil, nil, false)
	for _, worker := range []*Coordinator{first, second} {
		worker.RegisterExecutor(leaseLifecycleExecutor{})
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, worker.Shutdown(ctx))
		})
		require.NoError(t, worker.Start(context.Background()), "Kubernetes workers must coordinate through database leases, not a host installation lock")
	}
}
