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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	testLeaseName = "kfp-test"
	testNamespace = "kubeflow"
)

var testTimings = leaderElectionTimings{
	leaseDuration: time.Second,
	renewDeadline: 500 * time.Millisecond,
	retryPeriod:   100 * time.Millisecond,
}

// electionCandidate runs one replica's election in the background and
// records its callbacks.
type electionCandidate struct {
	cancel      context.CancelFunc
	electionEnd chan error
	running     atomic.Bool
	startCount  atomic.Int32
	lostCount   atomic.Int32
	runEnd      chan struct{}
}

func startCandidate(t *testing.T, client *fake.Clientset, identity string) *electionCandidate {
	t.Helper()
	candidateContext, cancelCandidate := context.WithCancel(context.Background())
	candidate := &electionCandidate{cancel: cancelCandidate, electionEnd: make(chan error, 1), runEnd: make(chan struct{})}
	t.Cleanup(cancelCandidate)
	go func() {
		candidate.electionEnd <- runWithLeaderElection(candidateContext, client, LeaderElectionConfig{
			LeaseName: testLeaseName,
			Namespace: testNamespace,
			Identity:  identity,
			Run: func(leaderContext context.Context) {
				candidate.startCount.Add(1)
				candidate.running.Store(true)
				<-leaderContext.Done()
				candidate.running.Store(false)
				close(candidate.runEnd)
			},
			OnLost: func() { candidate.lostCount.Add(1) },
		}, testTimings)
	}()
	return candidate
}

func waitForElectionEnd(t *testing.T, candidate *electionCandidate) {
	t.Helper()
	select {
	case electionError := <-candidate.electionEnd:
		assert.NoError(t, electionError)
	case <-time.After(5 * time.Second):
		t.Fatal("RunWithLeaderElection did not return")
	}
}

func leaseHolder(t *testing.T, client *fake.Clientset) string {
	t.Helper()
	lease, getError := client.CoordinationV1().Leases(testNamespace).Get(context.Background(), testLeaseName, metav1.GetOptions{})
	require.NoError(t, getError)
	if lease.Spec.HolderIdentity == nil {
		return ""
	}
	return *lease.Spec.HolderIdentity
}

func TestRunWithLeaderElection_StandbyTakesOverAfterLeaderStops(t *testing.T) {
	client := fake.NewClientset()
	leader := startCandidate(t, client, "leader")
	require.Eventually(t, leader.running.Load, 5*time.Second, 10*time.Millisecond)

	standby := startCandidate(t, client, "standby")
	// A standby must not run while the leader renews the lease.
	time.Sleep(3 * testTimings.leaseDuration)
	assert.Equal(t, int32(0), standby.startCount.Load(), "the standby must not run while the leader holds the lease")
	assert.Equal(t, "leader", leaseHolder(t, client))

	// A graceful shutdown stops Run without calling OnLost, and keeps the
	// lease (ReleaseOnCancel is false).
	leader.cancel()
	waitForElectionEnd(t, leader)
	assert.False(t, leader.running.Load(), "Run must have returned before RunWithLeaderElection returns")
	assert.Equal(t, int32(0), leader.lostCount.Load(), "OnLost must not be called on shutdown")
	assert.Equal(t, "leader", leaseHolder(t, client), "the lease must not be released on shutdown")

	// The standby takes over once the lease expires.
	assert.Eventually(t, standby.running.Load, 5*time.Second, 10*time.Millisecond, "the standby should take over")
	assert.Equal(t, "standby", leaseHolder(t, client))
	assert.Equal(t, int32(0), standby.lostCount.Load())
}

func TestRunWithLeaderElection_OnLostWhenRenewalFails(t *testing.T) {
	client := fake.NewClientset()
	// Reactors cannot be added while the fake clientset is in use, so the
	// failing reactor is installed up front and enabled later.
	var failRenewals atomic.Bool
	client.PrependReactor("update", "leases", func(k8stesting.Action) (bool, runtime.Object, error) {
		if failRenewals.Load() {
			return true, nil, errors.New("API server unavailable")
		}
		return false, nil, nil
	})
	leader := startCandidate(t, client, "leader")
	require.Eventually(t, leader.running.Load, 5*time.Second, 10*time.Millisecond)

	failRenewals.Store(true)

	select {
	case <-leader.runEnd:
	case <-time.After(5 * time.Second):
		t.Fatal("the context of Run was not canceled after the lease renewal failed")
	}
	assert.Eventually(t, func() bool { return leader.lostCount.Load() == 1 }, 5*time.Second, 10*time.Millisecond, "OnLost should be called once")
	waitForElectionEnd(t, leader)
}

func TestRunWithLeaderElection_MissingLeasePermissions(t *testing.T) {
	client := fake.NewClientset()
	client.PrependReactor("create", "leases", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, testLeaseName, errors.New("RBAC: access denied"))
	})
	candidate := startCandidate(t, client, "replica")

	// Without permission to create the Lease, the replica stays a standby:
	// it never runs, and it keeps retrying until it is stopped.
	time.Sleep(3 * testTimings.leaseDuration)
	assert.Equal(t, int32(0), candidate.startCount.Load(), "Run must not be called without the Lease")
	select {
	case <-candidate.electionEnd:
		t.Fatal("RunWithLeaderElection must keep retrying while the Lease cannot be created")
	default:
	}

	candidate.cancel()
	waitForElectionEnd(t, candidate)
	assert.Equal(t, int32(0), candidate.lostCount.Load(), "OnLost must not be called for a replica that never led")
}

func TestRunWithLeaderElection_ShutdownWaitsForInFlightWork(t *testing.T) {
	client := fake.NewClientset()
	electionContext, cancelElection := context.WithCancel(context.Background())
	t.Cleanup(cancelElection)
	var running, finished atomic.Bool
	electionEnd := make(chan error, 1)
	go func() {
		electionEnd <- runWithLeaderElection(electionContext, client, LeaderElectionConfig{
			LeaseName: testLeaseName,
			Namespace: testNamespace,
			Identity:  "leader",
			Run: func(leaderContext context.Context) {
				running.Store(true)
				<-leaderContext.Done()
				// Simulate a request that is still in flight when shutdown begins.
				time.Sleep(300 * time.Millisecond)
				finished.Store(true)
			},
			OnLost: func() { t.Error("OnLost must not be called on shutdown") },
		}, testTimings)
	}()
	require.Eventually(t, running.Load, 5*time.Second, 10*time.Millisecond)

	cancelElection()
	select {
	case electionError := <-electionEnd:
		assert.NoError(t, electionError)
	case <-time.After(5 * time.Second):
		t.Fatal("RunWithLeaderElection did not return after its context was canceled")
	}
	assert.True(t, finished.Load(), "RunWithLeaderElection must wait for the in-flight work of Run")
	assert.Equal(t, "leader", leaseHolder(t, client), "the lease must stay held after shutdown, until it expires")
}
