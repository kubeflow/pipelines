"""Submit a small parameter-and-artifact pipeline through the installed SDK."""

import argparse
import tempfile
from pathlib import Path

from kfp import client, compiler, dsl


@dsl.container_component
def produce(
    message: str,
    data: dsl.Output[dsl.Dataset],
    value: dsl.OutputPath(str),
) -> dsl.ContainerSpec:
    """Write the same message into an artifact and a declared output
    parameter."""
    return dsl.ContainerSpec(
        image='docker.io/alpine:3.23',
        command=['sh', '-ec'],
        args=[
            'mkdir -p "$(dirname "$1")" "$(dirname "$2")"; '
            'printf "%s\\n" "$0" | tee "$1"; printf "%s" "$0" > "$2"',
            message,
            data.path,
            value,
        ],
    )


@dsl.container_component
def consume(data: dsl.Input[dsl.Dataset], expected: str) -> dsl.ContainerSpec:
    """Check that the downstream task receives both resolved input forms."""
    return dsl.ContainerSpec(
        image='docker.io/alpine:3.23',
        command=['sh', '-ec'],
        args=[
            'test "$(cat "$0")" = "$1"; echo "verified packaged runtime"',
            data.path,
            expected,
        ],
    )


@dsl.pipeline(name='packaged-local-proof')
def pipeline(message: str = 'Hello from the installed local wheel') -> None:
    """Exercise the normal server/compiler contract rather than a local
    runner."""
    task = produce(message=message)
    consume(data=task.outputs['data'], expected=task.outputs['value'])


def main() -> None:
    """Compile, submit, and verify the packaged example through the public
    SDK."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--host', default='http://127.0.0.1:3002')
    args = parser.parse_args()
    api = client.Client(host=args.host)
    with tempfile.TemporaryDirectory(prefix='kfp-local-example-') as temporary:
        path = Path(temporary) / 'pipeline.yaml'
        compiler.Compiler().compile(pipeline, str(path))
        run = api.create_run_from_pipeline_package(
            str(path),
            experiment_name='packaged-local-proof',
            enable_caching=False,
        )
    completed = api.wait_for_run_completion(
        run.run_id,
        timeout=300,
        sleep_duration=1,
    )
    if completed.state != 'SUCCEEDED':
        raise RuntimeError(f'Run {run.run_id} finished in {completed.state}')
    print(f'Run {run.run_id}: {completed.state}')


if __name__ == '__main__':
    main()
