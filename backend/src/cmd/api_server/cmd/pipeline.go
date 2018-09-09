package cmd

import (
	"fmt"
	"math"

	params "github.com/googleprivate/ml/backend/api/pipeline_client/pipeline_service"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/spf13/cobra"
)

func NewPipelineCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "pipeline",
		Short: "Manage pipelines",
	}
	return command
}

func NewPipelineUploadCmd(root *RootCommand) *cobra.Command {
	var (
		filename string
		err      error
	)
	var command = &cobra.Command{
		Use:   "upload FILE",
		Short: "Upload a pipeline",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			filename, err = ValidateSingleString(args, "FILE")
			return err
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, err := root.PipelineUploadClient().Upload(filename)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}

			PrettyPrintResult(root.Writer(), root.NoColor(), root.OutputFormat(), pkg)
			return nil
		},
	}
	command.SetOutput(root.Writer())
	return command
}

func NewPipelineGetCmd(root *RootCommand) *cobra.Command {
	var (
		id  string
		err error
	)
	var command = &cobra.Command{
		Use:   "get ID",
		Short: "Display a pipeline",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			id, err = ValidateSingleString(args, "ID")
			return err
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewGetPipelineParams()
			params.ID = id
			pkg, err := root.PipelineClient().Get(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.NoColor(), root.OutputFormat(), pkg)
			return nil
		},
	}
	command.SetOutput(root.Writer())
	return command
}

func NewPipelineListCmd(root *RootCommand, pageSize int32) *cobra.Command {
	var (
		maxResultSize int
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "List all pipelines",

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
			params := params.NewListPipelinesParams()
			params.PageSize = util.Int32Pointer(pageSize)
			results, err := root.PipelineClient().ListAll(params, maxResultSize)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.NoColor(), root.OutputFormat(), results)
			return nil
		},
	}
	command.PersistentFlags().IntVarP(&maxResultSize, "max-items", "m", math.MaxInt32,
		"Maximum number of items to list")
	command.SetOutput(root.Writer())
	return command
}

func NewPipelineDeleteCmd(root *RootCommand) *cobra.Command {
	var (
		id  string
		err error
	)
	var command = &cobra.Command{
		Use:   "delete ID",
		Short: "Delete a pipeline",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			id, err = ValidateSingleString(args, "ID")
			return err
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewDeletePipelineParams()
			params.ID = id
			err := root.PipelineClient().Delete(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.NoColor(), root.OutputFormat(), "")
			return nil
		},
	}
	command.SetOutput(root.Writer())
	return command
}

func NewPipelineGetTemplateCmd(root *RootCommand) *cobra.Command {
	var (
		id  string
		err error
	)
	var command = &cobra.Command{
		Use:   "get-manifest ID",
		Short: "Display the manifest of a pipeline",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			id, err = ValidateSingleString(args, "ID")
			return err
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewGetTemplateParams()
			params.ID = id
			workflow, err := root.PipelineClient().GetTemplate(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.NoColor(), root.OutputFormat(), workflow)
			return nil
		},
	}
	command.SetOutput(root.Writer())
	return command

}
