package main

import (
	"context"
	"fmt"
	"ml/backend/api"
	"testing"

	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

type PaginationTestSuit struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	packageClient  api.PackageServiceClient
	pipelineClient api.PipelineServiceClient
	jobClient      api.JobServiceClient
}

// Check the namespace have ML pipeline installed and ready
func (s *PaginationTestSuit) SetupTest() {
	err := waitForReady(*namespace, *initializeTimeout)
	if err != nil {
		glog.Exit("Failed to initialize test. Error: %s", err.Error())
	}
	s.namespace = *namespace
	s.conn, err = getRpcConnection(s.namespace)
	if err != nil {
		glog.Exit("Failed to get RPC connection. Error: %s", err.Error())
	}
	s.packageClient = api.NewPackageServiceClient(s.conn)
	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
	s.jobClient = api.NewJobServiceClient(s.conn)
}

func (s *PaginationTestSuit) TearDownTest() {
	s.conn.Close()
}

func (s *PaginationTestSuit) TestPagination() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Upload three packages ---------- */
	pkgBody, writer := uploadPackageFileOrFail("resources/hello-world.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "packages/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pkgBody).Do().Raw()
	assert.Nil(t, err)

	pkgBody, writer = uploadPackageFileOrFail("resources/arguments-parameters.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "packages/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pkgBody).Do().Raw()
	assert.Nil(t, err)

	pkgBody, writer = uploadPackageFileOrFail("resources/loops.yaml")
	_, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "packages/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pkgBody).Do().Raw()
	assert.Nil(t, err)

	/* ---------- List package sorted by names ---------- */
	listFirstPagePackageResponse, err := s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePackageResponse.Packages))
	assert.Equal(t, "arguments-parameters.yaml", listFirstPagePackageResponse.Packages[0].Name)
	assert.Equal(t, "hello-world.yaml", listFirstPagePackageResponse.Packages[1].Name)
	assert.NotEmpty(t, listFirstPagePackageResponse.NextPageToken)

	listSecondPagePackageResponse, err := s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{PageToken: listFirstPagePackageResponse.NextPageToken, PageSize: 2, SortBy: "name"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPagePackageResponse.Packages))
	assert.Equal(t, "loops.yaml", listSecondPagePackageResponse.Packages[0].Name)
	assert.Empty(t, listSecondPagePackageResponse.NextPageToken)

	argumentsParametersPkgId := listFirstPagePackageResponse.Packages[0].Id
	helloWorldPkgId := listFirstPagePackageResponse.Packages[1].Id
	loopsPkgId := listSecondPagePackageResponse.Packages[0].Id

	/* ---------- List packages sort by unsupported description field. Should fail. ---------- */
	_, err = s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	/* ---------- Instantiate 3 pipeline using the packages ---------- */
	loopsPipeline := &api.Pipeline{
		Name:      "loops",
		PackageId: loopsPkgId,
	}
	loopsPipeline, err = s.pipelineClient.CreatePipeline(ctx, &api.CreatePipelineRequest{Pipeline: loopsPipeline})
	assert.Nil(t, err)

	argumentsParametersPipeline := &api.Pipeline{
		Name:      "arguments-parameters",
		PackageId: argumentsParametersPkgId,
		Parameters: []*api.Parameter{
			{Name: "param2", Value: "world"},
		},
	}
	argumentsParametersPipeline, err = s.pipelineClient.CreatePipeline(ctx, &api.CreatePipelineRequest{Pipeline: argumentsParametersPipeline})
	assert.Nil(t, err)

	helloWorldPipeline := &api.Pipeline{
		Name:      "hello-world",
		PackageId: helloWorldPkgId,
	}
	helloWorldPipeline, err = s.pipelineClient.CreatePipeline(ctx, &api.CreatePipelineRequest{Pipeline: helloWorldPipeline})
	assert.Nil(t, err)

	/* ---------- List pipelines sorted by names ---------- */
	listFirstPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(listFirstPagePipelineResponse.Pipelines))
	assert.Equal(t, "loops", listFirstPagePipelineResponse.Pipelines[0].Name)
	assert.Equal(t, "arguments-parameters", listFirstPagePipelineResponse.Pipelines[1].Name)
	assert.NotEmpty(t, listFirstPagePackageResponse.NextPageToken)

	listSecondPagePipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageToken: listFirstPagePipelineResponse.NextPageToken, PageSize: 2, SortBy: "created_at"})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(listSecondPagePipelineResponse.Pipelines))
	assert.Equal(t, "hello-world", listSecondPagePipelineResponse.Pipelines[0].Name)
	assert.Empty(t, listSecondPagePackageResponse.NextPageToken)

	/* ---------- List pipelines sort by unsupported description field. Should fail. ---------- */
	_, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{PageSize: 2, SortBy: "description"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "InvalidArgument")

	// TODO(https://github.com/googleprivate/ml/issues/473): Add tests for list jobs.

	/* ---------- Clean up ---------- */
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: argumentsParametersPipeline.Id})
	assert.Nil(t, err)
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: helloWorldPipeline.Id})
	assert.Nil(t, err)
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: loopsPipeline.Id})
	assert.Nil(t, err)

	_, err = s.packageClient.DeletePackage(ctx, &api.DeletePackageRequest{Id: argumentsParametersPkgId})
	assert.Nil(t, err)
	_, err = s.packageClient.DeletePackage(ctx, &api.DeletePackageRequest{Id: helloWorldPkgId})
	assert.Nil(t, err)
	_, err = s.packageClient.DeletePackage(ctx, &api.DeletePackageRequest{Id: loopsPkgId})
	assert.Nil(t, err)

}

func TestPagination(t *testing.T) {
	suite.Run(t, new(PaginationTestSuit))
}
