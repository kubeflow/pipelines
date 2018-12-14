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

// Package filter contains types and methods for parsing and applying filters to
// resources being queried by a ListXXX request.
package filter

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/golang/protobuf/ptypes"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
)

// Filter represents a filter that can be applied when querying an arbitrary API
// resource.
type Filter struct {
	filterProto *api.Filter

	eq  map[string]interface{}
	neq map[string]interface{}
	gt  map[string]interface{}
	gte map[string]interface{}
	lt  map[string]interface{}
	lte map[string]interface{}

	in map[string]interface{}
}

// New creates a new Filter from parsing the API filter protocol buffer.
func New(filterProto *api.Filter) (*Filter, error) {
	f := &Filter{
		filterProto: filterProto,
		eq:          make(map[string]interface{}),
		neq:         make(map[string]interface{}),
		gt:          make(map[string]interface{}),
		gte:         make(map[string]interface{}),
		lt:          make(map[string]interface{}),
		lte:         make(map[string]interface{}),
		in:          make(map[string]interface{}),
	}

	if err := f.parseFilterProto(); err != nil {
		return nil, err
	}
	return f, nil
}

// NewWithKeyMap is like New, but takes an additional map for mapping key names
// in the protocol buffer to an appropriate name for use when querying the
// model. For example, if the API name of a field is "foo" and the equivalent
// model name is "ModelFoo", then filterProto with predicates against key "foo"
// will be parsed as if the key value was "ModelFoo".
func NewWithKeyMap(filterProto *api.Filter, keyMap map[string]string) (*Filter, error) {
	for _, pred := range filterProto.Predicates {
		k, ok := keyMap[pred.Key]
		if !ok {
			return nil, fmt.Errorf("no support for filtering on unrecognized field %q", pred.Key)
		}
		pred.Key = k
	}
	return New(filterProto)
}

// AddToSelect builds a WHERE clause from the Filter f, adds it to the supplied
// SelectBuilder object and returns it for use in SQL queries.
func (f *Filter) AddToSelect(sb squirrel.SelectBuilder) squirrel.SelectBuilder {
	if len(f.eq) > 0 {
		sb = sb.Where(squirrel.Eq(f.eq))
	}

	if len(f.neq) > 0 {
		sb = sb.Where(squirrel.NotEq(f.neq))
	}

	if len(f.gt) > 0 {
		sb = sb.Where(squirrel.Gt(f.gt))
	}

	if len(f.gte) > 0 {
		sb = sb.Where(squirrel.GtOrEq(f.gte))
	}

	if len(f.lt) > 0 {
		sb = sb.Where(squirrel.Lt(f.lt))
	}

	if len(f.lte) > 0 {
		sb = sb.Where(squirrel.LtOrEq(f.lte))
	}

	// In
	if len(f.in) > 0 {
		sb = sb.Where(squirrel.Eq(f.in))
	}

	return sb
}

func checkPredicate(p *api.Predicate) error {
	switch p.Op {
	case api.Predicate_IN:
		switch t := p.Value.(type) {
		case *api.Predicate_IntValue, *api.Predicate_LongValue, *api.Predicate_StringValue, *api.Predicate_TimestampValue:
			return fmt.Errorf("cannot use IN operator with scalar type %T", t)
		}

	case api.Predicate_EQUALS, api.Predicate_NOT_EQUALS, api.Predicate_GREATER_THAN, api.Predicate_GREATER_THAN_EQUALS, api.Predicate_LESS_THAN, api.Predicate_LESS_THAN_EQUALS:
		switch t := p.Value.(type) {
		case *api.Predicate_IntValues, *api.Predicate_LongValues, *api.Predicate_StringValues:
			return fmt.Errorf("cannot use scalar operator %v on array type %T", p.Op, t)
		}

	default:
		return fmt.Errorf("invalid predicate operation: %v", p.Op)
	}

	return nil
}

func (f *Filter) parseFilterProto() error {
	for _, pred := range f.filterProto.Predicates {
		if err := checkPredicate(pred); err != nil {
			return err
		}

		var m map[string]interface{}
		switch pred.Op {
		case api.Predicate_EQUALS:
			m = f.eq
		case api.Predicate_NOT_EQUALS:
			m = f.neq
		case api.Predicate_GREATER_THAN:
			m = f.gt
		case api.Predicate_GREATER_THAN_EQUALS:
			m = f.gte
		case api.Predicate_LESS_THAN:
			m = f.lt
		case api.Predicate_LESS_THAN_EQUALS:
			m = f.lte
		case api.Predicate_IN:
			m = f.in
		default:
			return fmt.Errorf("invalid predicate operation: %v", pred.Op)
		}

		if err := addPredicateValue(m, pred); err != nil {
			return err
		}
	}

	return nil
}

func addPredicateValue(m map[string]interface{}, p *api.Predicate) error {
	switch t := p.Value.(type) {
	case *api.Predicate_IntValue:
		m[p.Key] = p.GetIntValue()
	case *api.Predicate_LongValue:
		m[p.Key] = p.GetLongValue()
	case *api.Predicate_StringValue:
		m[p.Key] = p.GetStringValue()
	case *api.Predicate_TimestampValue:
		ts, err := ptypes.Timestamp(p.GetTimestampValue())
		if err != nil {
			return fmt.Errorf("invalid timestamp: %v", err)
		}
		m[p.Key] = ts.Unix()

	case *api.Predicate_IntValues:
		var v []int32
		for _, i := range p.GetIntValues().GetValues() {
			v = append(v, i)
		}
		m[p.Key] = v

	case *api.Predicate_LongValues:
		var v []int64
		for _, i := range p.GetLongValues().GetValues() {
			v = append(v, i)
		}
		m[p.Key] = v

	case *api.Predicate_StringValues:
		var v []string
		for _, i := range p.GetStringValues().GetValues() {
			v = append(v, i)
		}
		m[p.Key] = v

	case nil:
		return fmt.Errorf("no value set for predicate on key %q", p.Key)

	default:
		return fmt.Errorf("unknown value type in Filter for predicate key %q: %T", p.Key, t)
	}

	return nil
}
