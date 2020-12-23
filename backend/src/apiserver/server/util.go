package server

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
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

func ValidatePipelineSpecAndResourceReferences(resourceManager *resource.ResourceManager, spec *api.PipelineSpec, resourceReferences []*api.ResourceReference) error {
	pipelineId := spec.GetPipelineId()
	workflowManifest := spec.GetWorkflowManifest()
	pipelineVersionId := getPipelineVersionIdFromResourceReferences(resourceManager, resourceReferences)

	if workflowManifest != "" {
		if pipelineId != "" || pipelineVersionId != "" {
			return util.NewInvalidInputError("Please don't specify a pipeline version or pipeline ID when you specify a workflow manifest.")
		}
		if err := validateWorkflowManifest(spec.GetWorkflowManifest()); err != nil {
			return err
		}
	} else {
		if pipelineId == "" && pipelineVersionId == "" {
			return util.NewInvalidInputError("Please specify a pipeline by providing a (workflow manifest) or (pipeline id or/and pipeline version).")
		}
		if err := validatePipelineId(resourceManager, pipelineId); err != nil {
			return err
		}
		if pipelineVersionId != "" {
			// verify pipelineVersionId exists
			pipelineVersion, err := resourceManager.GetPipelineVersion(pipelineVersionId)
			if err != nil {
				return util.Wrap(err, "Get pipelineVersionId failed.")
			}
			// verify pipelineId should be parent of pipelineVersionId
			if pipelineId != "" && pipelineVersion.PipelineId != pipelineId {
				return util.NewInvalidInputError("pipeline ID should be parent of pipeline version.")
			}
		}
	}
	return validateParameters(spec.GetParameters())
}
func validateParameters(parameters []*api.Parameter) error {
	if parameters != nil {
		paramsBytes, err := json.Marshal(parameters)
		if err != nil {
			return util.NewInternalServerError(err,
				"Failed to Marshall the pipeline parameters into bytes. Parameters: %s",
				printParameters(parameters))
		}
		if len(paramsBytes) > util.MaxParameterBytes {
			return util.NewInvalidInputError("The input parameter length exceed maximum size of %v.", util.MaxParameterBytes)
		}
	}
	return nil
}

func validatePipelineId(resourceManager *resource.ResourceManager, pipelineId string) error {
	if pipelineId != "" {
		// Verify pipeline exist
		if _, err := resourceManager.GetPipeline(pipelineId); err != nil {
			return util.Wrap(err, "Get pipelineId failed.")
		}
	}
	return nil
}

func validateWorkflowManifest(workflowManifest string) error {
	if workflowManifest != "" {
		// Verify valid workflow template
		var workflow util.Workflow
		if err := json.Unmarshal([]byte(workflowManifest), &workflow); err != nil {
			return util.NewInvalidInputErrorWithDetails(err,
				"Invalid argo workflow format. Workflow: "+workflowManifest)
		}
	}
	return nil
}

func getPipelineVersionIdFromResourceReferences(resourceManager *resource.ResourceManager, resourceReferences []*api.ResourceReference) string {
	var pipelineVersionId = ""
	for _, resourceReference := range resourceReferences {
		if resourceReference.Key.Type == api.ResourceType_PIPELINE_VERSION && resourceReference.Relationship == api.Relationship_CREATOR {
			pipelineVersionId = resourceReference.Key.Id
		}
	}
	return pipelineVersionId
}

func getUserIdentityFromHeader(userIdentityHeader, prefix string) (string, error) {
	if len(userIdentityHeader) > len(prefix) && userIdentityHeader[:len(prefix)] == prefix {
		return userIdentityHeader[len(prefix):], nil
	}
	return "", util.NewBadRequestError(
		errors.New("Request header error: user identity value is incorrectly formatted"),
		"Request header error: user identity value is incorrectly formatted. Expected prefix '%s', but got the header '%s'",
		prefix,
		userIdentityHeader,
	)
}

func getUserIdentity(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", util.NewUnauthenticatedError(errors.New("Request error: context is nil"), "Request error: context is nil.")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	// If the request header contains the user identity, requests are authorized
	// based on the namespace field in the request.
	if userIdentityHeader, ok := md[common.GetKubeflowUserIDHeader()]; ok {
		if len(userIdentityHeader) != 1 {
			return "", util.NewUnauthenticatedError(errors.New("Request header error: unexpected number of user identity header. Expect 1 got "+strconv.Itoa(len(userIdentityHeader))),
				"Request header error: unexpected number of user identity header. Expect 1 got "+strconv.Itoa(len(userIdentityHeader)))
		}
		return getUserIdentityFromHeader(userIdentityHeader[0], common.GetKubeflowUserIDPrefix())
	}
	return "", util.NewUnauthenticatedError(errors.New("Request header error: there is no user identity header."), "Request header error: there is no user identity header.")
}

// isAuthorized verifies whether the user identity, which is contained in the context object,
// can perform some action (verb) on a resource (resourceType/resourceName) living in the
// target namespace. If the returned error is nil, the authorization passes. Otherwise,
// authorization fails with a non-nil error.
func isAuthorized(resourceManager *resource.ResourceManager, ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if common.IsMultiUserMode() == false {
		// Skip authz if not multi-user mode.
		return nil
	}
	if common.IsMultiUserSharedReadMode() &&
		(resourceAttributes.Verb == common.RbacResourceVerbGet ||
			resourceAttributes.Verb == common.RbacResourceVerbList) {
		glog.Infof("Multi-user shared read mode is enabled. Request allowed: %+v", resourceAttributes)
		return nil
	}

	glog.Info("Getting user identity...")
	userIdentity, err := getUserIdentity(ctx)
	if err != nil {
		return util.Wrap(err, "Bad request.")
	}

	if len(userIdentity) == 0 {
		return util.NewUnauthenticatedError(errors.New("Request header error: user identity is empty."), "Request header error: user identity is empty.")
	}

	glog.Infof("User: %s, ResourceAttributes: %+v", userIdentity, resourceAttributes)
	glog.Info("Authorizing request...")
	err = resourceManager.IsRequestAuthorized(userIdentity, resourceAttributes)
	if err != nil {
		glog.Info(err.Error())
		return err
	}

	glog.Infof("Authorized user '%s': %+v", userIdentity, resourceAttributes)
	return nil
}
