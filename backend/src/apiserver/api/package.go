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

package api

import "ml/backend/src/model"

type Package struct {
	ID             uint        `json:"id"`
	CreatedAtInSec int64       `json:"createdAt"`
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	Parameters     []Parameter `json:"parameters,omitempty"`
}

func ToApiPackage(pkg *model.Package) *Package {
	return &Package{
		ID:             pkg.ID,
		CreatedAtInSec: pkg.CreatedAtInSec,
		Name:           pkg.Name,
		Description:    pkg.Description,
		Parameters:     toApiParameters(pkg.Parameters),
	}
}

func ToApiPackages(pkgs []model.Package) []Package {
	apiPkgs := make([]Package, 0)
	for _, pkg := range pkgs {
		apiPkgs = append(apiPkgs, *ToApiPackage(&pkg))
	}
	return apiPkgs
}
