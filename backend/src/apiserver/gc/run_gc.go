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

// Package gc implements background garbage collection for expired pipeline runs.
package gc

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	leaseName  = "kfp-apiserver-gc"
	drainPause = 100 * time.Millisecond
	// reelectionBackoff is how long a replica waits before re-entering the
	// election after LeaderElector.Run returns (e.g. after a lost lease).
	reelectionBackoff = 5 * time.Second
)

type RunGarbageCollector struct {
	runStore  storage.RunStoreInterface
	clientset kubernetes.Interface
	namespace string
	nowFunc   func() int64
	// indexReady reports whether idx_run_gc_lifecycle currently exists and is
	// usable. It is re-evaluated on every collection tick so an operator can
	// apply the index migration without restarting the API server. A nil
	// checker means "always ready" (tests).
	indexReady func() bool
}

// leaderLifecycle keeps lease renewal active until a graceful shutdown has
// drained the current collection. Its state also distinguishes that path from
// an unexpected lease loss, after which Start re-enters the election.
type leaderLifecycle struct {
	shutdownCtx    context.Context
	electionCtx    context.Context
	cancelElection context.CancelFunc
	runLoop        func(context.Context)

	mu               sync.Mutex
	callbackStarted  bool
	cancelCollection context.CancelFunc
	gracefulStop     bool
}

func newLeaderLifecycle(
	shutdownCtx context.Context,
	electionCtx context.Context,
	cancelElection context.CancelFunc,
	runLoop func(context.Context),
) *leaderLifecycle {
	return &leaderLifecycle{
		shutdownCtx:    shutdownCtx,
		electionCtx:    electionCtx,
		cancelElection: cancelElection,
		runLoop:        runLoop,
	}
}

func (l *leaderLifecycle) onShutdown() {
	l.mu.Lock()
	if l.callbackStarted {
		l.cancelCollection()
		l.mu.Unlock()
		return
	}
	l.gracefulStop = true
	l.mu.Unlock()
	l.cancelElection()
}

func (l *leaderLifecycle) onStartedLeading(leaderCtx context.Context) {
	collectionCtx, cancelCollection := context.WithCancel(leaderCtx)
	defer cancelCollection()

	l.mu.Lock()
	if l.gracefulStop || l.electionCtx.Err() != nil {
		l.mu.Unlock()
		return
	}
	l.callbackStarted = true
	l.cancelCollection = cancelCollection
	if l.shutdownCtx.Err() != nil {
		cancelCollection()
	}
	l.mu.Unlock()

	l.runLoop(collectionCtx)

	// leaderCtx remains active while this process still holds the lease. If it
	// was canceled first, the lease was lost unexpectedly; reset the callback
	// state so a later shutdown cancels the election rather than a stale
	// collection context, and let Start's election loop re-enter the election.
	l.mu.Lock()
	l.callbackStarted = false
	graceful := l.shutdownCtx.Err() != nil && leaderCtx.Err() == nil
	if graceful {
		l.gracefulStop = true
	}
	l.mu.Unlock()
	if graceful {
		l.cancelElection()
	}
}

func (l *leaderLifecycle) isGracefulStop() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.gracefulStop
}

func NewRunGarbageCollector(
	runStore storage.RunStoreInterface,
	clientset kubernetes.Interface,
	namespace string,
	indexReady func() bool,
) *RunGarbageCollector {
	return &RunGarbageCollector{
		runStore:   runStore,
		clientset:  clientset,
		namespace:  namespace,
		nowFunc:    func() int64 { return time.Now().Unix() },
		indexReady: indexReady,
	}
}

// Start launches the GC loop. It blocks until ctx is canceled and runLoop has
// finished draining its current batch while the leader lease is still renewed.
func (gc *RunGarbageCollector) Start(ctx context.Context) {
	archiveRetention := common.GetRunsRetentionTime()
	deleteRetention := common.GetArchivedRunsRetentionTime()
	if archiveRetention == 0 && deleteRetention == 0 {
		glog.Info("Run GC disabled: both RUNS_RETENTION_TIME and ARCHIVED_RUNS_RETENTION_TIME are empty")
		return
	}
	glog.Infof("Run GC enabled: archive after %v, delete after %v, interval %v, batch %d",
		archiveRetention, deleteRetention, common.GetRunsGCInterval(), common.GetRunsGCBatchSize())

	id := os.Getenv("POD_NAME")
	if id == "" {
		var err error
		id, err = os.Hostname()
		if err != nil {
			glog.Errorf("Run GC: cannot determine pod identity, disabling GC: %v", err)
			return
		}
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: gc.namespace,
		},
		Client: gc.clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	// Do not stop lease renewal when shutdown begins. The collection callback
	// cancels this context only after its in-flight database operation returns.
	electionCtx, cancelElection := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelElection()

	lifecycle := newLeaderLifecycle(ctx, electionCtx, cancelElection, gc.runLoop)
	stopShutdownHook := context.AfterFunc(ctx, lifecycle.onShutdown)
	defer stopShutdownHook()

	electionConfig := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderContext context.Context) {
				glog.Info("Run GC: acquired leader lease, starting collection loop")
				lifecycle.onStartedLeading(leaderContext)
			},
			OnStoppedLeading: func() {
				if lifecycle.isGracefulStop() {
					glog.Info("Run GC: collection loop drained, graceful shutdown complete")
					return
				}
				// Losing the lease never terminates the process: GC is opt-in
				// housekeeping inside the user-facing API server, and the
				// store passes are safe under overlap (SELECT FOR UPDATE plus
				// predicate re-checks). Start re-enters the election below.
				glog.Errorf("Run GC: lost leader lease unexpectedly; collection stopped, re-entering election in %v", reelectionBackoff)
			},
			OnNewLeader: func(identity string) {
				if identity != id {
					glog.Infof("Run GC: new leader elected: %s", identity)
				}
			},
		},
	}
	if _, electionError := leaderelection.NewLeaderElector(electionConfig); electionError != nil {
		glog.Errorf("Run GC: invalid leader election config, disabling GC: %v", electionError)
		return
	}

	// LeaderElector.Run returns when the lease is lost (acquire -> renew ->
	// return); it never re-enters the election on its own. Loop so a
	// transient renewal failure pauses GC instead of disabling it for the
	// pod's lifetime. A LeaderElector is single-use, so build a fresh one
	// per attempt; the config was validated once above so per-attempt
	// construction cannot fail persistently.
	wait.UntilWithContext(electionCtx, func(loopCtx context.Context) {
		attemptElector, attemptError := leaderelection.NewLeaderElector(electionConfig)
		if attemptError != nil {
			glog.Errorf("Run GC: failed to recreate leader elector, retrying in %v: %v", reelectionBackoff, attemptError)
			return
		}
		attemptElector.Run(loopCtx)
	}, reelectionBackoff)
}

func (gc *RunGarbageCollector) runLoop(ctx context.Context) {
	interval := common.GetRunsGCInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately, then on each tick.
	gc.collect(ctx)

	for {
		select {
		case <-ctx.Done():
			glog.Info("Run GC: context canceled, exiting collection loop")
			return
		case <-ticker.C:
			gc.collect(ctx)
		}
	}
}

// sleepUnlessDone pauses between drain batches but returns false immediately
// if ctx is canceled, so shutdown never waits on a drain pause.
func sleepUnlessDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func (gc *RunGarbageCollector) collect(ctx context.Context) {
	// Re-check the index on every tick so applying the index migration takes
	// effect without an API-server restart, and dropping the index stops the
	// unindexed scans instead of silently continuing them.
	if gc.indexReady != nil && !gc.indexReady() {
		glog.Warning("Run GC: skipping collection pass, idx_run_gc_lifecycle is missing or not usable. " +
			"Apply the online index migration in docs/agents/development.md.")
		return
	}

	now := gc.nowFunc()
	batchSize := common.GetRunsGCBatchSize()

	// Pass 1: drain expired terminal runs into archived state.
	archiveRetention := common.GetRunsRetentionTime()
	if archiveRetention > 0 {
		expirationCutoffEpoch := now - int64(archiveRetention/time.Second)
		for {
			if ctx.Err() != nil {
				return
			}
			archivedRunCount, archiveError := gc.runStore.ArchiveExpiredRuns(expirationCutoffEpoch, batchSize)
			if archiveError != nil {
				glog.Errorf("Run GC archive pass failed: %v", archiveError)
				break
			}
			if archivedRunCount > 0 {
				glog.Infof("Run GC: archived %d expired runs (cutoff: %v ago)", archivedRunCount, archiveRetention)
			}
			if archivedRunCount < int64(batchSize) {
				break
			}
			if !sleepUnlessDone(ctx, drainPause) {
				return
			}
		}
	}

	// Pass 2: drain expired archived runs into permanent deletion.
	deleteRetention := common.GetArchivedRunsRetentionTime()
	if deleteRetention > 0 {
		expirationCutoffEpoch := now - int64(deleteRetention/time.Second)
		for {
			if ctx.Err() != nil {
				return
			}
			deletedRunCount, deleteError := gc.runStore.DeleteExpiredArchivedRuns(expirationCutoffEpoch, batchSize)
			if deleteError != nil {
				glog.Errorf("Run GC delete pass failed: %v", deleteError)
				break
			}
			if deletedRunCount > 0 {
				glog.Infof("Run GC: deleted %d expired archived runs (cutoff: %v ago)", deletedRunCount, deleteRetention)
			}
			if deletedRunCount < int64(batchSize) {
				break
			}
			if !sleepUnlessDone(ctx, drainPause) {
				return
			}
		}
	}
}
