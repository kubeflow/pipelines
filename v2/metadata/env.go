package metadata

import "os"

type ServerConfig struct {
	Address string
	Port    string
}

func DefaultConfig() *ServerConfig {
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
		Address: "metadata-grpc-service.kubeflow",
		Port:    "8080",
	}
}
