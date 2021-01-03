// Code generated by go-swagger; DO NOT EDIT.

package pipeline_upload_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_upload_model"
)

// UploadPipelineVersionReader is a Reader for the UploadPipelineVersion structure.
type UploadPipelineVersionReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *UploadPipelineVersionReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewUploadPipelineVersionOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewUploadPipelineVersionDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewUploadPipelineVersionOK creates a UploadPipelineVersionOK with default headers values
func NewUploadPipelineVersionOK() *UploadPipelineVersionOK {
	return &UploadPipelineVersionOK{}
}

/*UploadPipelineVersionOK handles this case with default header values.

UploadPipelineVersionOK upload pipeline version o k
*/
type UploadPipelineVersionOK struct {
	Payload *pipeline_upload_model.APIPipelineVersion
}

func (o *UploadPipelineVersionOK) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/pipelines/upload_version][%d] uploadPipelineVersionOK  %+v", 200, o.Payload)
}

func (o *UploadPipelineVersionOK) GetPayload() *pipeline_upload_model.APIPipelineVersion {
	return o.Payload
}

func (o *UploadPipelineVersionOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_upload_model.APIPipelineVersion)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewUploadPipelineVersionDefault creates a UploadPipelineVersionDefault with default headers values
func NewUploadPipelineVersionDefault(code int) *UploadPipelineVersionDefault {
	return &UploadPipelineVersionDefault{
		_statusCode: code,
	}
}

/*UploadPipelineVersionDefault handles this case with default header values.

UploadPipelineVersionDefault upload pipeline version default
*/
type UploadPipelineVersionDefault struct {
	_statusCode int

	Payload *pipeline_upload_model.APIStatus
}

// Code gets the status code for the upload pipeline version default response
func (o *UploadPipelineVersionDefault) Code() int {
	return o._statusCode
}

func (o *UploadPipelineVersionDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/pipelines/upload_version][%d] UploadPipelineVersion default  %+v", o._statusCode, o.Payload)
}

func (o *UploadPipelineVersionDefault) GetPayload() *pipeline_upload_model.APIStatus {
	return o.Payload
}

func (o *UploadPipelineVersionDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_upload_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
