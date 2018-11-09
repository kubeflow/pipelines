package cmd

import (
	"fmt"
	"math"
	"time"

	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/go_http_client/job_client/job_service"
	model "github.com/kubeflow/pipelines/backend/api/go_http_client/job_model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

type jobCreateParams struct {
	name           string
	description    string
	disable        bool
	maxConcurrency int64
	pipelineId     string
	startTime      string
	endTime        string
	cron           string
	period         time.Duration
	parameters     []string
}

type jobCreateParamsValidated struct {
	name           string
	description    string
	enabled        bool
	maxConcurrency int64
	pipelineId     string
	startTime      *strfmt.DateTime
	endTime        *strfmt.DateTime
	cron           *string
	intervalSecond *int64
	parameters     map[string]string
}

func NewJobCmd() *cobra.Command {
	var command = &cobra.Command{
		Use:   "job",
		Short: "Manages jobs",
	}
	return command
}

func NewJobCreateCmd(root *RootCommand) *cobra.Command {
	var (
		raw       jobCreateParams
		validated jobCreateParamsValidated
	)
	const (
		flagNameName           = "name"
		flagNameDescription    = "description"
		flagNameDisable        = "disable"
		flagNameMaxConcurrency = "max-concurrency"
		flagNamePipelineID     = "pipeline-id"
		flagNameParameter      = "parameter"
		flagNameStartTime      = "start-time"
		flagNameEndTime        = "end-time"
		flagNameCron           = "cron"
		flagNamePeriod         = "period"
	)
	var command = &cobra.Command{
		Use:   "create",
		Short: "Create a job",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			if err != nil {
				return err
			}
			validated.name = raw.name               // Validation in API Server
			validated.description = raw.description // Validation in API Server
			validated.enabled = !raw.disable        // No validation needed
			validated.maxConcurrency, err = ValidateInt64Min(raw.maxConcurrency, 1,
				flagNameMaxConcurrency)
			if err != nil {
				return err
			}
			validated.pipelineId = raw.pipelineId // Validation by flags & API Server

			validated.startTime, err = ValidateStartTime(raw.startTime, flagNameStartTime,
				root.Time())
			if err != nil {
				return err
			}
			validated.endTime, err = ValidateEndTime(raw.endTime, flagNameEndTime)
			if err != nil {
				return err
			}
			validated.cron, err = ValidateCron(raw.cron, flagNameCron)
			if err != nil {
				return err
			}
			validated.intervalSecond = ValidateDuration(raw.period)
			validated.parameters, err = ValidateParams(raw.parameters)
			if err != nil {
				return err
			}

			// Validate combinations of arguments
			if validated.intervalSecond != nil && validated.cron != nil {
				return fmt.Errorf("Only one of period|cron can be specified")
			}

			return nil
		},

		// Execution
		RunE: func(cmd *cobra.Command, args []string) error {
			params := newCreateJobParams(&validated)
			job, err := root.JobClient().Create(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), job)
			return nil
		},
	}

	// Flags
	command.PersistentFlags().StringVar(&raw.name, flagNameName,
		"default", "The name of the job")
	command.PersistentFlags().StringVar(&raw.description, flagNameDescription,
		"No description provided", "A description of the job")
	command.PersistentFlags().BoolVar(&raw.disable, flagNameDisable, false,
		"Disable the job (it won't trigger any runs until it is enabled)")
	command.PersistentFlags().Int64Var(&raw.maxConcurrency, flagNameMaxConcurrency, 1,
		"The maximum number of concurrent runs")
	command.PersistentFlags().StringVar(&raw.pipelineId, flagNamePipelineID, "",
		"The ID of the pipeline used to create the job")
	command.MarkPersistentFlagRequired(flagNamePipelineID)
	// Note: Another example: "-p key=[[Index]]"
	command.Flags().StringArrayVarP(&raw.parameters, flagNameParameter, "p", []string{},
		"Provide an input parameter to the workflow (e.g.: '-p key=value', '-p key=[[ScheduledTime]]')")
	// Note: in strfmt format: https://godoc.org/github.com/go-openapi/strfmt
	command.Flags().StringVar(&raw.startTime, flagNameStartTime, "",
		"The start time of cron|periodic execution (e.g.: '2006-01-02T15:04:05.000Z'). Defaults to current time")
	// Note: in strfmt format: https://godoc.org/github.com/go-openapi/strfmt
	command.Flags().StringVar(&raw.endTime, flagNameEndTime, "",
		"The end time of a cron|periodic execution (e.g.: '2006-01-02T15:04:05.000Z'). Defaults to never")
	// Note: in the cron format detailed here: https://godoc.org/github.com/robfig/cron
	command.Flags().StringVar(&raw.cron, flagNameCron, "",
		"A cron schedule (e.g. '0 30 * * * *')")
	command.Flags().DurationVar(&raw.period, flagNamePeriod, 0*time.Second,
		"A period to trigger executions periodically")

	command.SetOutput(root.Writer())
	return command
}

func newCreateJobParams(validated *jobCreateParamsValidated) *params.CreateJobParams {
	result := params.NewCreateJobParams()
	result.Body = &model.APIJob{
		Name:           validated.name,
		Description:    validated.description,
		Enabled:        validated.enabled,
		MaxConcurrency: validated.maxConcurrency,
		PipelineSpec:   &model.APIPipelineSpec{PipelineID: validated.pipelineId},
	}

	var trigger *model.APITrigger

	if validated.cron != nil {
		trigger = &model.APITrigger{
			CronSchedule: &model.APICronSchedule{
				Cron:      *validated.cron,
				StartTime: *validated.startTime,
				EndTime:   *validated.endTime,
			},
		}
	} else if validated.intervalSecond != nil {
		trigger = &model.APITrigger{
			PeriodicSchedule: &model.APIPeriodicSchedule{
				IntervalSecond: *validated.intervalSecond,
				StartTime:      *validated.startTime,
				EndTime:        *validated.endTime,
			},
		}
	}

	if trigger != nil {
		result.Body.Trigger = trigger
	}

	result.Body.PipelineSpec.Parameters = toAPIParameters(validated.parameters)
	return result
}

func toAPIParameters(params map[string]string) []*model.APIParameter {
	result := make([]*model.APIParameter, 0)
	for key, value := range params {
		result = append(result, &model.APIParameter{
			Name:  key,
			Value: value,
		})
	}
	return result
}

func NewJobGetCmd(root *RootCommand) *cobra.Command {
	var (
		id string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "get",
		Short: "Display a job",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewGetJobParams()
			params.ID = id
			pkg, err := root.JobClient().Get(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), pkg)
			return nil
		},
	}
	command.PersistentFlags().StringVar(&id, flagNameID,
		"", "The id of the job")
	command.MarkPersistentFlagRequired(flagNameID)
	command.SetOutput(root.Writer())
	return command
}

func NewJobListCmd(root *RootCommand, pageSize int32) *cobra.Command {
	var (
		maxResultSize int
	)
	var command = &cobra.Command{
		Use:   "list",
		Short: "List all jobs",

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
			params := params.NewListJobsParams()
			params.PageSize = util.Int32Pointer(pageSize)
			results, err := root.JobClient().ListAll(params, maxResultSize)
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

func NewJobEnableCmd(root *RootCommand) *cobra.Command {
	var (
		id string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "enable",
		Short: "Enable a job",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewEnableJobParams()
			params.ID = id
			err := root.JobClient().Enable(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), "")
			return nil
		},
	}
	command.PersistentFlags().StringVar(&id, flagNameID,
		"", "The id of the job")
	command.MarkPersistentFlagRequired(flagNameID)
	command.SetOutput(root.Writer())
	return command
}

func NewJobDisableCmd(root *RootCommand) *cobra.Command {
	var (
		id string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "disable",
		Short: "Disable a job",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewDisableJobParams()
			params.ID = id
			err := root.JobClient().Disable(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), "")
			return nil
		},
	}
	command.PersistentFlags().StringVar(&id, flagNameID,
		"", "The id of the job")
	command.MarkPersistentFlagRequired(flagNameID)
	command.SetOutput(root.Writer())
	return command
}

func NewJobDeleteCmd(root *RootCommand) *cobra.Command {
	var (
		id string
	)
	const (
		flagNameID = "id"
	)
	var command = &cobra.Command{
		Use:   "delete",
		Short: "Delete a job",

		// Validation
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := ValidateArgumentCount(args, 0)
			return err
		},

		// Execute
		RunE: func(cmd *cobra.Command, args []string) error {
			params := params.NewDeleteJobParams()
			params.ID = id
			err := root.JobClient().Delete(params)
			if err != nil {
				return util.ExtractErrorForCLI(err, root.Debug())
			}
			PrettyPrintResult(root.Writer(), root.OutputFormat(), "")
			return nil
		},
	}
	command.PersistentFlags().StringVar(&id, flagNameID,
		"", "The id of the job")
	command.MarkPersistentFlagRequired(flagNameID)
	command.SetOutput(root.Writer())
	return command
}
