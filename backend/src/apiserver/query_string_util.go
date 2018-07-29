package main

import (
	"strings"

	"github.com/googleprivate/ml/backend/src/common/util"
)

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
