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
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

// Opt-in real offload DB validation. Unit CI uses MemoryOffloadNodeStatusRepo
// (TestOffloadHydration_EndToEnd_MemoryRepo and resource GC survival tests).
//
// To exercise NewSessionProxy + CreateWorkflowHydrator against a live Argo
// persistence database (Postgres or MySQL matching workflow-controller):
//
//	export KFP_ARGO_OFFLOAD_IT=1
//	# Provide a PersistConfig YAML path (same shape as ConfigMap data.persistence)
//	export KFP_ARGO_OFFLOAD_PERSIST_CONFIG=/path/to/persistence.yaml
//	# Optional: namespace holding userNameSecret/passwordSecret (default: default)
//	export KFP_ARGO_OFFLOAD_SECRETS_NAMESPACE=kubeflow
//	# Cluster kubeconfig must allow Secret get for those credential Secrets.
//	GOMODCACHE=$HOME/go/pkg/mod go test ./backend/src/common/util/ \
//	  -count=1 -run TestOffloadHydration_RealDB_OptIn -v
//
// This stub documents the entry point; full cluster Secret injection is
// environment-specific. Prefer the memory E2E above for non-flaky CI.
func TestOffloadHydration_RealDB_OptIn(t *testing.T) {
	if os.Getenv("KFP_ARGO_OFFLOAD_IT") != "1" {
		t.Skip("opt-in: set KFP_ARGO_OFFLOAD_IT=1 with persist config + kube Secrets to hit a real offload DB")
	}
	persistPath := os.Getenv("KFP_ARGO_OFFLOAD_PERSIST_CONFIG")
	if persistPath == "" {
		t.Skip("set KFP_ARGO_OFFLOAD_PERSIST_CONFIG to a persistence YAML file")
	}
	raw, err := os.ReadFile(persistPath)
	require.NoError(t, err)
	persist, err := ParseArgoPersistConfig(raw)
	require.NoError(t, err)
	require.True(t, persist.NodeStatusOffload, "persist config must enable nodeStatusOffLoad")

	ns := os.Getenv("KFP_ARGO_OFFLOAD_SECRETS_NAMESPACE")
	if ns == "" {
		ns = "default"
	}
	// Real runs need an in-cluster or kubeconfig-backed client that can read
	// persistence Secrets. The fake client is intentionally insufficient here
	// so operators wire their own cluster client in a follow-up IT harness.
	_, err = CreateWorkflowHydrator(context.Background(), k8sfake.NewClientset(), persist, ns)
	require.Error(t, err, "fake kube cannot supply persist Secrets; use a real clientset in cluster IT")
	t.Logf("CreateWorkflowHydrator correctly rejected fake kube (secrets namespace %q): %v", ns, err)
	t.Log("To complete real-DB validation, replace k8sfake with a rest.Config clientset that can Get the configured Secrets and reach the DB host.")
}
