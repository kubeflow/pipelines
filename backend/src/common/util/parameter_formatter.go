// Copyright 2018 The Kubeflow Authors
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

package util

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/lestrrat-go/strftime"
)

const (
	runUUIDExpression       = "[[RunUUID]]"
	scheduledTimeExpression = "[[ScheduledTime]]"
	currentTimeExpression   = "[[CurrentTime]]"
	IndexExpression         = "[[Index]]"
	scheduledTimePrefix     = "[[ScheduledTime."
	currentTimePrefix       = "[[CurrentTime."
	defaultTimeFormat       = "20060102150405"
	suffix                  = "]]"

	scheduledTimePrefix2 = "{{$.scheduledTime.strftime('"
	currentTimePrefix2   = "{{$.currentTime.strftime('"
	suffix2              = "')}}"
)

const (
	disabledField = -1
)

// ParameterFormatter is an object that substitutes specific strings
// in workflow parameters by information about the workflow execution (time at
// which the workflow was started, time at which the workflow was scheduled, etc.)
type ParameterFormatter struct {
	runUUID        string
	scheduledEpoch int64
	nowEpoch       int64
	index          int64
}

// NewRunParameterFormatter returns a new ParameterFormatter to substitute run macros.
func NewRunParameterFormatter(runUUID string, runAt int64) *ParameterFormatter {
	return &ParameterFormatter{
		runUUID:        runUUID,
		nowEpoch:       runAt,
		scheduledEpoch: disabledField,
		index:          disabledField,
	}
}

// NewSWFParameterFormatter returns a new ParameterFormatter to substitute recurring run macros.
func NewSWFParameterFormatter(runUUID string, scheduledEpoch int64, nowEpoch int64,
	index int64) *ParameterFormatter {
	return &ParameterFormatter{
		runUUID:        runUUID,
		scheduledEpoch: scheduledEpoch,
		nowEpoch:       nowEpoch,
		index:          index,
	}
}

func (p *ParameterFormatter) FormatWorkflowParameters(
	parameters map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range parameters {
		formatted := p.Format(value)
		result[key] = formatted
	}
	return result
}

// Format substitutes special strings in the provided string.
func (p *ParameterFormatter) Format(s string) string {
	re := regexp.MustCompile(`\[\[(.*?)\]\]|\{\{\$\.(.*?)\}\}`)
	matches := re.FindAllString(s, -1)
	if matches == nil {
		return s
	}

	result := s

	for _, match := range matches {
		substitute := p.createSubstitutes(match)
		result = strings.Replace(result, match, substitute, 1)
	}

	return result
}

func (p *ParameterFormatter) createSubstitutes(match string) string {
	// First ensure that the corresponding field is valid, then attempt to substitute
	if len(p.runUUID) > 0 && strings.HasPrefix(match, runUUIDExpression) {
		return p.runUUID
	} else if p.scheduledEpoch != disabledField && strings.HasPrefix(match, scheduledTimeExpression) {
		return time.Unix(p.scheduledEpoch, 0).UTC().Format(defaultTimeFormat)
	} else if p.nowEpoch != disabledField && strings.HasPrefix(match, currentTimeExpression) {
		return time.Unix(p.nowEpoch, 0).UTC().Format(defaultTimeFormat)
	} else if p.index != disabledField && strings.HasPrefix(match, IndexExpression) {
		return fmt.Sprintf("%v", p.index)
	} else if p.scheduledEpoch != disabledField && strings.HasPrefix(match, scheduledTimePrefix) {
		match = strings.Replace(match, scheduledTimePrefix, "", 1)
		match = strings.Replace(match, suffix, "", 1)
		return time.Unix(p.scheduledEpoch, 0).UTC().Format(match)
	} else if p.nowEpoch != disabledField && strings.HasPrefix(match, currentTimePrefix) {
		match = strings.Replace(match, currentTimePrefix, "", 1)
		match = strings.Replace(match, suffix, "", 1)
		return time.Unix(p.nowEpoch, 0).UTC().Format(match)
	} else if p.scheduledEpoch != disabledField && strings.HasPrefix(match, scheduledTimePrefix2) {
		format := strings.Replace(match, scheduledTimePrefix2, "", 1)
		format = strings.Replace(format, suffix2, "", 1)
		formatter, err := strftime.New(format, strftime.WithUnixSeconds('s'))
		if err != nil {
			glog.Errorf("Could not create the strftime formatter from '%v'. Error: %v", format, err)
			return match
		}
		return formatter.FormatString(time.Unix(p.scheduledEpoch, 0).UTC())
	} else if p.nowEpoch != disabledField && strings.HasPrefix(match, currentTimePrefix2) {
		format := strings.Replace(match, currentTimePrefix2, "", 1)
		format = strings.Replace(format, suffix2, "", 1)
		formatter, err := strftime.New(format, strftime.WithUnixSeconds('s'))
		if err != nil {
			glog.Errorf("Could not create the strftime formatter from '%v'. Error: %v", format, err)
			return match
		}
		return formatter.FormatString(time.Unix(p.nowEpoch, 0).UTC())
	} else {
		return match
	}
}
