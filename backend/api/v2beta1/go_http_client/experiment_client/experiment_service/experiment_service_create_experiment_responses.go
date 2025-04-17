// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	experiment_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/experiment_model"
)

// ExperimentServiceCreateExperimentReader is a Reader for the ExperimentServiceCreateExperiment structure.
type ExperimentServiceCreateExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ExperimentServiceCreateExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewExperimentServiceCreateExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewExperimentServiceCreateExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewExperimentServiceCreateExperimentOK creates a ExperimentServiceCreateExperimentOK with default headers values
func NewExperimentServiceCreateExperimentOK() *ExperimentServiceCreateExperimentOK {
	return &ExperimentServiceCreateExperimentOK{}
}

/*
ExperimentServiceCreateExperimentOK handles this case with default header values.

A successful response.
*/
type ExperimentServiceCreateExperimentOK struct {
	Payload *experiment_model.V2beta1Experiment
}

func (o *ExperimentServiceCreateExperimentOK) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/experiments][%d] experimentServiceCreateExperimentOK  %+v", 200, o.Payload)
}

func (o *ExperimentServiceCreateExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.V2beta1Experiment)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewExperimentServiceCreateExperimentDefault creates a ExperimentServiceCreateExperimentDefault with default headers values
func NewExperimentServiceCreateExperimentDefault(code int) *ExperimentServiceCreateExperimentDefault {
	return &ExperimentServiceCreateExperimentDefault{
		_statusCode: code,
	}
}

/*
ExperimentServiceCreateExperimentDefault handles this case with default header values.

An unexpected error response.
*/
type ExperimentServiceCreateExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.RuntimeError
}

// Code gets the status code for the experiment service create experiment default response
func (o *ExperimentServiceCreateExperimentDefault) Code() int {
	return o._statusCode
}

func (o *ExperimentServiceCreateExperimentDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/experiments][%d] ExperimentService_CreateExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *ExperimentServiceCreateExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.RuntimeError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
