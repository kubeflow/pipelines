// Copyright 2020 Google LLC
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

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	experiment_model "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
)

// ListExperimentReader is a Reader for the ListExperiment structure.
type ListExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *ListExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewListExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewListExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewListExperimentOK creates a ListExperimentOK with default headers values
func NewListExperimentOK() *ListExperimentOK {
	return &ListExperimentOK{}
}

/*ListExperimentOK handles this case with default header values.

A successful response.
*/
type ListExperimentOK struct {
	Payload *experiment_model.APIListExperimentsResponse
}

func (o *ListExperimentOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/experiments][%d] listExperimentOK  %+v", 200, o.Payload)
}

func (o *ListExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.APIListExperimentsResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewListExperimentDefault creates a ListExperimentDefault with default headers values
func NewListExperimentDefault(code int) *ListExperimentDefault {
	return &ListExperimentDefault{
		_statusCode: code,
	}
}

/*ListExperimentDefault handles this case with default header values.

ListExperimentDefault list experiment default
*/
type ListExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.APIStatus
}

// Code gets the status code for the list experiment default response
func (o *ListExperimentDefault) Code() int {
	return o._statusCode
}

func (o *ListExperimentDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/experiments][%d] ListExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *ListExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
