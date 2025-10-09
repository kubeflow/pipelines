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

var _ = PDescribe("Verify Pipeline Run >", Label(constants.POSITIVE, constants.PipelineScheduledRun, constants.APIServerTests, constants.FullRegression), func() {

	Context("Create reccurring pipeline run >", func() {
		It("Create a Pipeline Run with cron that runs every 5min", func() {
		})
		It("Create a Pipeline Run with cron that runs at a specific time and day", func() {
		})
		It("Create a Pipeline Run with cron that runs on alternate days", func() {
		})
		It("Create a Pipeline Run with cron that runs right now", func() {
		})
		It("Create a Pipeline Run with cache disabled", func() {
		})
	})

	Context("Disable reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, disable a run and make sure its not deleted", func() {
		})
	})
	Context("Enable a disabled reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, disable it and then enable it", func() {
		})
	})

	Context("Get a reccurring pipeline run >", func() {
		It("Create a Recurring pipeline Run, verify its details", func() {
		})
	})
})

var _ = PDescribe("List Recurring Pipeline Runs >", Label(constants.POSITIVE, constants.PipelineScheduledRun, "ListRecurringPipelineRun", constants.APIServerTests, constants.FullRegression), func() {

	Context("Basic Operations >", func() {
		It("Create 2 runs and list", func() {
		})
		It("When no recurring runs exist", func() {
		})
	})
	Context("Sorting >", func() {
		It("List Recurring pipeline Runs and sort by display name in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by display name in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by id in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by id in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by pipeline version id in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by pipeline version id in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by creation date in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by creation date in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by updated date in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by updated date in descending order", func() {
		})
		It("List Recurring pipeline Runs and sort by cron time in ascending order", func() {
		})
		It("List Recurring pipeline Runs and sort by cron time in descending order", func() {
		})
	})
	Context("Specify Page Size >", func() {
		It("List by page size", func() {
		})
		It("List by page size and iterate over at least 2 pages", func() {
		})
	})
	Context("Filtering >", func() {
		It("By run id", func() {
		})
		It("By run 'name' EQUALS", func() {
		})
		It("By run 'name' NOT EQUALS", func() {
		})
		It("By display name containing", func() {
		})
		It("By creation date", func() {
		})
		It("By cron time", func() {
		})
		It("By experiment id", func() {
		})
		It("By namespace", func() {
		})
		It("By name in ascending order", func() {
		})
	})
	Context("Sort and Filter >", func() {
		It("Filter and sort by created date in descending order", func() {
		})
		It("Filter by created date and sort by updated date in descending order", func() {
		})
	})
})

// ################## NEGATIVE TESTS ##################

var _ = PDescribe("Verify Pipeline Run Negative Tests >", Label(constants.NEGATIVE, constants.PipelineScheduledRun, constants.APIServerTests, constants.FullRegression), func() {

	Context("Create reccurring pipeline run >", func() {
		It("Create a Pipeline Run with invalid cron", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})

	if *config.KubeflowMode {
		Context("List reccurring pipeline runs in kubeflow mode >", func() {
			It("List reccurring pipeline runs in a namespace you don't have access to", func() {})
		})
	}

	Context("Disable a recurring pipeline run >", func() {
		It("Disable a deleted recurring run", func() {
		})
		It("Disable a non existent recurring run", func() {
		})
		It("Disable already disabled recurring run", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Enable a recurring pipeline run >", func() {
		It("Enable a deleted recurring run", func() {
		})
		It("Enable a non existent recurring run", func() {
		})
		It("Enable an already enabled recurring run", func() {
		})
		It("Enable an recurring run for a run that's associated with a delete experiment", func() {
		})
		It("Enable an recurring run for a run that's associated with an archived experiment", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
	Context("Delete a recurring pipeline run >", func() {
		It("Delete a deleted recurring run", func() {
		})
		It("Delete a non existent recurring run", func() {
		})
		It("Delete an already deleted recurring run", func() {
		})
	})
	Context("Associate a pipeline recurring run with invalid experiment >", func() {
		It("Associate a recurring run with an archived experiment", func() {
		})
		It("Associate a recurring run with non existent experiment", func() {
		})
		It("Associate a recurring run with deleted experiment", func() {
		})
	})
	Context("Get a reccurring pipeline run >", func() {
		It("Get Recurring pipeline Run for a deleted run", func() {
		})
		It("Get Recurring pipeline Run for a non existing run", func() {
		})
		It("Get Recurring pipeline Run for a non recurring run i.e. one-off run", func() {
		})
		if *config.KubeflowMode {
			It("In a namespace you don't have access to", func() {
			})
		}
	})
})
