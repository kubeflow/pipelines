# Kubernetes authoring configuration

`proto/kubernetes_executor_config.proto` defines typed configuration used by
`kfp.kubernetes` modifiers. The native compiler lowers supported settings into
Argo templates and rejects unsupported settings explicitly.

Run `make generate-python` from the repository root. The Python binding is in
`sdk/python/kfp/kubernetes`; there is no separate distribution or Go backend here.
