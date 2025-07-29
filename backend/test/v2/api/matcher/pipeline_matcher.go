package api

import (
	"sort"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/onsi/gomega"
)

// MatchPipelines - Deep compare 2 pipelines
func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline) {
	Expect(actual.PipelineID).To(Not(BeEmpty()), "Pipeline ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline name not matching")
	Expect(actual.Namespace).To(Equal(expected.Namespace), "Pipeline Namespace not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")

}

// MatchPipelineVersions - Deep compare 2 pipeline versions - even with deep comparison of pipeline specs
func MatchPipelineVersions(actual *model.V2beta1PipelineVersion, expected *model.V2beta1PipelineVersion) {
	Expect(actual.PipelineVersionID).To(Not(Equal(expected.PipelineVersionID)), "Pipeline Version ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline Version creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline Display Name not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
}

// MatchPipelineRuns - Shallow match 2 pipeline runs i.e. match only the fields that you do add to the payload when creating a run
func MatchPipelineRuns(actual *run_model.V2beta1Run, expected *run_model.V2beta1Run) {
	if expected.RunID != "" {
		Expect(actual.RunID).To(Equal(expected.RunID), "Run ID is not matching")
	} else {
		Expect(actual.RunID).To(Not(BeEmpty()), "Run ID is empty")
	}
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime) || actualTime.Equal(expectedTime)).To(BeTrue(), "Actual Run time is not before the expected time")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Run Name is not matching")
	Expect(actual.ExperimentID).To(Equal(expected.ExperimentID), "Experiment Id is not matching")
	Expect(actual.PipelineVersionID).To(Equal(expected.PipelineVersionID), "Pipeline Version Id is not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
	Expect(actual.PipelineVersionReference.PipelineVersionID).To(Equal(expected.PipelineVersionReference.PipelineVersionID), "Referred Pipeline Version Idis not matching")
	Expect(actual.PipelineVersionReference.PipelineID).To(Equal(expected.PipelineVersionReference.PipelineID), "Referred Pipeline Id is not matching")
	Expect(actual.ServiceAccount).To(Equal(expected.ServiceAccount), "Service Account is not matching")
	Expect(actual.StorageState).To(Equal(expected.StorageState), "Storage State is not matching")
}

func MatchPipelineRunDetails(actual *run_model.V2beta1RunDetails, expected *run_model.V2beta1RunDetails) {
	Expect(actual.PipelineContextID).To(Equal(expected.PipelineContextID), "Pipeline Context ID is not matching")
	Expect(actual.PipelineRunContextID).To(Equal(expected.PipelineRunContextID), "Pipeline Run Context ID is not matching")
	Expect(len(actual.TaskDetails)).To(Equal(len(expected.TaskDetails)), "Number of Tasks not matching")
	sort.Slice(actual.TaskDetails, func(i, j int) bool {
		return actual.TaskDetails[i].DisplayName < actual.TaskDetails[j].DisplayName // Sort Tasks by Name in ascending order
	})
	sort.Slice(expected.TaskDetails, func(i, j int) bool {
		return expected.TaskDetails[i].DisplayName < expected.TaskDetails[j].DisplayName // Sort Tasks by Name in ascending order
	})
	for index, task := range expected.TaskDetails {
		Expect(actual.TaskDetails[index].RunID).To(Equal(task.RunID), "Task Run ID is not matching")
		Expect(actual.TaskDetails[index].TaskID).To(Not(BeEmpty()), "Task ID is empty")
		if strings.Contains(task.DisplayName, "root") || strings.Contains(task.DisplayName, "driver") {
			Expect(actual.TaskDetails[index].DisplayName).To(Equal(task.DisplayName), "Task Display Name is not matching")
		} else {
			Expect(actual.TaskDetails[index].DisplayName).To(ContainSubstring(actual.TaskDetails[index].DisplayName), "Task Display Name does not match")
		}

		Expect(actual.TaskDetails[index].ParentTaskID).To(Equal(task.ParentTaskID), "Task Parent Task ID is not matching")
		actualCreationTime := time.Time(actual.TaskDetails[index].CreateTime).UTC()
		expectedCreationTime := time.Time(task.CreateTime).UTC()
		actualStartTime := time.Time(actual.TaskDetails[index].StartTime).UTC()
		expectedStartTimeRange := expectedCreationTime.Add(-1 * time.Second)
		expectedEndTimeRange := expectedCreationTime.Add(1 * time.Second)
		Expect(actualCreationTime.After(expectedStartTimeRange)).To(BeTrue(), "Task Create Time is before the expected creation time")
		Expect(actualCreationTime.Before(expectedEndTimeRange)).To(BeTrue(), "Task Create Time is after the expected creation time")
		Expect(actualStartTime.After(expectedStartTimeRange)).To(BeTrue(), "Task Start Time is before the expected start time")
		Expect(actualStartTime.Before(expectedEndTimeRange)).To(BeTrue(), "Task End Time is before the expected start time")
		Expect(actual.TaskDetails[index].State).To(BeElementOf([]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateCANCELED, run_model.V2beta1RuntimeStateCANCELING, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStatePENDING, run_model.V2beta1RuntimeStateRUNNING}), "Task State is not matching")
		for _, state := range actual.TaskDetails[index].StateHistory {
			Expect(state.State).To(BeElementOf([]run_model.V2beta1RuntimeState{run_model.V2beta1RuntimeStateCANCELED, run_model.V2beta1RuntimeStateCANCELING, run_model.V2beta1RuntimeStateFAILED, run_model.V2beta1RuntimeStateSUCCEEDED, run_model.V2beta1RuntimeStateSKIPPED, run_model.V2beta1RuntimeStatePENDING, run_model.V2beta1RuntimeStateRUNNING}), "Task State History is not matching")
		}
		if strings.Contains(task.DisplayName, "driver") {
			Expect(len(actual.TaskDetails[index].ChildTasks) > 0).To(BeTrue(), "No child tasks found for a Driver Task")
		}
	}
}
