package testutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"os"

	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func NewTestMlmdClient(testMlmdServerAddress string, testMlmdServerPort string, tlsEnabled bool, caCertPath string) (pb.MetadataStoreServiceClient, error) {
	dialOption := grpc.WithInsecure()
	if tlsEnabled {
		creds := credentials.NewTLS(&tls.Config{})
		if caCertPath == "" {
			return nil, errors.New("CA cert path is empty")
		}

		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("Error reading CA cert file %s: %w", caCertPath, err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		config := &tls.Config{
			RootCAs: caCertPool,
		}
		creds = credentials.NewTLS(config)
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
