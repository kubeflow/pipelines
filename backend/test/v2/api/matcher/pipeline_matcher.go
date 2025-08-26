package matcher

import (
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"time"

	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	"github.com/onsi/ginkgo/v2"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/gomega"
)

func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline, isKubeflowMode bool) {
	ginkgo.GinkgoHelper()
	Expect(actual.PipelineID).To(Not(BeEmpty()), "Pipeline ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	if !(actualTime.Equal(expectedTime) || actualTime.After(expectedTime)) {
		logger.Log("Pipeline creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt)
		ginkgo.Fail(fmt.Sprintf("Pipeline creation time %v is before the test start time %v", actual.CreatedAt, expected.CreatedAt))
	}
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline Display name not matching")
	if isKubeflowMode {
		// Validate namespace only if the mode is kubeflow otherwise everything is in the same namespace and
		// pipeline object in the response does not even have namespace
		Expect(actual.Namespace).To(Equal(expected.Namespace), "Pipeline Namespace not matching")
	}
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")

}

func MatchPipelineVersions(actual *model.V2beta1PipelineVersion, expected *model.V2beta1PipelineVersion) {
	ginkgo.GinkgoHelper()
	Expect(actual.PipelineVersionID).To(Not(BeEmpty()), "Pipeline Version ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	if !(actualTime.Equal(expectedTime) || actualTime.After(expectedTime)) {
		logger.Log("Pipeline Version creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt)
		ginkgo.Fail(fmt.Sprintf("Pipeline Version creation time %v is expected to be after test start time %v", actual.CreatedAt, expected.CreatedAt))
	}
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline Display Name not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")
	expectedPipelineSpec := expected.PipelineSpec.(*template.V2Spec)
	MatchPipelineSpecs(actual.PipelineSpec, expectedPipelineSpec)
}

func MatchPipelineSpecs(actual interface{}, expected *template.V2Spec) {
	ginkgo.GinkgoHelper()

	actualPipelineSpec := actual.(map[string]interface{})
	platformSpecs, exists := actualPipelineSpec["platform_spec"]
	if exists {
		actualPipelineSpecBytes := utils.ToBytes(actualPipelineSpec["pipeline_spec"])
		actualPlatformSpecBytes := utils.ToBytes(platformSpecs)
		Expect(actualPipelineSpecBytes).To(MatchYAML(utils.ProtoToBytes(expected.PipelineSpec())), "Pipeline specs do not match")
		Expect(actualPlatformSpecBytes).To(MatchYAML(utils.ProtoToBytes(expected.PlatformSpec())), "Platform specs do not match")
	} else {
		Expect(utils.ToBytes(actualPipelineSpec)).To(MatchYAML(expected.Bytes()), "Pipeline specs do not match")
	}
}
