// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	contextLoggerKey contextKey = "driver_log_key"
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

// WithLogger adds a logger that writes to stdout and logFile. The caller must close
// the returned file. It rejects nil contexts and contexts that already contain a logger.
func WithLogger(ctx context.Context, logFile string) (context.Context, io.Closer, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf(
			"cannot create logger for %q: context is nil; provide a non-nil context",
			logFile,
		)
	}

	if GetLoggerFrom(ctx) != nil {
		return nil, nil, fmt.Errorf("logger already exists in context; reuse it with GetLoggerFrom or provide a context without a logger")
	}

	logger, f, err := newFileLogger(logFile)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"cannot create log file %q; choose a writable file path with an existing parent directory: %w",
			logFile,
			err,
		)
	}

	ctx = context.WithValue(ctx, contextLoggerKey, logger)

	return ctx, f, nil
}

// GetLoggerFrom returns the logger in ctx, or nil if none is present. ctx must be non-nil.
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
