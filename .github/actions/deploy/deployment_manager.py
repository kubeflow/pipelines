#!/usr/bin/env python3
"""Kubernetes Deployment Management Utilities.

This module provides a generic deployment manager for Kubernetes
resources with built-in waiting, timeout handling, and debugging
capabilities.
"""

from enum import Enum
import os
import subprocess
import tempfile
from typing import List, Union


class WaitCondition(Enum):
    """Supported wait conditions for Kubernetes resources."""
    AVAILABLE = 'Available=true'
    READY = 'Ready'
    COMPLETE = 'Complete'
    ESTABLISHED = 'Established'


class ResourceType(Enum):
    """Supported Kubernetes resource types."""
    DEPLOYMENT = 'deployment'
    POD = 'pod'
    JOB = 'job'
    CERTIFICATE = 'certificate'
    STATEFULSET = 'statefulset'
    DAEMONSET = 'daemonset'
    SERVICE = 'service'
    CONFIGMAP = 'configmap'
    SECRET = 'secret'


class K8sDeploymentManager:
    """Generic Kubernetes deployment manager with apply, wait, and debug
    capabilities."""

    def __init__(self):
        """Initialize deployment manager."""
        pass

    def run_command(self,
                    cmd: List[str],
                    cwd: str = None,
                    check: bool = True,
                    timeout: int = None,
                    env: dict = None) -> subprocess.CompletedProcess:
        """Run shell command with streaming output to avoid memory issues."""
        cmd_str = ' '.join(cmd)
        print(f'🚀 Running: {cmd_str}')
        if cwd:
            print(f'📂 Working directory: {cwd}')
        if timeout:
            print(f'⏱️  Timeout: {timeout} seconds')

        process = None
        try:
            # Use Popen for streaming output instead of run() to avoid memory issues
            process = subprocess.Popen(
                cmd,
                cwd=cwd,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,  # Merge stderr into stdout
                text=True,
                bufsize=1,  # Line buffered
                universal_newlines=True,
                env=env)

            # Iterate over the process's stdout line by line in real time
            output_lines = []
            for line in process.stdout:
                print(
                    line, end='',
                    flush=True)  # Use flush=True to ensure immediate printing
                output_lines.append(line.rstrip())

            # Wait for the process to finish and get the return code
            return_code = process.wait(timeout=timeout)

            # Create a mock CompletedProcess for compatibility
            result = subprocess.CompletedProcess(
                args=cmd,
                returncode=return_code,
                stdout='\n'.join(output_lines),
                stderr=''  # Already merged into stdout
            )

            if check and return_code != 0:
                raise subprocess.CalledProcessError(
                    return_code, cmd, output=result.stdout)

            return result

        except subprocess.TimeoutExpired as e:
            print(f'⏰ Command timed out after {timeout} seconds')
            print(f'❌ Timeout command: {cmd_str}')
            raise
        except subprocess.CalledProcessError as e:
            # Error details already streamed during execution
            print(f'❌ Command failed with exit code {e.returncode}')
            raise
        except Exception as e:
            print(f'❌ Unexpected error running command: {e}')
            if process is not None:
                if process.poll() is None:  # Check if process is still running
                    process.kill()
                process.wait()
            raise

    def apply_resource(
        self,
        manifest_path: str = None,
        manifest_content: str = None,
        namespace: str = None,
        kustomize: bool = False,
        description: str = 'Kubernetes resource'
    ) -> subprocess.CompletedProcess:
        """Apply Kubernetes resource from file, content, or kustomize.

        Args:
            manifest_path: Path to manifest file or directory
            manifest_content: Raw YAML content to apply
            namespace: Namespace to apply in (optional)
            kustomize: Whether to use 'kubectl apply -k' for kustomize
            description: Human-readable description for logging

        Returns:
            CompletedProcess result from kubectl apply
        """
        print(f'📦 Applying {description}...')

        cmd = ['kubectl']
        temp_file = None

        if namespace:
            cmd.extend(['-n', namespace])

        cmd.append('apply')

        try:
            if kustomize:
                if not manifest_path:
                    raise ValueError(
                        'manifest_path required when using kustomize')
                cmd.extend(['-k', manifest_path])
            elif manifest_content:
                # Create temporary file for content
                temp_file = self._create_temp_file(manifest_content)
                cmd.extend(['-f', temp_file])
            elif manifest_path:
                cmd.extend(['-f', manifest_path])
            else:
                raise ValueError(
                    'Either manifest_path or manifest_content must be provided')

            result = self.run_command(cmd)
            print(f'✅ {description} applied successfully')
            return result
        except subprocess.CalledProcessError as e:
            print(f'❌ Failed to apply {description}: {e}')
            raise
        finally:
            # Clean up temporary file if created
            if temp_file:
                self.cleanup_temp_files(temp_file)

    def wait_for_resource_to_exist(self,
                                   resource_type: Union[ResourceType, str],
                                   resource_name: str,
                                   namespace: str = None,
                                   timeout_seconds: int = 300,
                                   description: str = None) -> bool:
        """Wait for a Kubernetes resource to exist.

        Args:
            resource_type: Type of resource (deployment, pod, etc.)
            resource_name: Specific resource name
            namespace: Namespace to check
            timeout_seconds: Timeout in seconds
            description: Human-readable description for logging

        Returns:
            True if resource exists within timeout, False otherwise
        """
        if isinstance(resource_type, ResourceType):
            resource_type = resource_type.value

        if not description:
            description = f'{resource_type} {resource_name}'

        print(
            f'⏳ Waiting for {description} to exist (timeout: {timeout_seconds}s)...'
        )

        import time
        start_time = time.time()

        while time.time() - start_time < timeout_seconds:
            # Check if resource exists
            cmd = ['kubectl', 'get', resource_type, resource_name]
            if namespace:
                cmd.extend(['-n', namespace])
            cmd.append('--no-headers')

            result = self.run_command(cmd, check=False)

            if result.returncode == 0:
                print(f'✅ {description} exists')
                return True

            print(
                f'⏳ {description} not found, waiting... ({int(time.time() - start_time)}s elapsed)'
            )
            time.sleep(5)  # Wait 5 seconds before checking again

        print(
            f'❌ {description} did not exist within {timeout_seconds}s timeout')
        return False

    def wait_for_resource(self,
                          resource_type: Union[ResourceType, str],
                          resource_name: str = None,
                          namespace: str = None,
                          condition: Union[WaitCondition,
                                           str] = WaitCondition.AVAILABLE,
                          timeout: str = '300s',
                          selector: str = None,
                          all_resources: bool = False,
                          description: str = None) -> bool:
        """Wait for Kubernetes resource to reach desired condition.

        Args:
            resource_type: Type of resource (deployment, pod, etc.)
            resource_name: Specific resource name (optional if using selector or all)
            namespace: Namespace to check
            condition: Condition to wait for
            timeout: Timeout string (e.g., "300s", "10m")
            selector: Label selector for multiple resources
            all_resources: Wait for all resources of type
            description: Human-readable description for logging

        Returns:
            True if wait succeeded, False if failed
        """
        if isinstance(resource_type, ResourceType):
            resource_type = resource_type.value
        if isinstance(condition, WaitCondition):
            condition = condition.value

        if not description:
            desc_parts = [resource_type]
            if resource_name:
                desc_parts.append(resource_name)
            elif selector:
                desc_parts.append(f'with selector {selector}')
            elif all_resources:
                desc_parts.append('(all)')
            description = ' '.join(desc_parts)

        print(
            f'⏳ Waiting for {description} to be {condition} (timeout: {timeout})...'
        )

        cmd = ['kubectl', 'wait', f'--for=condition={condition}']

        if namespace:
            cmd.extend(['-n', namespace])

        cmd.extend([f'--timeout={timeout}'])

        # Build resource specification
        if all_resources:
            cmd.append(f'{resource_type}')
            cmd.append('--all')
        elif selector:
            cmd.append(f'{resource_type}')
            cmd.extend(['-l', selector])
        elif resource_name:
            cmd.append(f'{resource_type}/{resource_name}')
        else:
            raise ValueError(
                'Must specify resource_name, selector, or all_resources=True')

        try:
            result = self.run_command(cmd, check=False)
            if result.returncode == 0:
                print(f'✅ {description} is ready')
                return True
            else:
                print(f'⚠️  {description} did not become ready within timeout')
                return False
        except Exception as e:
            print(f'❌ Error waiting for {description}: {e}')
            return False

    def debug_deployment_failure(self,
                                 namespace: str,
                                 deployment_name: str = None,
                                 resource_type: Union[
                                     ResourceType,
                                     str] = ResourceType.DEPLOYMENT,
                                 selector: str = None,
                                 include_events: bool = True,
                                 include_logs: bool = True,
                                 log_tail_lines: int = 50) -> None:
        """Debug deployment failure with comprehensive investigation.

        Args:
            namespace: Namespace to investigate
            deployment_name: Specific deployment name (optional)
            resource_type: Type of resource to investigate
            selector: Label selector for finding related pods
            include_events: Whether to include namespace events
            include_logs: Whether to include pod logs
            log_tail_lines: Number of log lines to show
        """
        if isinstance(resource_type, ResourceType):
            resource_type = resource_type.value

        print(
            f'🔍 Investigating {resource_type} failure in namespace: {namespace}'
        )
        if deployment_name:
            print(f'🔍 Context: {resource_type} {deployment_name} failed')

        # Get all pods in namespace
        print('🔍 All pods in namespace:')
        self.run_command(
            ['kubectl', 'get', 'pods', '-n', namespace, '-o', 'wide'],
            check=False)

        # Get pod details for investigation
        self._investigate_pods(namespace, deployment_name, selector,
                               include_logs, log_tail_lines)

        # Get resource status if deployment name provided
        if deployment_name:
            self._investigate_resource_status(namespace, deployment_name,
                                              resource_type)

        # Get recent events if requested
        if include_events:
            print('🔍 Recent events in namespace:')
            self.run_command([
                'kubectl', 'get', 'events', '-n', namespace,
                '--sort-by=.lastTimestamp'
            ],
                             check=False)

    def deploy_and_wait(self,
                        manifest_path: str = None,
                        manifest_content: str = None,
                        namespace: str = None,
                        kustomize: bool = False,
                        resource_type: Union[ResourceType,
                                             str] = ResourceType.DEPLOYMENT,
                        resource_name: str = None,
                        wait_condition: Union[WaitCondition,
                                              str] = WaitCondition.AVAILABLE,
                        wait_timeout: str = '300s',
                        wait_selector: str = None,
                        wait_all: bool = False,
                        debug_on_failure: bool = True,
                        description: str = 'resource') -> bool:
        """Apply resource and wait for it to be ready, with optional debugging.

        Args:
            manifest_path: Path to manifest file or directory
            manifest_content: Raw YAML content to apply
            namespace: Namespace to apply in
            kustomize: Whether to use kustomize
            resource_type: Type of resource to wait for
            resource_name: Name of resource to wait for
            wait_condition: Condition to wait for
            wait_timeout: Wait timeout
            wait_selector: Label selector for waiting
            wait_all: Wait for all resources of type
            debug_on_failure: Whether to debug on wait failure
            description: Human-readable description

        Returns:
            True if deployment and wait succeeded, False otherwise
        """
        try:
            # Apply the resource
            self.apply_resource(
                manifest_path=manifest_path,
                manifest_content=manifest_content,
                namespace=namespace,
                kustomize=kustomize,
                description=description)

            # Wait for resource to be ready
            wait_success = self.wait_for_resource(
                resource_type=resource_type,
                resource_name=resource_name,
                namespace=namespace,
                condition=wait_condition,
                timeout=wait_timeout,
                selector=wait_selector,
                all_resources=wait_all,
                description=f'{description} readiness')

            if not wait_success and debug_on_failure and namespace:
                print(
                    f'🔍 {description} failed to become ready, investigating...')
                self.debug_deployment_failure(
                    namespace=namespace,
                    deployment_name=resource_name,
                    resource_type=resource_type,
                    selector=wait_selector)

            return wait_success

        except Exception as e:
            print(f'❌ Failed to deploy {description}: {e}')
            if debug_on_failure and namespace:
                print(
                    f'🔍 Exception during {description} deployment, investigating...'
                )
                self.debug_deployment_failure(
                    namespace=namespace,
                    deployment_name=resource_name,
                    resource_type=resource_type,
                    selector=wait_selector)
            raise

    def _investigate_pods(self,
                          namespace: str,
                          deployment_name: str = None,
                          selector: str = None,
                          include_logs: bool = True,
                          log_tail_lines: int = 50) -> None:
        """Investigate pod status and logs."""
        # Build pod query
        pod_cmd = [
            'kubectl', 'get', 'pods', '-n', namespace, '--no-headers', '-o',
            'custom-columns=NAME:.metadata.name,STATUS:.status.phase'
        ]

        if selector:
            pod_cmd.extend(['-l', selector])
        elif deployment_name:
            # Try to find pods by deployment name
            pod_cmd.extend(['-l', f'app={deployment_name}'])

        pod_result = self.run_command(pod_cmd, check=False)

        if pod_result.returncode == 0 and pod_result.stdout.strip():
            failed_pods = []
            running_pods = []

            for line in pod_result.stdout.strip().split('\n'):
                if line.strip():
                    parts = line.strip().split()
                    if len(parts) >= 2:
                        pod_name = parts[0]
                        status = parts[1]

                        if status not in ['Running', 'Succeeded']:
                            failed_pods.append((pod_name, status))
                        elif status == 'Running':
                            running_pods.append((pod_name, status))

            # Investigate failed pods
            if failed_pods:
                print(f'🔍 Found {len(failed_pods)} failed/pending pods')
                for pod_name, status in failed_pods:
                    self._investigate_single_pod(namespace, pod_name, status,
                                                 include_logs, log_tail_lines)
            else:
                print('🔍 No failed/pending pods found')
                if running_pods and include_logs:
                    print(
                        f'🔍 Getting recent logs from {len(running_pods)} running pods...'
                    )
                    for pod_name, status in running_pods[:
                                                         3]:  # Limit to first 3
                        self._get_pod_logs(namespace, pod_name, 30, 'recent')
        else:
            # Fallback: Get all pods in namespace if label selector didn't work
            print(f'🔍 No pods found with selector, checking all pods in namespace {namespace}')
            self._investigate_all_pods_in_namespace(namespace, include_logs, log_tail_lines)

    def _investigate_single_pod(self,
                                namespace: str,
                                pod_name: str,
                                status: str,
                                include_logs: bool = True,
                                log_tail_lines: int = 50) -> None:
        """Investigate a single pod in detail."""
        print(f'🔍 Investigating {status} pod: {pod_name}')

        # Describe the pod
        print(f'🔍 Describing pod: {pod_name}')
        self.run_command(
            ['kubectl', 'describe', 'pod', pod_name, '-n', namespace],
            check=False)

        if include_logs:
            # Get current logs
            self._get_pod_logs(namespace, pod_name, log_tail_lines, 'current')
            # Get previous logs if available
            self._get_pod_logs(
                namespace,
                pod_name,
                min(log_tail_lines, 30),
                'previous',
                previous=True)

    def _get_pod_logs(self,
                      namespace: str,
                      pod_name: str,
                      tail_lines: int,
                      log_type: str = 'current',
                      previous: bool = False) -> None:
        """Get logs from a pod, handling multi-container pods."""
        print(f'🔍 {log_type.title()} logs from pod: {pod_name}')

        # First, check if this pod has multiple containers
        containers_result = self.run_command([
            'kubectl', 'get', 'pod', pod_name, '-n', namespace,
            '-o', 'jsonpath={.spec.containers[*].name}'
        ], check=False)

        if containers_result.returncode == 0 and containers_result.stdout.strip():
            containers = containers_result.stdout.strip().split()

            if len(containers) > 1:
                print(f'🔍 Pod {pod_name} has {len(containers)} containers')
                for container in containers:
                    print(f'🔍 {log_type.title()} logs from container {container} in pod {pod_name}:')
                    cmd = [
                        'kubectl', 'logs', pod_name, '-c', container, '-n', namespace,
                        '--tail', str(tail_lines)
                    ]
                    if previous:
                        cmd.append('--previous')
                    self.run_command(cmd, check=False)
            else:
                # Single container pod
                cmd = [
                    'kubectl', 'logs', pod_name, '-n', namespace, '--tail',
                    str(tail_lines)
                ]
                if previous:
                    cmd.append('--previous')
                self.run_command(cmd, check=False)
        else:
            # Fallback to simple logs command
            cmd = [
                'kubectl', 'logs', pod_name, '-n', namespace, '--tail',
                str(tail_lines)
            ]
            if previous:
                cmd.append('--previous')
            self.run_command(cmd, check=False)

    def _investigate_all_pods_in_namespace(self,
                                          namespace: str,
                                          include_logs: bool = True,
                                          log_tail_lines: int = 50) -> None:
        """Investigate all pods in a namespace as fallback."""
        print(f'🔍 Investigating all pods in namespace {namespace}')

        # Get all pods in the namespace
        pod_result = self.run_command([
            'kubectl', 'get', 'pods', '-n', namespace, '--no-headers', '-o',
            'custom-columns=NAME:.metadata.name,STATUS:.status.phase'
        ], check=False)

        if pod_result.returncode == 0 and pod_result.stdout.strip():
            failed_pods = []
            running_pods = []

            for line in pod_result.stdout.strip().split('\n'):
                if line.strip():
                    parts = line.strip().split()
                    if len(parts) >= 2:
                        pod_name = parts[0]
                        status = parts[1]

                        if status not in ['Running', 'Succeeded']:
                            failed_pods.append((pod_name, status))
                        elif status == 'Running':
                            running_pods.append((pod_name, status))

            # Investigate failed pods first
            if failed_pods:
                print(f'🔍 Found {len(failed_pods)} failed/pending pods in namespace {namespace}')
                for pod_name, status in failed_pods:
                    self._investigate_single_pod(namespace, pod_name, status,
                                                include_logs, log_tail_lines)

            # Also get logs from running pods if no failed pods or if specifically requested
            if running_pods and include_logs:
                print(f'🔍 Found {len(running_pods)} running pods in namespace {namespace}')
                # Get logs from a few running pods for context
                for pod_name, status in running_pods[:5]:  # Limit to first 5
                    self._get_pod_logs(namespace, pod_name, 30, 'recent')
        else:
            print(f'🔍 No pods found in namespace {namespace}')

    def _investigate_resource_status(self, namespace: str, resource_name: str,
                                     resource_type: str) -> None:
        """Get detailed status of a specific resource."""
        print(f'🔍 Checking {resource_type} status: {resource_name}')

        # Get resource details
        self.run_command([
            'kubectl', 'get', resource_type, resource_name, '-n', namespace,
            '-o', 'wide'
        ],
                         check=False)

        # Describe the resource
        self.run_command([
            'kubectl', 'describe', resource_type, resource_name, '-n', namespace
        ],
                         check=False)

    def _create_temp_file(self, content: str) -> str:
        """Create temporary file with content."""
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write(content)
            return f.name

    def cleanup_temp_files(self, *file_paths):
        """Clean up temporary files.

        Args:
            *file_paths: Variable number of file paths to delete
        """
        for file_path in file_paths:
            try:
                if file_path and os.path.exists(file_path):
                    os.unlink(file_path)
                    print(f'🧹 Cleaned up temporary file: {file_path}')
            except Exception as e:
                print(f'⚠️  Failed to clean up temporary file {file_path}: {e}')
