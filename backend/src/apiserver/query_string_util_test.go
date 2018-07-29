package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var fakeModelFieldsBySortableAPIFields = map[string]string{
	"":           "Empty",
	"id":         "UUID",
	"created_at": "CreatedAtInSec",
}

func TestParseSortByQueryString_EmptyString(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "Empty", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_FieldNameOnly(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("id", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "UUID", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_FieldNameWithDescFlag(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("id desc", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "UUID", modelField)
	assert.True(t, isDesc)
}

func TestParseSortByQueryString_FieldNameWithAscFlag(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("id asc", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "UUID", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_NotSortableFieldName(t *testing.T) {
	_, _, err := parseSortByQueryString("foobar", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
}

func TestParseSortByQueryString_IncorrectDescFlag(t *testing.T) {
	_, _, err := parseSortByQueryString("id foobar", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
}

func TestParseSortByQueryString_StringTooLong(t *testing.T) {
	_, _, err := parseSortByQueryString("id desc foo", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
}
