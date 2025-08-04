package metadata

import (
	"os"

	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
)

const (
	metadataGrpcServiceAddress = "metadata-grpc-service.kubeflow"
	metadataGrpcServicePort    = "8080"
)

type ServerConfig struct {
	Address string
	Port    string
}

func DefaultConfig() *ServerConfig {
	// If proxy is enabled, use DNS name `metadata-grpc-service.kubeflow:8080` as default.
	_, isHttpProxySet := os.LookupEnv(proxy.HttpProxyEnv)
	_, isHttpsProxySet := os.LookupEnv(proxy.HttpsProxyEnv)
	if isHttpProxySet || isHttpsProxySet {
		return &ServerConfig{
			Address: metadataGrpcServiceAddress,
			Port:    metadataGrpcServicePort,
		}
	}
	// The env vars exist when metadata-grpc-service Kubernetes service is
	// in the same namespace as the current Pod.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	hostEnv := os.Getenv("METADATA_GRPC_SERVICE_SERVICE_HOST")
	portEnv := os.Getenv("METADATA_GRPC_SERVICE_SERVICE_PORT")
	if hostEnv != "" && portEnv != "" {
		return &ServerConfig{
			Address: hostEnv,
			Port:    portEnv,
		}
	}
	return &ServerConfig{
		Address: metadataGrpcServiceAddress,
		Port:    metadataGrpcServicePort,
	}
}
