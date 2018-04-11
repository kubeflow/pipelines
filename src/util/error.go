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
	"github.com/pkg/errors"
)

type UserError struct {
	// Error for internal debugging.
	internalError error
	// Error message for the external client.
	externalMessage string
	// Status code for the external client.
	externalStatusCode int
}

func newUserError(internalError error, externalMessage string,
		externalStatusCode int) *UserError {
	return &UserError{
		internalError: internalError,
		externalMessage: externalMessage,
		externalStatusCode: externalStatusCode,
	}
}

func NewInternalServerError(err error, internalMessageFormat string,
	a ...interface{}) *UserError {
	internalMessage := fmt.Sprintf(internalMessageFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("InternalServerError: %v", internalMessage)),
		"Internal Server Error",
		http.StatusInternalServerError)
}

func NewResourceNotFoundError(resourceType string, resourceName string) *UserError {
	externalMessage := fmt.Sprintf("%s %s not found.", resourceType, resourceName)
	return newUserError(
		errors.New(fmt.Sprintf("ResourceNotFoundError: %v", externalMessage)),
		externalMessage,
		http.StatusNotFound)
}

func NewInvalidInputError(err error, externalMessage string, internalMessage string) *UserError {
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("InvalidInputError: %v: %v", externalMessage,
			internalMessage)),
		externalMessage,
		http.StatusBadRequest)
}

func NewBadRequestError(err error, externalFormat string, a ...interface{}) *UserError {
	externalMessage := fmt.Sprintf(externalFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("BadRequestError: %v", externalMessage)),
		externalMessage,
		http.StatusBadRequest)
}

func (e *UserError) ExternalMessage() string {
	return e.externalMessage
}

func (e *UserError) ExternalStatusCode() int {
	return e.externalStatusCode
}

func (e *UserError) Error() string {
	return e.internalError.Error()
}

func (e *UserError) String() string {
	return fmt.Sprintf("%v (code: %v): %+v", e.externalMessage, e.externalStatusCode,
		e.internalError)
}

func (e *UserError) wrapf(format string, args ...interface{}) *UserError {
	return newUserError(errors.Wrapf(e.internalError, format, args),
		e.externalMessage, e.externalStatusCode)
}

func (e *UserError) wrap(message string) *UserError {
	return newUserError(errors.Wrap(e.internalError, message),
		e.externalMessage, e.externalStatusCode)
}

func (e *UserError) PopulateContextAndLog(ctx iris.Context) {
	switch e.externalStatusCode {
	case http.StatusBadRequest:
		glog.Infof("%+v", e)
		ctx.StatusCode(e.externalStatusCode)
		ctx.WriteString(e.externalMessage)
	case http.StatusNotFound:
		glog.Infof("%+v", e)
		ctx.StatusCode(e.externalStatusCode)
		ctx.WriteString(e.externalMessage)
	default:
		// By default, we return an internal error since we did not handle this case.
		// We log all the details: both internal and external error.
		glog.Errorf("%+v", e)
		ctx.StatusCode(e.externalStatusCode)
		// Since this is OSS, we return the details of the internal error (instead of e.externalMessage)
		ctx.WriteString(fmt.Sprintf("%+v", e))
	}
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *UserError:
		return err.(*UserError).wrapf(format, args)
	default:
		return errors.Wrapf(err, format, args)
	}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *UserError:
		return err.(*UserError).wrapf(message)
	default:
		return errors.Wrapf(err, message)
	}
}

func PopulateContextAndLogError(ctx iris.Context, err error) {
	switch err.(type) {
	case *UserError:
		err.(*UserError).PopulateContextAndLog(ctx)
	default:
		// We log all the details.
		glog.Errorf("InternalError: %+v", err)
		ctx.StatusCode(http.StatusInternalServerError)
		// Since this is OSS, we return the details of the internal error.
		ctx.WriteString(fmt.Sprintf("Internal error: %+v", err))
	}
}

// TerminateIfError Check if error is nil. Terminate if not.
func TerminateIfError(err error) {
	if err != nil {
		glog.Fatalf("%v", err)
	}
}
