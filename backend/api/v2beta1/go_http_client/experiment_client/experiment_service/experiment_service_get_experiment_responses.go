// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
)

// ExperimentServiceGetExperimentReader is a Reader for the ExperimentServiceGetExperiment structure.
type ExperimentServiceGetExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ExperimentServiceGetExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewExperimentServiceGetExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewExperimentServiceGetExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewExperimentServiceGetExperimentOK creates a ExperimentServiceGetExperimentOK with default headers values
func NewExperimentServiceGetExperimentOK() *ExperimentServiceGetExperimentOK {
	return &ExperimentServiceGetExperimentOK{}
}

/*
ExperimentServiceGetExperimentOK describes a response with status code 200, with default header values.

A successful response.
*/
type ExperimentServiceGetExperimentOK struct {
	Payload *experiment_model.V2beta1Experiment
}

// IsSuccess returns true when this experiment service get experiment o k response has a 2xx status code
func (o *ExperimentServiceGetExperimentOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this experiment service get experiment o k response has a 3xx status code
func (o *ExperimentServiceGetExperimentOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this experiment service get experiment o k response has a 4xx status code
func (o *ExperimentServiceGetExperimentOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this experiment service get experiment o k response has a 5xx status code
func (o *ExperimentServiceGetExperimentOK) IsServerError() bool {
	return false
}

// IsCode returns true when this experiment service get experiment o k response a status code equal to that given
func (o *ExperimentServiceGetExperimentOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the experiment service get experiment o k response
func (o *ExperimentServiceGetExperimentOK) Code() int {
	return 200
}

func (o *ExperimentServiceGetExperimentOK) Error() string {
	return fmt.Sprintf("[GET /apis/v2beta1/experiments/{experimentId}][%d] experimentServiceGetExperimentOK  %+v", 200, o.Payload)
}

func (o *ExperimentServiceGetExperimentOK) String() string {
	return fmt.Sprintf("[GET /apis/v2beta1/experiments/{experimentId}][%d] experimentServiceGetExperimentOK  %+v", 200, o.Payload)
}

func (o *ExperimentServiceGetExperimentOK) GetPayload() *experiment_model.V2beta1Experiment {
	return o.Payload
}

func (o *ExperimentServiceGetExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.V2beta1Experiment)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewExperimentServiceGetExperimentDefault creates a ExperimentServiceGetExperimentDefault with default headers values
func NewExperimentServiceGetExperimentDefault(code int) *ExperimentServiceGetExperimentDefault {
	return &ExperimentServiceGetExperimentDefault{
		_statusCode: code,
	}
}

/*
ExperimentServiceGetExperimentDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type ExperimentServiceGetExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.GooglerpcStatus
}

// IsSuccess returns true when this experiment service get experiment default response has a 2xx status code
func (o *ExperimentServiceGetExperimentDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this experiment service get experiment default response has a 3xx status code
func (o *ExperimentServiceGetExperimentDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this experiment service get experiment default response has a 4xx status code
func (o *ExperimentServiceGetExperimentDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this experiment service get experiment default response has a 5xx status code
func (o *ExperimentServiceGetExperimentDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this experiment service get experiment default response a status code equal to that given
func (o *ExperimentServiceGetExperimentDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the experiment service get experiment default response
func (o *ExperimentServiceGetExperimentDefault) Code() int {
	return o._statusCode
}

func (o *ExperimentServiceGetExperimentDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v2beta1/experiments/{experimentId}][%d] ExperimentService_GetExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *ExperimentServiceGetExperimentDefault) String() string {
	return fmt.Sprintf("[GET /apis/v2beta1/experiments/{experimentId}][%d] ExperimentService_GetExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *ExperimentServiceGetExperimentDefault) GetPayload() *experiment_model.GooglerpcStatus {
	return o.Payload
}

func (o *ExperimentServiceGetExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
