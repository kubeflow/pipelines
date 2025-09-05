// Package test
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
	"github.com/onsi/ginkgo/v2"
	"math/rand"
	"strings"
	"time"
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

/*
*
Skip test if the provided string argument contains "GH-" (case insensitive)
*/
func SkipTest(stringValue string) {
	if strings.Contains(strings.ToLower(stringValue), "_gh-") {
		issue := strings.Split(strings.ToLower(stringValue), "_gh-")[1]
		ginkgo.Skip(fmt.Sprintf("Skipping pipeline run test because of a known issue: https://github.com/kubeflow/pipelines/issues/%s", issue))
	}
}

// ChunkSlice - Split slice into given number of chunk size and return a list of chunked slices
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		// Handle the case where the last chunk might be smaller than chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}
