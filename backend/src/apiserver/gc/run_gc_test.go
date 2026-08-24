// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gc

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRunStore records GC method calls; supports multi-batch sequences via archiveReturnSequence/deleteReturnSequence.
type fakeRunStore struct {
	archiveCalls          int
	archiveCutoff         int64
	archiveBatch          int
	archiveReturn         int64
	archiveReturnSequence []int64
	archiveErr            error

	deleteCalls          int
	deleteCutoff         int64
	deleteBatch          int
	deleteReturn         int64
	deleteReturnSequence []int64
	deleteErr            error
}

func (f *fakeRunStore) ArchiveExpiredRuns(archiveCutoffEpoch int64, batchSize int) (int64, error) {
	callIndex := f.archiveCalls
	f.archiveCalls++
	f.archiveCutoff = archiveCutoffEpoch
	f.archiveBatch = batchSize
	if f.archiveReturnSequence != nil && callIndex < len(f.archiveReturnSequence) {
		return f.archiveReturnSequence[callIndex], f.archiveErr
	}
	return f.archiveReturn, f.archiveErr
}

func (f *fakeRunStore) DeleteExpiredArchivedRuns(deleteCutoffEpoch int64, batchSize int) (int64, error) {
	callIndex := f.deleteCalls
	f.deleteCalls++
	f.deleteCutoff = deleteCutoffEpoch
	f.deleteBatch = batchSize
	if f.deleteReturnSequence != nil && callIndex < len(f.deleteReturnSequence) {
		return f.deleteReturnSequence[callIndex], f.deleteErr
	}
	return f.deleteReturn, f.deleteErr
}

// Stubs for the remaining RunStoreInterface methods.
func (f *fakeRunStore) CreateRun(_ *model.Run) (*model.Run, error) { return nil, nil }
func (f *fakeRunStore) GetRun(_ string) (*model.Run, error)        { return nil, nil }
func (f *fakeRunStore) ListRuns(_ *model.FilterContext, _ *list.Options) ([]*model.Run, int, string, error) {
	return nil, 0, "", nil
}
func (f *fakeRunStore) UpdateRun(_ *model.Run) error { return nil }
func (f *fakeRunStore) UpdateRunFromWorkflow(_ *model.Run, _ model.RuntimeState) (bool, error) {
	return false, nil
}
func (f *fakeRunStore) UpdateRunPluginsOutput(_ string, _ *model.LargeText) error { return nil }
func (f *fakeRunStore) ArchiveRun(_ string) error                                 { return nil }
func (f *fakeRunStore) UnarchiveRun(_ string) error                               { return nil }
func (f *fakeRunStore) DeleteRun(_ string) error                                  { return nil }
func (f *fakeRunStore) CreateMetric(_ *model.RunMetric) error                     { return nil }
func (f *fakeRunStore) TerminateRun(_ string) error                               { return nil }
func (f *fakeRunStore) GetRunByRecurringRunIDAndDisplayName(_, _ string) (string, error) {
	return "", nil
}
func (f *fakeRunStore) ClaimRunForRetry(_ string, _ bool) (string, string, int64, int64, error) {
	return "", "", 0, 0, nil
}
func (f *fakeRunStore) RollbackRetryClaim(_ string, _ string, _ string, _ int64, _ int64) error {
	return nil
}

func resetGCConfig() {
	viper.Set(common.RunsRetentionTime, "")
	viper.Set(common.ArchivedRunsRetentionTime, "")
	viper.Set(common.RunsGCBatchSize, "")
}

func TestCollect_BothDisabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	fake := &fakeRunStore{}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 1000000 },
	}

	gc.collect(context.Background())

	assert.Equal(t, 0, fake.archiveCalls, "archive should not be called when RUNS_RETENTION_TIME is empty")
	assert.Equal(t, 0, fake.deleteCalls, "delete should not be called when ARCHIVED_RUNS_RETENTION_TIME is empty")
}

func TestCollect_ArchiveOnlyEnabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")

	fake := &fakeRunStore{archiveReturn: 5}
	now := int64(3000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls, "archive pass should be invoked once")
	assert.Equal(t, now-2592000, fake.archiveCutoff, "archive cutoff = now minus 720h in seconds")
	assert.Equal(t, 100, fake.archiveBatch, "default batch size is 100")
	assert.Equal(t, 0, fake.deleteCalls, "delete should not be called when ARCHIVED_RUNS_RETENTION_TIME is empty")
}

func TestCollect_DeleteOnlyEnabled(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.ArchivedRunsRetentionTime, "2160h")

	fake := &fakeRunStore{deleteReturn: 3}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 0, fake.archiveCalls, "archive should not be called when RUNS_RETENTION_TIME is empty")
	assert.Equal(t, 1, fake.deleteCalls, "delete pass should be invoked once")
	assert.Equal(t, now-7776000, fake.deleteCutoff, "delete cutoff = now minus 2160h in seconds")
	assert.Equal(t, 100, fake.deleteBatch, "default batch size is 100")
}

func TestCollect_BothEnabled_CustomBatchSize(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.ArchivedRunsRetentionTime, "2160h")
	viper.Set(common.RunsGCBatchSize, "50")

	fake := &fakeRunStore{archiveReturn: 2, deleteReturn: 1}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls)
	assert.Equal(t, now-2592000, fake.archiveCutoff)
	assert.Equal(t, 50, fake.archiveBatch, "custom batch size should be respected")

	assert.Equal(t, 1, fake.deleteCalls)
	assert.Equal(t, now-7776000, fake.deleteCutoff)
	assert.Equal(t, 50, fake.deleteBatch, "custom batch size should be respected")
}

func TestCollect_ArchiveErrorDoesNotBlockDeletePass(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.ArchivedRunsRetentionTime, "2160h")

	fake := &fakeRunStore{
		archiveReturn: 0,
		archiveErr:    fmt.Errorf("db connection lost"),
		deleteReturn:  4,
	}

	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 10000000 },
	}

	gc.collect(context.Background())

	assert.Equal(t, 1, fake.archiveCalls, "archive pass should be attempted even if it will fail")
	assert.Equal(t, 1, fake.deleteCalls, "delete pass must still run after an archive error")
}

func TestCollect_CanceledContextExitsEarly(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")

	fake := &fakeRunStore{archiveReturn: 5}
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return 10000000 },
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	gc.collect(ctx)

	assert.Equal(t, 0, fake.archiveCalls, "canceled context should skip archive pass")
}

func TestCollect_ArchiveDrainsMultipleBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.RunsGCBatchSize, "100")

	fake := &fakeRunStore{
		archiveReturnSequence: []int64{100, 30},
	}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 2, fake.archiveCalls, "drain loop should call archive twice: full batch then partial batch")
}

func TestCollect_DeleteDrainsMultipleBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.ArchivedRunsRetentionTime, "2160h")
	viper.Set(common.RunsGCBatchSize, "100")

	fake := &fakeRunStore{
		deleteReturnSequence: []int64{100, 15},
	}
	now := int64(10000000)
	gc := &RunGarbageCollector{
		runStore: fake,
		nowFunc:  func() int64 { return now },
	}

	gc.collect(context.Background())

	assert.Equal(t, 2, fake.deleteCalls, "drain loop should call delete twice: full batch then partial batch")
}

func TestCollect_ContextCancelStopsDrainBetweenBatches(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")
	viper.Set(common.RunsGCBatchSize, "100")

	ctx, cancel := context.WithCancel(context.Background())

	cancelOnFirstCall := &cancellingFakeRunStore{
		fakeRunStore: fakeRunStore{
			archiveReturnSequence: []int64{100, 0},
		},
		cancelAfterArchiveN: 1,
		cancelFunc:          cancel,
	}

	gc := &RunGarbageCollector{
		runStore: cancelOnFirstCall,
		nowFunc:  func() int64 { return 10000000 },
	}

	gc.collect(ctx)

	assert.Equal(t, 1, cancelOnFirstCall.archiveCalls,
		"drain loop should stop after first batch when context is canceled between iterations")
}

// cancellingFakeRunStore wraps fakeRunStore and cancels a context after N archive calls.
type cancellingFakeRunStore struct {
	fakeRunStore
	cancelAfterArchiveN int
	cancelFunc          context.CancelFunc
}

func (f *cancellingFakeRunStore) ArchiveExpiredRuns(archiveCutoffEpoch int64, batchSize int) (int64, error) {
	result, archiveError := f.fakeRunStore.ArchiveExpiredRuns(archiveCutoffEpoch, batchSize)
	if f.archiveCalls >= f.cancelAfterArchiveN && f.cancelFunc != nil {
		f.cancelFunc()
	}
	return result, archiveError
}

type blockingRunStore struct {
	fakeRunStore
	archiveStarted chan struct{}
	releaseArchive chan struct{}
}

func (f *blockingRunStore) ArchiveExpiredRuns(_ int64, _ int) (int64, error) {
	close(f.archiveStarted)
	<-f.releaseArchive
	return 0, nil
}

func TestLeaderLifecycle_ShutdownKeepsElectionActiveUntilCollectionDrains(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()
	viper.Set(common.RunsRetentionTime, "1h")

	store := &blockingRunStore{
		archiveStarted: make(chan struct{}),
		releaseArchive: make(chan struct{}),
	}
	collector := &RunGarbageCollector{
		runStore: store,
		nowFunc:  func() int64 { return 1000000 },
	}

	shutdownCtx, cancelShutdown := context.WithCancel(context.Background())
	electionCtx, cancelElection := context.WithCancel(context.Background())
	defer cancelElection()

	lifecycle := newLeaderLifecycle(shutdownCtx, electionCtx, cancelElection, collector.runLoop)
	stopShutdownHook := context.AfterFunc(shutdownCtx, lifecycle.onShutdown)
	defer stopShutdownHook()

	callbackDone := make(chan struct{})
	go func() {
		defer close(callbackDone)
		lifecycle.onStartedLeading(electionCtx)
	}()

	requireSignal(t, store.archiveStarted, "collection did not reach the blocking store")

	cancelShutdown()

	require.Never(t, func() bool { return electionCtx.Err() != nil }, 50*time.Millisecond, time.Millisecond,
		"election stopped before the active collection drained")

	close(store.releaseArchive)

	requireSignal(t, callbackDone, "collection callback did not drain")
	requireSignal(t, electionCtx.Done(), "election did not stop after the collection drained")
	assert.True(t, lifecycle.isGracefulStop())
}

func TestLeaderLifecycle_ShutdownBeforeCallbackPreventsCollectionStart(t *testing.T) {
	shutdownCtx, cancelShutdown := context.WithCancel(context.Background())
	electionCtx, cancelElection := context.WithCancel(context.Background())
	defer cancelElection()

	var runLoopCalled atomic.Bool
	lifecycle := newLeaderLifecycle(shutdownCtx, electionCtx, cancelElection, func(context.Context) {
		runLoopCalled.Store(true)
	})
	stopShutdownHook := context.AfterFunc(shutdownCtx, lifecycle.onShutdown)
	defer stopShutdownHook()

	cancelShutdown()

	requireSignal(t, electionCtx.Done(), "idle election did not stop during shutdown")

	lifecycle.onStartedLeading(electionCtx)

	assert.False(t, runLoopCalled.Load(), "late callback started collection after shutdown")
	assert.True(t, lifecycle.isGracefulStop())
}

// A lease lost while shutdown is draining must still end with the election
// canceled once the collection callback returns; the pre-round-5 design kept
// gracefulStop=false here to arm a fatal fallback, which left the election
// loop running forever after shutdown (context.AfterFunc fires only once).
func TestLeaderLifecycle_UnexpectedLossDuringShutdownStopsElection(t *testing.T) {
	shutdownCtx, cancelShutdown := context.WithCancel(context.Background())
	electionCtx, cancelElection := context.WithCancel(context.Background())
	defer cancelElection()

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()

	runLoopStarted := make(chan struct{})
	releaseRunLoop := make(chan struct{})
	lifecycle := newLeaderLifecycle(shutdownCtx, electionCtx, cancelElection, func(ctx context.Context) {
		close(runLoopStarted)
		<-ctx.Done()
		<-releaseRunLoop
	})
	stopShutdownHook := context.AfterFunc(shutdownCtx, lifecycle.onShutdown)
	defer stopShutdownHook()

	callbackDone := make(chan struct{})
	go func() {
		defer close(callbackDone)
		lifecycle.onStartedLeading(leaderCtx)
	}()

	requireSignal(t, runLoopStarted, "collection callback did not start")

	cancelLeader()
	cancelShutdown()

	// While the collection is still draining, the stop is not yet graceful.
	require.Never(t, lifecycle.isGracefulStop, 50*time.Millisecond, time.Millisecond,
		"stop classified graceful while collection was still draining")

	close(releaseRunLoop)

	requireSignal(t, callbackDone, "collection callback did not exit")
	assert.True(t, lifecycle.isGracefulStop(), "shutdown after drain must be graceful even when the lease was also lost")
	select {
	case <-electionCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("election context not canceled; Start would re-enter the election after shutdown")
	}
}

func requireSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

// The index gate is evaluated on every tick, not once at startup, so applying
// the index migration takes effect without restarting the API server.
func TestCollect_IndexGateReevaluatedPerTick(t *testing.T) {
	resetGCConfig()
	defer resetGCConfig()

	viper.Set(common.RunsRetentionTime, "720h")

	fake := &fakeRunStore{}
	ready := false
	gc := &RunGarbageCollector{
		runStore:   fake,
		nowFunc:    func() int64 { return 3000000 },
		indexReady: func() bool { return ready },
	}

	gc.collect(context.Background())
	assert.Equal(t, 0, fake.archiveCalls, "collection must be skipped while the index is missing")

	ready = true
	gc.collect(context.Background())
	assert.Equal(t, 1, fake.archiveCalls, "collection must start once the index is ready, without a restart")
}

// Regression: a lease lost while a graceful shutdown is draining must still
// cancel the election once the collection loop returns. onShutdown has
// already fired (context.AfterFunc runs once), so if onStartedLeading does
// not cancel the election here, Start's re-election loop runs forever and
// the process's shutdown wait group never releases.
func TestLeaderLifecycle_LeaseLossDuringShutdownCancelsElection(t *testing.T) {
	shutdownCtx, shutdown := context.WithCancel(context.Background())
	electionCtx, cancelElection := context.WithCancel(context.Background())
	leaderCtx, loseLease := context.WithCancel(context.Background())

	collectionCanceled := make(chan struct{})
	lifecycle := newLeaderLifecycle(shutdownCtx, electionCtx, cancelElection, func(ctx context.Context) {
		// Simulate an in-flight collection: block until the shutdown path
		// cancels the collection context, then lose the lease before
		// returning — the race the regression targets.
		<-ctx.Done()
		close(collectionCanceled)
		loseLease()
	})

	done := make(chan struct{})
	go func() {
		lifecycle.onStartedLeading(leaderCtx)
		close(done)
	}()

	// Wait for the callback to register itself, then begin shutdown.
	for {
		lifecycle.mu.Lock()
		started := lifecycle.callbackStarted
		lifecycle.mu.Unlock()
		if started {
			break
		}
		time.Sleep(time.Millisecond)
	}
	shutdown()
	lifecycle.onShutdown()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("onStartedLeading did not return after shutdown")
	}
	<-collectionCanceled
	select {
	case <-electionCtx.Done():
		// Election canceled: Start's election loop can exit.
	case <-time.After(5 * time.Second):
		t.Fatal("election context was never canceled after lease loss during shutdown; Start would re-enter the election forever")
	}
	assert.True(t, lifecycle.isGracefulStop())
}

// Regression: client-go launches OnStartedLeading as an unjoined goroutine,
// so a re-entered election must not start a second collection loop while the
// previous callback is still draining, and Start's final join must not
// release until that drain completes.
func TestLeaderLifecycle_DrainGatePreventsOverlappingCallbacks(t *testing.T) {
	shutdownCtx, shutdown := context.WithCancel(context.Background())
	electionCtx, cancelElection := context.WithCancel(context.Background())
	leaderCtx, loseLease := context.WithCancel(context.Background())
	defer cancelElection()

	releaseRunLoop := make(chan struct{})
	runLoopStarted := make(chan struct{})
	lifecycle := newLeaderLifecycle(shutdownCtx, electionCtx, cancelElection, func(ctx context.Context) {
		close(runLoopStarted)
		<-ctx.Done()
		<-releaseRunLoop // simulate a database batch that cannot be interrupted
	})

	callbackDone := make(chan struct{})
	go func() {
		lifecycle.onStartedLeading(leaderCtx)
		close(callbackDone)
	}()
	requireSignal(t, runLoopStarted, "collection callback did not start")

	// Lease lost: LeaderElector.Run would return now, and the re-election
	// loop would attempt to acquire again while the callback still drains.
	loseLease()

	gateCtx, gateCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer gateCancel()
	assert.False(t, lifecycle.drainCallback(gateCtx),
		"drain gate must block re-election while the previous callback is draining")

	// Shutdown arrives mid-drain; the final join must wait for the drain.
	shutdown()
	lifecycle.onShutdown()
	joined := make(chan struct{})
	go func() {
		lifecycle.drainCallback(context.Background())
		close(joined)
	}()
	select {
	case <-joined:
		t.Fatal("final join released before the in-flight database work drained")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseRunLoop)
	requireSignal(t, callbackDone, "collection callback did not exit")
	requireSignal(t, joined, "final join did not release after the drain completed")
	select {
	case <-electionCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("election context not canceled after shutdown completed the drain")
	}
}

// Regression: client-go can dispatch OnStartedLeading and return from Run
// before the callback registers itself, so serialization must also be
// enforced inside onStartedLeading — a second callback must wait for the
// first to drain (or abandon its lease) before starting a collection loop.
func TestLeaderLifecycle_SecondCallbackWaitsForFirstToDrain(t *testing.T) {
	electionCtx, cancelElection := context.WithCancel(context.Background())
	defer cancelElection()
	leader1 := context.Background()
	leader2, cancelLeader2 := context.WithCancel(context.Background())

	var runLoopStarts atomic.Int32
	release := make(chan struct{})
	lifecycle := newLeaderLifecycle(context.Background(), electionCtx, cancelElection, func(ctx context.Context) {
		runLoopStarts.Add(1)
		<-release
	})

	go lifecycle.onStartedLeading(leader1)
	require.Eventually(t, func() bool { return runLoopStarts.Load() == 1 }, time.Second, time.Millisecond)

	// Second callback dispatched while the first is still draining: it must
	// wait, not start a second collection loop.
	secondDone := make(chan struct{})
	go func() {
		lifecycle.onStartedLeading(leader2)
		close(secondDone)
	}()
	require.Never(t, func() bool { return runLoopStarts.Load() > 1 }, 100*time.Millisecond, 5*time.Millisecond,
		"second callback must not start a collection loop while the first is draining")

	// Losing the waiting callback's lease releases it without running.
	cancelLeader2()
	requireSignal(t, secondDone, "waiting callback did not return after losing its lease")
	assert.Equal(t, int32(1), runLoopStarts.Load())

	// After the first drains, a fresh callback proceeds normally.
	close(release)
	thirdDone := make(chan struct{})
	go func() {
		lifecycle.onStartedLeading(context.Background())
		close(thirdDone)
	}()
	requireSignal(t, thirdDone, "third callback did not run after the first drained")
	assert.Equal(t, int32(2), runLoopStarts.Load())
}
