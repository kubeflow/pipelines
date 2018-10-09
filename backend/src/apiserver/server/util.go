package server

import (
	"bufio"
	"io"
	"net/url"

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

func ReadFile(fileReader io.Reader, maxFileLength int) ([]byte, error) {
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
