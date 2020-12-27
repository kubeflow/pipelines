// Code generated by go-swagger; DO NOT EDIT.

package visualization_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	visualization_model "github.com/kubeflow/pipelines/backend/api/go_http_client/visualization_model"
)

// CreateVisualizationReader is a Reader for the CreateVisualization structure.
type CreateVisualizationReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateVisualizationReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewCreateVisualizationOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewCreateVisualizationDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewCreateVisualizationOK creates a CreateVisualizationOK with default headers values
func NewCreateVisualizationOK() *CreateVisualizationOK {
	return &CreateVisualizationOK{}
}

/*CreateVisualizationOK handles this case with default header values.

A successful response.
*/
type CreateVisualizationOK struct {
	Payload *visualization_model.APIVisualization
}

func (o *CreateVisualizationOK) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/visualizations/{namespace}][%d] createVisualizationOK  %+v", 200, o.Payload)
}

func (o *CreateVisualizationOK) GetPayload() *visualization_model.APIVisualization {
	return o.Payload
}

func (o *CreateVisualizationOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(visualization_model.APIVisualization)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewCreateVisualizationDefault creates a CreateVisualizationDefault with default headers values
func NewCreateVisualizationDefault(code int) *CreateVisualizationDefault {
	return &CreateVisualizationDefault{
		_statusCode: code,
	}
}

/*CreateVisualizationDefault handles this case with default header values.

CreateVisualizationDefault create visualization default
*/
type CreateVisualizationDefault struct {
	_statusCode int

	Payload *visualization_model.APIStatus
}

// Code gets the status code for the create visualization default response
func (o *CreateVisualizationDefault) Code() int {
	return o._statusCode
}

func (o *CreateVisualizationDefault) Error() string {
	return fmt.Sprintf("[POST /apis/v1beta1/visualizations/{namespace}][%d] CreateVisualization default  %+v", o._statusCode, o.Payload)
}

func (o *CreateVisualizationDefault) GetPayload() *visualization_model.APIStatus {
	return o.Payload
}

func (o *CreateVisualizationDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(visualization_model.APIStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
