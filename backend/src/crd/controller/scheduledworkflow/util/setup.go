package util

import (
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
)

const (
	addressTemp = "%s:%s"
)

func NewRunPipelineClient(
	initializeTimeout time.Duration,
	timeout time.Duration,
	basePath string,
	mlPipelineServiceName string,
	mlPipelineServiceHttpPort string,
	mlPipelineServiceGRPCPort string) (api.RunServiceClient, error) {
	httpAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceHttpPort)
	grpcAddress := fmt.Sprintf(addressTemp, mlPipelineServiceName, mlPipelineServiceGRPCPort)
	err := util.WaitForAPIAvailable(initializeTimeout, basePath, httpAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to initialize pipeline client. Error: %s", err.Error())
	}
	connection, err := util.GetRpcConnection(grpcAddress)
	if err != nil {
		return nil, errors.Wrapf(err,
			"Failed to get RPC connection. Error: %s", err.Error())
	}

	return api.NewRunServiceClient(connection), nil
}
