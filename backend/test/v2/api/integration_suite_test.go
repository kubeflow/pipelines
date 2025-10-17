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

package api

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test/v2"

	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

// Test Context
var testContext *TestContext
var randomName string
var pipelineFilesRootDir = testutil.GetPipelineFilesDir()
var userToken string

var (
	pipelineUploadClient apiserver.PipelineUploadInterface
	pipelineClient       *apiserver.PipelineClient
	runClient            *apiserver.RunClient
	experimentClient     *apiserver.ExperimentClient
	recurringRunClient   *apiserver.RecurringRunClient
	k8Client             *kubernetes.Clientset
)

// Test Reporting Variables
var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "junit.xml"
	jsonReportFilename  = "api.json"
)

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))
	var newPipelineClient func() (*apiserver.PipelineClient, error)
	var newRunClient func() (*apiserver.RunClient, error)
	var newExperimentClient func() (*apiserver.ExperimentClient, error)
	var newRecurringRunClient func() (*apiserver.RecurringRunClient, error)
	clientConfig := testutil.GetClientConfig(*config.Namespace)
	k8Client, err = testutil.CreateK8sClient()
	Expect(err).To(BeNil(), "Failed to initialize K8s client")
	var tlsCfg *tls.Config
	if *config.TLSEnabled {
		tlsCfg, err = test.GetTLSConfig(*config.CaCertPath)
		if err != nil {
			log.Fatalf("Error getting TLS Config: %v", err)
		}
	}

	if *config.KubeflowMode {
		logger.Log("Creating API Clients for Kubeflow Mode")
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewKubeflowInClusterPipelineClient(*config.Namespace, *config.DebugMode, tlsCfg)
		}
		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewKubeflowInClusterExperimentClient(*config.Namespace, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewKubeflowInClusterRunClient(*config.Namespace, *config.DebugMode, tlsCfg)
		}
		newRecurringRunClient = func() (*apiserver.RecurringRunClient, error) {
			return apiserver.NewKubeflowInClusterRecurringRunClient(*config.Namespace, *config.DebugMode, tlsCfg)
		}
	} else if *config.MultiUserMode || *config.AuthToken != "" {
		if *config.AuthToken != "" {
			logger.Log("Creating API Clients With Auth Token")
			userToken = *config.AuthToken
		} else {
			logger.Log("Creating API Clients for Multi User Mode")
			userToken = testutil.CreateUserToken(k8Client, *config.UserNamespace, *config.UserServiceAccountName)
		}
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewMultiUserPipelineClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewMultiUserExperimentClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewMultiUserRunClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
		newRecurringRunClient = func() (*apiserver.RecurringRunClient, error) {
			return apiserver.NewMultiUserRecurringRunClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
	} else {
		logger.Log("Creating API Clients for Single User Mode")
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewPipelineClient(clientConfig, *config.DebugMode, tlsCfg)
		}
		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewExperimentClient(clientConfig, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewRunClient(clientConfig, *config.DebugMode, tlsCfg)
		}
		newRecurringRunClient = func() (*apiserver.RecurringRunClient, error) {
			return apiserver.NewRecurringRunClient(clientConfig, *config.DebugMode, tlsCfg)
		}
	}

	pipelineUploadClient, err = testutil.GetPipelineUploadClient(
		*config.UploadPipelinesWithKubernetes,
		*config.KubeflowMode,
		*config.DebugMode,
		*config.Namespace,
		clientConfig,
		tlsCfg,
	)

	Expect(err).To(BeNil(), "Failed to get Pipeline Upload Client")
	pipelineClient, err = newPipelineClient()
	Expect(err).To(BeNil(), "Failed to get Pipeline Client")
	experimentClient, err = newExperimentClient()
	Expect(err).To(BeNil(), "Failed to get Experiment client")
	runClient, err = newRunClient()
	Expect(err).To(BeNil(), "Failed to get Pipeline Run client")
	recurringRunClient, err = newRecurringRunClient()
	Expect(err).To(BeNil(), "Failed to get Recurring Run client")
})

var _ = BeforeEach(func() {
	logger.Log("################### Global Setup before each test #####################")
	testContext = &TestContext{
		TestStartTimeUTC: time.Now(),
	}
	randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
	testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
	testContext.Pipeline.UploadParams = uploadparams.NewUploadPipelineParams()
	testContext.PipelineRun.CreatedRunIds = make([]string, 0)
	testContext.Experiment.CreatedExperimentIds = make([]string, 0)
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	logger.Log("################### Global Cleanup after each test #####################")

	logger.Log("Deleting %d run(s)", len(testContext.PipelineRun.CreatedRunIds))
	for _, runID := range testContext.PipelineRun.CreatedRunIds {
		testutil.TerminatePipelineRun(runClient, runID)
		testutil.DeletePipelineRun(runClient, runID)
	}
	logger.Log("Deleting %d experiment(s)", len(testContext.Experiment.CreatedExperimentIds))
	if len(testContext.Experiment.CreatedExperimentIds) > 0 {
		for _, experimentID := range testContext.Experiment.CreatedExperimentIds {
			testutil.DeleteExperiment(experimentClient, experimentID)
		}
	}
	logger.Log("Deleting %d pipeline(s)", len(testContext.Pipeline.CreatedPipelines))
	for _, pipeline := range testContext.Pipeline.CreatedPipelines {
		testutil.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing pod logs from %v to %v", testContext.TestStartTimeUTC, time.Now().UTC())
		podLogs := testutil.ReadContainerLogs(k8Client, *config.Namespace, "pipeline-api-server", nil, &testContext.TestStartTimeUTC, config.PodLogLimit)
		AddReportEntry("Pod Log", podLogs)
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
		currentDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred(), "Failed to get current directory")
		testutil.WriteLogFile(specReport, GinkgoT().Name(), filepath.Join(currentDir, testLogsDirectory))
	} else {
		log.Printf("Test passed")
	}
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfigAPI, reporterConfigAPI := GinkgoConfiguration()
	suiteConfigAPI.FailFast = false
	reporterConfigAPI.ForceNewlines = true
	reporterConfigAPI.SilenceSkips = true
	reporterConfigAPI.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfigAPI.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "API Tests Suite", suiteConfigAPI, reporterConfigAPI)
}
