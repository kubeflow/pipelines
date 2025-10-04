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

package compiler

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

var pipelineFilesRootDir = testutil.GetPipelineFilesDir()
var pipelineDirectory = "valid"
var argoYAMLDir = filepath.Join(testutil.GetTestDataDir(), "compiled-workflows")
var updateGoldenFiles = flag.Bool("updateCompiledFiles", false, "update golden/expected compiled workflow files")
var createMissingGoldenFiles = flag.Bool("createGoldenFiles", false, "create missing golden/expected compiled workflow files")

// Test Reporting Variables
var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
	junitReportFilename = "junit.xml"
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
		currentDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred(), "Failed to get current directory")
		testName := GinkgoT().Name()
		testNameSplit := strings.Split(testName, ">")
		testutil.WriteLogFile(specReport, testNameSplit[len(testNameSplit)-1], filepath.Join(currentDir, testLogsDirectory))
	} else {
		log.Printf("Test passed")
	}
})

func TestCompilation(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfigCompiler, reporterConfigCompiler := GinkgoConfiguration()
	suiteConfigCompiler.FailFast = false
	reporterConfigCompiler.ForceNewlines = true
	reporterConfigCompiler.SilenceSkips = true
	reporterConfigCompiler.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
	reporterConfigCompiler.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
	RunSpecs(t, "Compiler Tests", suiteConfigCompiler, reporterConfigCompiler)
}
