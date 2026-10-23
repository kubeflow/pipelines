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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

// candidate runs one replica's election in the background and records its
// callbacks.
type candidate struct {
	cancel  context.CancelFunc
	done    chan error
	running atomic.Bool
	started atomic.Int32
	lost    atomic.Int32
	runDone chan struct{}
}

func startCandidate(t *testing.T, client *fake.Clientset, identity string) *candidate {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	c := &candidate{cancel: cancel, done: make(chan error, 1), runDone: make(chan struct{})}
	t.Cleanup(cancel)
	go func() {
		c.done <- runWithLeaderElection(ctx, client, LeaderElectionConfig{
			LeaseName: testLeaseName,
			Namespace: testNamespace,
			Identity:  identity,
			Run: func(leaderCtx context.Context) {
				c.started.Add(1)
				c.running.Store(true)
				<-leaderCtx.Done()
				c.running.Store(false)
				close(c.runDone)
			},
			OnLost: func() { c.lost.Add(1) },
		}, testTimings)
	}()
	return c
}

func leaseHolder(t *testing.T, client *fake.Clientset) string {
	t.Helper()
	lease, err := client.CoordinationV1().Leases(testNamespace).Get(context.Background(), testLeaseName, metav1.GetOptions{})
	require.NoError(t, err)
	if lease.Spec.HolderIdentity == nil {
		return ""
	}
	return *lease.Spec.HolderIdentity
}

func TestRunWithLeaderElection_InvalidConfig(t *testing.T) {
	run := func(context.Context) { t.Fatal("Run must not be called") }
	onLost := func() {}
	tests := []struct {
		name   string
		config LeaderElectionConfig
	}{
		{"missing lease name", LeaderElectionConfig{Namespace: testNamespace, Identity: "a", Run: run, OnLost: onLost}},
		{"missing namespace", LeaderElectionConfig{LeaseName: testLeaseName, Identity: "a", Run: run, OnLost: onLost}},
		{"missing identity", LeaderElectionConfig{LeaseName: testLeaseName, Namespace: testNamespace, Run: run, OnLost: onLost}},
		{"missing Run", LeaderElectionConfig{LeaseName: testLeaseName, Namespace: testNamespace, Identity: "a", OnLost: onLost}},
		{"missing OnLost", LeaderElectionConfig{LeaseName: testLeaseName, Namespace: testNamespace, Identity: "a", Run: run}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := runWithLeaderElection(context.Background(), fake.NewClientset(), test.config, testTimings)
			assert.Error(t, err)
		})
	}
}

func TestRunWithLeaderElection_StandbyTakesOverAfterLeaderStops(t *testing.T) {
	client := fake.NewClientset()
	a := startCandidate(t, client, "a")
	require.Eventually(t, a.running.Load, 5*time.Second, 10*time.Millisecond)

	b := startCandidate(t, client, "b")
	// A standby must not run while the leader renews the lease.
	time.Sleep(3 * testTimings.leaseDuration)
	assert.Equal(t, int32(0), b.started.Load(), "the standby must not run while the leader holds the lease")
	assert.Equal(t, "a", leaseHolder(t, client))

	// A graceful shutdown stops Run without calling OnLost, and keeps the
	// lease (ReleaseOnCancel is false).
	a.cancel()
	select {
	case err := <-a.done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("RunWithLeaderElection didn't return after its context was canceled")
	}
	assert.False(t, a.running.Load(), "Run must have returned before RunWithLeaderElection returns")
	assert.Equal(t, int32(0), a.lost.Load(), "OnLost must not be called on shutdown")
	assert.Equal(t, "a", leaseHolder(t, client), "the lease must not be released on shutdown")

	// The standby takes over once the lease expires.
	assert.Eventually(t, b.running.Load, 5*time.Second, 10*time.Millisecond, "the standby should take over")
	assert.Equal(t, "b", leaseHolder(t, client))
	assert.Equal(t, int32(0), b.lost.Load())
}

func TestRunWithLeaderElection_OnLostWhenRenewalFails(t *testing.T) {
	client := fake.NewClientset()
	// Reactors can't be added while the fake clientset is in use, so the
	// failing reactor is installed up front and enabled later.
	var failRenewals atomic.Bool
	client.PrependReactor("update", "leases", func(k8stesting.Action) (bool, runtime.Object, error) {
		if failRenewals.Load() {
			return true, nil, errors.New("API server unavailable")
		}
		return false, nil, nil
	})
	a := startCandidate(t, client, "a")
	require.Eventually(t, a.running.Load, 5*time.Second, 10*time.Millisecond)

	failRenewals.Store(true)

	select {
	case <-a.runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Run's context wasn't canceled after the lease renewal failed")
	}
	assert.Eventually(t, func() bool { return a.lost.Load() == 1 }, 5*time.Second, 10*time.Millisecond, "OnLost should be called once")
	select {
	case err := <-a.done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("RunWithLeaderElection didn't return after losing the lease")
	}
}

func TestLeaderElectionIdentity(t *testing.T) {
	t.Setenv("POD_NAME", "ml-pipeline-scheduledworkflow-abc")
	identity, err := LeaderElectionIdentity()
	require.NoError(t, err)
	assert.Equal(t, "ml-pipeline-scheduledworkflow-abc", identity)

	t.Setenv("POD_NAME", "")
	identity, err = LeaderElectionIdentity()
	require.NoError(t, err)
	assert.NotEmpty(t, identity, "falls back to the hostname")
}
