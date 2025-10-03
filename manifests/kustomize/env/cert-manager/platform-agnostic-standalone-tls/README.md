# Deploying KFP with Pod-to-Pod TLS Enabled

The manifests in this folder deploy Kubeflow Pipelines with pod-to-pod TLS enabled in standalone mode on a KinD cluster. These manifests can be easily deployed through the `kind-cluster-agnostic-tls` target located in the backend Makefile.  TLS configuration is outlined below:
### API Server
The API server is mounted with the TLS key/cert pair and accepts HTTPS requests.
### Scheduledworkflow Controller
The Scheduledworkflow Controller is mounted with the TLS cert CA and sends HTTPS requests to the API server.
### Persistence Agent 
The Persistence Agent is mounted with the TLS cert CA and sends HTTPS requests to the API server.
### KFP UI
The UI deployment is mounted with the TLS cert CA and sends HTTPS requests to the metadata-envoy deployment and the API server.
### Metadata-Envoy
The Metadata Envoy deployment is mounted with TLS key/cert and serves requests over TLS
### Metadata-gRPC
The Metadata gRPC deployment is mounted with TLS key/cert and serves requests over TLS
### Driver & Launcher
The driver and launcher pods are configured with the TLS cert CA in addition to system CAs in  order to send HTTPS requests to the both the API server and metadata-gRPC deployment.