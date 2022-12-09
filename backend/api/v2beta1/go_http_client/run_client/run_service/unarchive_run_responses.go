// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	run_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

// UnarchiveRunReader is a Reader for the UnarchiveRun structure.
type UnarchiveRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UnarchiveRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewUnarchiveRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewUnarchiveRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUnarchiveRunOK creates a UnarchiveRunOK with default headers values
func NewUnarchiveRunOK() *UnarchiveRunOK {
	return &UnarchiveRunOK{}
}

/*UnarchiveRunOK handles this case with default header values.

A successful response.
*/
type UnarchiveRunOK struct {
	Payload interface{}
}

func (o *UnarchiveRunOK) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:unarchive][%d] unarchiveRunOK  %+v", 200, o.Payload)
}

func (o *UnarchiveRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUnarchiveRunDefault creates a UnarchiveRunDefault with default headers values
func NewUnarchiveRunDefault(code int) *UnarchiveRunDefault {
	return &UnarchiveRunDefault{
		_statusCode: code,
	}
}

/*UnarchiveRunDefault handles this case with default header values.

UnarchiveRunDefault unarchive run default
*/
type UnarchiveRunDefault struct {
	_statusCode int

	Payload *run_model.RPCStatus
}

// Code gets the status code for the unarchive run default response
func (o *UnarchiveRunDefault) Code() int {
	return o._statusCode
}

func (o *UnarchiveRunDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/experiments/{experiment_id}/runs/{run_id}:unarchive][%d] UnarchiveRun default  %+v", o._statusCode, o.Payload)
}

func (o *UnarchiveRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.RPCStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
