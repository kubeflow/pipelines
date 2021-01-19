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

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	run_model "github.com/kubeflow/pipelines/backend/api/go_http_client/run_model"
)

// GetRunReader is a Reader for the GetRun structure.
type GetRunReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetRunReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetRunOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetRunDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetRunOK creates a GetRunOK with default headers values
func NewGetRunOK() *GetRunOK {
	return &GetRunOK{}
}

/*GetRunOK handles this case with default header values.

A successful response.
*/
type GetRunOK struct {
	Payload *run_model.APIRunDetail
}

func (o *GetRunOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/runs/{run_id}][%d] getRunOK  %+v", 200, o.Payload)
}

func (o *GetRunOK) GetPayload() *run_model.APIRunDetail {
	return o.Payload
}

func (o *GetRunOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.APIRunDetail)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetRunDefault creates a GetRunDefault with default headers values
func NewGetRunDefault(code int) *GetRunDefault {
	return &GetRunDefault{
		_statusCode: code,
	}
}

/*GetRunDefault handles this case with default header values.

GetRunDefault get run default
*/
type GetRunDefault struct {
	_statusCode int

	Payload *run_model.APIStatus
}

// Code gets the status code for the get run default response
func (o *GetRunDefault) Code() int {
	return o._statusCode
}

func (o *GetRunDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/runs/{run_id}][%d] GetRun default  %+v", o._statusCode, o.Payload)
}

func (o *GetRunDefault) GetPayload() *run_model.APIStatus {
	return o.Payload
}

func (o *GetRunDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(run_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
