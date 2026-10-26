"""Verify the cache-deployer kubectl's embedded version/provenance guard."""

import os
import pathlib
import re
import shlex
import subprocess
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]
DOCKERFILE = ROOT / "backend/src/cache/deployer/Dockerfile"


class CacheDeployerKubectlTest(unittest.TestCase):

    def test_source_checkout_preserves_verified_tag_metadata(self):
        source = DOCKERFILE.read_text().split("FROM alpine:")[0]
        self.assertIn("-buildvcs=true", source)
        self.assertNotIn("codeload.github.com", source)
        self.assertNotIn("rm -rf .git", source)

    def test_client_version_fields_follow_requested_version(self):
        source = DOCKERFILE.read_text().replace("\\\n", "")
        build = next(
            line.removeprefix("RUN ")
            for line in source.splitlines()
            if line.startswith("RUN ") and "go build" in line)
        # Stop after capturing the actual build arguments, before binary checks.
        stub = 'go() { if [ "$1" = build ]; then printf "%s\\n" "$@"; exit; fi; }; '
        for version, major, minor in (("v1.37.1", "1", "37"),
                                      ("v1.38.2", "1", "38"), ("v2.3.4", "2",
                                                               "3")):
            with self.subTest(version=version):
                result = subprocess.run(
                    ["sh", "-eu", "-c", stub + build],
                    env={
                        **os.environ, "KUBERNETES_VERSION": version,
                        "KUBERNETES_COMMIT": "0" * 40
                    },
                    text=True,
                    capture_output=True,
                    check=True)
                flags = next(
                    line.removeprefix("-ldflags=")
                    for line in result.stdout.splitlines()
                    if line.startswith("-ldflags="))
                fields = dict(
                    value.split("=", 1)
                    for value in shlex.split(flags)
                    if "=" in value)
                prefix = "k8s.io/component-base/version."
                self.assertEqual(fields[prefix + "gitVersion"], version)
                self.assertEqual(fields[prefix + "gitMajor"], major)
                self.assertEqual(fields[prefix + "gitMinor"], minor)

    def test_binary_guard_rejects_missing_or_incorrect_build_metadata(self):
        source = DOCKERFILE.read_text()
        version = re.search(r"ARG KUBERNETES_VERSION=(\S+)", source).group(1)
        commit = re.search(r"ARG KUBERNETES_COMMIT=([a-f0-9]{40})",
                           source).group(1)
        guard = re.search(r"awk .*?'(.*?)' /out/kubectl.buildinfo", source,
                          re.DOTALL).group(1).replace("\\\n", "")

        def validate(module_version, revision, module="k8s.io/kubernetes"):
            return subprocess.run(
                [
                    "awk", "-v", f"expected={version}+dirty", "-v",
                    f"commit={commit}", guard
                ],
                input=f"\tmod\t{module}\t{module_version}\n\tbuild\tvcs.revision={revision}\n",
                text=True,
                capture_output=True,
                check=False).returncode

        self.assertEqual(validate(version + "+dirty", commit), 0)
        for reported in ("(devel)", "", "v0.0.0", version):
            with self.subTest(version=reported):
                self.assertNotEqual(validate(reported, commit), 0)
        self.assertNotEqual(validate(version + "+dirty", "0" * 40), 0)
        self.assertNotEqual(validate(version + "+dirty", ""), 0)
        self.assertNotEqual(
            validate(version + "+dirty", commit, "other/module"), 0)


if __name__ == "__main__":
    unittest.main()
