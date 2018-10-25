package common

type ResourceType string
type Relationship string

const (
	Experiment ResourceType = "Experiment"
	Job        ResourceType = "Job"
	Rrn        ResourceType = "Run"
	Pipeline   ResourceType = "pipeline"
)

const (
	Owner   Relationship = "Owner"
	Creator Relationship = "Creator"
)
