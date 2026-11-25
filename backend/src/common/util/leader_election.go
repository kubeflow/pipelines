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

package util

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// LeaderElectionConfig configures RunWithLeaderElection.
type LeaderElectionConfig struct {
	// LeaseName is the name of the coordination.k8s.io Lease used as the lock.
	LeaseName string
	// Namespace is the namespace of the Lease.
	Namespace string
	// Identity identifies this replica in the Lease, usually the Pod name.
	Identity string
	// Run is called only on the replica that holds the lease. Its context is
	// canceled when the lease is lost or the parent context is done.
	Run func(leaderContext context.Context)
	// OnLost is called when the lease is lost before the parent context is
	// done. Callers exit the process here, because a replica that lost the
	// lease cannot safely keep or reuse its work queue.
	OnLost func()
}

type leaderElectionTimings struct {
	leaseDuration time.Duration
	renewDeadline time.Duration
	retryPeriod   time.Duration
}

// Kubernetes' default leader election timings, also used by the API server's
// run garbage collection.
var defaultLeaderElectionTimings = leaderElectionTimings{
	leaseDuration: 15 * time.Second,
	renewDeadline: 10 * time.Second,
	retryPeriod:   2 * time.Second,
}

// LeaderElectionIdentity returns the POD_NAME environment variable, or the
// hostname (which is the Pod name by default) if POD_NAME is not set.
func LeaderElectionIdentity() (string, error) {
	if podName := os.Getenv("POD_NAME"); podName != "" {
		return podName, nil
	}
	return os.Hostname()
}

// RunWithLeaderElection calls config.Run only while this replica holds the
// lease. It returns when electionContext is done or the lease is lost, after
// config.Run has returned. Losing the lease before electionContext is done
// also calls config.OnLost.
//
// ReleaseOnCancel is false: on shutdown, the lease is kept until it expires,
// so another replica does not start while requests sent by this one may still
// be in flight.
func RunWithLeaderElection(electionContext context.Context, client kubernetes.Interface, config LeaderElectionConfig) error {
	return runWithLeaderElection(electionContext, client, config, defaultLeaderElectionTimings)
}

func runWithLeaderElection(
	electionContext context.Context,
	client kubernetes.Interface,
	config LeaderElectionConfig,
	timings leaderElectionTimings,
) error {
	if config.LeaseName == "" || config.Namespace == "" || config.Identity == "" {
		return fmt.Errorf("leader election requires a lease name, namespace, and identity (got lease %q, namespace %q, identity %q): set the missing values",
			config.LeaseName, config.Namespace, config.Identity)
	}
	if config.Run == nil || config.OnLost == nil {
		return fmt.Errorf("leader election for lease %q requires Run and OnLost callbacks: set both", config.LeaseName)
	}

	// client-go calls OnStartedLeading in a goroutine and does not wait for it,
	// so wait for config.Run to return before returning. stopped prevents
	// config.Run from starting after elector.Run has returned.
	var (
		mutex            sync.Mutex
		stopped          bool
		runningCallbacks sync.WaitGroup
	)
	elector, electorError := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      config.LeaseName,
				Namespace: config.Namespace,
			},
			Client: client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: config.Identity,
			},
		},
		Name:            config.LeaseName,
		LeaseDuration:   timings.leaseDuration,
		RenewDeadline:   timings.renewDeadline,
		RetryPeriod:     timings.retryPeriod,
		ReleaseOnCancel: false,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderContext context.Context) {
				mutex.Lock()
				if stopped {
					mutex.Unlock()
					return
				}
				runningCallbacks.Add(1)
				mutex.Unlock()
				defer runningCallbacks.Done()
				glog.Infof("Leader election: %s acquired lease %s/%s", config.Identity, config.Namespace, config.LeaseName)
				config.Run(leaderContext)
			},
			OnStoppedLeading: func() {
				if electionContext.Err() != nil {
					glog.Infof("Leader election: %s stopped leading lease %s/%s for shutdown", config.Identity, config.Namespace, config.LeaseName)
					return
				}
				glog.Errorf("Leader election: %s lost lease %s/%s", config.Identity, config.Namespace, config.LeaseName)
				config.OnLost()
			},
			OnNewLeader: func(identity string) {
				if identity != config.Identity {
					glog.Infof("Leader election: lease %s/%s is held by %s", config.Namespace, config.LeaseName, identity)
				}
			},
		},
	})
	if electorError != nil {
		return fmt.Errorf("invalid leader election config for lease %q: %w", config.LeaseName, electorError)
	}

	glog.Infof("Leader election: %s is waiting for lease %s/%s", config.Identity, config.Namespace, config.LeaseName)
	elector.Run(electionContext)
	mutex.Lock()
	stopped = true
	mutex.Unlock()
	runningCallbacks.Wait()
	return nil
}
