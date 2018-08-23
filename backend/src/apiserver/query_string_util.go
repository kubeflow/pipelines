// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"

	"github.com/googleprivate/ml/backend/src/common/util"
)

var pipelineModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

var jobModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
	"package_id": "PipelineId",
}

var runModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

func parseSortByQueryString(queryString string, modelFieldByApiFieldMapping map[string]string) (string, bool, error) {
	// ignore the case of the letter. Split query string by space
	queryList := strings.Fields(strings.ToLower(queryString))
	// Check the query string format.
	if len(queryList) > 2 || (len(queryList) == 2 && queryList[1] != "desc" && queryList[1] != "asc") {
		return "", false, util.NewInvalidInputError(
			"Received invalid sort by field %v. Supported format: \"field_name\", \"field_name desc\", or \"field_name asc\"", queryString)
	}
	isDesc := false
	if len(queryList) == 2 && queryList[1] == "desc" {
		isDesc = true
	}
	sortByApiField := ""
	if len(queryList) > 0 {
		sortByApiField = queryList[0]
	}
	// Check if the field can be sorted.s
	sortByModelField, ok := modelFieldByApiFieldMapping[sortByApiField]
	if !ok {
		return "", false, util.NewInvalidInputError("Cannot sort on field %v.", sortByApiField)
	}
	return sortByModelField, isDesc, nil
}
