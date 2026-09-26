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

import copy
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest import mock

import arm64_smoke as smoke
import yaml

SHA = "a" * 40
DIGEST = "b" * 64


class WorkflowTests(unittest.TestCase):

    def test_rollout_wait_names_installed_deployments(self):
        kubectl = mock.Mock(side_effect=[
            "deployment.apps/ml-pipeline\ndeployment.apps/mysql\n", "success"
        ])
        smoke.wait_for_deployments(kubectl)
        self.assertEqual(kubectl.call_args_list, [
            mock.call("-n", "kubeflow", "get", "deployments", "-o", "name"),
            mock.call("-n", "kubeflow", "rollout", "status",
                      "deployment.apps/ml-pipeline", "deployment.apps/mysql",
                      "--timeout=600s"),
        ])

    def test_rollout_wait_rejects_empty_installation(self):
        with self.assertRaisesRegex(ValueError, "no deployments"):
            smoke.wait_for_deployments(mock.Mock(return_value=""))

    def test_master_smoke_consumes_same_publication_attempt(self):
        root = Path(__file__).resolve().parents[3]
        workflow = yaml.safe_load(
            (root / ".github/workflows/image-builds-master.yml").read_text())
        job = workflow["jobs"]["arm64-smoke"]
        self.assertEqual(job["needs"], "create-manifests")
        self.assertEqual(job["runs-on"], "ubuntu-24.04-arm")
        download = next(step for step in job["steps"] if step.get("uses") ==
                        "./.github/actions/download-artifact-with-retry")
        self.assertEqual(download["with"]["pattern"],
                         "published-index-*-${{ github.run_attempt }}")
        action = next(step for step in job["steps"]
                      if step.get("uses") == "./.github/actions/arm64-smoke")
        self.assertEqual(action["with"]["source_sha"], "${{ github.sha }}")
        self.assertEqual(action["with"]["image_records"],
                         download["with"]["path"])


class InventoryTests(unittest.TestCase):

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        for name in smoke.IMAGES:
            self.write(name)

    def write(self, name, **updates):
        record = {
            "image": f"ghcr.io/kubeflow/{name}",
            "reference": f"ghcr.io/kubeflow/{name}@sha256:{DIGEST}",
            "source_sha": SHA,
            "platforms": ["linux/amd64", "linux/arm64"],
        }
        record.update(updates)
        (self.directory / f"{name}.json").write_text(json.dumps(record))

    def read(self):
        return smoke.image_refs(self.directory, SHA, "published")

    def test_complete_shared_indexes(self):
        self.assertEqual(set(self.read()), smoke.IMAGES)

    def test_inverse_proxy_explicitly_outside_profile(self):
        self.write("inverse-proxy-agent", platforms=["linux/amd64"])
        self.assertEqual(set(self.read()), smoke.IMAGES)

    def test_missing_image_fails(self):
        (self.directory / "kfp-driver.json").unlink()
        with self.assertRaisesRegex(ValueError, "Missing images"):
            self.read()

    def test_wrong_source_fails(self):
        self.write("kfp-driver", source_sha="c" * 40)
        with self.assertRaisesRegex(ValueError, "not built from"):
            self.read()

    def test_single_architecture_fails(self):
        self.write("kfp-driver", platforms=["linux/arm64"])
        with self.assertRaisesRegex(ValueError, "both supported"):
            self.read()

    def test_mutable_reference_fails(self):
        self.write("kfp-driver", reference="ghcr.io/kubeflow/kfp-driver:master")
        with self.assertRaisesRegex(ValueError, "immutable shared"):
            self.read()

    def test_mismatched_image_reference_fails(self):
        self.write(
            "kfp-driver", reference=f"ghcr.io/other/kfp-driver@sha256:{DIGEST}")
        with self.assertRaisesRegex(ValueError, "immutable shared"):
            self.read()

    def test_duplicate_record_fails(self):
        (self.directory / "duplicate.json").write_text(
            (self.directory / "kfp-driver.json").read_text())
        with self.assertRaisesRegex(ValueError, "duplicate"):
            self.read()

    def test_local_registry_with_port(self):
        for name, ci_name in smoke.CI_IMAGES.items():
            self.write(
                name,
                image=name,
                reference=f"kind-registry:5000/{ci_name}:ci",
                platforms=["linux/arm64"])
        self.assertEqual(
            set(smoke.image_refs(self.directory, SHA, "local")), smoke.IMAGES)

    def test_overlay_maps_controls_and_runtime_refs(self):
        refs = self.read()
        spec = smoke.overlay_spec(
            refs, "/repo/manifests/kustomize/env/platform-agnostic")
        self.assertEqual(len(spec["images"]), 5)
        self.assertTrue(
            all(image["digest"] == f"sha256:{DIGEST}"
                for image in spec["images"]))
        patch = json.loads(spec["patches"][0]["patch"])
        env = patch["spec"]["template"]["spec"]["containers"][0]["env"]
        self.assertEqual({item["value"] for item in env},
                         {refs["kfp-driver"], refs["kfp-launcher"]})
        local = smoke.overlay_spec(refs, "/base", "local")
        viewer = json.loads(local["patches"][1]["patch"])
        self.assertEqual(
            viewer["spec"]["template"]["spec"]["containers"][0]
            ["imagePullPolicy"], "IfNotPresent")

    @unittest.skipUnless(
        shutil.which("kubectl"),
        "kubectl is needed to render the stock overlay")
    def test_rendered_stock_overlay_uses_all_seven_refs(self):
        base = Path(__file__).resolve(
        ).parents[3] / "manifests/kustomize/env/platform-agnostic"
        for mode in ("published", "local"):
            with self.subTest(mode=mode):
                refs = self.read() if mode == "published" else {
                    name: f"kind-registry:5000/{ci_name}:ci"
                    for name, ci_name in smoke.CI_IMAGES.items()
                }
                overlay = self.directory / mode
                overlay.mkdir()
                (overlay / "kustomization.yaml").write_text(
                    json.dumps(
                        smoke.overlay_spec(
                            refs, os.path.relpath(base, overlay.resolve()),
                            mode)))
                rendered = subprocess.check_output([
                    "kubectl", "kustomize", "--load-restrictor",
                    "LoadRestrictionsNone",
                    str(overlay)
                ],
                                                   text=True)
                deployments = [
                    item for item in yaml.safe_load_all(rendered)
                    if item["kind"] == "Deployment"
                ]
                containers = [
                    container for item in deployments for container in
                    item["spec"]["template"]["spec"]["containers"]
                ]
                self.assertTrue({refs[name] for name in smoke.CONTROL_IMAGES
                                }.issubset({c["image"] for c in containers}))
                api = next(c for c in containers
                           if c["name"] == "ml-pipeline-api-server")
                env = {item["name"]: item.get("value") for item in api["env"]}
                self.assertEqual(env["V2_DRIVER_IMAGE"], refs["kfp-driver"])
                self.assertEqual(env["V2_LAUNCHER_IMAGE"], refs["kfp-launcher"])
                if mode == "local":
                    for container in containers:
                        if container["image"] in refs.values():
                            self.assertEqual(container["imagePullPolicy"],
                                             "IfNotPresent")

    @mock.patch.object(smoke.subprocess, "run")
    @mock.patch.object(smoke.subprocess, "check_output")
    def test_archive_wrong_arch_fails_before_kind_load(self, inspect, run):
        (self.directory / "image.tar").touch()
        inspect.return_value = json.dumps([{
            "Architecture": "amd64",
            "Os": "linux"
        }])
        with self.assertRaisesRegex(ValueError, "not Linux ARM64"):
            smoke.load_images(self.directory, "test",
                              {"kfp-driver": "image:ci"})
        self.assertEqual(run.call_count, 1)
        self.assertEqual(run.call_args.args[0][:2], ["docker", "load"])


class ExecutionTests(unittest.TestCase):

    def setUp(self):
        self.refs = {
            name: f"ghcr.io/kubeflow/{name}@sha256:{DIGEST}"
            for name in smoke.IMAGES
        }
        self.nodes = {
            "items": [{
                "metadata": {
                    "name": "arm-node",
                    "labels": {
                        "kubernetes.io/arch": "arm64"
                    }
                },
                "status": {
                    "nodeInfo": {
                        "architecture": "arm64"
                    }
                },
            }]
        }

        def status(name):
            return {
                "name": name,
                "imageID": "containerd://sha256:abc",
                "state": {
                    "terminated": {
                        "exitCode": 0
                    }
                }
            }

        self.pods = {
            "items": [
                {
                    "metadata": {
                        "name": "driver-pod"
                    },
                    "spec": {
                        "nodeName":
                            "arm-node",
                        "containers": [{
                            "name": "main",
                            "image": self.refs["kfp-driver"]
                        }]
                    },
                    "status": {
                        "containerStatuses": [status("main")]
                    },
                },
                {
                    "metadata": {
                        "name": "component-pod"
                    },
                    "spec": {
                        "nodeName":
                            "arm-node",
                        "containers": [{
                            "name": "main",
                            "image": "alpine:3.23",
                            "command": ["/kfp-launcher/launch", "--"]
                        }],
                        "initContainers": [{
                            "name": "kfp-launcher",
                            "image": self.refs["kfp-launcher"]
                        }],
                    },
                    "status": {
                        "containerStatuses": [status("main")],
                        "initContainerStatuses": [status("kfp-launcher")]
                    },
                },
            ]
        }

    def verify(self, pods=None):
        return smoke.assert_execution(pods or self.pods, self.refs,
                                      smoke.assert_arm_nodes(self.nodes))

    def test_success_requires_driver_and_launcher(self):
        self.assertEqual(self.verify(), {
            "driver_pods": ["driver-pod"],
            "launcher_pods": ["component-pod"]
        })

    def test_argo_emissary_wrapped_launcher_succeeds(self):
        self.pods["items"][1]["spec"]["containers"][0]["command"] = [
            "/var/run/argo/argoexec", "emissary", "--loglevel", "info",
            "--log-format", "text", "--gloglevel", "0", "--",
            "/kfp-launcher/launch", "--executor_type", "container", "--"
        ]
        self.assertEqual(self.verify()["launcher_pods"], ["component-pod"])

    def test_wrapper_must_execute_launcher_not_merely_mention_it(self):
        for command in (
            ["/var/run/argo/argoexec", "emissary", "/kfp-launcher/launch"],
            [
                "/var/run/argo/argoexec", "emissary", "--", "echo",
                "/kfp-launcher/launch"
            ],
            ["unexpected-wrapper", "--", "/kfp-launcher/launch"],
        ):
            with self.subTest(command=command):
                self.pods["items"][1]["spec"]["containers"][0][
                    "command"] = command
                with self.assertRaisesRegex(ValueError, "Missing successful"):
                    self.verify()

    def test_amd64_node_fails(self):
        self.nodes["items"][0]["status"]["nodeInfo"]["architecture"] = "amd64"
        with self.assertRaisesRegex(ValueError, "only native ARM64"):
            self.verify()

    def test_missing_driver_fails(self):
        self.pods["items"].pop(0)
        with self.assertRaisesRegex(ValueError, "Missing successful"):
            self.verify()

    def test_wrong_driver_image_fails(self):
        self.pods["items"][0]["spec"]["containers"][0]["image"] = "other:ci"
        with self.assertRaisesRegex(ValueError, "Missing successful"):
            self.verify()

    def test_launcher_failure_or_missing_image_id_fails(self):
        for modification in ({
                "state": {
                    "terminated": {
                        "exitCode": 1
                    }
                }
        }, {
                "imageID": ""
        }):
            with self.subTest(modification=modification):
                pods = copy.deepcopy(self.pods)
                pods["items"][1]["status"]["initContainerStatuses"][0].update(
                    modification)
                with self.assertRaisesRegex(ValueError, "Missing successful"):
                    self.verify(pods)

    def test_main_must_execute_copied_launcher(self):
        self.pods["items"][1]["spec"]["containers"][0]["command"] = [
            "echo", "fake success"
        ]
        with self.assertRaisesRegex(ValueError, "Missing successful"):
            self.verify()

    def test_pod_on_unknown_node_fails(self):
        self.pods["items"][1]["spec"]["nodeName"] = "unknown"
        with self.assertRaisesRegex(ValueError, "verified ARM64 node"):
            self.verify()

    def test_pipeline_disables_caching(self):
        task = smoke.pipeline_spec()["root"]["dag"]["tasks"]["native"]
        self.assertFalse(task["cachingOptions"]["enableCache"])


if __name__ == "__main__":
    unittest.main()
