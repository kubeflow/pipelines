// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by go-swagger; DO NOT EDIT.

package pipeline_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	pipeline_model "github.com/kubeflow/pipelines/backend/api/go_http_client/pipeline_model"
)

// GetPipelineVersionTemplateReader is a Reader for the GetPipelineVersionTemplate structure.
type GetPipelineVersionTemplateReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetPipelineVersionTemplateReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewGetPipelineVersionTemplateOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewGetPipelineVersionTemplateDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetPipelineVersionTemplateOK creates a GetPipelineVersionTemplateOK with default headers values
func NewGetPipelineVersionTemplateOK() *GetPipelineVersionTemplateOK {
	return &GetPipelineVersionTemplateOK{}
}

/*GetPipelineVersionTemplateOK handles this case with default header values.

A successful response.
*/
type GetPipelineVersionTemplateOK struct {
	Payload *pipeline_model.APIGetTemplateResponse
}

func (o *GetPipelineVersionTemplateOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/pipeline_versions/{version_id}/templates][%d] getPipelineVersionTemplateOK  %+v", 200, o.Payload)
}

func (o *GetPipelineVersionTemplateOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIGetTemplateResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetPipelineVersionTemplateDefault creates a GetPipelineVersionTemplateDefault with default headers values
func NewGetPipelineVersionTemplateDefault(code int) *GetPipelineVersionTemplateDefault {
	return &GetPipelineVersionTemplateDefault{
		_statusCode: code,
	}
}

/*GetPipelineVersionTemplateDefault handles this case with default header values.

GetPipelineVersionTemplateDefault get pipeline version template default
*/
type GetPipelineVersionTemplateDefault struct {
	_statusCode int

	Payload *pipeline_model.APIStatus
}

// Code gets the status code for the get pipeline version template default response
func (o *GetPipelineVersionTemplateDefault) Code() int {
	return o._statusCode
}

func (o *GetPipelineVersionTemplateDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/pipeline_versions/{version_id}/templates][%d] GetPipelineVersionTemplate default  %+v", o._statusCode, o.Payload)
}

func (o *GetPipelineVersionTemplateDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(pipeline_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
