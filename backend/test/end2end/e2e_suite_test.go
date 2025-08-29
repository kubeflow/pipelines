package end2end

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/test_utils"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"k8s.io/client-go/kubernetes"

	apiserver "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2"
	apitests "github.com/kubeflow/pipelines/backend/test/v2/api"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

// Test Context
var testContext apitests.TestContext

var (
	pipelineUploadClient apiserver.PipelineUploadInterface
	pipelineClient       *apiserver.PipelineClient
	runClient            *apiserver.RunClient
	k8Client             *kubernetes.Clientset
)

// Test Reporting Variables
var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "e2e_junit.xml"
	jsonReportFilename  = "e2e.json"
)

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))
	var newPipelineClient func() (*apiserver.PipelineClient, error)
	var newRunClient func() (*apiserver.RunClient, error)

	if *config.IsKubeflowMode {
		logger.Log("Creating API Clients for Multi User Mode")
		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewKubeflowInClusterPipelineClient(*config.Namespace, *config.IsDebugMode)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewKubeflowInClusterRunClient(*config.Namespace, *config.IsDebugMode)
		}
	} else {
		logger.Log("Creating API Clients for Single User Mode")
		clientConfig := test_utils.GetClientConfig(*config.Namespace)

		newPipelineClient = func() (*apiserver.PipelineClient, error) {
			return apiserver.NewPipelineClient(clientConfig, *config.IsDebugMode)
		}
		newRunClient = func() (*apiserver.RunClient, error) {
			return apiserver.NewRunClient(clientConfig, *config.IsDebugMode)
		}
	}

	pipelineUploadClient, err = test.GetPipelineUploadClient(
		*config.UploadPipelinesWithKubernetes,
		*config.IsKubeflowMode,
		*config.IsDebugMode,
		*config.Namespace,
		test_utils.GetClientConfig(*config.Namespace),
	)

	Expect(err).To(BeNil(), "Failed to get Pipeline Upload Client")
	pipelineClient, err = newPipelineClient()
	Expect(err).To(BeNil(), "Failed to get Pipeline Client")
	runClient, err = newRunClient()
	Expect(err).To(BeNil(), "Failed to get Pipeline Run client")
	k8Client, err = initK8sClient()
	Expect(err).To(BeNil(), "Failed to initialize K8s client")
})

var _ = BeforeEach(func() {
	logger.Log("################### Global Setup before each test #####################")
	testContext = apitests.TestContext{
		TestStartTimeUTC: time.Now(),
	}
	testContext.PipelineRun.CreatedRunIds = make([]string, 0)
	testContext.Experiment.CreatedExperimentIds = make([]string, 0)
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	logger.Log("################### Global Cleanup after each test #####################")

	logger.Log("Deleting %d run(s)", len(testContext.PipelineRun.CreatedRunIds))
	for _, runID := range testContext.PipelineRun.CreatedRunIds {
		test_utils.TerminatePipelineRun(runClient, runID)
		test_utils.DeletePipelineRun(runClient, runID)
	}
	logger.Log("Deleting %d pipeline(s)", len(testContext.Pipeline.CreatedPipelines))
	for _, pipeline := range testContext.Pipeline.CreatedPipelines {
		test_utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing logs")
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
