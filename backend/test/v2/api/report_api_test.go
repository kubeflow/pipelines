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

package api

import (
	"fmt"
	"path/filepath"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
)

var projectDataDir = testutil.GetTestDataDir()
var workflowsDir = filepath.Join(projectDataDir, "compiled-workflows")

// ################## TESTS ##################

// ################## POSITIVE TESTS ##################

var _ = PDescribe("Create Workflow API Tests >", Label(constants.POSITIVE, constants.ReportTests, constants.APIServerTests, constants.FullRegression), func() {
	workflowFiles := testutil.GetListOfFilesInADir(workflowsDir)
	Context("Create Workflow >", func() {
		for _, workflowFile := range workflowFiles {
			It(fmt.Sprintf("Create a workflow from: %s", workflowFile), func() {
			})
			It(fmt.Sprintf("Create a scheduled workflow from: %s", workflowFile), func() {
			})
		}
	})
})

// ################## NEGATIVE TESTS ##################

var _ = PDescribe("Create Workflow Negative Tests >", Label(constants.NEGATIVE, constants.ReportTests, constants.APIServerTests, constants.FullRegression), func() {

	Context("Create workflow >", func() {
		It("With invalid workflow schema", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})

	Context("Create scheduled workflow >", func() {
		It("With invalid workflow schema", func() {
		})
		It("With valid workflow schema but invalid cron", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
})
