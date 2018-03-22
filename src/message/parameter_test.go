package message

import (
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/magiconair/properties/assert"
)

func TestToParameters(t *testing.T) {
	v1 := "abc"
	argoParam := []v1alpha1.Parameter{{Name: "parameter1"}, {Name: "parameter2", Value: &v1}}
	expectParam := []Parameter{
		{Name: "parameter1"},
		{Name: "parameter2", Value: &v1}}
	assert.Equal(t, expectParam, ToParameters(argoParam))
}
