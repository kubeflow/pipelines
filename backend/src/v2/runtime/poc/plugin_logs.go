package poc

import (
	"context"
	"fmt"
	"io"
	"os/exec"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dockerLogCommand func(context.Context, io.Writer, ...string) error

var (
	_ ExecutionLogReader = dockerExecutor{}
	_ ExecutionLogReader = kubernetesExecutor{}
)

func (d dockerExecutor) ReadLogs(ctx context.Context, reference ExecutionReference, follow bool, destination io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if destination == nil {
		return fmt.Errorf("a destination writer is required for execution logs")
	}
	data, err := parseDockerReference(reference)
	if err != nil {
		return err
	}
	container, exists, err := d.inspect(ctx, data.ContainerID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("docker execution container is missing; retained logs are unavailable")
	}
	if err := checkDockerIdentity(container, reference.Identity, data.ContainerID); err != nil {
		return err
	}
	args := []string{"logs"}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, data.ContainerID)
	if d.logCommand != nil {
		err = d.logCommand(ctx, destination, args...)
	} else {
		command := exec.CommandContext(ctx, "docker", args...)
		command.Stdout, command.Stderr = destination, destination
		err = command.Run()
	}
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("cannot stream retained Docker execution logs: %w", err)
	}
	return nil
}

func (k kubernetesExecutor) ReadLogs(ctx context.Context, reference ExecutionReference, follow bool, destination io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if destination == nil {
		return fmt.Errorf("a destination writer is required for execution logs")
	}
	if err := k.CheckRequirements(ctx); err != nil {
		return err
	}
	data, err := parsePodReference(reference)
	if err != nil {
		return err
	}
	client := k.coreClient.PodClient(data.Namespace)
	pod, err := client.Get(ctx, data.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err := checkReferencedPod(pod, reference, data); err != nil {
		return err
	}
	options := &corev1.PodLogOptions{Follow: follow}
	if len(pod.Spec.Containers) != 0 {
		options.Container = pod.Spec.Containers[0].Name
	}
	logRequest := client.GetLogs(data.PodName, options)
	if logRequest == nil {
		return fmt.Errorf("kubernetes client returned no execution log request")
	}
	stream, err := logRequest.Stream(ctx)
	if err != nil {
		return fmt.Errorf("cannot open retained Kubernetes execution logs: %w", err)
	}
	defer stream.Close()
	// Revalidate after opening the name-addressed log stream to reject a replacement race.
	current, err := client.Get(ctx, data.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if err := checkReferencedPod(current, reference, data); err != nil {
		return err
	}
	if _, err := io.Copy(destination, stream); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("cannot stream retained Kubernetes execution logs: %w", err)
	}
	return nil
}
