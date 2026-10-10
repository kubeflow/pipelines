package util

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type CtxKey string

const (
	contextLoggerKey CtxKey = "driver_log_key"
)

func newFileLogger(logFile string) (*logrus.Logger, io.Closer, error) {
	f, err := os.Create(logFile)
	if err != nil {
		return nil, nil, err
	}

	logger := logrus.New()
	logger.Out = io.MultiWriter(os.Stdout, f)
	logger.Formatter = &logrus.TextFormatter{}
	return logger, f, nil
}

// WithExistingLogger For testing only
func WithExistingLogger(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, logger)
}

func WithLogger(ctx context.Context, logFile string) (context.Context, io.Closer, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf(
			"error during creation of the logger for logId: %v. ctx can not be nil",
			logFile,
		)
	}

	if GetLoggerFrom(ctx) != nil {
		return nil, nil, fmt.Errorf("logger already exists in context")
	}

	logger, f, err := newFileLogger(logFile)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"error during creation of the logger for logId: %v details: %w",
			logFile,
			err,
		)
	}

	ctx = context.WithValue(ctx, contextLoggerKey, logger)

	return ctx, f, nil
}

func GetLoggerFrom(ctx context.Context) *logrus.Logger {
	v := ctx.Value(contextLoggerKey)
	if v == nil {
		return nil
	}

	logger, ok := v.(*logrus.Logger)
	if !ok {
		return nil
	}

	return logger
}
