package matcher

import (
	"fmt"
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
	if expected.Tags != nil {
		MatchMaps(actual.Tags, expected.Tags, "Pipeline Tags")
	}
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
	if expected.Tags != nil {
		MatchMaps(actual.Tags, expected.Tags, "Pipeline Version Tags")
	}
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
