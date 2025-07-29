package api

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"k8s.io/client-go/kubernetes"

	upload_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

const (
	helloWorldPipelineFileName = "hello-world.yaml"
	pipelineWithArgsFileName   = "arguments-parameters.yaml"
)

// API Clients declaration to make API calls
var k8Client *kubernetes.Clientset
var pipelineUploadClient *api_server.PipelineUploadClient
var pipelineClient *api_server.PipelineClient
var experimentClient *api_server.ExperimentClient
var runClient *api_server.RunClient

// Common test variables
var testStartTime strfmt.DateTime
var testStartTimeUTC time.Time
var randomName string
var pipelineFilesRootDir = utils.GetPipelineFilesDir()
var createdPipelines []*upload_model.V2beta1Pipeline
var createdRunIds []string
var createdExperimentIds []string
var pipelineUploadParams *upload_params.UploadPipelineParams
var pipelineGeneratedName string
var testLogsDirectory = "logs"
var testReportDirectory = "reports"
var junitReportFilename = "api.xml"
var jsonReportFilename = "api.json"

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	if err != nil {
		logger.Log("Error creating logs directory: %s", testLogsDirectory)
		return
	}
	err = os.MkdirAll(testReportDirectory, 0755)
	if err != nil {
		logger.Log("Error creating reports directory: %s", testReportDirectory)
		return
	}
	var newPipelineUploadClient func() (*api_server.PipelineUploadClient, error)
	var newPipelineClient func() (*api_server.PipelineClient, error)
	var newRunClient func() (*api_server.RunClient, error)
	var newExperimentClient func() (*api_server.ExperimentClient, error)

	if *isKubeflowMode {
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(*namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(*namespace, *isDebugMode)
		}
		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewKubeflowInClusterExperimentClient(*namespace, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewKubeflowInClusterRunClient(*namespace, *isDebugMode)
		}
	} else {
		clientConfig := utils.GetClientConfig(*namespace)

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
		}
		newExperimentClient = func() (*api_server.ExperimentClient, error) {
			return api_server.NewExperimentClient(clientConfig, *isDebugMode)
		}
		newRunClient = func() (*api_server.RunClient, error) {
			return api_server.NewRunClient(clientConfig, *isDebugMode)
		}
	}

	pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		logger.Log("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	pipelineClient, err = newPipelineClient()
	if err != nil {
		logger.Log("Failed to get pipeline client. Error: %s", err.Error())
	}
	experimentClient, err = newExperimentClient()
	if err != nil {
		logger.Log("Failed to get experiment client. Error: %v", err)
	}
	runClient, err = newRunClient()
	if err != nil {
		logger.Log("Failed to get run client. Error: %s", err.Error())
	}
	k8Client, err = initK8sClient()
	if err != nil {
		logger.Log("Failed to initialize K8s client. Error: %s", err.Error())
	}
})

var _ = BeforeEach(func() {
	logger.Log("################### Global Setup before each test #####################")
	testStartTime, _ = strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	testStartTimeUTC = time.Now().UTC()
	randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
	pipelineGeneratedName = "apitest-pipeline-" + randomName
	pipelineUploadParams = upload_params.NewUploadPipelineParams()
	createdPipelines = []*upload_model.V2beta1Pipeline{}
	createdRunIds = make([]string, 0)
	createdExperimentIds = make([]string, 0)
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	logger.Log("################### Global Cleanup after each test #####################")

	logger.Log("Deleting %d run(s)", len(createdRunIds))
	for _, runID := range createdRunIds {
		utils.TerminatePipelineRun(runClient, runID)
		utils.DeletePipelineRun(runClient, runID)
	}
	logger.Log("Deleting %d experiment(s)", len(createdExperimentIds))
	if len(createdExperimentIds) > 0 {
		for _, experimentID := range createdExperimentIds {
			utils.DeleteExperiment(experimentClient, experimentID)
		}
	}
	logger.Log("Deleting %d pipeline(s)", len(createdPipelines))
	for _, pipeline := range createdPipelines {
		utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
	}

})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing pod logs")
		podLogs := utils.ReadContainerLogs(k8Client, *namespace, "pipeline-api-server", nil, &testStartTimeUTC, podLogLimit)
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
	reporterConfig.GithubOutput = true
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
	if err != nil {
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
