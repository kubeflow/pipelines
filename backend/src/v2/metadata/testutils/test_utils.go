package testutils

import (
	"fmt"

	"github.com/kubeflow/pipelines/backend/test/v2"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func NewTestMlmdClient(testMlmdServerAddress string, testMlmdServerPort string, tlsEnabled bool, caCertPath string) (pb.MetadataStoreServiceClient, error) {
	dialOption := grpc.WithInsecure()
	if tlsEnabled {
		tlsCfg, err := test.GetTLSConfig(caCertPath)
		if err != nil {
			return nil, err
		}
		creds := credentials.NewTLS(tlsCfg)
		dialOption = grpc.WithTransportCredentials(creds)
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", testMlmdServerAddress, testMlmdServerPort),
		dialOption,
	)
	if err != nil {
		return nil, fmt.Errorf("NewMlmdClient() failed: %w", err)
	}
	return pb.NewMetadataStoreServiceClient(conn), nil
}
