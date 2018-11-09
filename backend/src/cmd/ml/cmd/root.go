package cmd

import (
	"fmt"
	"io"
	"os"

	client "github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type RootCommand struct {
	command              *cobra.Command
	outputFormat         string
	debug                bool
	clientConfig         clientcmd.ClientConfig
	pipelineUploadClient client.PipelineUploadInterface
	pipelineClient       client.PipelineInterface
	jobClient            client.JobInterface
	runClient            client.RunInterface
	time                 util.TimeInterface
	writer               io.Writer
}

func NewRootCmd(factory ClientFactoryInterface) *RootCommand {
	root := &RootCommand{}
	root.time = factory.Time()
	root.writer = factory.Writer()
	command := &cobra.Command{
		Use:   "ml",
		Short: "This is the command line interface to manipulate pipelines.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			pipelineUploadClient, err := factory.CreatePipelineUploadClient(root.ClientConfig(),
				root.debug)
			if err != nil {
				return fmt.Errorf("Could not create pipelineUploadClient: %v", err)
			}

			pipelineClient, err := factory.CreatePipelineClient(root.ClientConfig(), root.debug)
			if err != nil {
				fmt.Errorf("Could not create pipelineClient: %v", err)
			}

			jobClient, err := factory.CreateJobClient(root.ClientConfig(), root.debug)
			if err != nil {
				fmt.Errorf("Could not create jobClient: %v", err)
			}

			runClient, err := factory.CreateRunClient(root.ClientConfig(), root.debug)
			if err != nil {
				fmt.Errorf("Could not create runClient: %v", err)
			}
			root.pipelineUploadClient = pipelineUploadClient
			root.pipelineClient = pipelineClient
			root.jobClient = jobClient
			root.runClient = runClient
			return nil
		},
	}
	command.SetOutput(factory.Writer())
	root.command = command
	root.command.SilenceErrors = true
	root.command.SilenceUsage = true
	addStandardFlagsToCmd(command, &root.outputFormat, &root.debug)
	root.clientConfig = addKubectlFlagsToCmd(command)
	return root
}

func addStandardFlagsToCmd(cmd *cobra.Command, outputFormat *string, debug *bool) {
	cmd.PersistentFlags().StringVarP(outputFormat, "output", "o", "yaml",
		"Output format. One of: json|yaml|go")
	cmd.PersistentFlags().BoolVarP(debug, "debug", "d", false,
		"Enable debug mode")
}

func addKubectlFlagsToCmd(cmd *cobra.Command) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "",
		"Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules,
		&overrides, os.Stdin)
	return clientConfig
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (r *RootCommand) Execute() {
	if err := r.command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (r *RootCommand) Command() *cobra.Command {
	return r.command
}

func (r *RootCommand) OutputFormat() string {
	return r.outputFormat
}

func (r *RootCommand) Debug() bool {
	return r.debug
}

func (r *RootCommand) ClientConfig() clientcmd.ClientConfig {
	return r.clientConfig
}

func (r *RootCommand) PipelineUploadClient() client.PipelineUploadInterface {
	return r.pipelineUploadClient
}

func (r *RootCommand) PipelineClient() client.PipelineInterface {
	return r.pipelineClient
}

func (r *RootCommand) JobClient() client.JobInterface {
	return r.jobClient
}

func (r *RootCommand) RunClient() client.RunInterface {
	return r.runClient
}

func (r *RootCommand) Time() util.TimeInterface {
	return r.time
}

func (r *RootCommand) Writer() io.Writer {
	return r.writer
}

func (r *RootCommand) AddCommand(commands ...*cobra.Command) {
	r.command.AddCommand(commands...)
}

func CreateSubCommands(rootCmd *RootCommand, pageSize int32) *RootCommand {

	// Create commands
	pipelineCmd := NewPipelineCmd()
	pipelineUploadCmd := NewPipelineUploadCmd(rootCmd)
	pipelineCreateCmd := NewPipelineCreateCmd(rootCmd)
	pipelineListCmd := NewPipelineListCmd(rootCmd, pageSize)
	pipelineDeleteCmd := NewPipelineDeleteCmd(rootCmd)
	pipelineGetCmd := NewPipelineGetCmd(rootCmd)
	pipelineGetTemplateCmd := NewPipelineGetTemplateCmd(rootCmd)
	runCmd := NewRunCmd()
	runGetCmd := NewRunGetCmd(rootCmd)
	runListCmd := NewRunListCmd(rootCmd, pageSize)
	jobCmd := NewJobCmd()
	jobCreateCmd := NewJobCreateCmd(rootCmd)
	jobGetCmd := NewJobGetCmd(rootCmd)
	jobListCmd := NewJobListCmd(rootCmd, pageSize)
	jobEnableCmd := NewJobEnableCmd(rootCmd)
	jobDisableCmd := NewJobDisableCmd(rootCmd)
	jobDeleteCmd := NewJobDeleteCmd(rootCmd)

	// Specify subcommands
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(pipelineUploadCmd, pipelineCreateCmd, pipelineListCmd, pipelineDeleteCmd,
		pipelineGetCmd, pipelineGetTemplateCmd)
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runGetCmd, runListCmd)
	rootCmd.AddCommand(jobCmd)
	jobCmd.AddCommand(jobCreateCmd, jobGetCmd, jobListCmd, jobEnableCmd,
		jobDisableCmd, jobDeleteCmd)

	return rootCmd
}
