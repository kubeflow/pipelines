package api_server_v2

import (
	"testing"

	apimodel "github.com/kubeflow/pipelines/backend/src/apiserver/model"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateTags(t *testing.T) {
	tagsJSON := `{"team":"ml-ops","env":"prod"}`

	tags, err := parseAndValidateTags(&tagsJSON)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"team": "ml-ops", "env": "prod"}, tags)
}

func TestParseAndValidateTagsRejectsInvalidTags(t *testing.T) {
	testCases := map[string]string{
		"empty key":      `{"":"value"}`,
		"dot in key":     `{"team.name":"value"}`,
		"too many tags":  `{"k0":"v","k1":"v","k2":"v","k3":"v","k4":"v","k5":"v","k6":"v","k7":"v","k8":"v","k9":"v","k10":"v","k11":"v","k12":"v","k13":"v","k14":"v","k15":"v","k16":"v","k17":"v","k18":"v","k19":"v","k20":"v"}`,
		"invalid json":   `not-json`,
		"too long key":   `{"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl":"v"}`,
		"too long value": `{"team":"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl"}`,
	}

	for name, tagsJSON := range testCases {
		t.Run(name, func(t *testing.T) {
			tags, err := parseAndValidateTags(&tagsJSON)
			require.Error(t, err)
			assert.Nil(t, tags)
		})
	}
}

func TestUploadUsesCreatedPipelineIdentityForVersionAssociation(t *testing.T) {
	pipelineModel := apimodel.Pipeline{
		Name:        "test-pipeline",
		Namespace:   "kubeflow",
		DisplayName: "test-pipeline",
	}
	k8sPipeline := k8sapi.FromPipelineModel(pipelineModel)
	k8sPipeline.UID = "pipeline-uid"

	pipelineVersionModel := apimodel.PipelineVersion{
		Name:        "test-version",
		DisplayName: "test-version",
		Description: "desc",
		PipelineSpec: `pipelineInfo:
  name: test-pipeline
  displayName: Test Pipeline
root:
  dag:
    tasks: {}
schemaVersion: "2.1.0"
sdkVersion: kfp-2.13.0`,
	}

	pipelineVersion, err := k8sapi.FromPipelineVersionModel(*k8sPipeline.ToModel(), pipelineVersionModel)
	require.NoError(t, err)
	require.NotEmpty(t, pipelineVersion.OwnerReferences)
	assert.Equal(t, "pipeline-uid", string(pipelineVersion.OwnerReferences[0].UID))
	assert.Equal(t, "pipeline-uid", pipelineVersion.Labels["pipelines.kubeflow.org/pipeline-id"])
}
