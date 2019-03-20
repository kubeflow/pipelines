# OpenVINO Model Server deployment pipeline

This is an example of a one step pipeline implementation for [OpenVINO Model Server deployer](../../../components/openvino/ovms-deployer).

## Parameters

- model_export_path - local or gs path to the folder with numerical subfolders storing OpenVINO models
- server_name - Kubernetes service name to be deployed which is equal to service model name
- log_level - DEBUG/INFO/ERROR
- batch_size - which batch size should be loaded in the served model: 'auto' or numerical values
- model_version_policy - parameter defining which version should be served in OVMS. Examples: '{"latest": { "num_versions":2 }}') / 
{"specific": { "versions":[1, 3] }} / {"all": {}}
