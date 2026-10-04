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

package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type streamArchive struct {
	archive.LogArchiveInterface
	reads atomic.Int32
}

func (a *streamArchive) GetLogObjectKey(util.ExecutionSpec, string) (string, error) {
	a.reads.Add(1)
	return "archive.log", nil
}

type streamObjectStore struct{ storage.ObjectStore }

func (s *streamObjectStore) GetFileReader(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("archived log\n")), nil
}

type streamClientManager struct {
	resource.ClientManagerInterface
	logs *streamArchive
}

func (c *streamClientManager) LogArchive() archive.LogArchiveInterface { return c.logs }
func (c *streamClientManager) ObjectStore() storage.ObjectStore        { return &streamObjectStore{} }

type statusTrackingResponse struct {
	http.ResponseWriter
	statuses []int
}

func (w *statusTrackingResponse) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusTrackingResponse) Write(data []byte) (int, error) {
	if len(w.statuses) == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func (w *statusTrackingResponse) Flush() { w.ResponseWriter.(http.Flusher).Flush() }

func TestReadRunLog_BeforeFirstWrite(t *testing.T) {
	for _, cancelRequest := range []bool{false, true} {
		name := "archive fallback"
		if cancelRequest {
			name = "cancellation without fallback"
		}
		t.Run(name, func(t *testing.T) {
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			run.WorkflowRuntimeManifest = run.PipelineRuntimeManifest
			require.NoError(t, clients.RunStore().UpdateRun(run))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			kubeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/log") {
					if cancelRequest {
						cancel()
					}
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(&corev1.Pod{
					TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
					ObjectMeta: metav1.ObjectMeta{Name: "node-1", Namespace: "ns1", Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID}},
				})
			}))
			defer kubeServer.Close()
			kubeClient, err := kubernetes.NewForConfig(&rest.Config{Host: kubeServer.URL})
			require.NoError(t, err)
			clients.KubernetesCoreClientFake = &logTestKubernetesCore{Interface: kubeClient}
			logs := &streamArchive{LogArchiveInterface: archive.NewLogArchive("/logs", "main.log")}
			manager := resource.NewResourceManager(&streamClientManager{ClientManagerInterface: clients, logs: logs}, &resource.ResourceManagerOptions{})
			request := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/log?follow=true", nil).WithContext(ctx), map[string]string{RunKey: run.UUID, NodeKey: "node-1"})
			response := &statusTrackingResponse{ResponseWriter: httptest.NewRecorder()}
			NewRunLogServer(manager).ReadRunLog(response, request)
			body := response.ResponseWriter.(*httptest.ResponseRecorder).Body.String()
			if cancelRequest {
				assert.Empty(t, body)
				assert.Empty(t, response.statuses)
				assert.Zero(t, logs.reads.Load())
			} else {
				assert.Equal(t, "archived log\n", body)
				assert.Equal(t, []int{http.StatusOK}, response.statuses)
				assert.EqualValues(t, 1, logs.reads.Load())
			}
		})
	}
}

func TestReadRunLog_StreamsBeforePodExit(t *testing.T) {
	for _, tls := range []bool{false, true} {
		for _, ending := range []string{"complete", "disconnect", "cancel"} {
			name := "http/" + ending
			if tls {
				name = "https/" + ending
			}
			t.Run(name, func(t *testing.T) {
				clients, _, run := initWithOneTimeRun(t)
				defer clients.Close()
				// The archive fallback must be available, but never replay after a live line.
				run.WorkflowRuntimeManifest = run.PipelineRuntimeManifest
				require.NoError(t, clients.RunStore().UpdateRun(run))
				release := make(chan struct{})
				streamDone := make(chan struct{})
				kubeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/api/v1/namespaces/ns1/pods/node-1":
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(&corev1.Pod{
							TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
							ObjectMeta: metav1.ObjectMeta{Name: "node-1", Namespace: "ns1", Labels: map[string]string{util.LabelKeyWorkflowRunId: run.UUID}},
						})
					case "/api/v1/namespaces/ns1/pods/node-1/log":
						defer close(streamDone)
						_, _ = io.WriteString(w, "first live line\n")
						w.(http.Flusher).Flush()
						select {
						case <-release:
						case <-r.Context().Done():
						}
						if ending == "disconnect" {
							panic(http.ErrAbortHandler)
						}
					default:
						http.NotFound(w, r)
					}
				}))
				defer kubeServer.Close()
				defer close(release)
				kubeClient, err := kubernetes.NewForConfig(&rest.Config{Host: kubeServer.URL})
				require.NoError(t, err)
				clients.KubernetesCoreClientFake = &logTestKubernetesCore{Interface: kubeClient}
				logs := &streamArchive{LogArchiveInterface: archive.NewLogArchive("/logs", "main.log")}
				manager := resource.NewResourceManager(&streamClientManager{ClientManagerInterface: clients, logs: logs}, &resource.ResourceManagerOptions{})
				router := mux.NewRouter()
				router.HandleFunc("/apis/v2beta1/runs/{run_id}/nodes/{node_id}/log", NewRunLogServer(manager).ReadRunLog)
				handlerDone := make(chan []int, 1)
				apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tracked := &statusTrackingResponse{ResponseWriter: w}
					router.ServeHTTP(tracked, r)
					handlerDone <- tracked.statuses
				}))
				if tls {
					apiServer.EnableHTTP2 = true
					apiServer.StartTLS()
				} else {
					apiServer.Start()
				}
				defer apiServer.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiServer.URL+"/apis/v2beta1/runs/"+run.UUID+"/nodes/node-1/log?follow=true", nil)
				require.NoError(t, err)
				response, err := apiServer.Client().Do(req)
				require.NoError(t, err, "headers and a short line must arrive while the pod is still running")
				defer response.Body.Close()
				reader := bufio.NewReader(response.Body)
				line, err := reader.ReadString('\n')
				require.NoError(t, err)
				require.Equal(t, "first live line\n", line)
				select {
				case <-streamDone:
					t.Fatal("line arrived only after the pod stream ended")
				default:
				}
				require.Equal(t, "text/plain", response.Header.Get("Content-Type"))
				if ending == "cancel" {
					cancel()
				} else {
					release <- struct{}{}
				}
				if ending != "cancel" {
					tail, err := io.ReadAll(reader)
					require.NoError(t, err)
					assert.Empty(t, string(tail), "no archive replay or JSON error may be appended")
				}
				select {
				case statuses := <-handlerDone:
					assert.Equal(t, []int{http.StatusOK}, statuses)
				case <-time.After(5 * time.Second):
					t.Fatal("log handler did not stop")
				}
				assert.Zero(t, logs.reads.Load(), "a started/canceled stream must not fall back to the archive")
			})
		}
	}
}
