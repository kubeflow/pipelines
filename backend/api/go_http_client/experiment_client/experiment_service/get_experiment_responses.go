// Code generated by go-swagger; DO NOT EDIT.

package experiment_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
)

// GetExperimentReader is a Reader for the GetExperiment structure.
type GetExperimentReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetExperimentReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetExperimentOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewGetExperimentDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewGetExperimentOK creates a GetExperimentOK with default headers values
func NewGetExperimentOK() *GetExperimentOK {
	return &GetExperimentOK{}
}

/*GetExperimentOK handles this case with default header values.

A successful response.
*/
type GetExperimentOK struct {
	Payload *experiment_model.APIExperiment
}

func (o *GetExperimentOK) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/experiments/{id}][%d] getExperimentOK  %+v", 200, o.Payload)
}

func (o *GetExperimentOK) GetPayload() *experiment_model.APIExperiment {
	return o.Payload
}

func (o *GetExperimentOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.APIExperiment)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetExperimentDefault creates a GetExperimentDefault with default headers values
func NewGetExperimentDefault(code int) *GetExperimentDefault {
	return &GetExperimentDefault{
		_statusCode: code,
	}
}

/*GetExperimentDefault handles this case with default header values.

GetExperimentDefault get experiment default
*/
type GetExperimentDefault struct {
	_statusCode int

	Payload *experiment_model.APIStatus
}

// Code gets the status code for the get experiment default response
func (o *GetExperimentDefault) Code() int {
	return o._statusCode
}

func (o *GetExperimentDefault) Error() string {
	return fmt.Sprintf("[GET /apis/v1beta1/experiments/{id}][%d] GetExperiment default  %+v", o._statusCode, o.Payload)
}

func (o *GetExperimentDefault) GetPayload() *experiment_model.APIStatus {
	return o.Payload
}

func (o *GetExperimentDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(experiment_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
