package api

import (
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	test "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var pipelineUploadClient *api_server.PipelineUploadClient
var pipelineClient *api_server.PipelineClient
var testLogsDirectory = "logs"
var testReportDirectory = "reports"
var junitReportFilename = "junit.xml"
var jsonReportFilename = "report.json"

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

	if *isKubeflowMode {
		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewKubeflowInClusterPipelineUploadClient(*namespace, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewKubeflowInClusterPipelineClient(*namespace, *isDebugMode)
		}
	} else {
		clientConfig := test.GetClientConfig(*namespace)

		newPipelineUploadClient = func() (*api_server.PipelineUploadClient, error) {
			return api_server.NewPipelineUploadClient(clientConfig, *isDebugMode)
		}
		newPipelineClient = func() (*api_server.PipelineClient, error) {
			return api_server.NewPipelineClient(clientConfig, *isDebugMode)
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
})

var _ = ReportAfterEach(func(specReport types.SpecReport) {
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
	if specReport.Failed() {
		log.Printf("Test failed... Capture pod logs if you want to")
		AddReportEntry("Pod Log", "Pod Logs")
		AddReportEntry("Test Log", stdOutput)

	} else {
		log.Printf("Test passed")
	}
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.EmitSpecProgress = true
	suiteConfig.FailFast = false
	reporterConfig.Verbose = true
	reporterConfig.GithubOutput = true
	reporterConfig.ShowNodeEvents = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "API Tests Suite", suiteConfig, reporterConfig)
}
