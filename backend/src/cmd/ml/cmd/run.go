package cmd

import (
	"fmt"
	"math"

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/run_client/run_service"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func NewRunCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "run",
		Short: "Manage runs",
	}
	return command
}

func NewRunGetCmd(root *RootCommand) *cobra.Command {
	var (
		runID string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "get",
		Short: "Display a run",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewGetRunParams()
			params.RunID = runID
			pkg, workflow, err := root.RunClient().Get(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			pkg.PipelineRuntime.WorkflowManifest = ""
			PrettyPrintResult(root.Writer(), root.OutputFormat(), pkg,
				&WorkflowForDisplay{Workflow: workflow})
			return nil
		},
	}
	command.PersistentFlags().StringVar(&runID, flagNameID,
		"", "The ID of the run")
	command.MarkPersistentFlagRequired(flagNameID)
	command.SetOutput(root.Writer())
	return command
}

func NewRunListCmd(root *RootCommand, pageSize int32) *cobra.Command {
	var (
		maxResultSize int
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "List runs",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			if err != nil {
				return err
			}
			if maxResultSize < 0 {
				return fmt.Errorf("The flag 'max-items' cannot be negative")
			}
			return nil
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewListRunsParams()
			params.PageSize = util.Int32Pointer(pageSize)
			results, err := root.RunClient().ListAll(params, maxResultSize)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), results)
			return nil
		},
	}
	command.PersistentFlags().IntVarP(&maxResultSize, "max-items", "m", math.MaxInt32,
		"Maximum number of items to list")
	command.SetOutput(root.Writer())
	return command
}

func NewRunTerminateCmd(root *RootCommand) *cobra.Command {
	var (
		runID string
		err   error
	)
	var command = &cobra.Command{
		Use:   "terminate ID",
		Short: "Terminate a run",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			runID, err = ValidateSingleString(args, "ID")
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewTerminateRunParams()
			params.RunID = runID
			err := root.RunClient().Terminate(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), "Run was terminated successfully.")
			return nil
		},
	}
	command.SetOutput(root.Writer())
	return command
}

type WorkflowForDisplay struct {
	Workflow *workflowapi.Workflow `json:"workflow,omitempty"`
}
