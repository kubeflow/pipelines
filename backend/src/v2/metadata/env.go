package metadata

import "github.com/kubeflow/pipelines/backend/src/apiserver/common"

const (
	metadataGrpcServicePort = "8080"
)

type ServerConfig struct {
	Address string
	Port    string
}

func GetMetadataConfig() *ServerConfig {
	return &ServerConfig{
		Address: common.GetMetadataServiceName() + "." + common.GetPodNamespace() + ".svc." + common.GetClusterDomain(),
		Port:    metadataGrpcServicePort,
	}
}
