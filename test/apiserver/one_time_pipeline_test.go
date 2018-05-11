// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"ml/backend/api"
	"ml/backend/src/util"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

var namespace = flag.String("namespace", "default", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")
var testTimeout = flag.Duration("testTimeout", 2*time.Minute, "Duration to wait for the test to finish")

type OneTimePipelineTestSuite struct {
	suite.Suite
	namespace      string
	conn           *grpc.ClientConn
	packageClient  api.PackageServiceClient
	pipelineClient api.PipelineServiceClient
	jobClient      api.JobServiceClient
}

// Check the namespace have ML pipeline installed and ready
func (s *OneTimePipelineTestSuite) SetupTest() {
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

func (s *OneTimePipelineTestSuite) TearDownTest() {
	s.conn.Close()
}

// This test case tests a basic use case:
// - Upload a package
// - Create a pipeline with parameter
// - Verify an one-time job is automatically scheduled.
func (s *OneTimePipelineTestSuite) TestOneTimePipeline() {
	t := s.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	/* ---------- Verify no package exist ---------- */
	listPkgResponse, err := s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{})
	checkNoPackageExists(t, listPkgResponse, err)

	/* ---------- Verify no pipeline exist ---------- */
	listPipelineResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkNoPipelineExists(t, listPipelineResponse, err)

	/* ---------- Upload a package ---------- */
	requestStartTime := time.Now().Unix()
	pkgBody, writer := uploadPackageFileOrFail("resources/arguments-parameters.yaml")
	uploadPkgResponse, err := clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "packages/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pkgBody).Do().Raw()
	checkUploadPackageResponse(t, uploadPkgResponse, err, requestStartTime)

	/* ---------- Verify list package works ---------- */
	listPkgResponse, err = s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{})
	checkListPackagesResponse(t, listPkgResponse, err, requestStartTime)

	/* ---------- Verify get package works ---------- */
	getPkgResponse, err := s.packageClient.GetPackage(ctx, &api.GetPackageRequest{Id: 1})
	checkGetPackageResponse(t, getPkgResponse, err, requestStartTime)

	/* ---------- Verify get template works ---------- */
	getTmpResponse, err := s.packageClient.GetTemplate(ctx, &api.GetTemplateRequest{Id: 1})
	checkGetTemplateResponse(t, getTmpResponse, err)

	/* ---------- Instantiate pipeline using the package ---------- */
	requestStartTime = time.Now().Unix()
	pipeline := &api.Pipeline{
		Name:        "hello-world",
		PackageId:   1,
		Description: "this is my first pipeline",
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "goodbye"},
			{Name: "param2", Value: "world"},
		},
	}
	newPipeline, err := s.pipelineClient.CreatePipeline(ctx, &api.CreatePipelineRequest{Pipeline: pipeline})
	checkInstantiatePipelineResponse(t, newPipeline, err, requestStartTime)

	/* ---------- Verify list pipelines works ---------- */
	listPipResponse, err := s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkListPipelinesResponse(t, listPipResponse, err, requestStartTime)

	/* ---------- Verify get pipeline works ---------- */
	getPipResponse, err := s.pipelineClient.GetPipeline(ctx, &api.GetPipelineRequest{Id: 1})
	checkGetPipelineResponse(t, getPipResponse, err, requestStartTime)

	/* ---------- Verify list job works ---------- */
	listJobsResponse, err := s.jobClient.ListJobs(ctx, &api.ListJobsRequest{PipelineId: 1})
	checkListJobsResponse(t, listJobsResponse, err, requestStartTime)
	jobName := listJobsResponse.Jobs[0].Name

	/* ---------- Verify job complete successfully ---------- */
	err = s.checkJobSucceed(clientSet, s.namespace, "1", jobName)
	if err != nil {
		assert.Fail(t, "The job doesn't complete. Error: "+err.Error())
	}

	/* ---------- Verify get job works ---------- */
	getJobResponse, err := s.jobClient.GetJob(ctx, &api.GetJobRequest{PipelineId: 1, JobName: jobName})
	checkGetJobResponse(t, getJobResponse, err, requestStartTime)

	/* ---------- Verify delete pipeline works ---------- */
	_, err = s.pipelineClient.DeletePipeline(ctx, &api.DeletePipelineRequest{Id: 1})
	assert.Nil(t, err)

	/* ---------- Verify no pipeline exist ---------- */
	listPipResponse, err = s.pipelineClient.ListPipelines(ctx, &api.ListPipelinesRequest{})
	checkNoPipelineExists(t, listPipResponse, err)

	/* ---------- Verify can't retrieve the job ---------- */
	_, err = s.jobClient.GetJob(ctx, &api.GetJobRequest{PipelineId: 1, JobName: jobName})
	assert.NotNil(t, err)

	/* ---------- Verify delete package works ---------- */
	_, err = s.packageClient.DeletePackage(ctx, &api.DeletePackageRequest{Id: 1})
	assert.Nil(t, err)

	/* ---------- Verify no package exist ---------- */
	listPkgResponse, err = s.packageClient.ListPackages(ctx, &api.ListPackagesRequest{})
	checkNoPackageExists(t, listPkgResponse, err)
}

func (s *OneTimePipelineTestSuite) checkJobSucceed(clientSet *kubernetes.Clientset, namespace string, pipelineId string, jobName string) error {
	var waitForJobSucceed = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		getJobResponse, err := s.jobClient.GetJob(ctx, &api.GetJobRequest{PipelineId: 1, JobName: jobName})
		if err != nil {
			return errors.New("Can't get job.")
		}
		status := getJobStatus(getJobResponse)
		if status == nil || (*status) == "" {
			return errors.New("Job hasn't started yet.")
		}
		if *status == v1alpha1.NodeSucceeded {
			return nil
		}
		if *status == v1alpha1.NodeRunning {
			return errors.New("Job is still running. Waiting for job to finish.")
		}
		return backoff.Permanent(errors.Errorf("Job failed to run with status " + string(*status)))
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = *testTimeout
	return backoff.Retry(waitForJobSucceed, b)
}

func getJobStatus(response *api.JobDetail) *v1alpha1.NodePhase {
	if response.Workflow != nil {
		var workflow v1alpha1.Workflow
		yaml.Unmarshal(response.Workflow, &workflow)
		return &workflow.Status.Phase
	}
	return nil
}

func checkNoPackageExists(t *testing.T, response *api.ListPackagesResponse, err error) {
	assert.Nil(t, err)
	assert.Empty(t, response.Packages)
}

func checkNoPipelineExists(t *testing.T, response *api.ListPipelinesResponse, err error) {
	assert.Nil(t, err)
	assert.Empty(t, response.Pipelines)
}

func checkUploadPackageResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pkg api.Package
	util.UnmarshalJsonOrFail(string(response), &pkg)
	verifyPackage(t, &pkg, requestStartTime)
}

func checkListPackagesResponse(t *testing.T, response *api.ListPackagesResponse, err error, requestStartTime int64) {
	assert.Nil(t, err)
	assert.Equal(t, 1, len(response.Packages))
	verifyPackage(t, response.Packages[0], requestStartTime)
}

func checkGetPackageResponse(t *testing.T, pkg *api.Package, err error, requestStartTime int64) {
	assert.Nil(t, err)
	verifyPackage(t, pkg, requestStartTime)
}

func checkGetTemplateResponse(t *testing.T, response *api.GetTemplateResponse, err error) {
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	assert.Equal(t, string(expected), response.Template)
}

func checkInstantiatePipelineResponse(t *testing.T, response *api.Pipeline, err error, requestStartTime int64) {
	assert.Nil(t, err)
	verifyPipeline(t, response, requestStartTime)
}

func checkListPipelinesResponse(t *testing.T, response *api.ListPipelinesResponse, err error, requestStartTime int64) {
	assert.Nil(t, err)
	assert.Equal(t, 1, len(response.Pipelines))
	verifyPipeline(t, response.Pipelines[0], requestStartTime)
}

func checkGetPipelineResponse(t *testing.T, response *api.Pipeline, err error, requestStartTime int64) {
	assert.Nil(t, err)
	verifyPipeline(t, response, requestStartTime)
}

func checkListJobsResponse(t *testing.T, response *api.ListJobsResponse, err error, requestStartTime int64) {
	assert.Nil(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.Jobs))
	verifyJob(t, response.Jobs[0], requestStartTime)
}

func checkGetJobResponse(t *testing.T, response *api.JobDetail, err error, requestStartTime int64) {
	assert.Nil(t, err)
	verifyJob(t, response.Job, requestStartTime)

	// The Argo workflow might not be created. Only verify if it's created.
	if response.Workflow != nil {
		var workflow v1alpha1.Workflow
		err = yaml.Unmarshal(response.Workflow, &workflow)
		assert.Nil(t, err)
		// Do some very basic verification to make sure argo workflow is returned
		assert.Equal(t, response.Job.Name, workflow.Name)
	}
}

func verifyPackage(t *testing.T, pkg *api.Package, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.NotNil(t, *pkg)
	assert.NotNil(t, pkg.CreatedAt)
	assert.True(t, pkg.CreatedAt.GetSeconds() >= requestStartTime)
	expected := api.Package{
		Id:        1,
		CreatedAt: &timestamp.Timestamp{Seconds: pkg.CreatedAt.Seconds},
		Name:      "arguments-parameters.yaml",
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "hello"}, // Default value in the package template
			{Name: "param2"},                 // No default value in the package
		},
	}
	assert.Equal(t, expected, *pkg)
}

func verifyPipeline(t *testing.T, pipeline *api.Pipeline, requestStartTime int64) {
	assert.NotNil(t, *pipeline)
	assert.NotNil(t, pipeline.CreatedAt)
	assert.NotNil(t, pipeline.EnabledAt)
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, pipeline.CreatedAt.Seconds >= requestStartTime)
	assert.True(t, pipeline.EnabledAt.Seconds >= requestStartTime)

	expected := api.Pipeline{
		Id:          1,
		CreatedAt:   &timestamp.Timestamp{Seconds: pipeline.CreatedAt.Seconds},
		Name:        "hello-world",
		Description: "this is my first pipeline",
		PackageId:   1,
		Enabled:     true,
		EnabledAt:   &timestamp.Timestamp{Seconds: pipeline.EnabledAt.Seconds},
		Parameters: []*api.Parameter{
			{Name: "param1", Value: "goodbye"},
			{Name: "param2", Value: "world"},
		},
	}
	assert.Equal(t, expected, *pipeline)
}

func verifyJob(t *testing.T, actual *api.Job, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, actual.CreatedAt.Seconds >= requestStartTime)
	assert.True(t, actual.ScheduledAt.Seconds >= requestStartTime)
	assert.Contains(t, actual.Name, "arguments-parameters-")
}

func TestOneTimePipeline(t *testing.T) {
	suite.Run(t, new(OneTimePipelineTestSuite))
}
