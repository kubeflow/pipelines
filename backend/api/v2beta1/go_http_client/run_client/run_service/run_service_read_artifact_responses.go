// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/run_model"
)

// RunServiceReadArtifactReader is a Reader for the RunServiceReadArtifact structure.
type RunServiceReadArtifactReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RunServiceReadArtifactReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRunServiceReadArtifactOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewRunServiceReadArtifactDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRunServiceReadArtifactOK creates a RunServiceReadArtifactOK with default headers values
func NewRunServiceReadArtifactOK() *RunServiceReadArtifactOK {
	return &RunServiceReadArtifactOK{}
}

/*
RunServiceReadArtifactOK describes a response with status code 200, with default header values.

A successful response.
*/
type RunServiceReadArtifactOK struct {
	Payload *run_model.V2beta1ReadArtifactResponse
}

// IsSuccess returns true when this run service read artifact o k response has a 2xx status code
func (o *RunServiceReadArtifactOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this run service read artifact o k response has a 3xx status code
func (o *RunServiceReadArtifactOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this run service read artifact o k response has a 4xx status code
func (o *RunServiceReadArtifactOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this run service read artifact o k response has a 5xx status code
func (o *RunServiceReadArtifactOK) IsServerError() bool {
	return false
}

// IsCode returns true when this run service read artifact o k response a status code equal to that given
func (o *RunServiceReadArtifactOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the run service read artifact o k response
func (o *RunServiceReadArtifactOK) Code() int {
	return 200
}

func (o *RunServiceReadArtifactOK) Error() string {
	return fmt.Sprintf("[GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read][%d] runServiceReadArtifactOK  %+v", 200, o.Payload)
}

func (o *RunServiceReadArtifactOK) String() string {
	return fmt.Sprintf("[GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read][%d] runServiceReadArtifactOK  %+v", 200, o.Payload)
}

func (o *RunServiceReadArtifactOK) GetPayload() *run_model.V2beta1ReadArtifactResponse {
	return o.Payload
}

func (o *RunServiceReadArtifactOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.V2beta1ReadArtifactResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewRunServiceReadArtifactDefault creates a RunServiceReadArtifactDefault with default headers values
func NewRunServiceReadArtifactDefault(code int) *RunServiceReadArtifactDefault {
	return &RunServiceReadArtifactDefault{
		_statusCode: code,
	}
}

/*
RunServiceReadArtifactDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type RunServiceReadArtifactDefault struct {
	_statusCode int

	Payload *run_model.RuntimeError
}

// IsSuccess returns true when this run service read artifact default response has a 2xx status code
func (o *RunServiceReadArtifactDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this run service read artifact default response has a 3xx status code
func (o *RunServiceReadArtifactDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this run service read artifact default response has a 4xx status code
func (o *RunServiceReadArtifactDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this run service read artifact default response has a 5xx status code
func (o *RunServiceReadArtifactDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this run service read artifact default response a status code equal to that given
func (o *RunServiceReadArtifactDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the run service read artifact default response
func (o *RunServiceReadArtifactDefault) Code() int {
	return o._statusCode
}

func (o *RunServiceReadArtifactDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read][%d] RunService_ReadArtifact default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceReadArtifactDefault) String() string {
	return fmt.Sprintf("[GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/artifacts/{artifact_name}:read][%d] RunService_ReadArtifact default  %+v", o._statusCode, o.Payload)
}

func (o *RunServiceReadArtifactDefault) GetPayload() *run_model.RuntimeError {
	return o.Payload
}

func (o *RunServiceReadArtifactDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.RuntimeError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
