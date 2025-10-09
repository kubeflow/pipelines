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
	utils "github.com/kubeflow/pipelines/backend/test/testutil"

	. "github.com/onsi/ginkgo/v2"
)

// ###########################################
// ################## TESTS ##################
// ###########################################

// ################## POSITIVE TESTS ##################

var _ = PDescribe("List Pipelines API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineList", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic List Operations >", func() {
		It("When no pipelines exist", func() {
		})
		It("After creating a single pipeline", func() {
		})
		It("After creating multiple pipelines", func() {
		})
		It("By namespace", func() {
		})
	})
	Context("Pagination >", func() {
		It("List pipelines with page size limit", func() {
		})
		It("List pipelines with pagination - iterate through all pages (at least 2)", func() {
		})
	})
	Context("Sorting >", func() {
		It("Sort by name in ascending order", func() {
		})
		It("Sort by name in descending order", func() {
		})
		It("Sort by display name containing substring in ascending order", func() {
		})
		It("Sort by display name containing substring in descending order", func() {
		})
		It("Sort by creation date in ascending order", func() {
		})
		It("Sort by creation date in descending order", func() {
		})
	})
	Context("Filtering >", func() {
		It("Filter by pipeline id", func() {
		})
		It("Filter by name", func() {
		})
		It("Filter by created at", func() {
		})
		It("Filter by namespace", func() {
		})
		It("Filter by description", func() {
		})
	})
	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
		})
		It("Filter and sort by created date in descending order", func() {
		})
		It("Filter by created date and sort by updated date in descending order", func() {
		})
	})
})

var _ = PDescribe("List Pipelines Versions API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionList", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic List Operations >", func() {
		It("When no pipeline versions exist", func() {
		})
		It("After creating a single pipeline version", func() {
		})
		It("After creating multiple pipeline versions", func() {
		})
		It("By pipeline ID", func() {
		})
	})
	Context("Pagination >", func() {
		It("List pipelines with page size limit", func() {
		})
		It("List pipelines with pagination - iterate through all pages (at least 2)", func() {
		})
	})
	Context("Sorting >", func() {
		It("Sort by name in ascending order", func() {
		})
		It("Sort by name in descending order", func() {
		})
		It("Sort by display name containing substring in ascending order", func() {
		})
		It("Sort by display name containing substring in descending order", func() {
		})
		It("Sort by creation date in ascending order", func() {
		})
		It("Sort by creation date in descending order", func() {
		})
	})
	Context("Filtering >", func() {
		It("Filter by pipeline version id", func() {
		})
		It("Filter by pipeline id", func() {
		})
		It("Filter by name", func() {
		})
		It("Filter by created at", func() {
		})
		It("Filter by namespace", func() {
		})
		It("Filter by description", func() {
		})
	})
	Context("Combined Parameters >", func() {
		It("Filter and sort by name in ascending order", func() {
		})
		It("Filter and sort by created date in descending order", func() {
		})
		It("Filter by created date and sort by updated date in descending order", func() {
		})
	})
})

var _ = PDescribe("Create Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineCreate", constants.APIServerTests, constants.FullRegression), func() {

	Context("Create a pipeline using '/pipelines' >", func() {
		It("With just name", func() {
		})
		It("With name and description", func() {
		})
		It("With name length of 100 chars", func() {
		})
		It("With name containing ASCII characters", func() {
		})
	})

	Context("Create a pipeline with version using '/pipelines/create' >", func() {
		var pipelineDir = "valid/samples"
		pipelineFiles := utils.GetListOfFilesInADir(filepath.Join(pipelineFilesRootDir, pipelineDir))
		for _, pipelineFile := range pipelineFiles {
			It(fmt.Sprintf("Pipeline with name and Pipelineversion with name and pipeline spec from file: %s", pipelineFile), func() {
			})
		}
		pipelineURLs := []string{"Your actual pipeline URLs go here"}
		for _, pipelineURL := range pipelineURLs {
			It(fmt.Sprintf("Pipeline with name and Pipelineversion with name and pipeline spec from url: %s", pipelineURL), func() {
			})
		}
	})
})

var _ = PDescribe("Get Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineGet", constants.APIServerTests, constants.FullRegression), func() {

	Context("Get by name '/pipelines/{name}' >", func() {
		It("With full name", func() {
		})
		It("With name and namespace", func() {
		})
	})

	Context("Get by ID '/pipelines/{pipeline_id}' >", func() {
		It("With ID", func() {
		})
	})
})

var _ = PDescribe("Get Pipeline Version API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineVersionGet", constants.APIServerTests, constants.FullRegression), func() {

	Context("Get by id '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("With valid pipeline id and version id", func() {
		})
	})
})

var _ = PDescribe("Delete Pipeline API Tests >", Label(constants.POSITIVE, constants.Pipeline, "PipelineDelete", constants.APIServerTests, constants.FullRegression), func() {

	Context("Delete pipeline by ID '/pipelines/{pipeline_id}' >", func() {
		It("Delete pipeline by ID that does not have any versions", func() {
		})
	})
	Context("Delete pipeline version by ID '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("Delete pipeline version by ID", func() {
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = PDescribe("Verify Pipeline Negative Tests >", Label("Negative", constants.Pipeline, constants.APIServerTests, constants.FullRegression), func() {
	Context("Create a pipeline with version using '/pipelines/create' >", func() {
		It("With a valid pipeline and pipeline version name but invalid pipeline spec file", func() {
		})
		It("With a valid pipeline and pipeline version name but invalid pipeline spec url", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Create a pipeline using '/pipelines >", func() {
		It("With 500 char name", func() {
		})
		It("With CJK characters in the name", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Get pipeline by ID >", func() {
		It("By non existing ID", func() {
		})
		It("By ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Get pipeline version by ID >", func() {
		It("By non existing ID", func() {
		})
		It("By ID containing ASCII characters", func() {
		})
		It("By valid version ID but with the pipeline ID that does not contain this version", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Delete by ID '/pipelines/{pipeline_id}' >", func() {
		It("Delete by ID that does have pipeline version(s)", func() {
		})
		It("Delete by non existing ID", func() {
		})
		It("Delete by ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Delete pipeline version by ID '/pipelines/{pipeline_id}/versions/{pipeline_version_id}' >", func() {
		It("Delete pipeline version with an invalid ID", func() {
		})
		It("Delete pipeline version by ID but with the pipeline ID that does not contain this version", func() {
		})
		It("Delete by ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("List pipelines >", func() {
		It("By partial name", func() {
		})
		It("By invalid name", func() {
		})
		It("By invalid ID", func() {
		})
		It("By invalid ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
})
