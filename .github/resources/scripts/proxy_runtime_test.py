"""Guard the inverse-proxy build's source and dependency provenance."""

import pathlib
import re
import subprocess
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]


class ProxyRuntimeTest(unittest.TestCase):

    def test_agent_uses_verified_git_commit(self):
        dockerfile = (ROOT / "proxy/Dockerfile").read_text()
        agent = dockerfile.split("FROM builder AS kubectl")[0]
        self.assertIn("git clone", agent)
        self.assertIn("git rev-parse HEAD", agent)
        self.assertIn("${AGENT_COMMIT}", agent)
        self.assertNotIn("/archive/", agent)

    def test_kubectl_otel_modules_share_release(self):
        expected = {
            "go.opentelemetry.io/otel",
            "go.opentelemetry.io/otel/trace",
            "go.opentelemetry.io/otel/metric",
            "go.opentelemetry.io/otel/log",
            "go.opentelemetry.io/otel/sdk",
            "go.opentelemetry.io/otel/sdk/metric",
            "go.opentelemetry.io/otel/exporters/stdout/stdouttrace",
            "go.opentelemetry.io/otel/exporters/otlp/otlptrace",
            "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc",
        }
        for path in ("proxy/Dockerfile",
                     "backend/src/cache/deployer/Dockerfile"):
            with self.subTest(dockerfile=path):
                versions = dict(
                    re.findall(
                        r"(go\.opentelemetry\.io/otel(?:/[\w/]+)?)@(v[\d.]+)",
                        (ROOT / path).read_text()))
                self.assertEqual(set(versions), expected)
                self.assertEqual(set(versions.values()), {"v1.47.0"})

    def test_resolved_otel_graph_checks_stable_and_experimental_versions(self):
        for path in ("proxy/Dockerfile",
                     "backend/src/cache/deployer/Dockerfile"):
            dockerfile = (ROOT / path).read_text()
            check = re.search(r"awk '(.*?)' /out-modules.txt", dockerfile,
                              re.DOTALL).group(1).replace("\\\n", "")
            modules = dict(
                re.findall(
                    r"(go\.opentelemetry\.io/otel(?:/[\w/]+)?)@(v[\d.]+)",
                    dockerfile))
            modules["go.opentelemetry.io/otel/metric/x"] = "v0.69.0"

            def validate(graph):
                return subprocess.run(
                    ["awk", check],
                    input="\n".join(
                        f"{name} {version}" for name, version in graph.items()),
                    text=True,
                    check=False).returncode

            with self.subTest(dockerfile=path):
                self.assertEqual(validate(modules), 0)
                for name in modules:
                    with self.subTest(module=name):
                        self.assertNotEqual(
                            validate({
                                **modules, name: "v0.0.0"
                            }), 0)
                        self.assertNotEqual(
                            validate({
                                key: value
                                for key, value in modules.items()
                                if key != name
                            }), 0)

    def test_agent_otel_lock_uses_patched_stable_release(self):
        modules = dict(
            re.findall(
                r"^\s*(go\.opentelemetry\.io/otel(?:/[\w/]+)?) (v[\d.]+)",
                (ROOT / "proxy/agent.go.mod").read_text(), re.MULTILINE))
        self.assertEqual(set(modules.values()), {"v1.47.0"})
        self.assertTrue(
            {"go.opentelemetry.io/otel/sdk", "go.opentelemetry.io/otel/trace"
            } <= set(modules))

    def test_reporting_tests_run_outside_production_image(self):
        dockerfile = (ROOT / "proxy/Dockerfile").read_text()
        workflow = (ROOT / ".github/workflows/ci-scripts-tests.yml").read_text()
        self.assertNotIn("COPY security", dockerfile)
        self.assertIn("'proxy/**'", workflow)
        self.assertIn(
            "python3 -m unittest discover -s proxy/security -p '*_test.py'",
            workflow)


if __name__ == "__main__":
    unittest.main()
