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
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// identifierPattern matches valid SQL identifier names: start with a letter,
// followed by letters, digits, or underscores, max 128 characters.
// Used to validate pageToken fields before they are used in SQL queries.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,127}$`)

// metricNamePattern matches valid metric names. Metric names follow the same
// rules as SQL identifiers but additionally allow hyphens ("-"), since ML
// frameworks commonly use names like "log-loss" or "val-accuracy".
// Metric names are never used as SQL identifiers — they are passed as bind
// parameters — so allowing "-" here is safe.
var metricNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]{0,127}$`)

// validateIdentifierName validates that a field name or table name only contains
// safe characters to prevent SQL injection through pageToken parameters.
func validateIdentifierName(name, fieldType string) error {
	if name == "" {
		return nil
	}
	if !identifierPattern.MatchString(name) {
		return util.NewInvalidInputError(
			"Invalid %s: %q. Field names must start with a letter and contain only letters, numbers, and underscores (max 128 characters)",
			fieldType, name)
	}
	return nil
}

// validateMetricName validates that a metric name only contains safe characters.
// Unlike validateIdentifierName, hyphens are permitted.
func validateMetricName(name string) error {
	if name == "" {
		return nil
	}
	if !metricNamePattern.MatchString(name) {
		return util.NewInvalidInputError(
			"Invalid metric name: %q. Metric names must start with a letter and contain only letters, numbers, underscores, and hyphens (max 128 characters)",
			name)
	}
	return nil
}

// token represents a WHERE clause when making a ListXXX query. It can either
// represent a query for an initial set of results, in which page
// SortByFieldValue and KeyFieldValue are nil. If the latter fields are not nil,
// then token represents a query for a subsequent set of results (i.e., the next
// page of results), with the two values pointing to the first record in the
// next set of results.
type token struct {
	// SortByFieldName is the SQL-safe column name used in ORDER BY and WHERE
	// clauses. For regular fields it equals the model field name. For metric
	// sorts it is always the fixed constant model.MetricSortSQLAlias
	// ("sort_metric_value"), never the user-supplied metric name.
	SortByFieldName string
	// SortByMetricName is the original metric name supplied by the user when
	// sorting by a run metric (e.g. "log-loss"). It is used only as a bind
	// parameter value in CASE WHEN queries, never as a SQL identifier.
	// Empty for non-metric sorts.
	SortByMetricName string
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

	// SortByFieldIsString indicates whether the sort field is a string type.
	// Used to decide whether to apply LOWER() in ORDER BY and WHERE clauses.
	// This avoids relying on the runtime type of SortByFieldValue, which is nil
	// on the first page and therefore cannot be used for type inference.
	SortByFieldIsString bool

	// ModelName is the table where ***FieldName belongs to.
	ModelName string

	// Filter represents the filtering that should be applied in the query.
	Filter *filter.Filter
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

	// Migrate legacy tokens: before the SortByMetricName field was introduced,
	// the user-supplied metric name (e.g. "log-loss") was stored directly in
	// SortByFieldName. Such values fail SQL identifier validation, so detect
	// and migrate them to the new layout before validating.
	if t.SortByMetricName == "" && t.SortByFieldName != "" && !identifierPattern.MatchString(t.SortByFieldName) {
		t.SortByMetricName = t.SortByFieldName
		t.SortByFieldName = model.MetricSortSQLAlias
	}

	// Normalize prefixes: legacy tokens may carry a trailing dot; strip it so
	// downstream SQL quoting does not produce double dots.
	t.KeyFieldPrefix = strings.TrimSuffix(t.KeyFieldPrefix, ".")
	t.SortByFieldPrefix = strings.TrimSuffix(t.SortByFieldPrefix, ".")

	// Validate all identifier fields to prevent SQL injection attacks.
	// pageToken fields are used to construct SQL queries; unvalidated field
	// names allow injection of arbitrary SQL through the pageToken parameter.
	if err := validateIdentifierName(t.KeyFieldName, "key field name"); err != nil {
		return err
	}
	// SortByFieldName is the SQL column name (always a valid identifier).
	// SortByMetricName is the user-supplied metric name (allows hyphens).
	if err := validateIdentifierName(t.SortByFieldName, "sort field name"); err != nil {
		return err
	}
	if err := validateMetricName(t.SortByMetricName); err != nil {
		return err
	}
	if err := validateIdentifierName(t.ModelName, "model name"); err != nil {
		return err
	}
	if t.KeyFieldPrefix != "" {
		if err := validateIdentifierName(t.KeyFieldPrefix, "key field prefix"); err != nil {
			return err
		}
	}
	if t.SortByFieldPrefix != "" {
		if err := validateIdentifierName(t.SortByFieldPrefix, "sort field prefix"); err != nil {
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
		PageSize: math.MaxInt32,
		token:    &token{},
	}
}

// Matches returns trues if the sorting and filtering criteria in o matches that
// of the one supplied in opts.
func (o *Options) Matches(opts *Options) bool {
	return o.SortByFieldName == opts.SortByFieldName &&
		o.SortByMetricName == opts.SortByMetricName &&
		o.SortByFieldPrefix == opts.SortByFieldPrefix &&
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
	if len(queryList) > 0 {
		n, sqlCol, ok := listable.GetField(queryList[0])
		if ok {
			// sqlCol is the SQL-safe column name (for metrics: MetricSortSQLAlias).
			// n is the original field/metric name used for value lookups.
			token.SortByFieldName = sqlCol
			if n != sqlCol {
				// Metric sort: validate the name up front so an invalid metric name
				// fails on page 1 rather than on page 2 when the token is decoded.
				if err := validateMetricName(n); err != nil {
					return nil, err
				}
				token.SortByMetricName = n
			}
		} else {
			return nil, util.NewInvalidInputError("Invalid sorting field: %q on listable type %s", queryList[0], reflect.ValueOf(listable).Elem().Type().Name())
		}
	}
	token.SortByFieldPrefix = listable.GetSortByFieldPrefix(token.SortByFieldName)
	token.KeyFieldPrefix = listable.GetKeyFieldPrefix()

	// Probe the sort field type using the listable instance.
	// For metric sorts use the original metric name; for regular fields use SortByFieldName.
	// string fields return "" (string type); numeric fields return int64(0) or similar.
	probeFieldName := token.SortByFieldName
	if token.SortByMetricName != "" {
		probeFieldName = token.SortByMetricName
	}
	probeVal := listable.GetFieldValue(probeFieldName)
	_, token.SortByFieldIsString = probeVal.(string)

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
// The quote parameter is used to quote SQL identifiers (e.g., table and column names) based on the database dialect.
// If quote is nil, identifiers are not quoted.
// The collation parameter is appended after LOWER() expressions for string sorting and cursor
// comparisons (e.g., `COLLATE "C"` for PostgreSQL, "" for MySQL/SQLite).
func (o *Options) AddPaginationToSelect(sqlBuilder sq.SelectBuilder, quote func(string) string, collation string) sq.SelectBuilder {
	sqlBuilder = o.AddSortingToSelect(sqlBuilder, quote, collation)
	// Add one more item than what is requested.
	sqlBuilder = sqlBuilder.Limit(uint64(o.PageSize + 1))

	return sqlBuilder
}

// AddSortingToSelect adds Order By clause.
// The quote parameter is used to quote SQL identifiers (e.g., table and column names) based on the database dialect.
// If quote is nil, identifiers are not quoted.
// The collation parameter is appended after LOWER() expressions for string sorting and cursor
// comparisons to ensure consistent ordering across databases (e.g., `COLLATE "C"` for
// PostgreSQL byte-order sorting, "" for MySQL/SQLite which use their default collation).
func (o *Options) AddSortingToSelect(sqlBuilder sq.SelectBuilder, quote func(string) string, collation string) sq.SelectBuilder {
	if quote == nil {
		quote = func(s string) string { return s }
	}

	// lowerWithCollation wraps col in LOWER() and appends the collation clause if non-empty.
	// Used consistently in both ORDER BY and WHERE cursor comparisons so that pagination
	// boundaries are evaluated with the same collation as the sort order.
	lowerWithCollation := func(col string) string {
		if collation == "" {
			return fmt.Sprintf("LOWER(%s)", col)
		}
		return fmt.Sprintf("LOWER(%s) %s", col, collation)
	}

	sortByFieldNameWithPrefix := o.SortByFieldPrefix
	if sortByFieldNameWithPrefix != "" {
		sortByFieldNameWithPrefix = quote(sortByFieldNameWithPrefix) + "."
	}
	sortByFieldNameWithPrefix += quote(o.SortByFieldName)

	keyFieldNameWithPrefix := o.KeyFieldPrefix
	if keyFieldNameWithPrefix != "" {
		keyFieldNameWithPrefix = quote(keyFieldNameWithPrefix) + "."
	}
	keyFieldNameWithPrefix += quote(o.KeyFieldName)

	// When sorting by a direct field in the listable model (i.e., name in Run or uuid in Pipeline), a sortByFieldPrefix can be specified; when sorting by a field in an array-typed dictionary (i.e., a run metric inside the metrics in Run), a sortByFieldPrefix is not needed.
	// If next row's value is specified, set those values in the clause.
	if o.SortByFieldValue != nil && o.KeyFieldValue != nil {
		// Use SortByFieldIsString (set at Options creation time) to determine field type.
		// Also check the runtime value type as a fallback for tokens created before this field existed.
		_, valueIsString := o.SortByFieldValue.(string)
		isStringField := o.SortByFieldIsString || valueIsString

		strVal, sortValueIsString := o.SortByFieldValue.(string)
		floatVal, sortValueIsFloat := o.SortByFieldValue.(float64)

		sortCol := lowerWithCollation(sortByFieldNameWithPrefix)

		if o.IsDesc {
			if isStringField && sortValueIsString {
				// String field: use LOWER() with collation for case-insensitive comparison
				sqlBuilder = sqlBuilder.
					Where(sq.Or{
						sq.Expr(fmt.Sprintf("%s < %s", sortCol, lowerWithCollation("?")), strVal),
						sq.And{
							sq.Expr(fmt.Sprintf("%s = %s", sortCol, lowerWithCollation("?")), strVal),
							sq.LtOrEq{keyFieldNameWithPrefix: o.KeyFieldValue},
						},
					})
			} else if !isStringField && sortValueIsFloat {
				// Numeric field: use bind parameter to preserve full float64 precision and prevent injection
				sqlBuilder = sqlBuilder.
					Where(sq.Or{
						sq.Expr(sortByFieldNameWithPrefix+" < ?", floatVal),
						sq.And{
							sq.Expr(sortByFieldNameWithPrefix+" = ?", floatVal),
							sq.LtOrEq{keyFieldNameWithPrefix: o.KeyFieldValue},
						},
					})
			}
		} else {
			if isStringField && sortValueIsString {
				// String field: use LOWER() with collation for case-insensitive comparison
				sqlBuilder = sqlBuilder.
					Where(sq.Or{
						sq.Expr(fmt.Sprintf("%s > %s", sortCol, lowerWithCollation("?")), strVal),
						sq.And{
							sq.Expr(fmt.Sprintf("%s = %s", sortCol, lowerWithCollation("?")), strVal),
							sq.GtOrEq{keyFieldNameWithPrefix: o.KeyFieldValue},
						},
					})
			} else if !isStringField && sortValueIsFloat {
				// Numeric field: use bind parameter to preserve full float64 precision and prevent injection
				sqlBuilder = sqlBuilder.
					Where(sq.Or{
						sq.Expr(sortByFieldNameWithPrefix+" > ?", floatVal),
						sq.And{
							sq.Expr(sortByFieldNameWithPrefix+" = ?", floatVal),
							sq.GtOrEq{keyFieldNameWithPrefix: o.KeyFieldValue},
						},
					})
			}
		}
	}

	order := "ASC"
	if o.IsDesc {
		order = "DESC"
	}

	if o.SortByFieldName != "" {
		// Use SortByFieldIsString to decide whether to wrap with LOWER().
		// Also check runtime value type as fallback for old tokens that lack this field.
		_, valueIsString := o.SortByFieldValue.(string)
		if o.SortByFieldIsString || valueIsString {
			sortCol := lowerWithCollation(sortByFieldNameWithPrefix)
			sqlBuilder = sqlBuilder.OrderBy(fmt.Sprintf("%v %v", sortCol, order))
		} else {
			sqlBuilder = sqlBuilder.OrderBy(fmt.Sprintf("%v %v", sortByFieldNameWithPrefix, order))
		}
	}

	if o.KeyFieldName != "" {
		sqlBuilder = sqlBuilder.OrderBy(fmt.Sprintf("%v %v", keyFieldNameWithPrefix, order))
	}

	return sqlBuilder
}

// AddFilterToSelect adds WHERE clauses with the filtering criteria in the
// Options o to the supplied SelectBuilder, and returns the new SelectBuilder
// containing these.
// The quote parameter is used to quote SQL identifiers (e.g., table and column names) based on the database dialect.
// If quote is nil, identifiers are not quoted.
func (o *Options) AddFilterToSelect(sqlBuilder sq.SelectBuilder, quote func(string) string) sq.SelectBuilder {
	if o.Filter != nil {
		sqlBuilder = o.Filter.AddToSelect(sqlBuilder, quote)
	}

	return sqlBuilder
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

func (o *Options) GetSortByFieldValue() interface{} {
	if o.token == nil {
		return nil
	}
	return o.SortByFieldValue
}

// WithSortByFieldValue returns a new Options struct with the specific sort by field value.
// It maintains immutability of the original Options.
func (o *Options) WithSortByFieldValue(val interface{}) *Options {
	newOpts := *o
	if o.token != nil {
		newToken := *o.token
		newToken.SortByFieldValue = val
		newOpts.token = &newToken
	}
	return &newOpts
}

func (o *Options) nextPageToken(listable Listable) (*token, error) {
	elem := reflect.ValueOf(listable).Elem()
	elemName := elem.Type().Name()

	// For metric sorts, SortByFieldName is the fixed alias (e.g. "sort_metric_value"),
	// not a real struct field. Use the original metric name to look up the value.
	fieldNameForValue := o.SortByFieldName
	if o.SortByMetricName != "" {
		fieldNameForValue = o.SortByMetricName
	}

	var sortByField interface{}
	if sortByField = listable.GetFieldValue(fieldNameForValue); sortByField == nil {
		return nil, util.NewInvalidInputError("cannot sort by field %q on type %q", fieldNameForValue, elemName)
	}

	keyField := elem.FieldByName(listable.PrimaryKeyColumnName())
	if !keyField.IsValid() {
		return nil, util.NewInvalidInputError("type %q does not have key field %q", elemName, o.KeyFieldName)
	}

	return &token{
		SortByFieldName:     o.SortByFieldName,
		SortByMetricName:    o.SortByMetricName,
		SortByFieldValue:    sortByField,
		SortByFieldPrefix:   listable.GetSortByFieldPrefix(o.SortByFieldName),
		SortByFieldIsString: o.SortByFieldIsString,
		KeyFieldName:        listable.PrimaryKeyColumnName(),
		KeyFieldValue:       keyField.Interface(),
		KeyFieldPrefix:      listable.GetKeyFieldPrefix(),
		IsDesc:              o.IsDesc,
		Filter:              o.Filter,
		ModelName:           o.ModelName,
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
