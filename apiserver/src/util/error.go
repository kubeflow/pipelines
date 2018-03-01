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
	Resource     string
}

func NewResourceNotFoundError(resourceType string, resourceName string) *ResourceNotFoundError {
	return &ResourceNotFoundError{ResourceType: resourceType, Resource: resourceName}
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found.", e.ResourceType, e.Resource)
}

type InvalidInputError struct {
	Message string
}

func NewInvalidInputError(format string, a ...interface{}) *InvalidInputError {
	return &InvalidInputError{Message: fmt.Sprintf(format, a...)}
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("Invalid input. %v", e.Message)
}

func HandleError(action string, ctx iris.Context, err error) {
	glog.Errorf("%v failed. Error: %v", action, err.Error())
	switch err.(type) {
	case *InternalError:
		ctx.StatusCode(http.StatusInternalServerError)
		e, _ := err.(*InternalError)
		ctx.WriteString(e.Message)
	case *InvalidInputError, *ResourceNotFoundError:
		ctx.StatusCode(http.StatusBadRequest)
		ctx.WriteString(err.Error())
	default:
		ctx.StatusCode(http.StatusBadRequest)
	}
}

// TerminateIfError Check if error is nil. Terminate if not.
func TerminateIfError(err error) {
	if err != nil {
		glog.Fatalf("%v", err)
	}
}
