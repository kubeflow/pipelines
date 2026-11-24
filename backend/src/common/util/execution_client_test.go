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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// The Workflow CRD accepts both Workflows. The second one does not decode into
// the Go type, where spec.activeDeadlineSeconds is an integer.
const (
	ordinaryWorkflow = `{"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow",
		"metadata": {"namespace": "tenant-b", "name": "ordinary", "resourceVersion": "1"},
		"spec": {"activeDeadlineSeconds": 60}}`
	undecodableWorkflow = `{"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow",
		"metadata": {"namespace": "tenant-a", "name": "undecodable", "resourceVersion": "2"},
		"spec": {"activeDeadlineSeconds": "not-a-number"}}`
)

// A Workflow of one namespace that does not decode into the Go type must not
// keep the informer of all namespaces from syncing.
func TestNewExecutionInformerOrFatal_UndecodableWorkflow(t *testing.T) {
	tests := []struct {
		name string
		// Whether the API server sends the initial list on the watch, which
		// is what client-go asks for first.
		streamingList bool
	}{
		{name: "list", streamingList: false},
		{name: "streaming list", streamingList: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				query := r.URL.Query()
				switch {
				case query.Get("watch") != "true":
					fmt.Fprintf(w, `{"apiVersion": "argoproj.io/v1alpha1", "kind": "WorkflowList",
						"metadata": {"resourceVersion": "2"}, "items": [%s, %s]}`, undecodableWorkflow, ordinaryWorkflow)
					return
				case query.Get("sendInitialEvents") != "true":
					// The watch that follows a list: nothing changes.
				case !test.streamingList:
					w.WriteHeader(http.StatusUnprocessableEntity)
					fmt.Fprint(w, `{"apiVersion": "v1", "kind": "Status", "status": "Failure", "reason": "Invalid", "code": 422}`)
					return
				default:
					fmt.Fprintf(w, `{"type": "ADDED", "object": %s}
						{"type": "ADDED", "object": %s}
						{"type": "BOOKMARK", "object": {"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow",
							"metadata": {"resourceVersion": "2", "annotations": {"k8s.io/initial-events-end": "true"}}}}
						`, undecodableWorkflow, ordinaryWorkflow)
				}
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			t.Cleanup(server.CloseClientConnections)

			kubeconfig := clientcmdapi.NewConfig()
			kubeconfig.Clusters["test"] = &clientcmdapi.Cluster{Server: server.URL}
			kubeconfig.Contexts["test"] = &clientcmdapi.Context{Cluster: "test"}
			kubeconfig.CurrentContext = "test"
			kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
			require.NoError(t, clientcmd.WriteToFile(*kubeconfig, kubeconfigPath))
			t.Setenv("KUBECONFIG", kubeconfigPath)
			// Not the in-cluster configuration, wherever the test runs.
			t.Setenv("KUBERNETES_SERVICE_HOST", "")

			informer := NewExecutionInformerOrFatal(ArgoWorkflow, "", time.Second, ClientParameters{QPS: 5, Burst: 10})
			hasSynced := informer.HasSynced()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			informer.InformerFactoryStart(ctx.Done())
			require.True(t, cache.WaitForCacheSync(ctx.Done(), hasSynced),
				"The Workflow informer did not sync")

			everything := labels.Everything()
			workflows, err := informer.List("", &everything)
			require.NoError(t, err)
			require.Len(t, workflows, 1)
			assert.Equal(t, "tenant-b", workflows[0].ExecutionNamespace())
			assert.Equal(t, "ordinary", workflows[0].ExecutionName())
		})
	}
}
