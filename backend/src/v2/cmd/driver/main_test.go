package main

import (
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"os"
	"testing"
)

func Test_handleExecutionContainer(t *testing.T) {
	execution := &driver.Execution{}

	executionPaths := &ExecutionPaths{
		Condition: "condition.txt",
	}

	err := handleExecution(execution, CONTAINER, executionPaths)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	verifyFileContent(t, executionPaths.Condition, "nil")

	cleanup(t, executionPaths)
}

func Test_handleExecutionRootDAG(t *testing.T) {
	execution := &driver.Execution{}

	executionPaths := &ExecutionPaths{
		IterationCount: "iteration_count.txt",
		Condition:      "condition.txt",
	}

	err := handleExecution(execution, ROOT_DAG, executionPaths)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	verifyFileContent(t, executionPaths.IterationCount, "0")
	verifyFileContent(t, executionPaths.Condition, "nil")

	cleanup(t, executionPaths)
}

func cleanup(t *testing.T, executionPaths *ExecutionPaths) {
	removeIfExists(t, executionPaths.IterationCount)
	removeIfExists(t, executionPaths.ExecutionID)
	removeIfExists(t, executionPaths.Condition)
	removeIfExists(t, executionPaths.PodSpecPatch)
	removeIfExists(t, executionPaths.CachedDecision)
}

func removeIfExists(t *testing.T, filePath string) {
	_, err := os.Stat(filePath)
	if err == nil {
		err = os.Remove(filePath)
		if err != nil {
			t.Errorf("Unexpected error while removing the created file: %v", err)
		}
	}
}

func verifyFileContent(t *testing.T, filePath string, expectedContent string) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created, but it doesn't exist", filePath)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read file contents: %v", err)
	}

	if string(fileContent) != expectedContent {
		t.Errorf("Expected file fileContent to be %q, got %q", expectedContent, string(fileContent))
	}
}
