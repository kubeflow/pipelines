package argo

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodePhase is a label for the condition of a node at the current time.
type NodePhase string

// Workflow is the definition of a workflow resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Workflow struct {
	metav1.ObjectMeta `json:"metadata"`
	Status            WorkflowStatus `json:"status"`
}

// WorkflowList is list of Workflow resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowList struct {
	Items []Workflow `json:"items"`
}

// WorkflowSpec is the specification of a Workflow.
type WorkflowSpec struct {
	// Arguments contain the parameters and artifacts sent to the workflow entrypoint
	// Parameters are referencable globally using the 'workflow' variable prefix.
	// e.g. {{workflow.parameters.myparam}}
	Arguments Arguments `json:"arguments,omitempty"`
}

// Parameter indicate a passed string parameter to a service template with an optional default value
type Parameter struct {
	// Name is the parameter name
	Name string `json:"name"`

	// Default is the default value to use for an input parameter if a value was not supplied
	Default *string `json:"default,omitempty"`

	// Value is the literal value to use for the parameter.
	// If specified in the context of an input parameter, the value takes precedence over any passed values
	Value *string `json:"value,omitempty"`

	// ValueFrom is the source for the output parameter's value
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

// ValueFrom describes a location in which to obtain the value to a parameter
type ValueFrom struct {
	// Path in the container to retrieve an output parameter value from in container templates
	Path string `json:"path,omitempty"`

	// JSONPath of a resource to retrieve an output parameter value from in resource templates
	JSONPath string `json:"jsonPath,omitempty"`

	// JQFilter expression against the resource object in resource templates
	JQFilter string `json:"jqFilter,omitempty"`

	// Parameter reference to a step or dag task in which to retrieve an output parameter value from
	// (e.g. '{{steps.mystep.outputs.myparam}}')
	Parameter string `json:"parameter,omitempty"`
}

// Arguments to a template
type Arguments struct {
	// Parameters is the list of parameters to pass to the template or workflow
	Parameters []Parameter `json:"parameters,omitempty"`
}

// WorkflowStatus contains overall status information about a workflow
// +k8s:openapi-gen=false
type WorkflowStatus struct {
	// Phase a simple, high-level summary of where the workflow is in its lifecycle.
	Phase NodePhase `json:"phase"`

	// Time at which this workflow started
	StartedAt metav1.Time `json:"startedAt,omitempty"`

	// Time at which this workflow completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty"`
}
