package server

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/url"

	"strings"

	api "github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/common/util"
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

func isSupportedPipelineFormat(fileName string) bool {
	return isYamlFile(fileName) || strings.HasSuffix(fileName, ".tar.gz")
}

func isYamlFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
}

func decompressPipelineTarball(compressedFile []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedFile))
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
	}
	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil || header == nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error extracting pipeline from the tarball file. Not a valid tarball file.")
	}
	if !isYamlFile(header.Name) {
		return nil, util.NewInvalidInputError("Error extracting pipeline from the tarball file. Expecting a YAML file inside the tarball. Got: %v", header.Name)
	}
	decompressedFile, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Error reading pipeline YAML from the tarball file.")
	}
	return decompressedFile, err
}

func ReadPipelineFile(fileName string, fileReader io.Reader, maxFileLength int) ([]byte, error) {
	if !isSupportedPipelineFormat(fileName) {
		return nil, util.NewInvalidInputError("Unexpected pipeline file format. Support .tar.gz or YAML.")
	}

	// Read file into size limited byte array.
	pipelineFileBytes, err := loadFile(fileReader, maxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Error read pipeline file.")
	}

	// Return if file is YAML
	if isYamlFile(fileName) {
		return pipelineFileBytes, nil
	}

	// Decompress if file is tarball
	decompressedFile, err := decompressPipelineTarball(pipelineFileBytes)
	if err != nil {
		return nil, util.Wrap(err, "Error decompress the pipeline file")
	}
	return decompressedFile, nil
}

func printParameters(params []*api.Parameter) string {
	var s strings.Builder
	for _, p := range params {
		s.WriteString(p.String())
	}
	return s.String()
}
