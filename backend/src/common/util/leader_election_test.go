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

package util_test

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRunWithLeaderElection_InvalidConfig(t *testing.T) {
	run := func(context.Context) { t.Fatal("Run must not be called") }
	onLost := func() {}
	tests := []struct {
		name   string
		config util.LeaderElectionConfig
	}{
		{"missing lease name", util.LeaderElectionConfig{Namespace: "kubeflow", Identity: "replica", Run: run, OnLost: onLost}},
		{"missing namespace", util.LeaderElectionConfig{LeaseName: "kfp-test", Identity: "replica", Run: run, OnLost: onLost}},
		{"missing identity", util.LeaderElectionConfig{LeaseName: "kfp-test", Namespace: "kubeflow", Run: run, OnLost: onLost}},
		{"missing Run", util.LeaderElectionConfig{LeaseName: "kfp-test", Namespace: "kubeflow", Identity: "replica", OnLost: onLost}},
		{"missing OnLost", util.LeaderElectionConfig{LeaseName: "kfp-test", Namespace: "kubeflow", Identity: "replica", Run: run}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			electionError := util.RunWithLeaderElection(context.Background(), fake.NewClientset(), test.config)
			assert.Error(t, electionError)
		})
	}
}

func TestLeaderElectionIdentity(t *testing.T) {
	t.Setenv("POD_NAME", "ml-pipeline-scheduledworkflow-abc")
	identity, identityError := util.LeaderElectionIdentity()
	require.NoError(t, identityError)
	assert.Equal(t, "ml-pipeline-scheduledworkflow-abc", identity)

	t.Setenv("POD_NAME", "")
	identity, identityError = util.LeaderElectionIdentity()
	require.NoError(t, identityError)
	assert.NotEmpty(t, identity, "falls back to the hostname")
}
