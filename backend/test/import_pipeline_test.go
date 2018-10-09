package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"encoding/json"

	"github.com/golang/glog"
	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

// This test suit tests various methods to import pipeline to pipeline system, including
// - upload yaml file
// - (TODO) upload tarball file
// - providing YAML file url
// - (TODO) Providing tarball file url
type ImportPipelineTest struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	pipelineClient api.PipelineServiceClient
}

// Check the namespace have ML job installed and ready
func (s *ImportPipelineTest) SetupTest() {
	err := waitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	s.conn, err = getRpcConnection(s.namespace)
	if err != nil {
		glog.Exitf("Failed to get RPC connection. Error: %s", err.Error())
	}
	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
}

func (s *ImportPipelineTest) TearDownTest() {
	s.conn.Close()
}

func (s *ImportPipelineTest) TestImportPipeline() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Upload pipelines YAML ---------- */
	pipelineBody, writer := uploadPipelineFileOrFail("resources/hello-world.yaml")
	response, err := clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pipelineBody).Do().Raw()
	assert.Nil(t, err)
	var helloWorldPipeline api.Pipeline
	json.Unmarshal(response, &helloWorldPipeline)
	assert.Equal(t, "hello-world.yaml", helloWorldPipeline.Name)

	/* ---------- Import pipeline YAML by URL ---------- */
	sequentialPipeline, err := s.pipelineClient.CreatePipeline(
		ctx, &api.CreatePipelineRequest{
			Url:  &api.Url{PipelineUrl: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"},
			Name: "sequential"})
	assert.Nil(t, err)
	assert.Equal(t, "sequential", sequentialPipeline.Name)

	/* ---------- Clean up ---------- */
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: sequentialPipeline.Id})
	assert.Nil(t, err)
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: helloWorldPipeline.Id})
	assert.Nil(t, err)
}

func TestImportPipeline(t *testing.T) {
	suite.Run(t, new(ImportPipelineTest))
}
