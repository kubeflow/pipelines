// Package test_utils
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
package test_utils

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"

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

// CheckIfSkipping - test if the provided string argument contains "GH-" (case insensitive)
func CheckIfSkipping(stringValue string) {
	if strings.Contains(strings.ToLower(stringValue), "_gh-") {
		issue := strings.Split(strings.ToLower(stringValue), "_gh-")[1]
		ginkgo.Skip(fmt.Sprintf("Skipping pipeline run test because of a known issue: https://github.com/kubeflow/pipelines/issues/%s", issue))
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
