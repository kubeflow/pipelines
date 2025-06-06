package api

import (
	"flag"
	"github.com/golang/glog"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	test "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

var pipelineUploadClient *api_server.PipelineUploadClient
var pipelineClient *api_server.PipelineClient
var parallelProcesses = flag.Int("parallelProcesses", 1, "Number of tests to parallel in parallel")

var _ = BeforeSuite(func() {

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

	var err error
	pipelineUploadClient, err = newPipelineUploadClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline upload client. Error: %s", err.Error())
	}
	pipelineClient, err = newPipelineClient()
	if err != nil {
		glog.Exitf("Failed to get pipeline client. Error: %s", err.Error())
	}
})

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.ParallelTotal = *parallelProcesses
	suiteConfig.EmitSpecProgress = true
	suiteConfig.FailFast = false
	reporterConfig.Verbose = true
	reporterConfig.GithubOutput = true
	reporterConfig.ShowNodeEvents = true
	RunSpecs(t, "API Tests Suite", suiteConfig, reporterConfig)
}
