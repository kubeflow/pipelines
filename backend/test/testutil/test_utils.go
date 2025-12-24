// Copyright 2018-2023 The Kubeflow Authors
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

// Package testutil gathers shared helpers used across backend integration tests.
package testutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/onsi/gomega"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

// ParsePointersToString - convert a string pointer to string value
func ParsePointersToString(s *string) string {
	if s == nil {
		return ""
	} else {
		return *s
	}
}

// GetRandomString - Get a random string of length x
func GetRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CheckIfSkipping - checks input string against skip conditions, and skips pipeline run if applicable.
func CheckIfSkipping(stringValue string) {
	// Skip pipeline if name contains "GH-" (case-insensitive)
	if strings.Contains(strings.ToLower(stringValue), "_gh-") {
		issue := strings.Split(strings.ToLower(stringValue), "_gh-")[1]
		ginkgo.Skip("Skipping pipeline run")
		fmt.Printf("Skipping pipeline run test because of a known issue: https://github.com/kubeflow/pipelines/issues/%s", issue)

	}
	// Skip pipeline 'pipeline_submit_request' test if TLS is not enabled
	if !*config.TLSEnabled && strings.Contains(strings.ToLower(stringValue), "pipeline_submit_request") {
		ginkgo.Skip("Skipping pipeline run test because TLS is not enabled")
	}
}

func WriteLogFile(specReport types.SpecReport, testName, logDirectory string) {
	stdOutput := specReport.CapturedGinkgoWriterOutput
	testLogFile := filepath.Join(logDirectory, testName+".log")
	logFile, err := os.Create(testLogFile)
	if err != nil {
		logger.Log("Failed to create log file due to: %s", err.Error())
	}
	_, err = logFile.Write([]byte(stdOutput))
	if err != nil {
		logger.Log("Failed to write to the log file, due to: %s", err.Error())
	}
	err = logFile.Close()
	if err != nil {
		return
	}
}

// GetNamespace - Get Namespace based on the deployment mode
func GetNamespace() string {
	if *config.KubeflowMode || *config.MultiUserMode {
		return *config.UserNamespace
	}
	return *config.Namespace
}

// getPackagePath generates the package path based on environment variables
// Equivalent to the Python function get_package_path
func getPackagePath(subdir string) string {
	repoName := *config.REPO_NAME

	pullNumber := *config.PULL_NUMBER
	if pullNumber != "" {
		return fmt.Sprintf("git+https://github.com/%s.git@refs/pull/%s/merge#subdirectory=%s", repoName, pullNumber, subdir)
	}
	return fmt.Sprintf("git+https://github.com/%s.git@%s#subdirectory=%s", repoName, *config.BRANCH_NAME, subdir)
}

func ReplaceSDKInPipelineSpec(pipelineFilePath string) []byte {
	pipelineFileBytes, err := os.ReadFile(pipelineFilePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to read pipeline file: "+pipelineFilePath)
	pipelineFileString := string(pipelineFileBytes)

	// Define regex pattern to match kfp==[version] (e.g., kfp==2.8.0)
	kfpPattern := regexp.MustCompile(`kfp==[0-9]+\.[0-9]+\.[0-9]+`)

	// Replace all occurrences with the new package path
	newPackagePath := getPackagePath("sdk/python")
	modifiedPipelineSpec := kfpPattern.ReplaceAllString(pipelineFileString, newPackagePath)

	return []byte(modifiedPipelineSpec)
}

func ReplaceBaseImageInPipelineSpec(pipelineFilePath string) []byte {
	pipelineFileBytes, err := os.ReadFile(pipelineFilePath)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "failed to read pipeline file: "+pipelineFilePath)
	pipelineFileString := string(pipelineFileBytes)

	// Define regex pattern to match image: python:3.9
	imagePattern := regexp.MustCompile(`image: python:[0-9]+\.[0-9]+`)

	// Replace all occurrences with the new package path
	newBaseImage := fmt.Sprintf("image: %s", *config.BaseImage)
	modifiedPipelineSpec := imagePattern.ReplaceAllString(pipelineFileString, newBaseImage)

	return []byte(modifiedPipelineSpec)
}

func GetPipelineUploadClient(
	uploadPipelinesWithKubernetes bool,
	isKubeflowMode bool,
	isDebugMode bool,
	userToken string,
	namespace string,
	clientConfig clientcmd.ClientConfig,
	tlsCfg *tls.Config,
) (api_server.PipelineUploadInterface, error) {
	if uploadPipelinesWithKubernetes {
		return api_server.NewPipelineUploadClientKubernetes(clientConfig, namespace)
	}

	if isKubeflowMode {
		return api_server.NewKubeflowInClusterPipelineUploadClient(namespace, isDebugMode, tlsCfg)
	}
	if userToken != "" {
		return api_server.NewMultiUserPipelineUploadClient(clientConfig, userToken, isDebugMode, tlsCfg)
	}

	return api_server.NewPipelineUploadClient(clientConfig, isDebugMode, tlsCfg)
}

// GetTLSConfig returns TLS config set with system CA certs as well as custom CA stored at input caCertPath if provided.
func GetTLSConfig(caCertPath string) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, err
		}
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}

func GetRepoBranchURLRAW(repoName, branch, path string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/raw/refs/heads/%s/%s", repoName, branch, path)

	pullNumber := os.Getenv("PULL_NUMBER")
	if pullNumber != "" {
		url = fmt.Sprintf("https://raw.githubusercontent.com/%s/pull/%s/head/%s", repoName, pullNumber, path)
	}

	// Verify the URL exists
	resp, err := http.Head(url)
	if err != nil {
		return url, fmt.Errorf("failed to verify URL exists: %s\n"+
			"Error: %v\n"+
			"This may indicate network issues or the file doesn't exist at the specified location.\n"+
			"Repository: %s, Branch/PR: %s, Path: %s",
			url, err, repoName, getBranchOrPR(pullNumber, branch), path)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Log("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		msg := "URL verification failed with status %d: %s, " +
			"the file does not exist at the specified location, " +
			"repository: %s, Branch/PR: %s, Path: %s"
		return url, fmt.Errorf(msg, resp.StatusCode, url, repoName, getBranchOrPR(pullNumber, branch), path)
	}

	return url, nil
}

// getBranchOrPR returns a human-readable string indicating whether we're using a branch or PR
func getBranchOrPR(pullNumber, branch string) string {
	if pullNumber != "" {
		return fmt.Sprintf("PR #%s", pullNumber)
	}
	return fmt.Sprintf("branch '%s'", branch)
}
