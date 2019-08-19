package server

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
)

// These are valid conditions of a ScheduledWorkflow.
const (
	MaxFileNameLength = 100
	MaxFileLength     = 32 << 20 // 32Mb
)

// This method extract the common logic of naming the pipeline.
// API caller can either explicitly name the pipeline through query string ?name=foobar
// or API server can use the file name by default.
func GetPipelineName(queryString string, fileName string) (string, error) {
	pipelineName, err := url.QueryUnescape(queryString)
	if err != nil {
		return "", util.NewInvalidInputErrorWithDetails(err, "Pipeline name in the query string has invalid format.")
	}
	if pipelineName == "" {
		pipelineName = fileName
	}
	if len(pipelineName) > MaxFileNameLength {
		return "", util.NewInvalidInputError("Pipeline name too long. Support maximum length of %v", MaxFileNameLength)
	}
	return pipelineName, nil
}

func loadFile(fileReader io.Reader, maxFileLength int) ([]byte, error) {
	reader := bufio.NewReader(fileReader)
	pipelineFile := make([]byte, maxFileLength+1)
	size, err := reader.Read(pipelineFile)
	if err != nil && err != io.EOF {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error read pipeline file.")
	}
	if size == maxFileLength+1 {
		return nil, util.NewInvalidInputError("File size too large. Maximum supported size: %v", maxFileLength)
	}

	return pipelineFile[:size], nil
}

func isSupportedPipelineFormat(fileName string, compressedFile []byte) bool {
	return isYamlFile(fileName) || isCompressedTarballFile(compressedFile) || isZipFile(compressedFile)
}

func isYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

func isPipelineYamlFile(fileName string) bool {
	return fileName == "pipeline.yaml"
}

func isZipFile(compressedFile []byte) bool {
	return len(compressedFile) > 2 && compressedFile[0] == '\x50' && compressedFile[1] == '\x4B' //Signature of zip file is "PK"
}

func isCompressedTarballFile(compressedFile []byte) bool {
	return len(compressedFile) > 2 && compressedFile[0] == '\x1F' && compressedFile[1] == '\x8B'
}

func DecompressPipelineTarball(compressedFile []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedFile))
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
	}
	// New behavior: searching for the "pipeline.yaml" file.
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			tarReader = nil
			break
		}
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
		}
		if isPipelineYamlFile(header.Name) {
			//Found the pipeline file.
			break
		}
	}
	// Old behavior - taking the first file in the archive
	if tarReader == nil {
		// Resetting the reader
		gzipReader, err = gzip.NewReader(bytes.NewReader(compressedFile))
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
		}
		tarReader = tar.NewReader(gzipReader)
		header, err := tarReader.Next()
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
		}
		if !isYamlFile(header.Name) {
			return nil, util.NewInvalidInputError("Error extracting pipeline from the tarball file. Expecting a pipeline.yaml file inside the tarball. Got: %v", header.Name)
		}
	}

	decompressedFile, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error reading pipeline YAML from the tarball file.")
	}
	return decompressedFile, err
}

func DecompressPipelineZip(compressedFile []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(compressedFile), int64(len(compressedFile)))
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Not a valid zip file.")
	}
	if len(reader.File) < 1 {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Empty zip file.")
	}

	// Old behavior - taking the first file in the archive
	pipelineYamlFile := reader.File[0]
	// New behavior: searching for the "pipeline.yaml" file.
	for _, file := range reader.File {
		if isPipelineYamlFile(file.Name) {
			pipelineYamlFile = file
			break
		}
	}

	if !isYamlFile(pipelineYamlFile.Name) {
		return nil, util.NewInvalidInputError("Error extracting pipeline from the zip file. Expecting a pipeline.yaml file inside the zip. Got: %v", pipelineYamlFile.Name)
	}
	rc, err := pipelineYamlFile.Open()
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Failed to read the content.")
	}
	decompressedFile, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error reading pipeline YAML from the zip file.")
	}
	return decompressedFile, err
}

func ReadPipelineFile(fileName string, fileReader io.Reader, maxFileLength int) ([]byte, error) {
	// Read file into size limited byte array.
	pipelineFileBytes, err := loadFile(fileReader, maxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Error read pipeline file.")
	}

	var processedFile []byte
	switch {
	case isYamlFile(fileName):
		processedFile = pipelineFileBytes
	case isZipFile(pipelineFileBytes):
		processedFile, err = DecompressPipelineZip(pipelineFileBytes)
	case isCompressedTarballFile(pipelineFileBytes):
		processedFile, err = DecompressPipelineTarball(pipelineFileBytes)
	default:
		return nil, util.NewInvalidInputError("Unexpected pipeline file format. Support .zip, .tar.gz or YAML.")
	}
	if err != nil {
		return nil, util.Wrap(err, "Error decompress the pipeline file")
	}
	return processedFile, nil
}

func printParameters(params []*api.Parameter) string {
	var s strings.Builder
	for _, p := range params {
		s.WriteString(p.String())
	}
	return s.String()
}

// Verify the input resource references has one and only reference which is owner experiment.
func ValidateExperimentResourceReference(resourceManager *resource.ResourceManager, references []*api.ResourceReference) error {
	if references == nil || len(references) == 0 || references[0] == nil {
		return util.NewInvalidInputError("The resource reference is empty. Please specify which experiment owns this resource.")
	}
	if len(references) > 1 {
		return util.NewInvalidInputError("Got more resource references than expected. Please only specify which experiment owns this resource.")
	}
	if references[0].Key.Type != api.ResourceType_EXPERIMENT {
		return util.NewInvalidInputError("Unexpected resource type. Expected:%v. Got: %v",
			api.ResourceType_EXPERIMENT, references[0].Key.Type)
	}
	if references[0].Key.Id == "" {
		return util.NewInvalidInputError("Resource ID is empty. Please specify a valid ID")
	}
	if references[0].Relationship != api.Relationship_OWNER {
		return util.NewInvalidInputError("Unexpected relationship for the experiment. Expected: %v. Got: %v",
			api.Relationship_OWNER, references[0].Relationship)
	}
	if _, err := resourceManager.GetExperiment(references[0].Key.Id); err != nil {
		return util.Wrap(err, "Failed to get experiment.")
	}
	return nil
}

func ValidatePipelineSpec(resourceManager *resource.ResourceManager, spec *api.PipelineSpec) error {
	if spec == nil || (spec.GetPipelineId() == "" && spec.GetWorkflowManifest() == "") {
		return util.NewInvalidInputError("Please specify a pipeline by providing a pipeline ID or workflow manifest.")
	}
	if spec.GetPipelineId() != "" && spec.GetWorkflowManifest() != "" {
		return util.NewInvalidInputError("Please either specify a pipeline ID or a workflow manifest, not both.")
	}
	if spec.GetPipelineId() != "" {
		// Verify pipeline exist
		if _, err := resourceManager.GetPipeline(spec.GetPipelineId()); err != nil {
			return util.Wrap(err, "Get pipeline failed.")
		}
	}
	if spec.GetWorkflowManifest() != "" {
		// Verify valid workflow template
		var workflow util.Workflow
		if err := json.Unmarshal([]byte(spec.GetWorkflowManifest()), &workflow); err != nil {
			return util.NewInvalidInputErrorWithDetails(err,
				"Invalid argo workflow format. Workflow: "+spec.GetWorkflowManifest())
		}
	}
	paramsBytes, err := json.Marshal(spec.Parameters)
	if err != nil {
		return util.NewInternalServerError(err,
			"Failed to Marshall the pipeline parameters into bytes. Parameters: %s",
			printParameters(spec.Parameters))
	}
	if len(paramsBytes) > util.MaxParameterBytes {
		return util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
	}
	return nil
}