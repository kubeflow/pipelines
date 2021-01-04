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

package healthz_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
<<<<<<< HEAD

	strfmt "github.com/go-openapi/strfmt"

	healthz_model "github.com/kubeflow/pipelines/backend/api/go_http_client/healthz_model"
=======
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/go_http_client/healthz_model"
>>>>>>> c18a21eb (typo with folders fixed)
)

// GetHealthzReader is a Reader for the GetHealthz structure.
type GetHealthzReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetHealthzReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
<<<<<<< HEAD

=======
>>>>>>> c18a21eb (typo with folders fixed)
	case 200:
		result := NewGetHealthzOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
<<<<<<< HEAD

=======
>>>>>>> c18a21eb (typo with folders fixed)
	default:
		result := NewGetHealthzDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetHealthzOK creates a GetHealthzOK with default headers values
func NewGetHealthzOK() *GetHealthzOK {
	return &GetHealthzOK{}
}

/*GetHealthzOK handles this case with default header values.

A successful response.
*/
type GetHealthzOK struct {
	Payload *healthz_model.APIGetHealthzResponse
}

func (o *GetHealthzOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/healthz][%d] getHealthzOK  %+v", 200, o.Payload)
}

<<<<<<< HEAD
=======
func (o *GetHealthzOK) GetPayload() *healthz_model.APIGetHealthzResponse {
	return o.Payload
}

>>>>>>> c18a21eb (typo with folders fixed)
func (o *GetHealthzOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(healthz_model.APIGetHealthzResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetHealthzDefault creates a GetHealthzDefault with default headers values
func NewGetHealthzDefault(code int) *GetHealthzDefault {
	return &GetHealthzDefault{
		_statusCode: code,
	}
}

/*GetHealthzDefault handles this case with default header values.

GetHealthzDefault get healthz default
*/
type GetHealthzDefault struct {
	_statusCode int

	Payload *healthz_model.APIStatus
}

// Code gets the status code for the get healthz default response
func (o *GetHealthzDefault) Code() int {
	return o._statusCode
}

func (o *GetHealthzDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/healthz][%d] GetHealthz default  %+v", o._statusCode, o.Payload)
}

<<<<<<< HEAD
=======
func (o *GetHealthzDefault) GetPayload() *healthz_model.APIStatus {
	return o.Payload
}

>>>>>>> c18a21eb (typo with folders fixed)
func (o *GetHealthzDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(healthz_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
