package pipelinemanager

import "ml/apiserver/src/message/argo"

type Parameter struct {
	Name      string `json:"name" gorm:"not null"`
	Value     string `json:"value"`
	OwnerID   uint   `json:"-"`
	OwnerType string `json:"-"`
}

func ToParameters(parameters []argo.Parameter) []Parameter {
	newParams := make([]Parameter, 0)
	for _, param := range parameters {
		param := Parameter{
			Name:  param.Name,
			Value: *param.Value,
		}
		newParams = append(newParams, param)
	}
	return newParams
}
