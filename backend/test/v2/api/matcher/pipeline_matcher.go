package api

import (
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	. "github.com/onsi/gomega"
	"time"
)

func MatchPipelines(actual *model.V2beta1Pipeline, expected *model.V2beta1Pipeline) {
	Expect(actual.PipelineID).To(Not(BeEmpty()), "Pipeline ID is empty")
	expected.PipelineID = actual.PipelineID
	actualTime := time.Time(actual.CreatedAt).UTC()
	expectedTime := time.Time(expected.CreatedAt).UTC()
	Expect(actualTime.After(expectedTime)).To(BeTrue(), "Actual Pipeline creation time is not as expected")
	expected.CreatedAt = actual.CreatedAt
	Expect(actual).To(Equal(expected), "Pipeline Object not matching")

}
