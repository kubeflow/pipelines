package metadata

import "github.com/kubeflow/pipelines/backend/src/apiserver/common"

const (
	metadataGrpcServiceName = "metadata-grpc-service"
	metadataGrpcServicePort = "8080"
)

type ServerConfig struct {
	Address string
	Port    string
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Address: metadataGrpcServiceName + "." + common.GetPodNamespace() + ".svc.cluster.local",
		Port:    metadataGrpcServicePort,
	}
}
