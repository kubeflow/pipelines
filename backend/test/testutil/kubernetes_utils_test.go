// Copyright 2021-2023 The Kubeflow Authors
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

package testutil_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/testutil"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestReadPodLogsPreservesStreamErrorAndContinues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/namespaces/ns/pods" {
			_ = json.NewEncoder(w).Encode(corev1.PodList{Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "ns"}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "unavailable"}, {Name: "available"}}}}}})
			return
		}
		if r.URL.Path != "/api/v1/namespaces/ns/pods/test-pod/log" {
			t.Errorf("unexpected log request path %q", r.URL.Path)
		}
		if r.URL.Query().Get("container") == "unavailable" {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: "Failure", Reason: metav1.StatusReasonForbidden, Message: "log access denied by diagnostic-test policy", Code: http.StatusForbidden})
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "remaining container log")
	}))
	defer server.Close()
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL})
	require.NoError(t, err)
	var diagnostics bytes.Buffer
	ginkgo.GinkgoWriter.TeeTo(&diagnostics)
	defer ginkgo.GinkgoWriter.ClearTeeWriters()
	logs := testutil.ReadPodLogs(client, "ns", "test-pod", nil, nil, nil)
	require.Equal(t, "remaining container log", logs)
	require.Contains(t, diagnostics.String(), "Failed to stream pod logs for container 'unavailable'")
	require.Contains(t, diagnostics.String(), "log access denied by diagnostic-test policy")
	require.NotContains(t, diagnostics.String(), "stream is nil")
}
