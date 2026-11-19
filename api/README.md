# Shared authoring schema

`v2alpha1/pipeline_spec.proto` supplies the SDK's type/interface and portable
container-definition messages. Native pipeline execution does not use an IR
builder or KFP RPC API. The backend's REST/gRPC service definitions are removed.

Run `make generate-python` at the repository root to regenerate the checked-in
Python binding. There are no Go bindings or backend generators in this directory.
