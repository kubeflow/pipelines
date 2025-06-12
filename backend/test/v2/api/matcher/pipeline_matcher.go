package api

import (
	"fmt"
	"reflect"
	"time"

	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/onsi/gomega"
)

func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline) {
	Expect(actual.PipelineID).To(Not(BeEmpty()), "Pipeline ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline name not matching")
	Expect(actual.Namespace).To(Equal(expected.Namespace), "Pipeline Namespace not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")

}

func MatchPipelineVersions(actual *model.V2beta1PipelineVersion, expected *model.V2beta1PipelineVersion) {
	Expect(actual.PipelineVersionID).To(Not(Equal(expected.PipelineVersionID)), "Pipeline Version ID is empty")
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline Version creation time is not as expected")
	Expect(actual.DisplayName).To(Equal(expected.DisplayName), "Pipeline Display Name not matching")
	Expect(actual.Description).To(Equal(expected.Description), "Pipeline Description not matching")
	MatchMaps(actual.PipelineSpec, expected.PipelineSpec, "Pipeline Spec")
}

func MatchPipelineVersionSpec(actual any, expected map[string]interface{}) {
	if reflect.TypeOf(actual).Kind() == reflect.Map {
		for key, value := range expected {
			Expect(actual).To(HaveKeyWithValue(key, value), fmt.Sprintf("%s value not matching", key))
		}
	}
}
