// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	pipeline_model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_model"
)

// PipelineServiceCreatePipelineV1Reader is a Reader for the PipelineServiceCreatePipelineV1 structure.
type PipelineServiceCreatePipelineV1Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PipelineServiceCreatePipelineV1Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewPipelineServiceCreatePipelineV1OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewPipelineServiceCreatePipelineV1Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPipelineServiceCreatePipelineV1OK creates a PipelineServiceCreatePipelineV1OK with default headers values
func NewPipelineServiceCreatePipelineV1OK() *PipelineServiceCreatePipelineV1OK {
	return &PipelineServiceCreatePipelineV1OK{}
}

/*
PipelineServiceCreatePipelineV1OK handles this case with default header values.

A successful response.
*/
type PipelineServiceCreatePipelineV1OK struct {
	Payload *pipeline_model.APIPipeline
}

func (o *PipelineServiceCreatePipelineV1OK) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/pipelines][%d] pipelineServiceCreatePipelineV1OK  %+v", 200, o.Payload)
}

func (o *PipelineServiceCreatePipelineV1OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIPipeline)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPipelineServiceCreatePipelineV1Default creates a PipelineServiceCreatePipelineV1Default with default headers values
func NewPipelineServiceCreatePipelineV1Default(code int) *PipelineServiceCreatePipelineV1Default {
	return &PipelineServiceCreatePipelineV1Default{
		_statusCode: code,
	}
}

/*
PipelineServiceCreatePipelineV1Default handles this case with default header values.

An unexpected error response.
*/
type PipelineServiceCreatePipelineV1Default struct {
	_statusCode int

	Payload *pipeline_model.GatewayruntimeError
}

// Code gets the status code for the pipeline service create pipeline v1 default response
func (o *PipelineServiceCreatePipelineV1Default) Code() int {
	return o._statusCode
}

func (o *PipelineServiceCreatePipelineV1Default) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/pipelines][%d] PipelineService_CreatePipelineV1 default  %+v", o._statusCode, o.Payload)
}

func (o *PipelineServiceCreatePipelineV1Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.GatewayruntimeError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
