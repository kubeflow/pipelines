package pipelinemanager

import "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

type Parameter struct {
	Name      string  `json:"name" gorm:"not null"`
	Value     *string `json:"value"`
	OwnerID   uint    `json:"-"`
	OwnerType string  `json:"-"`
}

func ToParameters(argoParameters []v1alpha1.Parameter) []Parameter {
	newParams := make([]Parameter, 0)
	for _, argoParam := range argoParameters {
		param := Parameter{
			Name:  argoParam.Name,
			Value: argoParam.Value,
		}
		newParams = append(newParams, param)
	}
	return newParams
}
