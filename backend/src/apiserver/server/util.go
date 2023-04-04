// Copyright 2018 The Kubeflow Authors
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
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

func loadFile(fileReader io.Reader, MaxFileLength int) ([]byte, error) {
	// TODO(lingqinggan): investigate ways to increase the buffer size, so we don't have to use a loop.
	reader := bufio.NewReaderSize(fileReader, MaxFileLength)
	var pipelineFile []byte
	for {
		currentRead := make([]byte, bufio.MaxScanTokenSize)
		size, err := reader.Read(currentRead)
		pipelineFile = append(pipelineFile, currentRead[:size]...)
		if err == io.EOF {
			// there is no more data to read
			break
		}
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error read pipeline file")
		}
	}
	if len(pipelineFile) > MaxFileLength {
		return nil, util.NewInvalidInputError("File size too large. Maximum supported size: %v", MaxFileLength)
	}
	return pipelineFile, nil
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
	return len(compressedFile) > 2 && compressedFile[0] == '\x50' && compressedFile[1] == '\x4B' // Signature of zip file is "PK"
}

func isCompressedTarballFile(compressedFile []byte) bool {
	return len(compressedFile) > 2 && compressedFile[0] == '\x1F' && compressedFile[1] == '\x8B'
}

func DecompressPipelineTarball(compressedFile []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedFile))
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file")
	}
	// New behavior: searching for the "pipeline.yaml" file.
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			tarReader = nil
			break
		}
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file")
		}
		if isPipelineYamlFile(header.Name) {
			// Found the pipeline file.
			break
		}
	}
	// Old behavior - taking the first file in the archive
	if tarReader == nil {
		// Resetting the reader
		gzipReader, err = gzip.NewReader(bytes.NewReader(compressedFile))
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file")
		}
		tarReader = tar.NewReader(gzipReader)
		header, err := tarReader.Next()
		if err != nil {
			return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file")
		}
		if !isYamlFile(header.Name) {
			return nil, util.NewInvalidInputError("Error extracting pipeline from the tarball file. Expecting a pipeline.yaml file inside the tarball. Got: %v", header.Name)
		}
	}

	decompressedFile, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error reading pipeline YAML from the tarball file")
	}
	return decompressedFile, err
}

func DecompressPipelineZip(compressedFile []byte) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(compressedFile), int64(len(compressedFile)))
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Not a valid zip file")
	}
	if len(reader.File) < 1 {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Empty zip file")
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
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the zip file. Failed to read the content")
	}
	decompressedFile, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error reading pipeline YAML from the zip file")
	}
	return decompressedFile, err
}

func ReadPipelineFile(fileName string, fileReader io.Reader, MaxFileLength int) ([]byte, error) {
	// Read file into size limited byte array.
	pipelineFileBytes, err := loadFile(fileReader, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Error read pipeline file")
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
		return nil, util.NewInvalidInputError("Unexpected pipeline file format. Support .zip, .tar.gz, .json or YAML")
	}
	if err != nil {
		return nil, util.Wrap(err, "Error decompress the pipeline file")
	}
	return processedFile, nil
}
