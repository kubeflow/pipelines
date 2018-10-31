package test

//
//import (
//	"context"
//	"fmt"
//	"testing"
//	"time"
//
//	"encoding/json"
//
//	"net/url"
//
//	"io/ioutil"
//
//	"github.com/golang/glog"
//	"github.com/golang/protobuf/ptypes/timestamp"
//	api "github.com/googleprivate/ml/backend/api/go_client"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/suite"
//	"google.golang.org/grpc"
//)
//
//// This test suit tests various methods to import pipeline to pipeline system, including
//// - upload yaml file
//// - upload tarball file
//// - providing YAML file url
//// - Providing tarball file url
//type PipelineApiTest struct {
//	suite.Suite
//	namespace      string
//	conn           *grpc.ClientConn
//	pipelineClient api.PipelineServiceClient
//}
//
//// Check the namespace have ML job installed and ready
//func (s *PipelineApiTest) SetupTest() {
//	err := waitForReady(*namespace, *initializeTimeout)
//	if err != nil {
//		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
//	}
//	s.namespace = *namespace
//	s.conn, err = getRpcConnection(s.namespace)
//	if err != nil {
//		glog.Exitf("Failed to get RPC connection. Error: %s", err.Error())
//	}
//	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
//}
//
//func (s *PipelineApiTest) TearDownTest() {
//	s.conn.Close()
//}
//
//func (s *PipelineApiTest) TestPipelineAPI() {
//	t := s.T()
//	clientSet, err := getKubernetesClient()
//	if err != nil {
//		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
//	}
//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//	defer cancel()
//
//	requestStartTime := time.Now().Unix()
//
//	/* ---------- Upload pipelines YAML ---------- */
//	pipelineBody, writer := uploadPipelineFileOrFail("resources/arguments-parameters.yaml")
//	response, err := clientSet.RESTClient().Post().
//		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
//		SetHeader("Content-Type", writer.FormDataContentType()).
//		Body(pipelineBody).Do().Raw()
//	assert.Nil(t, err)
//	var argumentYAMLPipeline api.Pipeline
//	json.Unmarshal(response, &argumentYAMLPipeline)
//	assert.Equal(t, "arguments-parameters.yaml", argumentYAMLPipeline.Name)
//
//	/* ---------- Upload the same pipeline again. Should fail due to name uniqueness ---------- */
//	pipelineBody, writer = uploadPipelineFileOrFail("resources/arguments-parameters.yaml")
//	_, err = clientSet.RESTClient().Post().
//		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
//		SetHeader("Content-Type", writer.FormDataContentType()).
//		Body(pipelineBody).Do().Raw()
//	assert.NotNil(t, err)
//	assert.Contains(t, err.Error(), "Please specify a new name.")
//
//	/* ---------- Import pipeline YAML by URL ---------- */
//	sequentialPipeline, err := s.pipelineClient.CreatePipeline(
//		ctx, &api.CreatePipelineRequest{
//			Url:  &api.Url{PipelineUrl: "https://storage.googleapis.com/ml-pipeline-dataset/sequential.yaml"},
//			Name: "sequential"})
//	assert.Nil(t, err)
//	assert.Equal(t, "sequential", sequentialPipeline.Name)
//
//	/* ---------- Upload pipelines tarball ---------- */
//	pipelineBody, writer = uploadPipelineFileOrFail("resources/arguments.tar.gz")
//	response, err = clientSet.RESTClient().Post().
//		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
//		Param("name", url.PathEscape("arguments-parameters")).
//		SetHeader("Content-Type", writer.FormDataContentType()).
//		Body(pipelineBody).Do().Raw()
//	assert.Nil(t, err)
//	var argumentUploadPipeline api.Pipeline
//	json.Unmarshal(response, &argumentUploadPipeline)
//	assert.Equal(t, "arguments-parameters", argumentUploadPipeline.Name)
//
//	/* ---------- Import pipeline tarball by URL ---------- */
//	argumentUrlPipeline, err := s.pipelineClient.CreatePipeline(
//		ctx, &api.CreatePipelineRequest{
//			Url:  &api.Url{PipelineUrl: "https://storage.googleapis.com/ml-pipeline-dataset/arguments.tar.gz"},
//			Name: "url-arguments-parameters"})
//	assert.Nil(t, err)
//	assert.Equal(t, "url-arguments-parameters", argumentUrlPipeline.Name)
//
//	/* ---------- Verify list pipeline works ---------- */
//	listPipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
//	checkListPipelinesResponse(t, listPipelineResponse, err, requestStartTime)
//
//	/* ---------- Verify list pipeline sorted by names ---------- */
//	listFirstPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "name"})
//	assert.Nil(t, err)
//	assert.Equal(t, 2, len(listFirstPagePipelineResponse.Pipelines))
//	assert.Equal(t, "arguments-parameters", listFirstPagePipelineResponse.Pipelines[0].Name)
//	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePipelineResponse.Pipelines[1].Name)
//	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)
//
//	listSecondPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "name"})
//	assert.Nil(t, err)
//	assert.Equal(t, 2, len(listSecondPagePipelineResponse.Pipelines))
//	assert.Equal(t, "sequential", listSecondPagePipelineResponse.Pipelines[0].Name)
//	assert.Equal(t, "url-arguments-parameters", listSecondPagePipelineResponse.Pipelines[1].Name)
//	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)
//
//	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
//	_, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "description"})
//	assert.NotNil(t, err)
//	assert.Contains(t, err.Error(), "InvalidArgument")
//
//	/* ---------- List pipelines sorted by names descend order ---------- */
//	listFirstPagePipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "name desc"})
//	assert.Nil(t, err)
//	assert.Equal(t, 2, len(listFirstPagePipelineResponse.Pipelines))
//	assert.Equal(t, "url-arguments-parameters", listFirstPagePipelineResponse.Pipelines[0].Name)
//	assert.Equal(t, "sequential", listFirstPagePipelineResponse.Pipelines[1].Name)
//	assert.NotEmpty(t, listFirstPagePipelineResponse.NextPageToken)
//
//	listSecondPagePipelineResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "name desc"})
//	assert.Nil(t, err)
//	assert.Equal(t, 2, len(listSecondPagePipelineResponse.Pipelines))
//	assert.Equal(t, "arguments-parameters.yaml", listSecondPagePipelineResponse.Pipelines[0].Name)
//	assert.Equal(t, "arguments-parameters", listSecondPagePipelineResponse.Pipelines[1].Name)
//	assert.Empty(t, listSecondPagePipelineResponse.NextPageToken)
//
//	/* ---------- Verify get pipeline works ---------- */
//	getPipelineResponse, err := s.pipelineClient.GetPipeline(ctx, &api.GetPipelineRequest{Id: argumentYAMLPipeline.Id})
//	checkGetPipelineResponse(t, getPipelineResponse, err, requestStartTime)
//
//	/* ---------- Verify get template works ---------- */
//	getTmpResponse, err := s.pipelineClient.GetTemplate(ctx, &api.GetTemplateRequest{Id: argumentYAMLPipeline.Id})
//	checkGetTemplateResponse(t, getTmpResponse, err)
//
//	/* ---------- Clean up ---------- */
//	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: sequentialPipeline.Id})
//	assert.Nil(t, err)
//	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: argumentYAMLPipeline.Id})
//	assert.Nil(t, err)
//	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: argumentUploadPipeline.Id})
//	assert.Nil(t, err)
//	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: argumentUrlPipeline.Id})
//	assert.Nil(t, err)
//}
//
//func checkListPipelinesResponse(t *testing.T, response *api.ListPipelinesResponse, err error, requestStartTime int64) {
//	assert.Nil(t, err)
//	assert.Equal(t, 4, len(response.Pipelines))
//	verifyPipeline(t, response.Pipelines[0], requestStartTime)
//}
//
//func checkGetPipelineResponse(t *testing.T, pipeline *api.Pipeline, err error, requestStartTime int64) {
//	assert.Nil(t, err)
//	verifyPipeline(t, pipeline, requestStartTime)
//}
//
//func checkGetTemplateResponse(t *testing.T, response *api.GetTemplateResponse, err error) {
//	assert.Nil(t, err)
//	expected, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
//	assert.Nil(t, err)
//	assert.Equal(t, string(expected), response.Template)
//}
//
//func verifyPipeline(t *testing.T, pipeline *api.Pipeline, requestStartTime int64) {
//	// Only verify the time fields have valid value and in the right range.
//	assert.NotNil(t, *pipeline)
//	assert.NotNil(t, pipeline.CreatedAt)
//	// TODO: Investigate this. This is flaky for some reason.
//	//assert.True(t, pipeline.CreatedAt.GetSeconds() >= requestStartTime)
//	expected := api.Pipeline{
//		Id:        pipeline.Id,
//		CreatedAt: &timestamp.Timestamp{Seconds: pipeline.CreatedAt.Seconds},
//		Name:      "arguments-parameters.yaml",
//		Parameters: []*api.Parameter{
//			{Name: "param1", Value: "hello"}, // Default value in the pipeline template
//			{Name: "param2"},                 // No default value in the pipeline
//		},
//	}
//	assert.Equal(t, expected, *pipeline)
//}
//
//func TestPipelineAPI(t *testing.T) {
//	//suite.Run(t, new(PipelineApiTest))
//}
