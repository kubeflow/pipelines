package filter

import (
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/golang/protobuf/ptypes"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
)

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

// New creates ...
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

// AddToSelect blah blah
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
		m[p.Key] = ts

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
