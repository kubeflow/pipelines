"""Verify the cache-deployer kubectl's embedded version/provenance guard."""

import pathlib
import re
import subprocess
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]
DOCKERFILE = ROOT / "backend/src/cache/deployer/Dockerfile"


class CacheDeployerKubectlTest(unittest.TestCase):

    def test_source_checkout_preserves_verified_tag_metadata(self):
        source = DOCKERFILE.read_text().split("FROM alpine:")[0]
        self.assertIn('git clone --depth=1 --branch "${KUBERNETES_VERSION}"',
                      source)
        self.assertIn('test "$(git rev-parse HEAD)" = "${KUBERNETES_COMMIT}"',
                      source)
        self.assertIn("-buildvcs=true", source)
        self.assertNotIn("codeload.github.com", source)
        self.assertNotIn("rm -rf .git", source)

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
