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

package server

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	defaultPageSize = 20
	maxPageSize     = 200
)

var experimentModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"id":         "UUID",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

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
	"name":       "DisplayName",
	"created_at": "CreatedAtInSec",
	"package_id": "PipelineId",
}

var runModelFieldsBySortableAPIFields = map[string]string{
	// Sort by CreatedAtInSec by default
	"":           "CreatedAtInSec",
	"name":       "Name",
	"created_at": "CreatedAtInSec",
}

func ValidateFilter(referenceKey *api.ResourceKey) (*common.FilterContext, error) {
	filterContext := &common.FilterContext{}
	if referenceKey != nil {
		refType, err := common.ToModelResourceType(referenceKey.Type)
		if err != nil {
			return nil, util.Wrap(err, "Unrecognized resource reference type.")
		}
		filterContext.ReferenceKey = &common.ReferenceKey{Type: refType, ID: referenceKey.Id}
	}
	return filterContext, nil
}

func ValidatePagination(pageToken string, pageSize int, keyFieldName string, queryString string,
	modelFieldByApiFieldMapping map[string]string) (*common.PaginationContext, error) {
	sortByFieldName, isDesc, err := parseSortByQueryString(queryString, modelFieldByApiFieldMapping)
	if err != nil {
		return nil, util.Wrap(err, "Invalid query string.")
	}
	if pageSize < 0 {
		return nil, util.NewInvalidInputError("The page size should be greater than 0. Got %v", strconv.Itoa(pageSize))
	}
	if pageSize == 0 {
		// Use default page size if not provided.
		pageSize = defaultPageSize
	}
	if pageSize > defaultPageSize {
		pageSize = maxPageSize
	}
	if sortByFieldName == "" {
		// By default, sort by key field.
		sortByFieldName = keyFieldName
	}
	token, err := deserializePageToken(pageToken)
	if err != nil {
		return nil, util.Wrap(err, "Invalid page token.")
	}
	return &common.PaginationContext{
		PageSize:        pageSize,
		SortByFieldName: sortByFieldName,
		KeyFieldName:    keyFieldName,
		IsDesc:          isDesc,
		Token:           token}, nil
}

func parseSortByQueryString(queryString string, modelFieldByApiFieldMapping map[string]string) (string, bool, error) {
	// ignore the case of the letter. Split query string by space
	queryList := strings.Fields(strings.ToLower(queryString))
	// Check the query string format.
	if len(queryList) > 2 || (len(queryList) == 2 && queryList[1] != "desc" && queryList[1] != "asc") {
		return "", false, util.NewInvalidInputError(
			"Received invalid sort by format `%v`. Supported format: \"field_name\", \"field_name desc\", or \"field_name asc\"", queryString)
	}
	isDesc := false
	if len(queryList) == 2 && queryList[1] == "desc" {
		isDesc = true
	}
	sortByApiField := ""
	if len(queryList) > 0 {
		sortByApiField = queryList[0]
	}
	// Check if the field can be sorted.
	sortByFieldName, ok := modelFieldByApiFieldMapping[sortByApiField]
	if !ok {
		return "", false, util.NewInvalidInputError("Cannot sort on field %v. Supported fields %v.",
			sortByApiField, keysString(modelFieldByApiFieldMapping))
	}
	return sortByFieldName, isDesc, nil
}

func keysString(modelFieldByApiFieldMapping map[string]string) string {
	keys := make([]string, 0, len(modelFieldByApiFieldMapping))
	for k := range modelFieldByApiFieldMapping {
		if k != "" {
			keys = append(keys, k)
		}
	}
	return "[" + strings.Join(keys, ", ") + "]"
}

// Decode page token. If page token is empty, we assume listing the first page and return a nil Token.
func deserializePageToken(pageToken string) (*common.Token, error) {
	if pageToken == "" {
		return nil, nil
	}
	tokenBytes, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Invalid package token.")
	}
	var token common.Token
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Invalid package token.")
	}
	return &token, nil
}
