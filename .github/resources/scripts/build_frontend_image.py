#!/usr/bin/env python3
"""Build the frontend with bounded attempts and isolated BuildKit recovery."""

import argparse
from contextlib import contextmanager
import os
from pathlib import Path
import signal
import subprocess
import tempfile
import time
import uuid


class CommandTimeout(RuntimeError):
    pass


@contextmanager
def ignore_cancellation():
    previous = {
        sig: signal.signal(sig, signal.SIG_IGN)
        for sig in (signal.SIGINT, signal.SIGTERM)
    }
    try:
        yield
    finally:
        for sig, handler in previous.items():
            signal.signal(sig, handler)


def run(command, seconds, grace=10, stdout=None):
    """Bound the command and terminate its client process group."""
    process = subprocess.Popen(command, start_new_session=True, stdout=stdout)
    try:
        return process.wait(timeout=max(0.01, seconds))
    except BaseException as error:
        with ignore_cancellation():
            try:
                os.killpg(process.pid, signal.SIGTERM)
                # Children may outlive the leader. Always kill the group.
                stop_grace = min(grace, 1) if isinstance(
                    error, KeyboardInterrupt) else grace
                time.sleep(stop_grace)
                os.killpg(process.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            process.wait()
        if isinstance(error, subprocess.TimeoutExpired):
            raise CommandTimeout(command[0]) from error
        raise


def bounded(command, seconds, grace):
    try:
        return run(command, seconds, grace)
    except CommandTimeout:
        return 124


def build(context,
          dockerfile,
          tag,
          node_version,
          output,
          attempt_seconds=480,
          maintenance_seconds=15,
          grace=10):
    output = Path(output)
    partial = output.with_name(output.name + '.partial')
    output.parent.mkdir(parents=True, exist_ok=True)
    output.unlink(missing_ok=True)
    setup = Path(__file__).with_name('setup-buildx-with-retry.sh')
    for attempt in (1, 2):
        builder = f'kfp-frontend-{uuid.uuid4().hex[:12]}'
        container = f'buildx_buildkit_{builder}0'
        partial.unlink(missing_ok=True)
        deadline = time.monotonic() + attempt_seconds
        succeeded = False
        cleaned = False
        cancelled = False
        try:
            print(f'Frontend build attempt {attempt}/2: {builder}', flush=True)
            status = run(['bash', str(setup), builder],
                         deadline - time.monotonic(), grace)
            if status == 0:
                command = [
                    'docker',
                    'buildx',
                    'build',
                    '--builder',
                    builder,
                    '--progress',
                    'plain',
                    '--file',
                    dockerfile,
                    '--build-arg',
                    f'NODE_VERSION={node_version}',
                    '--tag',
                    tag,
                    '--output',
                    f'type=docker,dest={partial}',
                ]
                if attempt == 2:
                    command.append('--no-cache')
                status = run(command + [context], deadline - time.monotonic(),
                             grace)
            succeeded = (
                status == 0 and partial.is_file() and
                partial.stat().st_size > 0)
        except CommandTimeout:
            print(
                f'::warning::Frontend attempt {attempt} exceeded '
                f'{attempt_seconds}s; collecting diagnostics before recovery.',
                flush=True)
        except KeyboardInterrupt:
            cancelled = True
        finally:
            try:
                if cancelled:
                    raise KeyboardInterrupt
                if not succeeded:
                    for command in (
                        ['docker', 'top', container],
                        ['docker', 'logs', '--tail', '100', container],
                    ):
                        bounded(command, maintenance_seconds, grace)
                # BuildKit runs outside the client process group. Remove it before
                # retrying so an abandoned build cannot consume the next attempt.
                cleaned = bounded(
                    ['docker', 'buildx', 'rm', '--force', builder],
                    maintenance_seconds, grace) == 0
                if not cleaned:
                    cleaned = bounded(['docker', 'rm', '--force', container],
                                      maintenance_seconds, grace) == 0
                if not cleaned:
                    # Setup may fail before creating the daemon. Verified absence
                    # is safe to retry; a Docker query failure is not.
                    with tempfile.TemporaryFile() as listing:
                        try:
                            status = run([
                                'docker',
                                'ps',
                                '--all',
                                '--filter',
                                f'name=^/{container}$',
                                '--format',
                                '{{.ID}}',
                            ],
                                         maintenance_seconds,
                                         grace,
                                         stdout=listing)
                            listing.seek(0)
                            cleaned = status == 0 and not listing.read().strip()
                        except CommandTimeout:
                            pass
                if not succeeded or not cleaned:
                    partial.unlink(missing_ok=True)
            except KeyboardInterrupt:
                # Cancellation can also arrive during diagnostics or cleanup.
                # Remove the daemon first and ignore repeated cancellation.
                with ignore_cancellation():
                    bounded(['docker', 'rm', '--force', container], 2, 0.5)
                    bounded(['docker', 'buildx', 'rm', '--force', builder], 2,
                            0.5)
                    partial.unlink(missing_ok=True)
                raise
        if succeeded and cleaned:
            partial.replace(output)
            return 0
        if not cleaned:
            print(
                '::error::Could not remove the frontend builder; retry the '
                'job on a fresh runner.',
                flush=True)
            return 1
    print(
        '::error::Both frontend build attempts failed. Inspect the build and '
        'BuildKit diagnostics before retrying the job.',
        flush=True)
    return 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('context', 'dockerfile', 'tag', 'node-version', 'output'):
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()

    def cancelled(_signum, _frame):
        raise KeyboardInterrupt

    signal.signal(signal.SIGTERM, cancelled)
    try:
        return build(args.context, args.dockerfile, args.tag, args.node_version,
                     args.output)
    except KeyboardInterrupt:
        return 130


if __name__ == '__main__':
    raise SystemExit(main())
