package common
import (
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/stretchr/testify/assert"
)

func TestGetNamespaceFromResourceReferences(t *testing.T) {
	references := []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_NAMESPACE, Id: "ns"},
			Relationship: api.Relationship_OWNER,
		},
	}
	namespace := GetNamespaceFromResourceReferences(references)
	assert.Equal(t, "ns", namespace)

	references = []*api.ResourceReference{
		{
			Key: &api.ResourceKey{
				Type: api.ResourceType_EXPERIMENT, Id: "123"},
			Relationship: api.Relationship_CREATOR,
		},
	}
	namespace = GetNamespaceFromResourceReferences(references)
	assert.Equal(t, "", namespace)
}
