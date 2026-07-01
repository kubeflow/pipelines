"""Local runner tests to validate SubprocessRunner/DockerRunner behavior.

These tests are lightweight and deterministic; they do not require a
Kubernetes cluster or external network access. They exercise the local
runners using simple Python components.
"""

import unittest

from kfp import dsl
from kfp import local


@dsl.component(base_image='python:3.11')
def echo(msg: str) -> str:
    return msg


@dsl.pipeline(name="local-run-test")
def local_pipeline():
    """Pipeline used for local runner validation."""
    # Execute a simple component; do not return outputs to keep compilation trivial
    _ = echo(msg="hello")


class LocalRunnerTest(unittest.TestCase):

    def test_local_subprocess_runner_executes(self):
        """Verify that SubprocessRunner executes a simple pipeline without a
        cluster."""
        local.init(runner=local.SubprocessRunner())
        # Executing the pipeline should not raise
        try:
            local_pipeline()
        except Exception as e:
            self.fail(f"Local pipeline execution raised an exception: {e}")


if __name__ == "__main__":
    unittest.main(verbosity=2)
