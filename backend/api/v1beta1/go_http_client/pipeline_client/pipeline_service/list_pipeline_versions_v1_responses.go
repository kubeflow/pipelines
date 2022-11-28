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

// ListPipelineVersionsV1Reader is a Reader for the ListPipelineVersionsV1 structure.
type ListPipelineVersionsV1Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListPipelineVersionsV1Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewListPipelineVersionsV1OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewListPipelineVersionsV1Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListPipelineVersionsV1OK creates a ListPipelineVersionsV1OK with default headers values
func NewListPipelineVersionsV1OK() *ListPipelineVersionsV1OK {
	return &ListPipelineVersionsV1OK{}
}

/*ListPipelineVersionsV1OK handles this case with default header values.

A successful response.
*/
type ListPipelineVersionsV1OK struct {
	Payload *pipeline_model.APIListPipelineVersionsResponse
}

func (o *ListPipelineVersionsV1OK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/pipeline_versions][%d] listPipelineVersionsV1OK  %+v", 200, o.Payload)
}

func (o *ListPipelineVersionsV1OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIListPipelineVersionsResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListPipelineVersionsV1Default creates a ListPipelineVersionsV1Default with default headers values
func NewListPipelineVersionsV1Default(code int) *ListPipelineVersionsV1Default {
	return &ListPipelineVersionsV1Default{
		_statusCode: code,
	}
}

/*ListPipelineVersionsV1Default handles this case with default header values.

ListPipelineVersionsV1Default list pipeline versions v1 default
*/
type ListPipelineVersionsV1Default struct {
	_statusCode int

	Payload *pipeline_model.APIStatus
}

// Code gets the status code for the list pipeline versions v1 default response
func (o *ListPipelineVersionsV1Default) Code() int {
	return o._statusCode
}

func (o *ListPipelineVersionsV1Default) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/pipeline_versions][%d] ListPipelineVersionsV1 default  %+v", o._statusCode, o.Payload)
}

func (o *ListPipelineVersionsV1Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
