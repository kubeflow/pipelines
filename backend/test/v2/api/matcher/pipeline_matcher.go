package matcher

import (
	"fmt"
	"sort"
	"strings"
	"time"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/kubeflow/pipelines/backend/test/logger"
	"github.com/kubeflow/pipelines/backend/test/testutil"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
)

// MatchPipelines - Deep compare 2 pipelines
func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline) {
	ginkgo.GinkgoHelper()
	gomega.Expect(actual.PipelineID).To(gomega.Not(gomega.BeEmpty()), "Pipeline ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	if !actualTime.Equal(expectedTime) && !actualTime.After(expectedTime) {
		logger.Log("Pipeline creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt)
		ginkgo.Fail(fmt.Sprintf("Pipeline creation time %v is before the test start time %v", actual.CreatedAt, expected.CreatedAt))
	}
	gomega.Expect(actual.DisplayName).To(gomega.Equal(expected.DisplayName), "Pipeline Display name not matching")
	if *config.KubeflowMode {
		// Validate namespace only if the mode is kubeflow otherwise everything is in the same namespace and
		// pipeline object in the response does not even have namespace
		gomega.Expect(actual.Namespace).To(gomega.Equal(expected.Namespace), "Pipeline Namespace not matching")
	}
	gomega.Expect(actual.Description).To(gomega.Equal(expected.Description), "Pipeline Description not matching")

}

// MatchPipelineVersions - Deep compare 2 pipeline versions - even with deep comparison of pipeline specs
func MatchPipelineVersions(actual *model.V2beta1PipelineVersion, expected *model.V2beta1PipelineVersion) {
	ginkgo.GinkgoHelper()
	gomega.Expect(actual.PipelineVersionID).To(gomega.Not(gomega.BeEmpty()), "Pipeline Version ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	if !actualTime.Equal(expectedTime) && !actualTime.After(expectedTime) {
		logger.Log("Pipeline Version creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt)
		ginkgo.Fail(fmt.Sprintf("Pipeline Version creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt))
	}
	gomega.Expect(actual.DisplayName).To(gomega.Equal(expected.DisplayName), "Pipeline Display Name not matching")
	gomega.Expect(actual.Description).To(gomega.Equal(expected.Description), "Pipeline Description not matching")
	expectedPipelineSpec := expected.PipelineSpec.(*template.V2Spec)
	MatchPipelineSpecs(actual.PipelineSpec, expectedPipelineSpec)
}

func MatchPipelineSpecs(actual interface{}, expected *template.V2Spec) {
	ginkgo.GinkgoHelper()

	actualPipelineSpec := actual.(map[string]interface{})
	platformSpecs, exists := actualPipelineSpec["platform_spec"]
	if exists {
		actualPipelineSpecBytes := testutil.ToBytes(actualPipelineSpec["pipeline_spec"])
		actualPlatformSpecBytes := testutil.ToBytes(platformSpecs)
		gomega.Expect(actualPipelineSpecBytes).To(gomega.MatchYAML(testutil.ProtoToBytes(expected.PipelineSpec())), "Pipeline specs do not match")
		gomega.Expect(actualPlatformSpecBytes).To(gomega.MatchYAML(testutil.ProtoToBytes(expected.PlatformSpec())), "Platform specs do not match")
	} else {
		gomega.Expect(testutil.ToBytes(actualPipelineSpec)).To(gomega.MatchYAML(expected.Bytes()), "Pipeline specs do not match")
	}
}

// MatchPipelineRuns - Shallow match 2 pipeline runs i.e. match only the fields that you do add to the payload when creating a run
func MatchPipelineRuns(actual *run_model.V2beta1Run, expected *run_model.V2beta1Run) {
	ginkgo.GinkgoHelper()
	if expected.RunID != "" {
		gomega.Expect(actual.RunID).To(gomega.Equal(expected.RunID), "Run ID is not matching")
	} else {
		gomega.Expect(actual.RunID).To(gomega.Not(gomega.BeEmpty()), "Run ID is empty")
	}
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	gomega.Expect(actualTime.After(expectedTime) || actualTime.Equal(expectedTime)).To(gomega.BeTrue(), "Actual Run time is not before the expected time")
	gomega.Expect(actual.DisplayName).To(gomega.Equal(expected.DisplayName), "Run Name is not matching")
	gomega.Expect(actual.ExperimentID).To(gomega.Equal(expected.ExperimentID), "Experiment Id is not matching")
	gomega.Expect(actual.PipelineVersionID).To(gomega.Equal(expected.PipelineVersionID), "Pipeline Version Id is not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
	gomega.Expect(actual.PipelineVersionReference.PipelineVersionID).To(gomega.Equal(expected.PipelineVersionReference.PipelineVersionID), "Referred Pipeline Version Idis not matching")
	gomega.Expect(actual.PipelineVersionReference.PipelineID).To(gomega.Equal(expected.PipelineVersionReference.PipelineID), "Referred Pipeline Id is not matching")
	gomega.Expect(actual.ServiceAccount).To(gomega.Equal(expected.ServiceAccount), "Service Account is not matching")
	gomega.Expect(actual.StorageState).To(gomega.Equal(expected.StorageState), "Storage State is not matching")
}

// MatchPipelineRunDetails - NOTE: Not yet used but once we start populating Run Details, this matcher will come in very handy
func MatchPipelineRunDetails(actual *run_model.V2beta1RunDetails, expected *run_model.V2beta1RunDetails) {
	gomega.Expect(actual.PipelineContextID).To(gomega.Equal(expected.PipelineContextID), "Pipeline Context ID is not matching")
	gomega.Expect(actual.PipelineRunContextID).To(gomega.Equal(expected.PipelineRunContextID), "Pipeline Run Context ID is not matching")
	gomega.Expect(len(actual.TaskDetails)).To(gomega.Equal(len(expected.TaskDetails)), "Number of Tasks not matching")
	sort.Slice(actual.TaskDetails, func(i, j int) bool {
		return actual.TaskDetails[i].DisplayName < actual.TaskDetails[j].DisplayName // Sort Tasks by Name in ascending order
	})
	sort.Slice(expected.TaskDetails, func(i, j int) bool {
		return expected.TaskDetails[i].DisplayName < expected.TaskDetails[j].DisplayName // Sort Tasks by Name in ascending order
	})
	for index, task := range expected.TaskDetails {
		gomega.Expect(actual.TaskDetails[index].RunID).To(gomega.Equal(task.RunID), "Task Run ID is not matching")
		gomega.Expect(actual.TaskDetails[index].TaskID).To(gomega.Not(gomega.BeEmpty()), "Task ID is empty")
		if strings.Contains(task.DisplayName, "root") || strings.Contains(task.DisplayName, "driver") {
			gomega.Expect(actual.TaskDetails[index].DisplayName).To(gomega.Equal(task.DisplayName), "Task Display Name is not matching")
		} else {
			gomega.Expect(actual.TaskDetails[index].DisplayName).To(gomega.ContainSubstring(actual.TaskDetails[index].DisplayName), "Task Display Name does not match")
		}

		gomega.Expect(actual.TaskDetails[index].ParentTaskID).To(gomega.Equal(task.ParentTaskID), "Task Parent Task ID is not matching")
		actualCreationTime := time.Time(actual.TaskDetails[index].CreateTime).UTC()
		expectedCreationTime := time.Time(task.CreateTime).UTC()
		expectedStartTimeRange := expectedCreationTime.Add(-1 * time.Second)
		expectedEndTimeRange := expectedCreationTime.Add(1 * time.Second)
		gomega.Expect(actualCreationTime.After(expectedStartTimeRange)).To(gomega.BeTrue(), "Task Create Time is before the expected creation time")
		gomega.Expect(actualCreationTime.Before(expectedEndTimeRange)).To(gomega.BeTrue(), "Task Create Time is after the expected creation time")
		gomega.Expect(*actual.TaskDetails[index].State).To(gomega.BeElementOf([]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateCANCELED, run_model.V2beta1RuntimeStateCANCELING, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStatePENDING, run_model.V2beta1RuntimeStateRUNNING}), "Task State is not matching")
		for _, state := range actual.TaskDetails[index].StateHistory {
			gomega.Expect(*state.State).To(gomega.BeElementOf([]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateCANCELED, run_model.V2beta1RuntimeStateCANCELING, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStatePENDING, run_model.V2beta1RuntimeStateRUNNING}), "Task State History is not matching")
		}
		if strings.Contains(task.DisplayName, "driver") {
			gomega.Expect(len(actual.TaskDetails[index].ChildTasks) > 0).To(gomega.BeTrue(), "No child tasks found for a Driver Task")
		}
	}
}
