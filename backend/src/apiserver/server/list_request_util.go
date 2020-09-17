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
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
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
	"name":       "DisplayName",
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
	if pageSize > maxPageSize {
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

// parseAPIFilter attempts to decode a url-encoded JSON-stringified api
// filter object. An empty string is considered valid input, and equivalent to
// the nil filter, which trivially does nothing.
func parseAPIFilter(encoded string) (*api.Filter, error) {
	if encoded == "" {
		return nil, nil
	}

	errorF := func(err error) (*api.Filter, error) {
		return nil, util.NewInvalidInputError("failed to parse valid filter from %q: %v", encoded, err)
	}

	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return errorF(err)
	}

	f := &api.Filter{}
	if err := jsonpb.UnmarshalString(string(decoded), f); err != nil {
		return errorF(err)
	}
	return f, nil
}

func validatedListOptions(listable list.Listable, pageToken string, pageSize int, sortBy string, filterSpec string) (*list.Options, error) {
	defaultOpts := func() (*list.Options, error) {
		if listable == nil {
			return nil, util.NewInvalidInputError("Please specify a valid type to list. E.g., list runs or list jobs.")
		}

		f, err := parseAPIFilter(filterSpec)
		if err != nil {
			return nil, err
		}

		return list.NewOptions(listable, pageSize, sortBy, f)
	}

	if pageToken == "" {
		return defaultOpts()
	}

	opts, err := list.NewOptionsFromToken(pageToken, pageSize)
	if err != nil {
		return nil, err
	}

	if sortBy != "" || filterSpec != "" {
		// Sanity check that these match the page token.
		do, err := defaultOpts()
		if err != nil {
			return nil, err
		}

		if !opts.Matches(do) {
			return nil, util.NewInvalidInputError("page token does not match the supplied sort by and/or filtering criteria. Either specify the same criteria or leave the latter empty if page token is specified.")
		}
	}

	return opts, nil
}
