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

package model

import "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

type Parameter struct {
	Name      string  `gorm:"column:Name; not null;primary_key"`
	Value     *string `gorm:"column:Value"`
	OwnerID   uint    `gorm:"column:OwnerID; primary_key; auto_increment:false"` /* Foreign key */
	OwnerType string  `gorm:"column:OwnerType; primary_key"`
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
