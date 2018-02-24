package util

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/kataras/iris"
)

type InternalError struct {
	Message string
}

func NewInternalError(format string, a ...interface{}) *InternalError {
	return &InternalError{Message: fmt.Sprintf(format, a...)}
}

func (e *InternalError) Error() string {
	return e.Message
}

func HandleError(action string, ctx iris.Context, err error) {
	glog.Errorf("%v failed. Error: %v", action, err.Error())

	switch err.(type) {
	case *InternalError:
		ctx.StatusCode(http.StatusInternalServerError)
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
