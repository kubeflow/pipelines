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

package localdocker

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

const localPipelineRootEnvVar = "KFP_LOCAL_PIPELINE_ROOT"

var randomName string
var experimentID *string
var testContext *apitests.TestContext

var (
	pipelineUploadClient apiserver.PipelineUploadInterface
	pipelineClient       *apiserver.PipelineClient
	runClient            *apiserver.RunClient
	experimentClient     *apiserver.ExperimentClient
)

var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "junit.xml"
	jsonReportFilename  = "local-docker-e2e.json"
)

func localPipelineRoot() string {
	return os.Getenv(localPipelineRootEnvVar)
}

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))

	Expect(localPipelineRoot()).NotTo(BeEmpty(), fmt.Sprintf("Expected %s to be set", localPipelineRootEnvVar))

	clientConfig := testutil.GetClientConfig(*config.Namespace)
	var tlsCfg *tls.Config
	if *config.TLSEnabled {
		tlsCfg, err = testutil.GetTLSConfig(*config.CaCertPath)
		if err != nil {
			log.Fatalf("Error getting TLS Config: %v", err)
		}
	}

	pipelineUploadClient, err = testutil.GetPipelineUploadClient(
		*config.UploadPipelinesWithKubernetes,
		*config.KubeflowMode,
		*config.DebugMode,
		*config.AuthToken,
		*config.Namespace,
		clientConfig,
		tlsCfg,
	)
	Expect(err).To(BeNil(), "Failed to get Pipeline Upload Client")

	pipelineClient, err = apiserver.NewPipelineClient(clientConfig, *config.DebugMode, tlsCfg)
	Expect(err).To(BeNil(), "Failed to get Pipeline Client")
	experimentClient, err = apiserver.NewExperimentClient(clientConfig, *config.DebugMode, tlsCfg)
	Expect(err).To(BeNil(), "Failed to get Experiment client")
	runClient, err = apiserver.NewRunClient(clientConfig, *config.DebugMode, tlsCfg)
	Expect(err).To(BeNil(), "Failed to get Pipeline Run client")
})

var _ = BeforeEach(func() {
	logger.Log("################### Local Docker E2E setup #####################")
	testContext = &apitests.TestContext{
		TestStartTimeUTC: time.Now(),
	}
	randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
	testContext.Pipeline.UploadParams = uploadparams.NewUploadPipelineParams()
	testContext.Pipeline.CreatedPipelines = make([]*pipeline_upload_model.V2beta1Pipeline, 0)
	testContext.PipelineRun.CreatedRunIds = make([]string, 0)
	testContext.Experiment.CreatedExperimentIds = make([]string, 0)

	experiment := testutil.CreateExperiment(experimentClient, "LocalDockerE2E-"+randomName, testutil.GetNamespace())
	testContext.Experiment.CreatedExperimentIds = append(testContext.Experiment.CreatedExperimentIds, experiment.ExperimentID)
	experimentID = &experiment.ExperimentID

	DeferCleanup(cleanupLocalDockerTestResources)
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if testContext == nil || !specReport.Failed() {
		return
	}
	AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
	currentDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred(), "Failed to get current directory")
	testutil.WriteLogFile(specReport, GinkgoT().Name(), filepath.Join(currentDir, testLogsDirectory))
})

func cleanupLocalDockerTestResources() {
	logger.Log("################### Local Docker E2E cleanup #####################")
	for _, runID := range testContext.PipelineRun.CreatedRunIds {
		testutil.TerminatePipelineRun(runClient, runID)
		testutil.ArchivePipelineRun(runClient, runID)
		testutil.DeletePipelineRun(runClient, runID)
	}
	for _, experimentID := range testContext.Experiment.CreatedExperimentIds {
		testutil.DeleteExperiment(experimentClient, experimentID)
	}
	for _, pipeline := range testContext.Pipeline.CreatedPipelines {
		testutil.DeletePipeline(pipelineClient, pipeline.PipelineID, true)
	}
}

func TestLocalDockerE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.FailFast = false
	reporterConfig.ForceNewlines = true
	reporterConfig.SilenceSkips = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "Local Docker E2E Suite", suiteConfig, reporterConfig)
}
