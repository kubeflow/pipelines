#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from collections.abc import Callable, Iterable
import subprocess
import time

PodIdentity = tuple[str, str]


def container_creating_pod_identities(
        pods: Iterable[object]) -> set[PodIdentity]:
    """Return pod identities with an init or regular container being created."""
    identities = set()
    for pod in pods:
        pod_status = getattr(pod, 'status', None)
        metadata = getattr(pod, 'metadata', None)
        if pod_status is None or metadata is None:
            continue
        container_statuses = (
            (getattr(pod_status, 'init_container_statuses', None) or []) +
            (getattr(pod_status, 'container_statuses', None) or []))
        for container_status in container_statuses:
            state = getattr(container_status, 'state', None)
            waiting = getattr(state, 'waiting', None)
            if getattr(waiting, 'reason', None) == 'ContainerCreating':
                identities.add((str(metadata.name), str(metadata.uid)))
                break
    return identities


class SlowContainerCreatingDiagnostics:
    """Capture one bounded snapshot for each sustained ContainerCreating episode."""

    def __init__(
        self,
        threshold_seconds: float = 60,
        clock: Callable[[], float] = time.monotonic,
        command_runner: Callable[..., subprocess.CompletedProcess[str]] = subprocess.run,
        output: Callable[[str], None] = print,
    ) -> None:
        self._threshold_seconds = threshold_seconds
        self._clock = clock
        self._command_runner = command_runner
        self._output = output
        self._first_seen: dict[PodIdentity, float] = {}
        self._reported: set[PodIdentity] = set()

    def observe(self, namespace: str,
                container_creating_pods: Iterable[PodIdentity]) -> None:
        """Observe the current stalled pods and report newly sustained stalls."""
        now = self._clock()
        current_pods = set(container_creating_pods)

        for pod in set(self._first_seen) - current_pods:
            self._first_seen.pop(pod, None)
            self._reported.discard(pod)

        for pod in current_pods:
            first_seen = self._first_seen.setdefault(pod, now)
            if (pod in self._reported or
                    now - first_seen < self._threshold_seconds):
                continue

            self._capture(namespace, *pod)
            # Do not repeatedly run diagnostics if kubectl itself is unhealthy.
            self._reported.add(pod)

    def _capture(self, namespace: str, pod_name: str, pod_uid: str) -> None:
        self._output(
            f'Pod {namespace}/{pod_name} has remained in ContainerCreating '
            f'for at least {self._threshold_seconds:g}s; collecting diagnostics.'
        )
        commands = (
            (
                'describe',
                [
                    'kubectl', '--request-timeout=10s', '-n', namespace,
                    'describe', 'pod', pod_name
                ],
            ),
            (
                'events',
                [
                    'kubectl', '--request-timeout=10s', '-n', namespace,
                    'get', 'events', '--field-selector',
                    f'involvedObject.uid={pod_uid}',
                    '--sort-by=.metadata.creationTimestamp'
                ],
            ),
        )

        for label, command in commands:
            self._output(f'----- {namespace}/{pod_name} {label} -----')
            try:
                result = self._command_runner(
                    command,
                    stdin=subprocess.DEVNULL,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.STDOUT,
                    text=True,
                    check=False,
                    timeout=15,
                )
                if result.stdout:
                    self._output(result.stdout.rstrip())
                if result.returncode != 0:
                    self._output(
                        f'WARNING: {label} diagnostics exited with status '
                        f'{result.returncode}.')
            except (OSError, subprocess.SubprocessError) as error:
                self._output(
                    f'WARNING: unable to collect {label} diagnostics: {error}')
