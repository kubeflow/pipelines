// Code generated by go-swagger; DO NOT EDIT.

package job_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/job_model"
)

// JobServiceListJobsReader is a Reader for the JobServiceListJobs structure.
type JobServiceListJobsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *JobServiceListJobsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewJobServiceListJobsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	default:
		result := NewJobServiceListJobsDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewJobServiceListJobsOK creates a JobServiceListJobsOK with default headers values
func NewJobServiceListJobsOK() *JobServiceListJobsOK {
	return &JobServiceListJobsOK{}
}

/*
JobServiceListJobsOK describes a response with status code 200, with default header values.

A successful response.
*/
type JobServiceListJobsOK struct {
	Payload *job_model.APIListJobsResponse
}

// IsSuccess returns true when this job service list jobs o k response has a 2xx status code
func (o *JobServiceListJobsOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this job service list jobs o k response has a 3xx status code
func (o *JobServiceListJobsOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this job service list jobs o k response has a 4xx status code
func (o *JobServiceListJobsOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this job service list jobs o k response has a 5xx status code
func (o *JobServiceListJobsOK) IsServerError() bool {
	return false
}

// IsCode returns true when this job service list jobs o k response a status code equal to that given
func (o *JobServiceListJobsOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the job service list jobs o k response
func (o *JobServiceListJobsOK) Code() int {
	return 200
}

func (o *JobServiceListJobsOK) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /apis/v1beta1/jobs][%d] jobServiceListJobsOK %s", 200, payload)
}

func (o *JobServiceListJobsOK) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /apis/v1beta1/jobs][%d] jobServiceListJobsOK %s", 200, payload)
}

func (o *JobServiceListJobsOK) GetPayload() *job_model.APIListJobsResponse {
	return o.Payload
}

func (o *JobServiceListJobsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(job_model.APIListJobsResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewJobServiceListJobsDefault creates a JobServiceListJobsDefault with default headers values
func NewJobServiceListJobsDefault(code int) *JobServiceListJobsDefault {
	return &JobServiceListJobsDefault{
		_statusCode: code,
	}
}

/*
JobServiceListJobsDefault describes a response with status code -1, with default header values.

An unexpected error response.
*/
type JobServiceListJobsDefault struct {
	_statusCode int

	Payload *job_model.GooglerpcStatus
}

// IsSuccess returns true when this job service list jobs default response has a 2xx status code
func (o *JobServiceListJobsDefault) IsSuccess() bool {
	return o._statusCode/100 == 2
}

// IsRedirect returns true when this job service list jobs default response has a 3xx status code
func (o *JobServiceListJobsDefault) IsRedirect() bool {
	return o._statusCode/100 == 3
}

// IsClientError returns true when this job service list jobs default response has a 4xx status code
func (o *JobServiceListJobsDefault) IsClientError() bool {
	return o._statusCode/100 == 4
}

// IsServerError returns true when this job service list jobs default response has a 5xx status code
func (o *JobServiceListJobsDefault) IsServerError() bool {
	return o._statusCode/100 == 5
}

// IsCode returns true when this job service list jobs default response a status code equal to that given
func (o *JobServiceListJobsDefault) IsCode(code int) bool {
	return o._statusCode == code
}

// Code gets the status code for the job service list jobs default response
func (o *JobServiceListJobsDefault) Code() int {
	return o._statusCode
}

func (o *JobServiceListJobsDefault) Error() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /apis/v1beta1/jobs][%d] JobService_ListJobs default %s", o._statusCode, payload)
}

func (o *JobServiceListJobsDefault) String() string {
	payload, _ := json.Marshal(o.Payload)
	return fmt.Sprintf("[GET /apis/v1beta1/jobs][%d] JobService_ListJobs default %s", o._statusCode, payload)
}

func (o *JobServiceListJobsDefault) GetPayload() *job_model.GooglerpcStatus {
	return o.Payload
}

func (o *JobServiceListJobsDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(job_model.GooglerpcStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
