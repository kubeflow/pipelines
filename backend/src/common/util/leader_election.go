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
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	authorizationv1 "k8s.io/api/authorization/v1"
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
	Run func(ctx context.Context)
	// OnLost is called when the lease is lost before the parent context is
	// done. Callers exit the process here, because a replica that lost the
	// lease can't safely keep or reuse its work queue.
	OnLost func()
}

type leaderElectionTimings struct {
	leaseDuration time.Duration
	renewDeadline time.Duration
	retryPeriod   time.Duration
}

// The same timings as the API server's run garbage collection lease
// (backend/src/apiserver/gc/run_gc.go).
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

// RunWithLeaderElection blocks until ctx is done. Only the replica holding
// the lease calls config.Run; losing the lease calls config.OnLost.
//
// ReleaseOnCancel is false: on shutdown, the lease is kept until it expires,
// so another replica doesn't start while requests sent by this one may still
// be in flight.
func RunWithLeaderElection(ctx context.Context, client kubernetes.Interface, config LeaderElectionConfig) error {
	return runWithLeaderElection(ctx, client, config, defaultLeaderElectionTimings)
}

func runWithLeaderElection(
	ctx context.Context,
	client kubernetes.Interface,
	config LeaderElectionConfig,
	timings leaderElectionTimings,
) error {
	if config.LeaseName == "" || config.Namespace == "" || config.Identity == "" {
		return fmt.Errorf("leader election requires a lease name, namespace and identity, got lease %q, namespace %q, identity %q",
			config.LeaseName, config.Namespace, config.Identity)
	}
	if config.Run == nil || config.OnLost == nil {
		return fmt.Errorf("leader election for lease %q requires Run and OnLost callbacks", config.LeaseName)
	}
	if err := checkLeasePermissions(ctx, client, config.Namespace, config.LeaseName); err != nil {
		return err
	}

	// client-go calls OnStartedLeading in a goroutine and doesn't wait for it,
	// so track it to return only after config.Run has returned. stopped keeps
	// a goroutine that starts after elector.Run returned from calling Run.
	var (
		mu      sync.Mutex
		stopped bool
		running sync.WaitGroup
	)
	elector, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
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
			OnStartedLeading: func(leaderCtx context.Context) {
				mu.Lock()
				if stopped {
					mu.Unlock()
					return
				}
				running.Add(1)
				mu.Unlock()
				defer running.Done()
				glog.Infof("Leader election: %s acquired lease %s/%s", config.Identity, config.Namespace, config.LeaseName)
				config.Run(leaderCtx)
			},
			OnStoppedLeading: func() {
				if ctx.Err() != nil {
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
	if err != nil {
		return fmt.Errorf("invalid leader election config for lease %q: %w", config.LeaseName, err)
	}

	glog.Infof("Leader election: %s is waiting for lease %s/%s", config.Identity, config.Namespace, config.LeaseName)
	elector.Run(ctx)
	mu.Lock()
	stopped = true
	mu.Unlock()
	running.Wait()
	return nil
}

// checkLeasePermissions fails fast when the Lease RBAC rules are missing.
// Without this check, client-go retries forever and the replica stays a
// standby that only logs errors.
func checkLeasePermissions(ctx context.Context, client kubernetes.Interface, namespace, leaseName string) error {
	checks := []struct {
		verb string
		name string
	}{
		// RBAC can't restrict create by resource name.
		{verb: "create"},
		{verb: "get", name: leaseName},
		{verb: "update", name: leaseName},
	}
	var missing []string
	for _, check := range checks {
		review := &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      check.verb,
					Group:     "coordination.k8s.io",
					Resource:  "leases",
					Name:      check.name,
				},
			},
		}
		response, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
		if err != nil {
			// Some clusters don't allow access reviews. Lease errors are
			// still logged by client-go.
			glog.Warningf("Leader election: could not check permissions for lease %s/%s, continuing: %v", namespace, leaseName, err)
			return nil
		}
		if !response.Status.Allowed {
			missing = append(missing, check.verb)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("leader election: the service account can't %s leases.coordination.k8s.io %q in namespace %q; add the Lease rules to its Role",
			strings.Join(missing, ", "), leaseName, namespace)
	}
	return nil
}
