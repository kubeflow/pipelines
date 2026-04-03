// Copyright 2018 The Kubeflow Authors
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

// Package list contains types and methods for performing ListXXX operations. In
// particular, the package exports the Options struct, which can be used for
// applying listing, filtering and pagination logic.
package list

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// token represents a WHERE clause when making a ListXXX query. It can either
// represent a query for an initial set of results, in which page
// SortByFieldValue and KeyFieldValue are nil. If the latter fields are not nil,
// then token represents a query for a subsequent set of results (i.e., the next
// page of results), with the two values pointing to the first record in the
// next set of results.
type token struct {
	// SortByFieldName is the user-facing field name used for pagination state
	// and GetFieldValue lookups. For metric sorts this is the raw metric name
	// (e.g. "accuracy"). Never use this field directly in SQL identifiers.
	SortByFieldName string
	// SortBySQLColumn is the safe SQL column name used in ORDER BY and WHERE
	// clauses. For regular fields it equals SortByFieldName. For metric sorts
	// it is always the fixed alias "sort_metric_value", never user input.
	SortBySQLColumn string
	// SortByFieldValue is the value of the sorted field of the next row to be
	// returned.
	SortByFieldValue  interface{}
	SortByFieldPrefix string

	// KeyFieldName is the name of the primary key for the model being queried.
	KeyFieldName string
	// KeyFieldValue is the value of the sorted field of the next row to be
	// returned.
	KeyFieldValue  interface{}
	KeyFieldPrefix string

	// IsDesc is true if the sorting order should be descending.
	IsDesc bool

	// ModelName is the table where ***FieldName belongs to.
	ModelName string

	// Filter represents the filtering that should be applied in the query.
	Filter *filter.Filter
}

// identifierPattern matches valid SQL identifier names: start with a letter,
// followed by letters, digits, or underscores, max 128 characters.
// Used to validate pageToken fields before they are used in SQL queries.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

// validateIdentifierName validates that a field name or table name only contains
// safe characters to prevent SQL injection through pageToken parameters.
func validateIdentifierName(name, fieldType string) error {
	if name == "" {
		return nil // empty names are handled elsewhere
	}
	if !identifierPattern.MatchString(name) {
		return util.NewInvalidInputError(
			"Invalid %s: %q. Field names must start with a letter and contain only letters, numbers, and underscores (max 128 characters)",
			fieldType, name)
	}
	return nil
}

func (t *token) unmarshal(pageToken string) error {
	errorF := func(err error) error {
		return util.NewInvalidInputErrorWithDetails(err, "Invalid page token")
	}
	b, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return errorF(err)
	}

	if err = json.Unmarshal(b, t); err != nil {
		return errorF(err)
	}

	// Validate all identifier fields to prevent SQL injection attacks.
	// pageToken fields are used to construct SQL queries; unvalidated field names
	// allow injection of arbitrary SQL through the pageToken parameter.
	if err := validateIdentifierName(t.KeyFieldName, "key field name"); err != nil {
		return err
	}
	if err := validateIdentifierName(t.SortByFieldName, "sort field name"); err != nil {
		return err
	}
	if err := validateIdentifierName(t.SortBySQLColumn, "sort SQL column"); err != nil {
		return err
	}
	if err := validateIdentifierName(t.ModelName, "model name"); err != nil {
		return err
	}
	if t.KeyFieldPrefix != "" {
		prefix := strings.TrimSuffix(t.KeyFieldPrefix, ".")
		if err := validateIdentifierName(prefix, "key field prefix"); err != nil {
			return err
		}
	}
	if t.SortByFieldPrefix != "" {
		prefix := strings.TrimSuffix(t.SortByFieldPrefix, ".")
		if err := validateIdentifierName(prefix, "sort field prefix"); err != nil {
			return err
		}
	}

	if t.Filter != nil {
		if err := t.Filter.ValidateKeys(func(segment string) error {
			return validateIdentifierName(segment, "filter key")
		}); err != nil {
			return err
		}
	}

	return nil
}

func (t *token) marshal() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to serialize page token")
	}
	// return string(b), nil
	return base64.StdEncoding.EncodeToString(b), nil
}

// Options represents options used when making a ListXXX query. In particular,
// it contains information on how to sort and filter results. It also
// encapsulates all the logic required for making the query for an initial set
// of results as well as subsequent pages of results.
type Options struct {
	PageSize int
	*token
}

func EmptyOptions() *Options {
	return &Options{
		math.MaxInt32,
		&token{},
	}
}

// Matches returns trues if the sorting and filtering criteria in o matches that
// of the one supplied in opts.
func (o *Options) Matches(opts *Options) bool {
	return o.SortByFieldName == opts.SortByFieldName && o.SortByFieldPrefix == opts.SortByFieldPrefix &&
		o.IsDesc == opts.IsDesc &&
		reflect.DeepEqual(o.Filter, opts.Filter)
}

// NewOptionsFromToken creates a new Options struct from the passed in token
// which represents the next page of results. An empty nextPageToken will result
// in an error.
func NewOptionsFromToken(nextPageToken string, pageSize int) (*Options, error) {
	if nextPageToken == "" {
		return nil, util.NewInvalidInputError("cannot create list.Options from empty page token")
	}
	pageSize, err := validatePageSize(pageSize)
	if err != nil {
		return nil, err
	}

	t := &token{}
	if err := t.unmarshal(nextPageToken); err != nil {
		return nil, err
	}
	return &Options{PageSize: pageSize, token: t}, nil
}

// NewOptions creates a new Options struct for the given listable. It uses
// sorting and filtering criteria parsed from sortBy and filterProto
// respectively.
func NewOptions(listable Listable, pageSize int, sortBy string, filter *filter.Filter) (*Options, error) {
	pageSize, err := validatePageSize(pageSize)
	if err != nil {
		return nil, err
	}

	token := &token{
		KeyFieldName: listable.PrimaryKeyColumnName(),
		ModelName:    listable.GetModelName(),
	}

	// Ignore the case of the letter. Split query string by space.
	queryList := strings.Fields(strings.ToLower(sortBy))
	// Check the query string format.
	if len(queryList) > 2 || (len(queryList) == 2 && queryList[1] != "desc" && queryList[1] != "asc") {
		return nil, util.NewInvalidInputError(
			"Received invalid sort by format %q. Supported format: \"field_name\", \"field_name desc\", or \"field_name asc\"", sortBy)
	}

	token.SortByFieldName = listable.DefaultSortField()
	token.SortBySQLColumn = token.SortByFieldName
	if len(queryList) > 0 {
		n, sqlCol, ok := listable.GetField(queryList[0])
		if ok {
			token.SortByFieldName = n
			token.SortBySQLColumn = sqlCol
		} else {
			return nil, util.NewInvalidInputError("Invalid sorting field: %q on listable type %s", queryList[0], reflect.ValueOf(listable).Elem().Type().Name())
		}
	}
	token.SortByFieldPrefix = listable.GetSortByFieldPrefix(token.SortByFieldName)
	token.KeyFieldPrefix = listable.GetKeyFieldPrefix()

	if len(queryList) == 2 {
		token.IsDesc = queryList[1] == "desc"
	}

	// Filtering.
	if filter != nil {
		if err := filter.ReplaceKeys(listable.APIToModelFieldMap(), listable.GetModelName()); err != nil {
			return nil, err
		}
		token.Filter = filter
	}
	return &Options{PageSize: pageSize, token: token}, nil
}

// AddPaginationToSelect adds WHERE clauses with the sorting and pagination criteria in the
// Options o to the supplied SelectBuilder, and returns the new SelectBuilder
// containing these.
func (o *Options) AddPaginationToSelect(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
	sqlBuilder = o.AddSortingToSelect(sqlBuilder)
	// Add one more item than what is requested.
	sqlBuilder = sqlBuilder.Limit(uint64(o.PageSize + 1))

	return sqlBuilder
}

// AddSortingToSelect adds Order By clause.
func (o *Options) AddSortingToSelect(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
	// When sorting by a direct field in the listable model (i.e., name in Run or uuid in Pipeline), a sortByFieldPrefix can be specified; when sorting by a field in an array-typed dictionary (i.e., a run metric inside the metrics in Run), a sortByFieldPrefix is not needed.
	// If next row's value is specified, set those values in the clause.
	if o.SortByFieldValue != nil && o.KeyFieldValue != nil {
		if o.IsDesc {
			sqlBuilder = sqlBuilder.
				Where(sq.Or{
					sq.Lt{o.SortByFieldPrefix + o.SortBySQLColumn: o.SortByFieldValue},
					sq.And{
						sq.Eq{o.SortByFieldPrefix + o.SortBySQLColumn: o.SortByFieldValue},
						sq.LtOrEq{o.KeyFieldPrefix + o.KeyFieldName: o.KeyFieldValue},
					},
				})
		} else {
			sqlBuilder = sqlBuilder.
				Where(sq.Or{
					sq.Gt{o.SortByFieldPrefix + o.SortBySQLColumn: o.SortByFieldValue},
					sq.And{
						sq.Eq{o.SortByFieldPrefix + o.SortBySQLColumn: o.SortByFieldValue},
						sq.GtOrEq{o.KeyFieldPrefix + o.KeyFieldName: o.KeyFieldValue},
					},
				})
		}
	}

	order := "ASC"
	if o.IsDesc {
		order = "DESC"
	}

	if o.SortBySQLColumn != "" {
		sqlBuilder = sqlBuilder.OrderBy(fmt.Sprintf("%v %v", o.SortByFieldPrefix+o.SortBySQLColumn, order))
	}

	if o.KeyFieldName != "" {
		sqlBuilder = sqlBuilder.OrderBy(fmt.Sprintf("%v %v", o.KeyFieldPrefix+o.KeyFieldName, order))
	}

	return sqlBuilder
}

// AddFilterToSelect adds WHERE clauses with the filtering criteria in the
// Options o to the supplied SelectBuilder, and returns the new SelectBuilder
// containing these.
func (o *Options) AddFilterToSelect(sqlBuilder sq.SelectBuilder) sq.SelectBuilder {
	if o.Filter != nil {
		sqlBuilder = o.Filter.AddToSelect(sqlBuilder)
	}

	return sqlBuilder
}

// FilterOnResourceReference filters the given resource's table by rows from the ResourceReferences
// table that match an optional given filter, and returns the rebuilt SelectBuilder.
func FilterOnResourceReference(tableName string, columns []string, resourceType model.ResourceType,
	selectCount bool, filterContext *model.FilterContext,
) (sq.SelectBuilder, error) {
	selectBuilder := sq.Select(columns...)
	if selectCount {
		selectBuilder = sq.Select("count(*)")
	}
	selectBuilder = selectBuilder.From(tableName)
	if filterContext.ReferenceKey != nil && (filterContext.ReferenceKey.ID != "" || common.IsMultiUserMode()) {
		resourceReferenceFilter, args, err := sq.Select("ResourceUUID").
			From("resource_references as rf").
			Where(sq.And{
				sq.Eq{"rf.ResourceType": resourceType},
				sq.Eq{"rf.ReferenceUUID": filterContext.ID},
				sq.Eq{"rf.ReferenceType": filterContext.Type},
			}).ToSql()
		if err != nil {
			return selectBuilder, util.NewInternalServerError(
				err, "Failed to create subquery to filter by resource reference: %v", err.Error())
		}
		return selectBuilder.Where(fmt.Sprintf("UUID in (%s)", resourceReferenceFilter), args...), nil
	}
	return selectBuilder, nil
}

// FilterOnExperiment filters the given table by rows based on provided experiment ID,
// and returns the rebuilt SelectBuilder.
func FilterOnExperiment(
	tableName string,
	columns []string,
	selectCount bool,
	experimentID string,
) (sq.SelectBuilder, error) {
	return filterByColumnValue(tableName, columns, selectCount, "ExperimentUUID", experimentID), nil
}

func FilterOnNamespace(
	tableName string,
	columns []string,
	selectCount bool,
	namespace string,
) (sq.SelectBuilder, error) {
	return filterByColumnValue(tableName, columns, selectCount, "Namespace", namespace), nil
}

func filterByColumnValue(
	tableName string,
	columns []string,
	selectCount bool,
	columnName string,
	filterValue interface{},
) sq.SelectBuilder {
	selectBuilder := sq.Select(columns...)
	if selectCount {
		selectBuilder = sq.Select("count(*)")
	}
	selectBuilder = selectBuilder.From(tableName).Where(
		sq.Eq{columnName: filterValue},
	)
	return selectBuilder
}

// Scans the one given row into a number, and returns the number.
func ScanRowToTotalSize(rows *sql.Rows) (int, error) {
	var total_size int
	rows.Next()
	err := rows.Scan(&total_size)
	if err != nil {
		return 0, util.NewInternalServerError(err, "Failed to scan row total_size")
	}
	return total_size, nil
}

// Listable is an interface that should be implemented by any resource/model
// that wants to support listing queries.
type Listable interface {
	// PrimaryKeyColumnName returns the primary key for model.
	PrimaryKeyColumnName() string
	// DefaultSortField returns the default field name to be used when sorting list
	// query results.
	DefaultSortField() string
	// APIToModelFieldMap returns a map from field names in the API representation
	// of the model to its corresponding field name in the model itself.
	APIToModelFieldMap() map[string]string
	// GetModelName returns table name used as sort field prefix.
	GetModelName() string
	// Get the prefix of sorting field.
	GetSortByFieldPrefix(string) string
	// Get the prefix of key field.
	GetKeyFieldPrefix() string
	// GetField returns the model field name and safe SQL column name for the
	// given API field name. For regular fields fieldName and sqlColumn are
	// identical. For metric fields (e.g. "metric:accuracy") sqlColumn is the
	// fixed alias "sort_metric_value" so user input never reaches SQL structure.
	GetField(name string) (fieldName string, sqlColumn string, ok bool)
	// Find the value of a given field in a listable object.
	GetFieldValue(name string) interface{}
}

// NextPageToken returns a string that can be used to fetch the subsequent set
// of results using the same listing options in o, starting with listable as the
// first record.
func (o *Options) NextPageToken(listable Listable) (string, error) {
	t, err := o.nextPageToken(listable)
	if err != nil {
		return "", err
	}
	return t.marshal()
}

func (o *Options) nextPageToken(listable Listable) (*token, error) {
	elem := reflect.ValueOf(listable).Elem()
	elemName := elem.Type().Name()

	var sortByField interface{}
	if sortByField = listable.GetFieldValue(o.SortByFieldName); sortByField == nil {
		return nil, util.NewInvalidInputError("cannot sort by field %q on type %q", o.SortByFieldName, elemName)
	}

	keyField := elem.FieldByName(listable.PrimaryKeyColumnName())
	if !keyField.IsValid() {
		return nil, util.NewInvalidInputError("type %q does not have key field %q", elemName, o.KeyFieldName)
	}

	return &token{
		SortByFieldName:   o.SortByFieldName,
		SortBySQLColumn:   o.SortBySQLColumn,
		SortByFieldValue:  sortByField,
		SortByFieldPrefix: listable.GetSortByFieldPrefix(o.SortByFieldName),
		KeyFieldName:      listable.PrimaryKeyColumnName(),
		KeyFieldValue:     keyField.Interface(),
		KeyFieldPrefix:    listable.GetKeyFieldPrefix(),
		IsDesc:            o.IsDesc,
		Filter:            o.Filter,
		ModelName:         o.ModelName,
	}, nil
}

const (
	defaultPageSize = 20
	maxPageSize     = 200
)

func validatePageSize(pageSize int) (int, error) {
	if pageSize < 0 {
		return 0, util.NewInvalidInputError("The page size should be greater than 0. Got %d", pageSize)
	}

	if pageSize == 0 {
		// Use default page size if not provided.
		return defaultPageSize, nil
	}

	if pageSize > maxPageSize {
		return maxPageSize, nil
	}

	return pageSize, nil
}
