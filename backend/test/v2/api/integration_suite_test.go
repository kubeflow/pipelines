package api

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/client-go/kubernetes"

	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

// Test Context
var testContext TestContext
var pipelineFilesRootDir = utils.GetPipelineFilesDir()

// API Client Objects
var (
	k8Client             *kubernetes.Clientset
	pipelineUploadClient api_server.PipelineUploadInterface
	pipelineClient       *api_server.PipelineClient
)

// Test Reporting Variables
var testLogsDirectory = "logs"
var testReportDirectory = "reports"
var junitReportFilename = "junit.xml"
var jsonReportFilename = "report.json"

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))
	var newPipelineClient func() (*api_server.PipelineClient, error)

	if *isKubeflowMode {
		logger.Log("Creating API Clients for Multi User Mode")
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(*namespace, *isDebugMode)
		}
	} else {
		logger.Log("Creating API Clients for Single User Mode")
		clientConfig := utils.GetClientConfig(*namespace)

		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
	}

	pipelineUploadClient, err = test.GetPipelineUploadClient(
		*uploadPipelinesWithKubernetes,
		*isKubeflowMode,
		*isDebugMode,
		*namespace,
		utils.GetClientConfig(*namespace),
	)

	Expect(err).To(BeNil(), "Failed to get Pipeline Upload Client")
	pipelineClient, err = newPipelineClient()
	Expect(err).To(BeNil(), "Failed to get Pipeline Client")
	k8Client, err = initK8sClient()
	Expect(err).To(BeNil(), "Failed to initialize K8s client")
})

var _ = BeforeEach(func() {
	logger.Log("################### Global Setup before each test #####################")
	testContext = TestContext{
		TestStartTimeUTC: time.Now(),
	}
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	logger.Log("################### Cleanup after each test #####################")
	logger.Log("Deleting %d pipeline(s)", len(testContext.CreatedPipelines))
	for _, pipeline := range testContext.CreatedPipelines {
		utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing pod logs from %v to %v", testContext.TestStartTimeUTC, time.Now().UTC())
		podLogs := utils.ReadContainerLogs(k8Client, *namespace, "pipeline-api-server", nil, &testContext.TestStartTimeUTC, podLogLimit)
		AddReportEntry("Pod Log", podLogs)
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)
		writeLogFile(specReport)
	} else {
		log.Printf("Test passed")
	}
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.FailFast = false
	reporterConfig.ForceNewlines = true
	reporterConfig.SilenceSkips = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "API Tests Suite", suiteConfig, reporterConfig)
}

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

func initK8sClient() (*kubernetes.Clientset, error) {
	restConfig, configErr := util.GetKubernetesConfig()
	if configErr != nil {
		return nil, configErr
	}
	k8sClient, clientErr := kubernetes.NewForConfig(restConfig)
	if clientErr != nil {
		return nil, clientErr
	}
	return k8sClient, nil
}

func writeLogFile(specReport types.SpecReport) {
	stdOutput := specReport.CapturedGinkgoWriterOutput
	testName := GinkgoT().Name()
	testLogFile := filepath.Join(testLogsDirectory, testName+".log")
	logFile, err := os.Create(testLogFile)
	if err != nil {
		logger.Log("Failed to create log file due to: %s", err.Error())
	}
	_, err = logFile.Write([]byte(stdOutput))
	if err != nil {
		logger.Log("Failed to write to the log file, due to: %s", err.Error())
	}
	logFile.Close()
}
