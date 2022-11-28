// Code generated by go-swagger; DO NOT EDIT.

package pipeline_upload_model

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"encoding/json"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/validate"
)

// ResourceType resource type
// swagger:model ResourceType
type ResourceType string

const (

	// ResourceTypeUNKNOWNRESOURCETYPE captures enum value "UNKNOWN_RESOURCE_TYPE"
	ResourceTypeUNKNOWNRESOURCETYPE ResourceType = "UNKNOWN_RESOURCE_TYPE"

	// ResourceTypeEXPERIMENT captures enum value "EXPERIMENT"
	ResourceTypeEXPERIMENT ResourceType = "EXPERIMENT"

	// ResourceTypeJOB captures enum value "JOB"
	ResourceTypeJOB ResourceType = "JOB"

	// ResourceTypePIPELINE captures enum value "PIPELINE"
	ResourceTypePIPELINE ResourceType = "PIPELINE"

	// ResourceTypePIPELINEVERSION captures enum value "PIPELINE_VERSION"
	ResourceTypePIPELINEVERSION ResourceType = "PIPELINE_VERSION"

	// ResourceTypeNAMESPACE captures enum value "NAMESPACE"
	ResourceTypeNAMESPACE ResourceType = "NAMESPACE"
)

// for schema
var resourceTypeEnum []interface{}

func init() {
	var res []ResourceType
	if err := json.Unmarshal([]byte(`["UNKNOWN_RESOURCE_TYPE","EXPERIMENT","JOB","PIPELINE","PIPELINE_VERSION","NAMESPACE"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		resourceTypeEnum = append(resourceTypeEnum, v)
	}
}

func (m ResourceType) validateResourceTypeEnum(path, location string, value ResourceType) error {
	if err := validate.Enum(path, location, value, resourceTypeEnum); err != nil {
		return err
	}
	return nil
}

// Validate validates this resource type
func (m ResourceType) Validate(formats strfmt.Registry) error {
	var res []error

	// value enum
	if err := m.validateResourceTypeEnum("", "body", m); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
