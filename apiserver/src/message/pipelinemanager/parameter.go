// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelinemanager

import "ml/apiserver/src/message/argo"

type Parameter struct {
	Name      string  `json:"name" gorm:"not null"`
	Value     *string `json:"value"`
	OwnerID   uint    `json:"-"`
	OwnerType string  `json:"-"`
}

func ToParameters(argoParameters []argo.Parameter) []Parameter {
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
