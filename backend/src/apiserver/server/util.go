// Copyright 2018-2022 The Kubeflow Authors
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

package server

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func loadFile(fileReader io.Reader, MaxFileLength int) ([]byte, error) {
	reader := bufio.NewReader(fileReader)
	pipelineFile := make([]byte, MaxFileLength+1)
	size, err := reader.Read(pipelineFile)
	if err != nil && err != io.EOF {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error read pipeline file.")
	}
	if size == MaxFileLength+1 {
		return nil, util.NewInvalidInputError("File size too large. Maximum supported size: %v", MaxFileLength)
	}

	return pipelineFile[:size], nil
}

func isYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

func isJSONFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".json")
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

func ReadPipelineFile(fileName string, fileReader io.Reader, MaxFileLength int) ([]byte, error) {
	// Read file into size limited byte array.
	pipelineFileBytes, err := loadFile(fileReader, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Error read pipeline file.")
	}

	var processedFile []byte
	switch {
	case isYamlFile(fileName):
		processedFile = pipelineFileBytes
	case isJSONFile(fileName):
		processedFile = pipelineFileBytes
	case isZipFile(pipelineFileBytes):
		processedFile, err = DecompressPipelineZip(pipelineFileBytes)
	case isCompressedTarballFile(pipelineFileBytes):
		processedFile, err = DecompressPipelineTarball(pipelineFileBytes)
	default:
		return nil, util.NewInvalidInputError("Unexpected pipeline file format. Support .zip, .tar.gz, .json or YAML.")
	}
	if err != nil {
		return nil, util.Wrap(err, "Error decompress the pipeline file")
	}
	return processedFile, nil
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
	userIdentity, err := resourceManager.AuthenticateRequest(ctx)
	if err != nil {
		return err
	}

	if len(userIdentity) == 0 {
		return util.NewUnauthenticatedError(errors.New("Request header error: user identity is empty."), "Request header error: user identity is empty.")
	}

	glog.Infof("User: %s, ResourceAttributes: %+v", userIdentity, resourceAttributes)
	glog.Info("Authorizing request...")
	err = resourceManager.IsRequestAuthorized(ctx, userIdentity, resourceAttributes)
	if err != nil {
		glog.Info(err.Error())
		return err
	}

	glog.Infof("Authorized user '%s': %+v", userIdentity, resourceAttributes)
	return nil
}
