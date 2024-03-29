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

// PipelineServiceGetPipelineByNameV1Reader is a Reader for the PipelineServiceGetPipelineByNameV1 structure.
type PipelineServiceGetPipelineByNameV1Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PipelineServiceGetPipelineByNameV1Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewPipelineServiceGetPipelineByNameV1OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewPipelineServiceGetPipelineByNameV1Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewPipelineServiceGetPipelineByNameV1OK creates a PipelineServiceGetPipelineByNameV1OK with default headers values
func NewPipelineServiceGetPipelineByNameV1OK() *PipelineServiceGetPipelineByNameV1OK {
	return &PipelineServiceGetPipelineByNameV1OK{}
}

/*PipelineServiceGetPipelineByNameV1OK handles this case with default header values.

A successful response.
*/
type PipelineServiceGetPipelineByNameV1OK struct {
	Payload *pipeline_model.APIPipeline
}

func (o *PipelineServiceGetPipelineByNameV1OK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/namespaces/{namespace}/pipelines/{name}][%d] pipelineServiceGetPipelineByNameV1OK  %+v", 200, o.Payload)
}

func (o *PipelineServiceGetPipelineByNameV1OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIPipeline)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPipelineServiceGetPipelineByNameV1Default creates a PipelineServiceGetPipelineByNameV1Default with default headers values
func NewPipelineServiceGetPipelineByNameV1Default(code int) *PipelineServiceGetPipelineByNameV1Default {
	return &PipelineServiceGetPipelineByNameV1Default{
		_statusCode: code,
	}
}

/*PipelineServiceGetPipelineByNameV1Default handles this case with default header values.

An unexpected error response.
*/
type PipelineServiceGetPipelineByNameV1Default struct {
	_statusCode int

	Payload *pipeline_model.GatewayruntimeError
}

// Code gets the status code for the pipeline service get pipeline by name v1 default response
func (o *PipelineServiceGetPipelineByNameV1Default) Code() int {
	return o._statusCode
}

func (o *PipelineServiceGetPipelineByNameV1Default) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/namespaces/{namespace}/pipelines/{name}][%d] PipelineService_GetPipelineByNameV1 default  %+v", o._statusCode, o.Payload)
}

func (o *PipelineServiceGetPipelineByNameV1Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.GatewayruntimeError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
