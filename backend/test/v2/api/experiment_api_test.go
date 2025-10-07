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
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/constants"

	. "github.com/onsi/ginkgo/v2"
)

// ###########################################
// ################## TESTS ##################
// ###########################################

// ################## POSITIVE TESTS ##################

var _ = PDescribe("List Experiments API Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentList", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic List Operations >", func() {
		It("When no experiments exist", func() {
		})
		It("After creating a single experiment", func() {
		})
		It("After creating multiple experiments", func() {
		})
		It("By namespace", func() {
		})
	})
	Context("Pagination >", func() {
		It("List experiments with page size limit", func() {
		})
		It("List experiments with pagination - iterate through all pages (at least 2)", func() {
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
		It("Filter by experiment id", func() {
		})
		It("Filter by pipeline run id", func() {
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

var _ = PDescribe("Create Experiment API Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentCreate", constants.APIServerTests, constants.FullRegression), func() {

	Context("Create an experiment >", func() {
		It("With just name", func() {
		})
		It("With name and description", func() {
		})
		It("With name length of 100 chars", func() {
		})
		It("With name containing ASCII characters", func() {
		})
	})
})

var _ = PDescribe("Get Experiment API Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentGet", constants.APIServerTests, constants.FullRegression), func() {

	Context("Get by ID >", func() {
		It("With ID", func() {
		})
	})
})

var _ = PDescribe("Archive an experiment Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentArchive", constants.APIServerTests, constants.FullRegression), func() {

	Context("By ID >", func() {
		It("One that does not have any run(s) or recurring run(s)", func() {
		})
		It("One that does have run(s) or recurring run(s)", func() {
		})
		It("One that have currently RUNNING run(s) or recurring run(s)", func() {
		})
		It("One that is associated to deleted run(s) or recurring run(s)", func() {
		})
	})
})
var _ = PDescribe("UnArchive an experiment Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentUnarchive", constants.APIServerTests, constants.FullRegression), func() {

	Context("By ID >", func() {
		It("One that does not have any run(s) or recurring run(s)", func() {
		})
		It("One that does have run(s) or recurring run(s)", func() {
		})
		It("One that is associated to deleted run(s) or recurring run(s)", func() {
		})
	})
})

var _ = PDescribe("Delete Experiment API Tests >", Label(constants.POSITIVE, constants.Experiment, "ExperimentDelete", constants.APIServerTests, constants.FullRegression), func() {

	Context("Delete by ID >", func() {
		It("Delete an experiment by ID that does not have any run(s) or recurring run(s)", func() {
		})
		It("Delete an experiment by ID that does have run(s) or recurring run(s), and validate that the runs still exists", func() {
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = PDescribe("Verify Pipeline Negative Tests >", Label("Negative", constants.Experiment, constants.APIServerTests, constants.FullRegression), func() {

	Context("Create experiment >", func() {
		It("With 500 char name", func() {
		})
		It("With CJK characters in the name", func() {
		})
		It("With 10000k char description", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Delete by ID >", func() {
		It("Delete by non existing ID", func() {
		})
		It("Delete by ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Archive >", func() {
		It("Archive by non existing ID", func() {
		})
		It("Archive by ID containing ASCII characters", func() {
		})
		It("Archive an archived experiment", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("UnArchive >", func() {
		It("UnArchive by non existing ID", func() {
		})
		It("UnArchive an active experiment", func() {
		})
		It("UnArchive by ID containing ASCII characters", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("List experiments >", func() {
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
