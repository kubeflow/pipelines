package commands

import (
	"github.com/kubeflow/pipelines/install"
	"github.com/spf13/cobra"
)

var (
	project  = "pipelines"
	imageTag = "latest"
	DefaultNamespace       = "kube-system"
	DefaultControllerImage = project + "/scheduledworkflow-controller:" + imageTag
)

func NewInstallCommand() *cobra.Command {
	var installOption install.InstallOption
	command := &cobra.Command{
		Use: "install",
		Short: "install ScheduledWorkflow components",
		RunE: func(cmd *cobra.Command, args []string) error {
			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}
			installer, err := install.NewInstaller(restConfig, installOption)
			if err != nil {
				return err
			}
			installer.Install()
			return nil
		},
	}
	command.Flags().StringVar(&installOption.Namespace, "namespace", DefaultNamespace,
		"use a specified namespace to deploy controller")
	command.Flags().StringVar(&installOption.ControllerImage, "controller-image", DefaultControllerImage,
		"use a specified controller image")
	command.Flags().StringVar(&installOption.ImagePullPolicy, "image-pull-policy", "",
		"imagePullPolicy for deployments")
	return command
}
