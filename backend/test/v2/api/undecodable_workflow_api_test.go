// Copyright 2026 The Kubeflow Authors
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

package api

import (
	"context"
	"path/filepath"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	undecodableWorkflowName = "kfp-test-undecodable-workflow"
	// Long enough for the hello-world pipeline to run and for the persistence
	// agent to report it, in seconds as WaitForRunToBeInState expects.
	undecodableWorkflowRunTimeout = 600
)

// The Workflow CRD that KFP installs does not type the contents of spec, so the
// Kubernetes API server stores a Workflow whose field has the wrong JSON type.
// The persistence agent and the ScheduledWorkflow controller watch Workflows
// through one shared cache each. This verifies, against the real components and
// a real API server, that such an object does not keep them from reporting
// another Workflow.
var _ = Describe("Undecodable Workflow >", Serial, Label(constants.POSITIVE, constants.APIServerTests, "UndecodableWorkflow"), func() {
	workflows := schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}
	var client dynamic.ResourceInterface

	BeforeEach(func() {
		restConfig, err := commonutil.GetKubernetesConfig()
		Expect(err).NotTo(HaveOccurred())
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		Expect(err).NotTo(HaveOccurred())
		client = dynamicClient.Resource(workflows).Namespace(testutil.GetNamespace())

		undecodable := &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]interface{}{
				"name": undecodableWorkflowName,
				// Argo's own controller does not pick up completed Workflows.
				"labels": map[string]interface{}{"workflows.argoproj.io/completed": "true"},
			},
			// An integer field with a string value.
			"spec": map[string]interface{}{"activeDeadlineSeconds": "not-a-number"},
		}}
		_, err = client.Create(context.Background(), undecodable, metav1.CreateOptions{})
		if apierrors.IsInvalid(err) {
			Skip("The installed Workflow CRD validates spec, so this Workflow cannot be stored")
		}
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			err := client.Delete(context.Background(), undecodableWorkflowName, metav1.DeleteOptions{})
			if !apierrors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	It("does not keep a later run from being reported as succeeded", func() {
		pipelineFile := filepath.Join(testutil.GetValidPipelineFilesDir(), helloWorldPipelineFileName)
		createdExperiment := createExperiment(experimentName)
		createdPipeline := uploadAPipeline(pipelineFile, &testContext.Pipeline.PipelineGeneratedName)
		createdPipelineVersion := testutil.GetLatestPipelineVersion(pipelineClient, &createdPipeline.PipelineID)
		createdRun := createPipelineRun(
			&createdPipeline.PipelineID,
			&createdPipelineVersion.PipelineVersionID,
			&createdExperiment.ExperimentID,
			testutil.GetPipelineRunTimeInputs(pipelineFile),
		)

		timeout := time.Duration(undecodableWorkflowRunTimeout)
		testutil.WaitForRunToBeInState(
			runClient,
			&createdRun.RunID,
			[]run_model.V2RuntimeState{run_model.V2RuntimeStateSUCCEEDED},
			&timeout,
		)

		// The object was there the whole time.
		_, err := client.Get(context.Background(), undecodableWorkflowName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})
