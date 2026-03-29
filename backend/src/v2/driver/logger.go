package driver

import (
	"context"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/sirupsen/logrus"
)

func driverLogger(ctx context.Context) *logrus.Logger {
	if logger := util.GetLoggerFrom(ctx); logger != nil {
		return logger
	}
	return logrus.StandardLogger()
}
