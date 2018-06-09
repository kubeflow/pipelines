package commands

import (
	"k8s.io/client-go/tools/clientcmd"
	"github.com/spf13/cobra"
)

var (
	clientConfig clientcmd.ClientConfig
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use: "pipelines",
		Short: "pipelines is the command line for Pipelines",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewInstallCommand())
	addKubectlFlagsToCmd(command)
	return command
}

func addKubectlFlagsToCmd(cmd *cobra.Command) {
	// The "usual" clientcmd/kubectl flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &overrides)
}
