package server

import (
	"github.com/ghodss/yaml"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestGetPipelineName_QueryStringNotEmpty(t *testing.T) {
	pipelineName, err := GetPipelineName("pipeline%20one", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "pipeline one", pipelineName)
}

func TestGetPipelineName(t *testing.T) {
	pipelineName, err := GetPipelineName("", "file one")
	assert.Nil(t, err)
	assert.Equal(t, "file one", pipelineName)
}

func TestGetPipelineName_InvalidQueryString(t *testing.T) {
	_, err := GetPipelineName("pipeline!$%one", "file one")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestGetPipelineName_NameTooLong(t *testing.T) {
	_, err := GetPipelineName("",
		"this is a loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog name")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "name too long")
}

func TestLoadFile(t *testing.T) {
	file := "12345"
	bytes, err := loadFile(strings.NewReader(file), 5)
	assert.Nil(t, err)
	assert.Equal(t, []byte(file), bytes)
}

func TestLoadFile_ExceedSizeLimit(t *testing.T) {
	file := "12345"
	_, err := loadFile(strings.NewReader(file), 4)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "File size too large")
}

func TestDecompressPipelineTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := DecompressPipelineTarball(tarballByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineTarball_MalformattedTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/malformatted_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineTarball_NonYamlTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/non_yaml_tarball/non_yaml_tarball.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the tarball")
}

func TestDecompressPipelineTarball_EmptyTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/empty_tarball/empty.tar.gz")
	_, err := DecompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := DecompressPipelineZip(zipByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineZip_MalformattedZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformatted_zip.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_MalformedZip2(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/malformed_zip2.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestDecompressPipelineZip_NonYamlZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/non_yaml_zip/non_yaml_file.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a pipeline.yaml file inside the zip")
}

func TestDecompressPipelineZip_EmptyZip(t *testing.T) {
	zipByte, _ := ioutil.ReadFile("test/empty_tarball/empty.zip")
	_, err := DecompressPipelineZip(zipByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid zip file")
}

func TestReadPipelineFile_YAML(t *testing.T) {
	file, _ := os.Open("test/arguments-parameters.yaml")
	fileBytes, err := ReadPipelineFile("arguments-parameters.yaml", file, MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_Zip(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.zip", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Zip_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_zip/arguments-parameters.zip")
	pipelineFile, err := ReadPipelineFile("arguments-parameters.pipeline", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_MultifileZip(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.zip")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.tar.gz", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_Tarball_AnyExtension(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.pipeline", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_MultifileTarball(t *testing.T) {
	file, _ := os.Open("test/pipeline_plus_component/pipeline_plus_component.tar.gz")
	pipelineFile, err := ReadPipelineFile("pipeline_plus_component.ai-hub-package", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/pipeline_plus_component/pipeline.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_UnknownFileFormat(t *testing.T) {
	file, _ := os.Open("test/unknown_format.foo")
	_, err := ReadPipelineFile("unknown_format.foo", file, MaxFileLength)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected pipeline file format")
}

func TestValidateExperimentResourceReference(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	assert.Nil(t, ValidateExperimentResourceReference(manager, validReference))
}

func TestValidateExperimentResourceReference_MoreThanOneRef(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "456"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "more resource references than expected")
}

func TestValidateExperimentResourceReference_UnexpectedType(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123"},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected resource type")
}

func TestValidateExperimentResourceReference_EmptyID(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT},
			Relationship: api.Relationship_OWNER,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Resource ID is empty")
}

func TestValidateExperimentResourceReference_UnexpectedRelationship(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
	}
	err := ValidateExperimentResourceReference(manager, references)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unexpected relationship for the experiment")
}

func TestValidateExperimentResourceReference_ExperimentNotExist(t *testing.T) {
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	manager := resource.NewResourceManager(clients)
	defer clients.Close()
	err := ValidateExperimentResourceReference(manager, validReference)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to get experiment")
}

func TestValidatePipelineSpec_PipelineID(t *testing.T) {
	clients, manager, pipeline := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{PipelineId: pipeline.UUID}
	assert.Nil(t, ValidatePipelineSpec(manager, spec))
}

func TestValidatePipelineSpec_WorkflowSpec(t *testing.T) {
	clients, manager, _ := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{WorkflowManifest: testWorkflow.ToStringForStore()}
	assert.Nil(t, ValidatePipelineSpec(manager, spec))
}

func TestValidatePipelineSpec_EmptySpec(t *testing.T) {
	clients, manager, _ := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{}
	err := ValidatePipelineSpec(manager, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please specify a pipeline by providing a pipeline ID or workflow manifest")
}

func TestValidatePipelineSpec_MoreThanOneSpec(t *testing.T) {
	clients, manager, pipeline := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{PipelineId: pipeline.UUID, WorkflowManifest: testWorkflow.ToStringForStore()}
	err := ValidatePipelineSpec(manager, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Please either specify a pipeline ID or a workflow manifest, not both.")
}

func TestValidatePipelineSpec_PipelineIDNotFound(t *testing.T) {
	clients, manager, _ := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{PipelineId: "not-found"}
	err := ValidatePipelineSpec(manager, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Get pipeline failed")
}

func TestValidatePipelineSpec_InvalidWorkflowManifest(t *testing.T) {
	clients, manager, _ := initWithPipeline(t)
	defer clients.Close()
	spec := &api.PipelineSpec{WorkflowManifest: "I am an invalid manifest"}
	err := ValidatePipelineSpec(manager, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid argo workflow format")
}

func TestValidatePipelineSpec_ParameterTooLong(t *testing.T) {
	clients, manager, pipeline := initWithPipeline(t)
	defer clients.Close()

	var params []*api.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, &api.Parameter{Name: "param2", Value: "world"})
	}
	spec := &api.PipelineSpec{PipelineId: pipeline.UUID, Parameters: params}
	err := ValidatePipelineSpec(manager, spec)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "The input parameter length exceed maximum size")
}

func TestResubmitWorkflowWith(t *testing.T) {
	wf := `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2019-08-02T07:15:14Z"
  generateName: resubmit-
  generation: 1
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: resubmit-hl9ft
  namespace: kubeflow
  resourceVersion: "13488984"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/kubeflow/workflows/resubmit-hl9ft
  uid: 4628dce4-b4f5-11e9-b75e-42010a8001b8
spec:
  arguments: {}
  entrypoint: rand-fail-dag
  templates:
  - dag:
      tasks:
      - arguments: {}
        name: A
        template: random-fail
      - arguments: {}
        dependencies:
        - A
        name: B
        template: random-fail
    inputs: {}
    metadata: {}
    name: rand-fail-dag
    outputs: {}
  - container:
      args:
      - import random; import sys; exit_code = random.choice([0, 0, 1]); print('exiting
        with code {}'.format(exit_code)); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: random-fail
    outputs: {}
status:
  finishedAt: "2019-08-02T07:15:19Z"
  nodes:
    resubmit-hl9ft:
      children:
      - resubmit-hl9ft-3929423573
      displayName: resubmit-hl9ft
      finishedAt: "2019-08-02T07:15:19Z"
      id: resubmit-hl9ft
      name: resubmit-hl9ft
      phase: Failed
      startedAt: "2019-08-02T07:15:14Z"
      templateName: rand-fail-dag
      type: DAG
    resubmit-hl9ft-3879090716:
      boundaryID: resubmit-hl9ft
      displayName: B
      finishedAt: "2019-08-02T07:15:18Z"
      id: resubmit-hl9ft-3879090716
      message: failed with exit code 1
      name: resubmit-hl9ft.B
      phase: Failed
      startedAt: "2019-08-02T07:15:17Z"
      templateName: random-fail
      type: Pod
    resubmit-hl9ft-3929423573:
      boundaryID: resubmit-hl9ft
      children:
      - resubmit-hl9ft-3879090716
      displayName: A
      finishedAt: "2019-08-02T07:15:16Z"
      id: resubmit-hl9ft-3929423573
      name: resubmit-hl9ft.A
      phase: Succeeded
      startedAt: "2019-08-02T07:15:15Z"
      templateName: random-fail
      type: Pod
  phase: Failed
  startedAt: "2019-08-02T07:15:14Z"
`

	var workflow util.Workflow
	err := yaml.Unmarshal([]byte( wf), &workflow)
	assert.Nil(t, err)
	newWf, err := formulateResubmitWorkflow(workflow.Workflow, util.NewFakeRandomString("12345"))

	newWfString, err := yaml.Marshal(newWf)
	assert.Nil(t, err)
	expectedNewWfString :=
`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: null
  generateName: resubmit-
  name: resubmit-12345
spec:
  arguments: {}
  entrypoint: rand-fail-dag
  templates:
  - dag:
      tasks:
      - arguments: {}
        name: A
        template: random-fail
      - arguments: {}
        dependencies:
        - A
        name: B
        template: random-fail
    inputs: {}
    metadata: {}
    name: rand-fail-dag
    outputs: {}
  - container:
      args:
      - import random; import sys; exit_code = random.choice([0, 0, 1]); print('exiting
        with code {}'.format(exit_code)); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: random-fail
    outputs: {}
status:
  finishedAt: null
  nodes:
    resubmit-12345-1329222707:
      boundaryID: resubmit-12345
      children:
      - resubmit-12345-1346000326
      displayName: A
      finishedAt: "2019-08-02T07:15:16Z"
      id: resubmit-12345-1329222707
      name: resubmit-12345.A
      phase: Skipped
      startedAt: "2019-08-02T07:15:15Z"
      templateName: random-fail
      type: Skipped
  startedAt: null
`

	assert.Equal(t, expectedNewWfString,string(newWfString))
}
