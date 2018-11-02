package server

import (
	"testing"

	"strings"

	"os"

	"io/ioutil"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
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
	pipelineFile, err := decompressPipelineTarball(tarballByte)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments_tarball/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestDecompressPipelineTarball_MalformattedTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/malformatted_tarball.tar.gz")
	_, err := decompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestDecompressPipelineTarball_NonYamlTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/non_yaml_tarball/non_yaml_tarball.tar.gz")
	_, err := decompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Expecting a YAML file inside the tarball")
}

func TestDecompressPipelineTarball_EmptyTarball(t *testing.T) {
	tarballByte, _ := ioutil.ReadFile("test/empty_tarball/empty.tar.gz")
	_, err := decompressPipelineTarball(tarballByte)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Not a valid tarball file")
}

func TestReadPipelineFile_YAML(t *testing.T) {
	file, _ := os.Open("test/arguments-parameters.yaml")
	fileBytes, err := ReadPipelineFile("arguments-parameters.yaml", file, MaxFileLength)
	assert.Nil(t, err)

	expectedFileBytes, _ := ioutil.ReadFile("test/arguments-parameters.yaml")
	assert.Equal(t, expectedFileBytes, fileBytes)
}

func TestReadPipelineFile_Tarball(t *testing.T) {
	file, _ := os.Open("test/arguments_tarball/arguments.tar.gz")
	pipelineFile, err := ReadPipelineFile("arguments.tar.gz", file, MaxFileLength)
	assert.Nil(t, err)

	expectedPipelineFile, _ := ioutil.ReadFile("test/arguments_tarball/arguments-parameters.yaml")
	assert.Equal(t, expectedPipelineFile, pipelineFile)
}

func TestReadPipelineFile_UnknownFileFormat(t *testing.T) {
	file, _ := os.Open("test/unknown_extension.foo")
	_, err := ReadPipelineFile("unknown_extension.foo", file, MaxFileLength)
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
