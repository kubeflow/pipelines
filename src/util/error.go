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

package util

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

type InternalError struct {
	// Error message returned to client
	Message string
	// The error details for logging only
	ErrorDetail string
}

func NewInternalError(message string, errorDetailFormat string, a ...interface{}) *InternalError {
	return &InternalError{Message: message, ErrorDetail: fmt.Sprintf(errorDetailFormat, a...)}
}

func (e *InternalError) Error() string {
	return fmt.Sprintf("%s. Error: <%s>", e.Message, e.ErrorDetail)
}

type ResourceNotFoundError struct {
	ResourceType string
	ResourceName string
}

func NewResourceNotFoundError(resourceType string, resourceName string) *ResourceNotFoundError {
	return &ResourceNotFoundError{ResourceType: resourceType, ResourceName: resourceName}
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found.", e.ResourceType, e.ResourceName)
}

type InvalidInputError struct {
	// Error message returned to client
	Message string
	// The error details for logging only
	ErrorDetail string
}

func NewInvalidInputError(message string, errorDetailFormat string, a ...interface{}) *InvalidInputError {
	return &InvalidInputError{Message: fmt.Sprintf(message, a...), ErrorDetail: fmt.Sprintf(errorDetailFormat, a...)}
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid input: %v. Error: <%s>", e.Message, e.ErrorDetail)
}

type BadRequestError struct {
	// Error message returned to client
	Message string
}

func NewBadRequestError(message string, a ...interface{}) *BadRequestError {
	return &BadRequestError{Message: fmt.Sprintf(message, a...)}
}

func (e *BadRequestError) Error() string {
	return fmt.Sprintf("Bad request. Error: <%s>", e.Message)
}

// TODO(yangpa): Consider add a flag so that when a flag is true, user can see the internal message
func HandleError(action string, ctx iris.Context, err error) {
	switch err.(type) {
	case *InternalError:
		glog.Errorf("%v failed. Error: %v", action, err.Error())
		ctx.StatusCode(http.StatusInternalServerError)
		e, _ := err.(*InternalError)
		ctx.WriteString(e.Message)
	case *InvalidInputError:
		glog.Infof("%v failed. Error: %v", action, err.Error())
		ctx.StatusCode(http.StatusBadRequest)
		e, _ := err.(*InvalidInputError)
		ctx.WriteString(e.Message)
	case *ResourceNotFoundError:
		glog.Infof("%v failed. Error: %v", action, err.Error())
		ctx.StatusCode(http.StatusBadRequest)
		ctx.WriteString(err.Error())
	case *BadRequestError:
		glog.Infof("%v failed. Error: %v", action, err.Error())
		ctx.StatusCode(http.StatusBadRequest)
		ctx.WriteString(err.Error())
	default:
		glog.Infof("%v failed. Error: %v", action, err.Error())
		ctx.StatusCode(http.StatusBadRequest)
	}
}

// TerminateIfError Check if error is nil. Terminate if not.
func TerminateIfError(err error) {
	if err != nil {
		glog.Fatalf("%v", err)
	}
}
