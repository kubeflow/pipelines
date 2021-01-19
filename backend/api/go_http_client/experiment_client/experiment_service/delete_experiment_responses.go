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

// DeleteExperimentReader is a Reader for the DeleteExperiment structure.
type DeleteExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeleteExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeleteExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewDeleteExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewDeleteExperimentOK creates a DeleteExperimentOK with default headers values
func NewDeleteExperimentOK() *DeleteExperimentOK {
	return &DeleteExperimentOK{}
}

/*DeleteExperimentOK handles this case with default header values.

A successful response.
*/
type DeleteExperimentOK struct {
	Payload interface{}
}

func (o *DeleteExperimentOK) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/experiments/{id}][%d] deleteExperimentOK  %+v", 200, o.Payload)
}

func (o *DeleteExperimentOK) GetPayload() interface{} {
	return o.Payload
}

func (o *DeleteExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeleteExperimentDefault creates a DeleteExperimentDefault with default headers values
func NewDeleteExperimentDefault(code int) *DeleteExperimentDefault {
	return &DeleteExperimentDefault{
		_statusCode: code,
	}
}

/*DeleteExperimentDefault handles this case with default header values.

DeleteExperimentDefault delete experiment default
*/
type DeleteExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.APIStatus
}

// Code gets the status code for the delete experiment default response
func (o *DeleteExperimentDefault) Code() int {
	return o._statusCode
}

func (o *DeleteExperimentDefault) Error() string {
	return fmt.Sprintf("[DELETE /apis/v1beta1/experiments/{id}][%d] DeleteExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *DeleteExperimentDefault) GetPayload() *experiment_model.APIStatus {
	return o.Payload
}

func (o *DeleteExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
