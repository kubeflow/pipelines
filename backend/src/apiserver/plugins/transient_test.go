package plugins

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type timeoutNetError struct{ timeout bool }

func (e timeoutNetError) Error() string   { return "network error" }
func (e timeoutNetError) Timeout() bool   { return e.timeout }
func (e timeoutNetError) Temporary() bool { return false }

var _ net.Error = timeoutNetError{}

func TestIsTransientConfigError(t *testing.T) {
	secrets := schema.GroupResource{Resource: "secrets"}
	wrap := func(err error) error {
		return util.NewInternalServerError(err, "failed to read MLflow credentials secret")
	}

	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"service unavailable":   {wrap(apierrors.NewServiceUnavailable("unavailable")), true},
		"server timeout":        {wrap(apierrors.NewServerTimeout(secrets, "get", 1)), true},
		"gateway timeout":       {wrap(apierrors.NewTimeoutError("timeout", 1)), true},
		"too many requests":     {wrap(apierrors.NewTooManyRequests("throttled", 1)), true},
		"deadline exceeded":     {fmt.Errorf("resolve: %w", context.DeadlineExceeded), true},
		"network timeout":       {fmt.Errorf("resolve: %w", timeoutNetError{timeout: true}), true},
		"network non-timeout":   {fmt.Errorf("resolve: %w", timeoutNetError{timeout: false}), false},
		"internal error":        {wrap(apierrors.NewInternalError(errors.New("storage"))), false},
		"not found":             {wrap(apierrors.NewNotFound(secrets, "kfp-mlflow-credentials")), false},
		"forbidden":             {wrap(apierrors.NewForbidden(secrets, "kfp-mlflow-credentials", errors.New("denied"))), false},
		"invalid configuration": {util.NewInvalidInputError("unsupported plugins.mlflow.settings.authType %q", "x"), false},
		"context canceled":      {fmt.Errorf("resolve: %w", context.Canceled), false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isTransientConfigError(tc.err))
		})
	}
}
