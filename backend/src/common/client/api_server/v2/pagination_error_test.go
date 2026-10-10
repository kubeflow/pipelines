// Copyright 2018-2023 The Kubeflow Authors
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

package api_server_v2 //nolint:staticcheck // ST1003: package name matches existing convention in this directory

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	experimentclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client"
	experimentparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	pipelineclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client"
	pipelineparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	recurring_runclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client"
	recurring_runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	runclient "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client"
	runparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
)

const restartResponse = `{"code":9,"message":"Clear page_token and restart this listing from the first page.","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"PAGINATION_RESTART_REQUIRED","domain":"kubeflow.org"}]}`

func TestListPreservesPaginationRestart(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*httptransport.Runtime) error
	}{
		{"Run", func(rt *httptransport.Runtime) error {
			c := &RunClient{apiClient: runclient.New(rt, strfmt.Default)}
			_, _, _, err := c.List(&runparams.RunServiceListRunsParams{})
			return err
		}},
		{"Pipeline", func(rt *httptransport.Runtime) error {
			c := &PipelineClient{apiClient: pipelineclient.New(rt, strfmt.Default)}
			_, _, _, err := c.List(&pipelineparams.PipelineServiceListPipelinesParams{})
			return err
		}},
		{"PipelineVersion", func(rt *httptransport.Runtime) error {
			c := &PipelineClient{apiClient: pipelineclient.New(rt, strfmt.Default)}
			_, _, _, err := c.ListPipelineVersions(&pipelineparams.PipelineServiceListPipelineVersionsParams{})
			return err
		}},
		{"Experiment", func(rt *httptransport.Runtime) error {
			c := &ExperimentClient{apiClient: experimentclient.New(rt, strfmt.Default)}
			_, _, _, err := c.List(&experimentparams.ExperimentServiceListExperimentsParams{})
			return err
		}},
		{"RecurringRun", func(rt *httptransport.Runtime) error {
			c := &RecurringRunClient{apiClient: recurring_runclient.New(rt, strfmt.Default)}
			_, _, _, err := c.List(&recurring_runparams.RecurringRunServiceListRecurringRunsParams{})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = w.Write([]byte(restartResponse))
			}))
			defer srv.Close()
			err := tc.call(httptransport.New(strings.TrimPrefix(srv.URL, "http://"), "/", []string{"http"}))
			if !api_server.IsPaginationRestartRequired(err) {
				t.Fatalf("lost restart condition: %v", err)
			}
			if !strings.Contains(err.Error(), "Clear page_token") {
				t.Fatalf("lost actionable message: %v", err)
			}
			if calls != 1 {
				t.Fatalf("unexpected retry: %d calls", calls)
			}
		})
	}
}
func TestListAllDoesNotReturnPartialResultsOrRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_, _ = w.Write([]byte(`{"runs":[{"run_id":"first"}],"next_page_token":"old-token"}`))
			return
		}
		if r.URL.Query().Get("page_token") != "old-token" {
			t.Error("did not continue original traversal")
		}
		w.WriteHeader(400)
		_, _ = w.Write([]byte(restartResponse))
	}))
	defer srv.Close()
	c := &RunClient{apiClient: runclient.New(httptransport.New(strings.TrimPrefix(srv.URL, "http://"), "/", []string{"http"}), strfmt.Default)}
	rows, err := c.ListAll(&runparams.RunServiceListRunsParams{}, 100)
	if rows != nil || !api_server.IsPaginationRestartRequired(err) {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if calls != 2 {
		t.Fatalf("unexpected retry: %d calls", calls)
	}
}

func TestListPreservesUnrelatedGeneratedErrors(t *testing.T) {
	for _, resource := range []string{"experiments", "recurring runs"} {
		t.Run(resource, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"code":9,"message":"some other precondition","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo","reason":"OTHER","domain":"kubeflow.org"}]}`))
			}))
			defer srv.Close()
			rt := httptransport.New(strings.TrimPrefix(srv.URL, "http://"), "/", []string{"http"})
			var err error
			if resource == "experiments" {
				c := &ExperimentClient{apiClient: experimentclient.New(rt, strfmt.Default)}
				_, _, _, err = c.List(&experimentparams.ExperimentServiceListExperimentsParams{})
				var original *experimentparams.ExperimentServiceListExperimentsDefault
				if !errors.As(err, &original) {
					t.Fatalf("lost original generated error: %v", err)
				}
			} else {
				c := &RecurringRunClient{apiClient: recurring_runclient.New(rt, strfmt.Default)}
				_, _, _, err = c.List(&recurring_runparams.RecurringRunServiceListRecurringRunsParams{})
				var original *recurring_runparams.RecurringRunServiceListRecurringRunsDefault
				if !errors.As(err, &original) {
					t.Fatalf("lost original generated error: %v", err)
				}
			}
			if api_server.IsPaginationRestartRequired(err) {
				t.Fatalf("misclassified unrelated precondition: %v", err)
			}
		})
	}
}
