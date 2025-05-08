package testutils

import (
	"fmt"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/grpc"
)

func NewTestMlmdClient(testMlmdServerAddress string, testMlmdServerPort string) (pb.MetadataStoreServiceClient, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", testMlmdServerAddress, testMlmdServerPort),
		grpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("NewMlmdClient() failed: %w", err)
	}
	return pb.NewMetadataStoreServiceClient(conn), nil
}
