// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
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
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	leaseName  = "kfp-apiserver-gc"
	drainPause = 100 * time.Millisecond
)

type RunGarbageCollector struct {
	runStore  storage.RunStoreInterface
	clientset kubernetes.Interface
	namespace string
	nowFunc   func() int64
}

func NewRunGarbageCollector(
	runStore storage.RunStoreInterface,
	clientset kubernetes.Interface,
	namespace string,
) *RunGarbageCollector {
	return &RunGarbageCollector{
		runStore:  runStore,
		clientset: clientset,
		namespace: namespace,
		nowFunc:   func() int64 { return time.Now().Unix() },
	}
}

// Start launches the GC loop. It blocks until ctx is canceled and
// runLoop has finished draining its current batch.
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

	// loopDone is closed when runLoop exits so OnStoppedLeading can
	// wait for the current batch to drain.
	loopDone := make(chan struct{})
	loopStarted := false

	leaderElector, electionError := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderContext context.Context) {
				glog.Info("Run GC: acquired leader lease, starting collection loop")
				loopStarted = true
				gc.runLoop(leaderContext)
				close(loopDone)
			},
			OnStoppedLeading: func() {
				if ctx.Err() != nil {
					glog.Info("Run GC: leader lease released during graceful shutdown, waiting for collection loop to drain")
					if loopStarted {
						<-loopDone
					}
					glog.Info("Run GC: collection loop drained, shutdown complete")
					return
				}
				// Terminate to guarantee no concurrent collection (matches
				// kube-controller-manager pattern). K8s restarts the pod.
				glog.Fatalf("Run GC: lost leader lease unexpectedly, terminating to prevent concurrent collection")
			},
			OnNewLeader: func(identity string) {
				if identity != id {
					glog.Infof("Run GC: new leader elected: %s", identity)
				}
			},
		},
	})
	if electionError != nil {
		glog.Errorf("Run GC: failed to create leader elector, disabling GC: %v", electionError)
		return
	}

	leaderElector.Run(ctx)
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

func (gc *RunGarbageCollector) collect(ctx context.Context) {
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
			time.Sleep(drainPause)
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
			time.Sleep(drainPause)
		}
	}
}
