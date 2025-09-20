package compiler

import (
	"flag"
	"fmt"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/test_utils"
	"log"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

var pipelineFilesRootDir = test_utils.GetPipelineFilesDir()
var pipelineDirectory = "valid"
var argoYAMLDir = filepath.Join(test_utils.GetTestDataDir(), "compiled-workflows")
var updateGoldenFiles = flag.Bool("updateCompiledFiles", false, "update golden/expected compiled workflow files")
var createMissingGoldenFiles = flag.Bool("createGoldenFiles", false, "create missing golden/expected compiled workflow files")

// Test Reporting Variables
var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "compiler_junit.xml"
	jsonReportFilename  = "compiler.json"
)

var _ = BeforeSuite(func() {
	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))
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

func TestCompilation(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.FailFast = false
	reporterConfig.ForceNewlines = true
	reporterConfig.SilenceSkips = true
	reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "Compiler Tests", suiteConfig, reporterConfig)
}

// ####################################################################################################################################################################
// ################################################################### UTILITY METHODS ################################################################################
// ####################################################################################################################################################################

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
