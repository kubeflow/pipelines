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
	leaseName = "kfp-apiserver-gc"
)

type RunGarbageCollector struct {
	runStore  storage.RunStoreInterface
	clientset kubernetes.Interface
	namespace string
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
	}
}

// Start launches the GC loop. It blocks until ctx is canceled.
func (gc *RunGarbageCollector) Start(ctx context.Context) {
	archiveRetention := common.GetRunsRetentionTime()
	deleteRetention := common.GetArchivedRunsRetentionTime()

	if archiveRetention == 0 && deleteRetention == 0 {
		glog.Info("Run GC disabled: both RunsRetentionTime and ArchivedRunsRetentionTime are 0")
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

	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				glog.Info("Run GC: acquired leader lease, starting collection loop")
				gc.runLoop(ctx)
			},
			OnStoppedLeading: func() {
				glog.Info("Run GC: lost leader lease, stopping collection loop")
			},
			OnNewLeader: func(identity string) {
				if identity != id {
					glog.Infof("Run GC: new leader elected: %s", identity)
				}
			},
		},
	})
	if err != nil {
		glog.Errorf("Run GC: failed to create leader elector, disabling GC: %v", err)
		return
	}
	le.Run(ctx)
}

func (gc *RunGarbageCollector) runLoop(ctx context.Context) {
	interval := common.GetRunsGCInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately, then on each tick.
	gc.collect()

	for {
		select {
		case <-ctx.Done():
			glog.Info("Run GC: context canceled, exiting collection loop")
			return
		case <-ticker.C:
			gc.collect()
		}
	}
}

func (gc *RunGarbageCollector) collect() {
	now := time.Now().Unix()
	batchSize := common.GetRunsGCBatchSize()

	// Pass 1: archive terminal active runs past retention.
	archiveRetention := common.GetRunsRetentionTime()
	if archiveRetention > 0 {
		cutoff := now - int64(archiveRetention.Seconds())
		archived, err := gc.runStore.ArchiveExpiredRuns(cutoff, batchSize)
		if err != nil {
			glog.Errorf("Run GC archive pass failed: %v", err)
		} else if archived > 0 {
			glog.Infof("Run GC: archived %d expired runs (cutoff: %v ago)", archived, archiveRetention)
		}
	}

	// Pass 2: delete archived runs past retention.
	deleteRetention := common.GetArchivedRunsRetentionTime()
	if deleteRetention > 0 {
		cutoff := now - int64(deleteRetention.Seconds())
		deleted, err := gc.runStore.DeleteExpiredArchivedRuns(cutoff, batchSize)
		if err != nil {
			glog.Errorf("Run GC delete pass failed: %v", err)
		} else if deleted > 0 {
			glog.Infof("Run GC: deleted %d expired archived runs (cutoff: %v ago)", deleted, deleteRetention)
		}
	}
}
