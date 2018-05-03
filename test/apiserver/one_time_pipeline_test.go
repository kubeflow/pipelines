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
	"flag"
	"fmt"
	"ml/backend/src/apiserver/api"
	"ml/backend/src/util"
	"testing"
	"time"

	"io/ioutil"

	"net/http"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
)

var namespace = flag.String("namespace", "default", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")
var testTimeout = flag.Duration("testTimeout", 2*time.Minute, "Duration to wait for the test to finish")

type OneTimePipelineTestSuite struct {
	suite.Suite
	namespace string
}

// Check the namespace have ML pipeline installed and ready
func (suite *OneTimePipelineTestSuite) SetupTest() {
	if err := initTest(*namespace, *initializeTimeout); err != nil {
		glog.Exit("Failed to initialize test. Error: %s", err.Error())
	}
	suite.namespace = *namespace
}

func (suite *OneTimePipelineTestSuite) TearDownTest() {
}

// This test case tests a basic use case:
// - Upload a package
// - Create a pipeline with parameter
// - Verify an one-time job is automatically scheduled.
func (suite *OneTimePipelineTestSuite) TestOneTimePipeline() {
	t := suite.T()
	clientSet, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
	}

	/* ---------- Verify no package exist ---------- */
	response, err := clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages")).Do().Raw()
	checkNoPackageExists(t, response, err)

	/* ---------- Verify no pipeline exist ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines")).Do().Raw()
	checkNoPipelineExists(t, response, err)

	/* ---------- Upload a package ---------- */
	requestStartTime := time.Now().Unix()
	pkgBody, writer := uploadPackageFileOrFail("resources/arguments-parameters.yaml")
	response, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages/upload")).
		SetHeader("Content-Type", writer.FormDataContentType()).
		Body(pkgBody).Do().Raw()
	checkUpdatePackageResponse(t, response, err, requestStartTime)

	/* ---------- Verify list package works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages")).Do().Raw()
	checkListPackagesResponse(t, response, err, requestStartTime)

	/* ---------- Verify get package works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages/1")).Do().Raw()
	checkGetPackageResponse(t, response, err, requestStartTime)

	/* ---------- Verify get template works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages/1/templates")).Do().Raw()
	checkGetTemplateResponse(t, response, err)

	/* ---------- Instantiate pipeline using the package ---------- */
	requestStartTime = time.Now().Unix()
	pipeline := api.Pipeline{
		Name:        "hello-world",
		PackageId:   1,
		Description: "this is my first pipeline",
		Parameters: []api.Parameter{
			{Name: "param1", Value: util.StringPointer("goodbye")},
			{Name: "param2", Value: util.StringPointer("world")},
		},
	}
	response, err = clientSet.RESTClient().Post().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines")).
		Body(util.MarshalJsonOrFail(pipeline)).Do().Raw()
	checkInstantiatePipelineResponse(t, response, err, requestStartTime)

	/* ---------- Verify list pipelines works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines")).Do().Raw()
	checkListPipelinesResponse(t, response, err, requestStartTime)

	/* ---------- Verify get pipeline works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines/1")).Do().Raw()
	checkGetPipelineResponse(t, response, err, requestStartTime)

	/* ---------- Verify list job works ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines/1/jobs")).Do().Raw()
	checkListJobsResponse(t, response, err, requestStartTime)

	/* ---------- Verify get job works ---------- */
	jobName := getJobName(response)
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines/1/jobs/"+jobName)).Do().Raw()
	checkGetJobResponse(t, response, err, requestStartTime)

	/* ---------- Verify job complete successfully ---------- */
	err = checkJobSucceed(clientSet, suite.namespace, "1", jobName)
	if err != nil {
		assert.Fail(t, "The job doesn't complete. Error: <%s>", err.Error())
	}

	/* ---------- Verify delete pipeline works ---------- */
	var statusCode int
	clientSet.RESTClient().Delete().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines/1")).Do().StatusCode(&statusCode)
	assert.Equal(t, http.StatusOK, statusCode)

	/* ---------- Verify no pipeline exist ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines")).Do().Raw()
	checkNoPipelineExists(t, response, err)

	/* ---------- Verify can't retrieve the job ---------- */
	clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "pipelines/1/jobs/"+jobName)).Do().StatusCode(&statusCode)
	assert.Equal(t, http.StatusNotFound, statusCode)

	/* ---------- Verify delete package works ---------- */
	clientSet.RESTClient().Delete().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages/1")).Do().StatusCode(&statusCode)
	assert.Equal(t, http.StatusOK, statusCode)

	/* ---------- Verify no package exist ---------- */
	response, err = clientSet.RESTClient().Get().
		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, suite.namespace, "packages")).Do().Raw()
	checkNoPackageExists(t, response, err)
}

func checkNoPackageExists(t *testing.T, response []byte, err error) {
	assert.Nil(t, err)
	var pkgs []api.Package
	util.UnmarshalJsonOrFail(string(response), &pkgs)
	assert.Empty(t, pkgs)
}

func checkNoPipelineExists(t *testing.T, response []byte, err error) {
	assert.Nil(t, err)
	var pipelines []api.Package
	util.UnmarshalJsonOrFail(string(response), &pipelines)
	assert.Empty(t, pipelines)
}

func checkUpdatePackageResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pkg api.Package
	util.UnmarshalJsonOrFail(string(response), &pkg)
	verifyPackage(t, pkg, requestStartTime)
}

func checkListPackagesResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pkgs []api.Package
	util.UnmarshalJsonOrFail(string(response), &pkgs)
	assert.Equal(t, 1, len(pkgs))
	verifyPackage(t, pkgs[0], requestStartTime)
}
func checkGetPackageResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pkg api.Package
	util.UnmarshalJsonOrFail(string(response), &pkg)
	verifyPackage(t, pkg, requestStartTime)
}

func checkGetTemplateResponse(t *testing.T, response []byte, err error) {
	assert.Nil(t, err)
	expected, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
	assert.Nil(t, err)
	assert.Equal(t, expected, response)
}

func checkInstantiatePipelineResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pipeline api.Pipeline
	util.UnmarshalJsonOrFail(string(response), &pipeline)
	verifyPipeline(t, pipeline, requestStartTime)
}

func checkListPipelinesResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pipelines []api.Pipeline
	util.UnmarshalJsonOrFail(string(response), &pipelines)
	assert.Equal(t, 1, len(pipelines))
	verifyPipeline(t, pipelines[0], requestStartTime)
}

func checkGetPipelineResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var pipeline api.Pipeline
	util.UnmarshalJsonOrFail(string(response), &pipeline)
	verifyPipeline(t, pipeline, requestStartTime)
}

func checkListJobsResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var jobs []api.Job
	util.UnmarshalJsonOrFail(string(response), &jobs)
	assert.Equal(t, 1, len(jobs))
	verifyJob(t, jobs[0], requestStartTime)
}

func checkGetJobResponse(t *testing.T, response []byte, err error, requestStartTime int64) {
	assert.Nil(t, err)
	var jobDetail api.JobDetail
	util.UnmarshalJsonOrFail(string(response), &jobDetail)
	verifyJob(t, *jobDetail.Job, requestStartTime)

	// The Argo workflow might not be created. Only verify if it's created.
	if jobDetail.Workflow != nil {
		// Do some very basic verification to make sure argo workflow is returned
		assert.Equal(t, jobDetail.Job.Name, jobDetail.Workflow.Name)
	}
}
func checkJobSucceed(clientSet *kubernetes.Clientset, namespace string, pipelineId string, jobName string) error {
	var waitForJobSucceed = func() error {
		response, err := clientSet.RESTClient().Get().
			AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, namespace, "pipelines/"+pipelineId+"/jobs/"+jobName)).Do().Raw()
		if err != nil {
			return err
		}
		status := getJobStatus(response)
		if status == nil {
			return errors.New("Job hasn't started yet.")
		}
		if *status == v1alpha1.NodeSucceeded {
			return nil
		}
		if *status == v1alpha1.NodeRunning {
			return errors.New("Job is still running. Waiting for job to finish.")
		}
		return backoff.Permanent(errors.Errorf("Job failed to run with status %s.", *status))
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = *testTimeout
	return backoff.Retry(waitForJobSucceed, b)
}

func getJobStatus(response []byte) *v1alpha1.NodePhase {
	var jobDetail api.JobDetail
	util.UnmarshalJsonOrFail(string(response), &jobDetail)
	if jobDetail.Workflow != nil {
		return &jobDetail.Workflow.Status.Phase
	}
	return nil
}

func verifyPackage(t *testing.T, actual api.Package, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, actual.CreatedAtInSec >= requestStartTime)
	expected := api.Package{
		ID:             1,
		CreatedAtInSec: actual.CreatedAtInSec,
		Name:           "arguments-parameters.yaml",
		Parameters: []api.Parameter{
			{Name: "param1", Value: util.StringPointer("hello")}, // Default value in the package template
			{Name: "param2"},                                     // No default value in the package
		},
	}
	assert.Equal(t, expected, actual)
}

func verifyPipeline(t *testing.T, actual api.Pipeline, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, actual.CreatedAtInSec >= requestStartTime)
	assert.True(t, actual.EnabledAtInSec >= requestStartTime)

	expected := api.Pipeline{
		ID:             1,
		CreatedAtInSec: actual.CreatedAtInSec,
		Name:           "hello-world",
		Description:    "this is my first pipeline",
		PackageId:      1,
		Enabled:        true,
		EnabledAtInSec: actual.EnabledAtInSec,
		Parameters: []api.Parameter{
			{Name: "param1", Value: util.StringPointer("goodbye")},
			{Name: "param2", Value: util.StringPointer("world")},
		},
	}
	assert.Equal(t, expected, actual)
}

func verifyJob(t *testing.T, actual api.Job, requestStartTime int64) {
	// Only verify the time fields have valid value and in the right range.
	assert.True(t, actual.CreatedAtInSec >= requestStartTime)
	assert.True(t, actual.ScheduledAtInSec >= requestStartTime)
	assert.Contains(t, actual.Name, "arguments-parameters-")
}

func getJobName(response []byte) string {
	var jobs []api.Job
	util.UnmarshalJsonOrFail(string(response), &jobs)
	return jobs[0].Name
}

func TestOneTimePipeline(t *testing.T) {
	suite.Run(t, new(OneTimePipelineTestSuite))
}
