// Code generated by go-swagger; DO NOT EDIT.

package visualization_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	visualization_model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/visualization_model"
)

// VisualizationServiceCreateVisualizationV1Reader is a Reader for the VisualizationServiceCreateVisualizationV1 structure.
type VisualizationServiceCreateVisualizationV1Reader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *VisualizationServiceCreateVisualizationV1Reader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewVisualizationServiceCreateVisualizationV1OK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		result := NewVisualizationServiceCreateVisualizationV1Default(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewVisualizationServiceCreateVisualizationV1OK creates a VisualizationServiceCreateVisualizationV1OK with default headers values
func NewVisualizationServiceCreateVisualizationV1OK() *VisualizationServiceCreateVisualizationV1OK {
	return &VisualizationServiceCreateVisualizationV1OK{}
}

/*VisualizationServiceCreateVisualizationV1OK handles this case with default header values.

A successful response.
*/
type VisualizationServiceCreateVisualizationV1OK struct {
	Payload *visualization_model.V2beta1Visualization
}

func (o *VisualizationServiceCreateVisualizationV1OK) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/visualizations/{namespace}][%d] visualizationServiceCreateVisualizationV1OK  %+v", 200, o.Payload)
}

func (o *VisualizationServiceCreateVisualizationV1OK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(visualization_model.V2beta1Visualization)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewVisualizationServiceCreateVisualizationV1Default creates a VisualizationServiceCreateVisualizationV1Default with default headers values
func NewVisualizationServiceCreateVisualizationV1Default(code int) *VisualizationServiceCreateVisualizationV1Default {
	return &VisualizationServiceCreateVisualizationV1Default{
		_statusCode: code,
	}
}

/*VisualizationServiceCreateVisualizationV1Default handles this case with default header values.

An unexpected error response.
*/
type VisualizationServiceCreateVisualizationV1Default struct {
	_statusCode int

	Payload *visualization_model.RuntimeError
}

// Code gets the status code for the visualization service create visualization v1 default response
func (o *VisualizationServiceCreateVisualizationV1Default) Code() int {
	return o._statusCode
}

func (o *VisualizationServiceCreateVisualizationV1Default) Error() string {
	return fmt.Sprintf("[POST /apis/v2beta1/visualizations/{namespace}][%d] VisualizationService_CreateVisualizationV1 default  %+v", o._statusCode, o.Payload)
}

func (o *VisualizationServiceCreateVisualizationV1Default) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(visualization_model.RuntimeError)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
