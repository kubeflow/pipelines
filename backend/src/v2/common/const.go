package common

// publisher type enum
const (
	PublisherType_DAG      = "DAG"
	PublisherType_EXECUTOR = "EXECUTOR"
)

// executor output parameters path
const (
	ExecutorOutputPathParameters = "/kfp/outputs/parameters"
	ExecutorEntrypointPath       = "/kfp/entrypoint/entrypoint"
	ExecutorEntrypointVolumePath = "/kfp/entrypoint"
)

// execution custom properties
const (
	ExecutionPropertyPrefixOutputParam = "output:"
)
