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

	"github.com/go-openapi/runtime"
	"github.com/golang/glog"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	k8metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CustomCode uint32

const (
	CUSTOM_CODE_TRANSIENT CustomCode = 0
	CUSTOM_CODE_PERMANENT CustomCode = 1
	CUSTOM_CODE_NOT_FOUND CustomCode = 2
	CUSTOM_CODE_GENERIC   CustomCode = 3
)

type APICode int

const (
	API_CODE_NOT_FOUND = 404
)

type CustomError struct {
	error error
	code  CustomCode
}

func NewCustomError(err error, code CustomCode, format string, a ...interface{}) *CustomError {
	message := fmt.Sprintf(format, a...)
	return &CustomError{
		error: errors.Wrapf(err, fmt.Sprintf("CustomError (code: %v): %v", code, message)),
		code:  code,
	}
}

func NewCustomErrorf(code CustomCode, format string, a ...interface{}) *CustomError {
	message := fmt.Sprintf(format, a...)
	return &CustomError{
		error: errors.Errorf("CustomError (code: %v): %v", code, message),
		code:  code,
	}
}

func (e *CustomError) Error() string {
	return e.error.Error()
}

func HasCustomCode(err error, code CustomCode) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case *CustomError:
		return err.(*CustomError).code == code
	default:
		return false
	}
}

type UserError struct {
	// Error for internal debugging.
	internalError error
	// Error message for the external client.
	externalMessage string
	// Status code for the external client.
	externalStatusCode codes.Code
}

func newUserError(internalError error, externalMessage string,
	externalStatusCode codes.Code) *UserError {
	return &UserError{
		internalError:      internalError,
		externalMessage:    externalMessage,
		externalStatusCode: externalStatusCode,
	}
}

func NewUserErrorWithSingleMessage(err error, message string) *UserError {
	return NewUserError(err, message, message)
}

func NewUserError(err error, internalMessage string, externalMessage string) *UserError {
	// Note apiError.Response is of type github.com/go-openapi/runtime/client
	if apiError, ok := err.(*runtime.APIError); ok {
		if apiError.Code == API_CODE_NOT_FOUND {
			return newUserError(
				errors.Wrapf(err, internalMessage),
				fmt.Sprintf("%v: %v", externalMessage, "Resource not found"),
				codes.Code(apiError.Code))
		} else {
			return newUserError(
				errors.Wrapf(err, internalMessage),
				fmt.Sprintf("%v. Raw error from the service: %v", externalMessage, err.Error()),
				codes.Code(apiError.Code))
		}
	}

	return newUserError(
		errors.Wrapf(err, internalMessage),
		fmt.Sprintf("%v. Raw error from the service: %v", externalMessage, err.Error()),
		codes.Internal)
}

func ExtractErrorForCLI(err error, isDebugMode bool) error {
	if userError, ok := err.(*UserError); ok {
		if isDebugMode {
			return fmt.Errorf("%+v", userError.internalError)
		} else {
			return fmt.Errorf("%v", userError.externalMessage)
		}
	} else {
		return err
	}
}

func NewInternalServerError(err error, internalMessageFormat string,
	a ...interface{}) *UserError {
	internalMessage := fmt.Sprintf(internalMessageFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("InternalServerError: %v", internalMessage)),
		"Internal Server Error",
		codes.Internal)
}

func NewNotFoundError(err error, externalMessageFormat string,
	a ...interface{}) *UserError {
	externalMessage := fmt.Sprintf(externalMessageFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("NotFoundError: %v", externalMessage)),
		externalMessage,
		codes.NotFound)
}

func NewResourceNotFoundError(resourceType string, resourceName string) *UserError {
	externalMessage := fmt.Sprintf("%s %s not found.", resourceType, resourceName)
	return newUserError(
		errors.New(fmt.Sprintf("ResourceNotFoundError: %v", externalMessage)),
		externalMessage,
		codes.NotFound)
}

func NewResourcesNotFoundError(resourceTypesFormat string, resourceNames ...interface{}) *UserError {
	externalMessage := fmt.Sprintf("%s not found.", fmt.Sprintf(resourceTypesFormat, resourceNames...))
	return newUserError(
		errors.New(fmt.Sprintf("ResourceNotFoundError: %v", externalMessage)),
		externalMessage,
		codes.NotFound)
}

func NewInvalidInputError(messageFormat string, a ...interface{}) *UserError {
	message := fmt.Sprintf(messageFormat, a...)
	return newUserError(errors.Errorf("Invalid input error: %v", message), message, codes.InvalidArgument)
}

func NewInvalidInputErrorWithDetails(err error, externalMessage string) *UserError {
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("InvalidInputError: %v", externalMessage)),
		externalMessage,
		codes.InvalidArgument)
}

func NewAlreadyExistError(messageFormat string, a ...interface{}) *UserError {
	message := fmt.Sprintf(messageFormat, a...)
	return newUserError(errors.Errorf("Already exist error: %v", message), message, codes.AlreadyExists)
}

func NewBadRequestError(err error, externalFormat string, a ...interface{}) *UserError {
	externalMessage := fmt.Sprintf(externalFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("BadRequestError: %v", externalMessage)),
		externalMessage,
		codes.Aborted)
}

func NewUnauthenticatedError(err error, externalFormat string, a ...interface{}) *UserError {
	externalMessage := fmt.Sprintf(externalFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("Unauthenticated: %v", externalMessage)),
		externalMessage,
		codes.Unauthenticated)
}

func NewPermissionDeniedError(err error, externalFormat string, a ...interface{}) *UserError {
	externalMessage := fmt.Sprintf(externalFormat, a...)
	return newUserError(
		errors.Wrapf(err, fmt.Sprintf("PermissionDenied: %v", externalMessage)),
		externalMessage,
		codes.PermissionDenied)
}

func (e *UserError) ExternalMessage() string {
	return e.externalMessage
}

func (e *UserError) ExternalStatusCode() codes.Code {
	return e.externalStatusCode
}

func (e *UserError) Error() string {
	return e.internalError.Error()
}

func (e *UserError) Cause() error {
	return e.internalError
}

func (e *UserError) String() string {
	return fmt.Sprintf("%v (code: %v): %+v", e.externalMessage, e.externalStatusCode,
		e.internalError)
}

func (e *UserError) wrapf(format string, args ...interface{}) *UserError {
	return newUserError(errors.Wrapf(e.internalError, format, args...),
		e.externalMessage, e.externalStatusCode)
}

func (e *UserError) wrap(message string) *UserError {
	return newUserError(errors.Wrap(e.internalError, message),
		e.externalMessage, e.externalStatusCode)
}

func (e *UserError) Log() {
	switch e.externalStatusCode {
	case codes.Aborted, codes.InvalidArgument, codes.NotFound, codes.Internal:
		glog.Infof("%+v", e.internalError)
	default:
		glog.Errorf("%+v", e.internalError)
	}
}

func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *UserError:
		return err.(*UserError).wrapf(format, args...)
	default:
		return errors.Wrapf(err, format, args...)
	}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case *UserError:
		return err.(*UserError).wrap(message)
	default:
		return errors.Wrapf(err, message)
	}
}

func LogError(err error) {
	switch err.(type) {
	case *UserError:
		err.(*UserError).Log()
	default:
		// We log all the details.
		glog.Errorf("InternalError: %+v", err)
	}
}

func ToGRPCError(err error) error {
	switch err.(type) {
	case *UserError:
		userError := err.(*UserError)
		stat := status.New(userError.externalStatusCode, userError.internalError.Error())
		statWithDetail, statErr := stat.
			WithDetails(&api.Error{
				ErrorMessage: userError.externalMessage,
				ErrorDetails: userError.internalError.Error(),
			})

		if statErr != nil {
			// Failed to stream error message as proto.
			glog.Errorf("Failed to stream gRpc error. Error to be streamed: %v Error: %v",
				userError.String(), statErr)
			return stat.Err()
		}
		return statWithDetail.Err()
	default:
		externalMessage := fmt.Sprintf("Internal error: %+v", err)
		stat := status.New(codes.Internal, externalMessage)
		statWithDetail, statErr := stat.
			WithDetails(&api.Error{ErrorMessage: externalMessage, ErrorDetails: externalMessage})
		if statErr != nil {
			// Failed to stream error message as proto.
			glog.Errorf("Failed to stream gRpc error. Error to be streamed: %v Error: %v",
				externalMessage, statErr)
			return stat.Err()
		}
		return statWithDetail.Err()
	}
}

// TerminateIfError Check if error is nil. Terminate if not.
func TerminateIfError(err error) {
	if err != nil {
		glog.Fatalf("%v", err)
	}
}

// IsNotFound returns whether an error indicates that a resource was "not found".
func IsNotFound(err error) bool {
	return reasonForError(err) == k8metav1.StatusReasonNotFound
}

// IsUserErrorCodeMatch returns whether the error is a user error with specified code.
func IsUserErrorCodeMatch(err error, code codes.Code) bool {
	userError, ok := err.(*UserError)
	return ok && userError.externalStatusCode == code
}

// ReasonForError returns the HTTP status for a particular error.
func reasonForError(err error) k8metav1.StatusReason {
	switch t := err.(type) {
	case k8errors.APIStatus:
		return t.Status().Reason
	case *k8errors.StatusError:
		return t.Status().Reason
	}
	return k8metav1.StatusReasonUnknown
}
