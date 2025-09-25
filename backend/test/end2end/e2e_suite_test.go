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

package end2end

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/test_utils"
	"github.com/kubeflow/pipelines/backend/test/v2"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

var randomName string
var experimentID *string = nil
var userToken string

const maxPipelineWaitTime = 900 // In Seconds

var (
	pipelineUploadClient apiserver.PipelineUploadInterface
	pipelineClient       *apiserver.PipelineClient
	runClient            *apiserver.RunClient
	experimentClient     *apiserver.ExperimentClient
	k8Client             *kubernetes.Clientset
)

// Test Reporting Variables
var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "junit.xml"
	jsonReportFilename  = "e2e.json"
)

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))
	var newPipelineClient func() (*apiserver.PipelineClient, error)
	var newRunClient func() (*apiserver.RunClient, error)
	var newExperimentClient func() (*apiserver.ExperimentClient, error)
	clientConfig := test_utils.GetClientConfig(*config.Namespace)
	k8Client, err = test_utils.CreateK8sClient()
	Expect(err).To(BeNil(), "Failed to initialize K8s client")
	var tlsCfg *tls.Config
	if *config.TlsEnabled {
		tlsCfg = test.GetTLSConfig(*config.CaCertPath)
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
	} else if *config.MultiUserMode {
		logger.Log("Creating API Clients for Multi User Mode")
		userToken = test_utils.CreateUserToken(k8Client, *config.UserNamespace, *config.UserServiceAccountName)
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewMultiUserPipelineClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
		newExperimentClient = func() (*apiserver.ExperimentClient, error) {
			return apiserver.NewMultiUserExperimentClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewMultiUserRunClient(clientConfig, userToken, *config.DebugMode, tlsCfg)
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
	}

	pipelineUploadClient, err = test.GetPipelineUploadClient(
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
})

var _ = BeforeEach(func() {

	// Create Experiment so that we can use it to associate pipeline runs with
	experimentName := fmt.Sprintf("E2EExperiment-%s", strconv.FormatInt(time.Now().UnixNano(), 10))
	experiment := test_utils.CreateExperiment(experimentClient, experimentName, test_utils.GetNamespace())
	experimentID = &experiment.ExperimentID
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing logs")
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
		currentDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred(), "Failed to get current directory")
		test_utils.WriteLogFile(specReport, GinkgoT().Name(), filepath.Join(currentDir, testLogsDirectory))
	} else {
		log.Printf("Test passed")
	}
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfigE2E, reporterConfigE2E := GinkgoConfiguration()
	suiteConfigE2E.FailFast = false
	reporterConfigE2E.ForceNewlines = true
	reporterConfigE2E.SilenceSkips = true
	reporterConfigE2E.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfigE2E.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "E2E Tests Suite", suiteConfigE2E, reporterConfigE2E)
}
