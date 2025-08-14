package metadata

const (
	metadataGrpcServiceAddress = "metadata-grpc-service.kubeflow"
	metadataGrpcServicePort    = "8080"
)

type ServerConfig struct {
	Address string
	Port    string
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Address: metadataGrpcServiceAddress,
		Port:    metadataGrpcServicePort,
	}
}
