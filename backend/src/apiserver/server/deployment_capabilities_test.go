package server

import (
	"context"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestRecurringRunManagementWithoutController(t *testing.T) {
	viper.Set("SCHEDULED_WORKFLOW_CONTROLLER_ENABLED", false)
	t.Cleanup(func() { viper.Set("SCHEDULED_WORKFLOW_CONTROLLER_ENABLED", nil) })
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewJobServer(manager, &JobServerOptions{})
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"create": func() error {
			_, err := server.CreateRecurringRun(ctx, &apiv2.CreateRecurringRunRequest{})
			return err
		},
		"get": func() error {
			_, err := server.GetRecurringRun(ctx, &apiv2.GetRecurringRunRequest{RecurringRunId: "missing"})
			return err
		},
		"list": func() error {
			_, err := server.ListRecurringRuns(ctx, &apiv2.ListRecurringRunsRequest{})
			return err
		},
		"enable": func() error {
			_, err := server.EnableRecurringRun(ctx, &apiv2.EnableRecurringRunRequest{RecurringRunId: "missing"})
			return err
		},
		"disable": func() error {
			_, err := server.DisableRecurringRun(ctx, &apiv2.DisableRecurringRunRequest{RecurringRunId: "missing"})
			return err
		},
		"delete": func() error {
			_, err := server.DeleteRecurringRun(ctx, &apiv2.DeleteRecurringRunRequest{RecurringRunId: "missing"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			require.True(t, util.IsUserErrorCodeMatch(err, codes.FailedPrecondition), "got %v", err)
			require.Contains(t, err.Error(), "The scheduled-workflow controller is not enabled.")
		})
	}
}
