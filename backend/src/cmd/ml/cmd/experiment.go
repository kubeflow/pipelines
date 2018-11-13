package cmd

import (
	"fmt"
	"math"

	params "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_client/experiment_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/experiment_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type experimentCreateParams struct {
	name        string
	description string
}

type experimentCreateParamsValidated struct {
	name        string
	description string
}

func NewExperimentCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "experiment",
		Short: "Manages experiments",
	}
	return command
}

func NewExperimentCreateCmd(root *RootCommand) *cobra.Command {
	var (
		raw       experimentCreateParams
		validated experimentCreateParamsValidated
	)
	const (
		flagNameName        = "name"
		flagNameDescription = "description"
	)
	var command = &cobra.Command{
		Use:   "create",
		Short: "Create a experiment",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			if err != nil {
				return err
			}
			validated.name = raw.name               // Validation in API Server
			validated.description = raw.description // Validation in API Server
			return nil
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := newCreateExperimentParams(&validated)
			experiment, err := root.ExperimentClient().Create(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), experiment)
			return nil
		},
	}

	// Flags
	command.PersistentFlags().StringVar(&raw.name, flagNameName,
		"default", "The name of the experiment")
	command.PersistentFlags().StringVar(&raw.description, flagNameDescription,
		"No description provided", "A description of the experiment")

	command.SetOutput(root.Writer())
	return command
}

func newCreateExperimentParams(validated *experimentCreateParamsValidated) *params.CreateExperimentParams {
	result := params.NewCreateExperimentParams()
	result.Body = &model.APIExperiment{
		Name:        validated.name,
		Description: validated.description,
	}
	return result
}

func NewExperimentGetCmd(root *RootCommand) *cobra.Command {
	var (
		id string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "get",
		Short: "Display an experiment",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewGetExperimentParams()
			params.ID = id
			pkg, err := root.ExperimentClient().Get(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), pkg)
			return nil
		},
	}

	// Flags
	command.PersistentFlags().StringVar(&id, flagNameID,
		"default", "The id of the experiment")
	command.MarkPersistentFlagRequired(flagNameID)

	command.SetOutput(root.Writer())
	return command
}

func NewExperimentListCmd(root *RootCommand, pageSize int32) *cobra.Command {
	var (
		maxResultSize int
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "List all experiments",

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
			params := params.NewListExperimentParams()
			params.PageSize = util.Int32Pointer(pageSize)
			results, err := root.ExperimentClient().ListAll(params, maxResultSize)
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
