# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from pathlib import Path
import shutil
import subprocess
import unittest

import arm64_smoke
import yaml

ROOT = Path(__file__).resolve().parents[3]


class PublicationTests(unittest.TestCase):

    def test_master_and_release_publish_only_supported_runtime_images(self):
        for line in ("master", "release"):
            with self.subTest(line=line):
                workflow = ROOT / f".github/workflows/image-builds-{line}.yml"
                jobs = yaml.safe_load(workflow.read_text())["jobs"]
                matrix = jobs[f"build-images-for-{line}"]["strategy"]["matrix"]
                self.assertEqual({image["name"] for image in matrix["image"]},
                                 arm64_smoke.IMAGES)
                self.assertNotIn("exclude", matrix)
                self.assertEqual({arch["platform"] for arch in matrix["arch"]},
                                 {"linux/amd64", "linux/arm64"})
                manifest = jobs["create-manifests"]
                self.assertEqual(
                    {
                        item["image"]
                        for item in manifest["strategy"]["matrix"]["component"]
                    }, arm64_smoke.IMAGES)
                self.assertEqual(manifest["with"]["expected_platforms"],
                                 "linux/amd64,linux/arm64")


@unittest.skipUnless(
    shutil.which("kubectl"), "kubectl is needed to render overlays")
class ManifestTests(unittest.TestCase):

    def test_former_consumers_retain_kfp_without_bundled_inverse_proxy(self):
        removed = {
            ("Deployment", "proxy-agent"),
            ("ConfigMap", "inverse-proxy-config"),
            ("ServiceAccount", "proxy-agent-runner"),
            ("Role", "proxy-agent-runner"),
            ("RoleBinding", "proxy-agent-runner"),
        }
        for overlay in ("manifests/kustomize/env/gcp",
                        "manifests/kustomize/env/dev",
                        "manifests/kustomize/env/dev/postgresql",
                        "manifests/kustomize/sample",
                        "manifests/kustomize/env/cert-manager/dev",
                        "test/manifests/dev"):
            with self.subTest(overlay=overlay):
                result = subprocess.run(
                    ["kubectl", "kustomize",
                     str(ROOT / overlay)],
                    capture_output=True,
                    text=True,
                    check=True,
                    timeout=120)
                resources = list(yaml.safe_load_all(result.stdout))
                identities = {(item["kind"], item["metadata"]["name"])
                              for item in resources
                              if item}
                self.assertFalse(identities & removed)
                self.assertNotIn("inverse-proxy", result.stdout)
                self.assertIn(("Deployment", "ml-pipeline"), identities)
                self.assertIn(("Deployment", "ml-pipeline-ui"), identities)
                if overlay.endswith(("/gcp", "/sample")):
                    self.assertIn(("Deployment", "cloudsqlproxy"), identities)


if __name__ == "__main__":
    unittest.main()
