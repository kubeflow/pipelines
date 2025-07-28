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

// Package filter contains types and methods for parsing and applying filters to
// resources being queried by a ListXXX request.
package filter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
)

const filterMessage = "Filter %v is not implemented for Kubernetes pipeline store. Only substring is supported."

// Internal representation of a predicate.
type Predicate struct {
	operation string
	key       string
	value     interface{}
}

// Filter represents a filter that can be applied when querying an arbitrary API
// resource.
type Filter struct {
	eq  map[string][]interface{}
	neq map[string][]interface{}
	gt  map[string][]interface{}
	gte map[string][]interface{}
	lt  map[string][]interface{}
	lte map[string][]interface{}

	in map[string][]interface{}

	substring map[string][]interface{}
}

// filterForMarshaling is a helper struct for marshaling Filter into JSON. This
// is needed as we don't want to export the fields in Filter.
type filterForMarshaling struct {
	EQ  map[string][]interface{}
	NEQ map[string][]interface{}
	GT  map[string][]interface{}
	GTE map[string][]interface{}
	LT  map[string][]interface{}
	LTE map[string][]interface{}

	IN map[string][]interface{}

	SUBSTRING map[string][]interface{}
}

// MarshalJSON implements JSON Marshaler for Filter.
func (f *Filter) MarshalJSON() ([]byte, error) {
	return json.Marshal(&filterForMarshaling{
		EQ:        f.eq,
		NEQ:       f.neq,
		GT:        f.gt,
		GTE:       f.gte,
		LT:        f.lt,
		LTE:       f.lte,
		IN:        f.in,
		SUBSTRING: f.substring,
	})
}

// UnmarshalJSON implements JSON Unmarshaler for Filter.
func (f *Filter) UnmarshalJSON(b []byte) error {
	ffm := filterForMarshaling{}
	err := json.Unmarshal(b, &ffm)
	if err != nil {
		return err
	}

	f.eq = ffm.EQ
	f.neq = ffm.NEQ
	f.gt = ffm.GT
	f.gte = ffm.GTE
	f.lt = ffm.LT
	f.lte = ffm.LTE
	f.in = ffm.IN
	f.substring = ffm.SUBSTRING

	return nil
}

// New creates a new Filter from parsing the API filter protocol buffer.
func New(filterProto interface{}) (*Filter, error) {
	predicates, err := toPredicates(filterProto)
	if err != nil {
		return nil, err
	}
	return NewFromPredicate(predicates)
}

// NewWithKeyMap is like New, but takes an additional map and model name for mapping key names
// in the protocol buffer to an appropriate name for use when querying the
// model. For example, if the API name of a field is "name", the model name is "pipelines", and
// the equivalent column name is "Name", then filterProto with predicates against key "name"
// will be parsed as if the key value was "pipelines.Name".
func NewWithKeyMap(filterProto interface{}, keyMap map[string]string, modelName string) (*Filter, error) {
	// Fully qualify column name to avoid "ambiguous column name" error.
	var modelNamePrefix string
	if modelName != "" {
		modelNamePrefix = modelName + "."
	}

	predicates, err := toPredicates(filterProto)
	if err != nil {
		return nil, err
	}

	for _, pred := range predicates {
		k, ok := keyMap[pred.key]
		if !ok {
			return nil, util.NewInvalidInputError("no support for filtering on unrecognized field %q", pred.key)
		}
		pred.key = modelNamePrefix + k
	}
	return NewFromPredicate(predicates)
}

// New creates a new Filter from parsed predicates.
func NewFromPredicate(predicates []*Predicate) (*Filter, error) {
	if len(predicates) == 0 {
		return nil, nil
	}

	f := &Filter{
		eq:        make(map[string][]interface{}, 0),
		neq:       make(map[string][]interface{}, 0),
		gt:        make(map[string][]interface{}, 0),
		gte:       make(map[string][]interface{}, 0),
		lt:        make(map[string][]interface{}, 0),
		lte:       make(map[string][]interface{}, 0),
		in:        make(map[string][]interface{}, 0),
		substring: make(map[string][]interface{}, 0),
	}

	if err := f.parsePredicates(predicates); err != nil {
		return nil, err
	}
	return f, nil
}

// Replaces and adds a prefix to the keys for an existing filter.
// This is useful when someone wants to extend the filter with a table name.
func (f *Filter) ReplaceKeys(keyMap map[string]string, prefix string) error {
	if prefix != "" {
		prefix = prefix + "."
	}
	if err := replaceMapKeys(f.eq, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.neq, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.gt, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.gte, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.lt, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.lte, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.in, keyMap, prefix); err != nil {
		return err
	}
	if err := replaceMapKeys(f.substring, keyMap, prefix); err != nil {
		return err
	}
	return nil
}

// Replaces string keys in a map and adds a prefix.
func replaceMapKeys(m map[string][]interface{}, keyMap map[string]string, prefix string) error {
	keys := make([]string, 0)
	for k := range m {
		keys = append(keys, k)
	}
	for _, k := range keys {
		newKey, ok := keyMap[k]
		if !ok {
			return util.NewInvalidInputError("no support for filtering on unrecognized field %q", k)
		}
		m[prefix+newKey] = m[k]
		delete(m, k)
	}
	return nil
}

func (f *Filter) FilterK8sPipelines(pipeline v2beta1.Pipeline) (bool, error) {
	if len(f.eq) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "eq"))
	}

	if len(f.neq) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "neq"))
	}

	if len(f.gt) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "gt"))
	}

	if len(f.gte) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "gte"))
	}

	if len(f.lt) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "lt"))
	}

	if len(f.lte) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "lte"))
	}

	if len(f.in) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "in"))
	}

	for k := range f.substring {
		for _, v := range f.substring[k] {
			if strings.Contains(fmt.Sprint(pipeline.GetField(k)), fmt.Sprint(v)) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (f *Filter) FilterK8sPipelineVersions(pipelineVersion v2beta1.PipelineVersion) (bool, error) {
	if len(f.eq) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "eq"))
	}

	if len(f.neq) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "neq"))
	}

	if len(f.gt) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "gt"))
	}

	if len(f.gte) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "gte"))
	}

	if len(f.lt) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "lt"))
	}

	if len(f.lte) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "lte"))
	}

	if len(f.in) > 0 {
		return false, util.NewInvalidInputError("%s", fmt.Sprintf(filterMessage, "in"))
	}

	for k := range f.substring {
		for _, v := range f.substring[k] {
			if strings.Contains(fmt.Sprint(pipelineVersion.GetField(k)), fmt.Sprint(v)) {
				return true, nil
			}
		}
	}

	return false, nil
}

// AddToSelect builds a WHERE clause from the Filter f, adds it to the supplied
// SelectBuilder object and returns it for use in SQL queries.
func (f *Filter) AddToSelect(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
	for k := range f.eq {
		for _, v := range f.eq[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.Eq(m))
		}
	}

	for k := range f.neq {
		for _, v := range f.neq[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.NotEq(m))
		}
	}

	for k := range f.gt {
		for _, v := range f.gt[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.Gt(m))
		}
	}

	for k := range f.gte {
		for _, v := range f.gte[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.GtOrEq(m))
		}
	}

	for k := range f.lt {
		for _, v := range f.lt[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.Lt(m))
		}
	}

	for k := range f.lte {
		for _, v := range f.lte[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.LtOrEq(m))
		}
	}

	// In
	for k := range f.in {
		for _, v := range f.in[k] {
			m := map[string]interface{}{k: v}
			sb = sb.Where(squirrel.Eq(m))
		}
	}

	for k := range f.substring {
		// Modify each string value v so it looks like %v% so we are doing a substring
		// match with the LIKE operator.
		for _, v := range f.substring[k] {
			like := make(squirrel.Like)
			like[k] = fmt.Sprintf("%%%s%%", v)
			sb = sb.Where(like)
		}
	}

	return sb
}

func checkPredicate(p *Predicate) error {
	switch p.operation {
	case apiv1beta1.Predicate_IN.String(), apiv2beta1.Predicate_IN.String():
		switch t := p.value.(type) {
		case int32, int64, string:
			return util.NewInvalidInputError("cannot use IN operator with scalar type %T", t)
		}
	case apiv1beta1.Predicate_EQUALS.String(), apiv1beta1.Predicate_NOT_EQUALS.String(), apiv1beta1.Predicate_GREATER_THAN.String(), apiv1beta1.Predicate_GREATER_THAN_EQUALS.String(), apiv1beta1.Predicate_LESS_THAN.String(), apiv1beta1.Predicate_LESS_THAN_EQUALS.String(), apiv2beta1.Predicate_EQUALS.String(), apiv2beta1.Predicate_NOT_EQUALS.String(), apiv2beta1.Predicate_GREATER_THAN.String(), apiv2beta1.Predicate_GREATER_THAN_EQUALS.String(), apiv2beta1.Predicate_LESS_THAN.String(), apiv2beta1.Predicate_LESS_THAN_EQUALS.String():
		switch t := p.value.(type) {
		case []int32, []int64, []string:
			return util.NewInvalidInputError("cannot use scalar operator %v on array type %T", p.operation, t)
		}
	case apiv1beta1.Predicate_IS_SUBSTRING.String(), apiv2beta1.Predicate_IS_SUBSTRING.String():
		switch t := p.value.(type) {
		case string:
			return nil
		default:
			return util.NewInvalidInputError("cannot use non string value type %T with operator %v", p.operation, t)
		}
	default:
		return util.NewInvalidInputError("invalid predicate operation: %v", p.operation)
	}

	return nil
}

func (f *Filter) parsePredicates(preds []*Predicate) error {
	for _, pred := range preds {
		if err := checkPredicate(pred); err != nil {
			return err
		}
		switch pred.operation {
		case "EQUALS":
			f.eq[pred.key] = append(f.eq[pred.key], pred.value)
		case "NOT_EQUALS":
			f.neq[pred.key] = append(f.neq[pred.key], pred.value)
		case "GREATER_THAN":
			f.gt[pred.key] = append(f.gt[pred.key], pred.value)
		case "GREATER_THAN_EQUALS":
			f.gte[pred.key] = append(f.gte[pred.key], pred.value)
		case "LESS_THAN":
			f.lt[pred.key] = append(f.lt[pred.key], pred.value)
		case "LESS_THAN_EQUALS":
			f.lte[pred.key] = append(f.lte[pred.key], pred.value)
		case "IN":
			f.in[pred.key] = append(f.in[pred.key], pred.value)
		case "IS_SUBSTRING":
			f.substring[pred.key] = append(f.substring[pred.key], pred.value)
		default:
			return util.NewInvalidInputError("invalid predicate operation: %v", pred.operation)
		}
	}

	return nil
}

func toPredicates(filterProto interface{}) ([]*Predicate, error) {
	if filterProto == nil {
		return nil, nil
	}
	predicates := make([]*Predicate, 0)
	switch filterProto := filterProto.(type) {
	case *apiv2beta1.Filter:
		for _, p := range filterProto.GetPredicates() {
			if pred, err := toPredicate(p); err != nil {
				return nil, err
			} else {
				predicates = append(predicates, pred)
			}
		}
	case *apiv1beta1.Filter:
		for _, p := range filterProto.GetPredicates() {
			if pred, err := toPredicate(p); err != nil {
				return nil, err
			} else {
				predicates = append(predicates, pred)
			}
		}
	default:
		return nil, util.NewUnknownApiVersionError("Filter", filterProto)
	}
	return predicates, nil
}

func toPredicate(p interface{}) (*Predicate, error) {
	if p == nil {
		return nil, nil
	}
	operation := ""
	key := ""
	var value interface{}
	switch p := p.(type) {
	case *apiv2beta1.Predicate:
		key = p.GetKey()
		if temp, err := toOperation(p.GetOperation()); err != nil {
			return nil, err
		} else {
			operation = temp
		}
		if temp, err := toValue(p.GetValue()); err != nil {
			return nil, err
		} else {
			value = temp
		}
	case *apiv1beta1.Predicate:
		key = p.GetKey()
		if temp, err := toOperation(p.GetOp()); err != nil {
			return nil, err
		} else {
			operation = temp
		}
		if temp, err := toValue(p.GetValue()); err != nil {
			return nil, err
		} else {
			value = temp
		}
	default:
		return nil, util.NewUnknownApiVersionError("Filter.Predicate", p)
	}
	if key == "" {
		return nil, util.NewInvalidInputError("Predicate key cannot be empty for operation %v and value %v", operation, value)
	}
	return &Predicate{
		operation: operation,
		key:       key,
		value:     value,
	}, nil
}

func toOperation(o interface{}) (string, error) {
	switch o {
	case apiv2beta1.Predicate_EQUALS, apiv1beta1.Predicate_EQUALS:
		return "EQUALS", nil
	case apiv2beta1.Predicate_NOT_EQUALS, apiv1beta1.Predicate_NOT_EQUALS:
		return "NOT_EQUALS", nil
	case apiv2beta1.Predicate_GREATER_THAN, apiv1beta1.Predicate_GREATER_THAN:
		return "GREATER_THAN", nil
	case apiv2beta1.Predicate_GREATER_THAN_EQUALS, apiv1beta1.Predicate_GREATER_THAN_EQUALS:
		return "GREATER_THAN_EQUALS", nil
	case apiv2beta1.Predicate_LESS_THAN, apiv1beta1.Predicate_LESS_THAN:
		return "LESS_THAN", nil
	case apiv2beta1.Predicate_LESS_THAN_EQUALS, apiv1beta1.Predicate_LESS_THAN_EQUALS:
		return "LESS_THAN_EQUALS", nil
	case apiv2beta1.Predicate_IN, apiv1beta1.Predicate_IN:
		return "IN", nil
	case apiv2beta1.Predicate_IS_SUBSTRING, apiv1beta1.Predicate_IS_SUBSTRING:
		return "IS_SUBSTRING", nil
	default:
		return "", util.NewUnknownApiVersionError("Filter.Predicate.Operation", o)
	}
}

func toValue(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case *apiv2beta1.Predicate_IntValue:
		return v.IntValue, nil
	case *apiv2beta1.Predicate_LongValue:
		return v.LongValue, nil
	case *apiv2beta1.Predicate_StringValue:
		return v.StringValue, nil
	case *apiv2beta1.Predicate_TimestampValue:
		return v.TimestampValue.AsTime().Unix(), nil
	case *apiv2beta1.Predicate_IntValues_:
		return v.IntValues.GetValues(), nil
	case *apiv2beta1.Predicate_StringValues_:
		return v.StringValues.GetValues(), nil
	case *apiv2beta1.Predicate_LongValues_:
		return v.LongValues.GetValues(), nil

	case *apiv1beta1.Predicate_IntValue:
		return v.IntValue, nil
	case *apiv1beta1.Predicate_LongValue:
		return v.LongValue, nil
	case *apiv1beta1.Predicate_StringValue:
		return v.StringValue, nil
	case *apiv1beta1.Predicate_TimestampValue:
		return v.TimestampValue.AsTime().Unix(), nil
	case *apiv1beta1.Predicate_IntValues:
		return v.IntValues.GetValues(), nil
	case *apiv1beta1.Predicate_StringValues:
		return v.StringValues.GetValues(), nil
	case *apiv1beta1.Predicate_LongValues:
		return v.LongValues.GetValues(), nil

	default:
		return nil, util.NewUnknownApiVersionError("Filter.Predicate.Value", v)
	}
}
