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

package integration

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/eapache/go-resiliency/retrier"
	experiment_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_client/experiment_service"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	recurring_run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_client/recurring_run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/recurring_run_model"
	run_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	test "github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// TestRecurringRunCustomServiceAccount requires a multi-user installation with
// KFP_SCHEDULE_TEST_SERVICE_ACCOUNT naming an existing custom runner in the test
// resource namespace. Allowlist that account and grant serviceaccounts/use on
// that exact account to both the test API caller and scheduled-workflow controller.
// The runner needs normal pipeline execution permissions; the test Kubernetes
// identity needs get/list/update on ScheduledWorkflows to exercise CR tampering.
func (s *RecurringRunApiTestSuite) TestRecurringRunCustomServiceAccount() {
	t := s.T()
	serviceAccount := os.Getenv("KFP_SCHEDULE_TEST_SERVICE_ACCOUNT")
	if !*isKubeflowMode || serviceAccount == "" {
		t.Skip("requires multi-user mode and KFP_SCHEDULE_TEST_SERVICE_ACCOUNT")
	}
	defer s.cleanUp()

	pipeline, err := s.pipelineUploadClient.UploadFile("../resources/arguments-parameters.yaml", upload_params.NewUploadPipelineParams())
	require.NoError(t, err)
	// Pipeline-version creation timestamps have second precision.
	time.Sleep(time.Second)
	version, err := s.pipelineUploadClient.UploadPipelineVersion("../resources/arguments-parameters.yaml", &upload_params.UploadPipelineVersionParams{
		Name:       util.StringPointer("authorized-schedule-version"),
		Pipelineid: util.StringPointer(pipeline.PipelineID),
	})
	require.NoError(t, err)

	for _, pinned := range []bool{false, true} {
		name := "latest"
		if pinned {
			name = "pinned"
		}
		s.Run(name, func() {
			t := s.T()
			experiment, err := s.experimentClient.Create(&experiment_params.ExperimentServiceCreateExperimentParams{
				Experiment: test.MakeExperiment("authorized-schedule-"+name, "", s.resourceNamespace),
			})
			require.NoError(t, err)
			reference := &recurring_run_model.V2beta1PipelineVersionReference{PipelineID: pipeline.PipelineID}
			if pinned {
				reference.PipelineVersionID = version.PipelineVersionID
			}
			schedule, err := s.recurringRunClient.Create(&recurring_run_params.RecurringRunServiceCreateRecurringRunParams{
				RecurringRun: &recurring_run_model.V2beta1RecurringRun{
					DisplayName:              "authorized-schedule-" + name,
					ExperimentID:             experiment.ExperimentID,
					PipelineVersionReference: reference,
					RuntimeConfig: &recurring_run_model.V2beta1RuntimeConfig{Parameters: map[string]interface{}{
						"param1": "authorized", "param2": "schedule",
					}},
					ServiceAccount: serviceAccount,
					MaxConcurrency: 1,
					Mode:           recurring_run_model.RecurringRunModeDISABLE.Pointer(),
				},
			})
			require.NoError(t, err)

			// The UID binds a CR to its API recurring run; never select by a user label.
			workflowClient := s.swfClient.ScheduledWorkflow(s.resourceNamespace)
			workflows, err := workflowClient.List(context.Background(), metav1.ListOptions{})
			require.NoError(t, err)
			workflowName := ""
			for _, workflow := range workflows.Items {
				if string(workflow.UID) == schedule.RecurringRunID {
					workflowName = workflow.Name
					break
				}
			}
			require.NotEmpty(t, workflowName, "the disabled schedule must have a backing ScheduledWorkflow")
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				workflow, err := workflowClient.Get(context.Background(), workflowName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				workflow.Spec.ServiceAccount = "tampered-schedule-account"
				workflow.Spec.PipelineId = "tampered-pipeline-id"
				workflow.Spec.PipelineVersionId = "tampered-version-id"
				if workflow.Spec.Workflow == nil {
					workflow.Spec.Workflow = &swfapi.WorkflowResource{}
				}
				workflow.Spec.Workflow.Parameters = []swfapi.Parameter{{Name: "param1", Value: `"tampered"`}}
				_, err = workflowClient.Update(context.Background(), workflow)
				return err
			})
			require.NoError(t, err)
			require.NoError(t, s.recurringRunClient.Enable(&recurring_run_params.RecurringRunServiceEnableRecurringRunParams{
				RecurringRunID: schedule.RecurringRunID,
			}))

			patched, err := workflowClient.Get(context.Background(), workflowName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, "tampered-schedule-account", patched.Spec.ServiceAccount)
			require.Equal(t, "tampered-pipeline-id", patched.Spec.PipelineId)

			var executed *run_model.V2beta1Run
			err = retrier.New(retrier.ConstantBackoff(120, 5*time.Second), nil).Run(func() error {
				runs, _, _, err := s.runClient.List(&run_params.RunServiceListRunsParams{ExperimentID: util.StringPointer(experiment.ExperimentID)})
				if err != nil {
					return err
				}
				if len(runs) != 1 {
					return fmt.Errorf("expected one scheduled run, got %d", len(runs))
				}
				executed = runs[0]
				if executed.State == nil {
					return fmt.Errorf("scheduled run %s has no state yet", executed.RunID)
				}
				if *executed.State != run_model.V2beta1RuntimeStateSUCCEEDED {
					return fmt.Errorf("scheduled run %s has not succeeded: %s", executed.RunID, *executed.State)
				}
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, schedule.RecurringRunID, executed.RecurringRunID)
			require.Equal(t, serviceAccount, executed.ServiceAccount)
			require.NotNil(t, executed.RuntimeConfig)
			require.Equal(t, "authorized", executed.RuntimeConfig.Parameters["param1"])
			require.Equal(t, "schedule", executed.RuntimeConfig.Parameters["param2"])
			require.NotNil(t, executed.PipelineVersionReference)
			require.Equal(t, pipeline.PipelineID, executed.PipelineVersionReference.PipelineID)
			require.Equal(t, version.PipelineVersionID, executed.PipelineVersionReference.PipelineVersionID)
		})
	}
}
