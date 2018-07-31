package util

import (
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	yaml "gopkg.in/yaml.v2"
)

func TestGetParameters(t *testing.T) {
	template := v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
		Parameters: []v1alpha1.Parameter{{Name: "name1", Value: StringPointer("value1")}}}}}
	templateBytes, _ := yaml.Marshal(template)
	paramString, err := GetParameters(templateBytes)
	assert.Nil(t, err)
	assert.Equal(t, `[{"name":"name1","value":"value1"}]`, paramString)
}

func TestGetParameters_ParametersTooLong(t *testing.T) {
	var params []v1alpha1.Parameter
	// Create a long enough parameter string so it exceed the length limit of parameter.
	for i := 0; i < 10000; i++ {
		params = append(params, v1alpha1.Parameter{Name: "name1", Value: StringPointer("value1")})
	}
	template := v1alpha1.Workflow{Spec: v1alpha1.WorkflowSpec{Arguments: v1alpha1.Arguments{
		Parameters: params}}}
	templateBytes, _ := yaml.Marshal(template)
	_, err := GetParameters(templateBytes)
	assert.Equal(t, codes.InvalidArgument, err.(*UserError).ExternalStatusCode())
}
