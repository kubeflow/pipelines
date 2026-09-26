#!/usr/bin/env python3
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
"""Prove native ARM execution with the public v2 API and standalone overlay."""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import time
import urllib.error
import urllib.request

CI_IMAGES = {
    "kfp-api-server": "apiserver",
    "kfp-persistence-agent": "persistenceagent",
    "kfp-scheduled-workflow-controller": "scheduledworkflow",
    "kfp-frontend": "frontend",
    "kfp-viewer-crd-controller": "viewer-crd-controller",
    "kfp-driver": "driver",
    "kfp-launcher": "launcher",
}
IMAGES = set(CI_IMAGES)
CONTROL_IMAGES = IMAGES - {"kfp-driver", "kfp-launcher"}
MARKER = "KFP_NATIVE_ARM64_EXECUTION_OK"


def image_refs(directory, source_sha, mode):
    """Require complete, source-matched inventories and shared index
    digests."""
    if not re.fullmatch(r"[0-9a-f]{40}", source_sha):
        raise ValueError("source_sha must be a full commit SHA")
    refs = {}
    for path in sorted(Path(directory).rglob("*.json")):
        record = json.loads(path.read_text())
        image = record["image"]
        name = image.rsplit("/", 1)[-1]
        # The legacy GCP inverse-proxy overlay is outside this standalone profile.
        if name in {"inverse-proxy-agent", "kfp-inverse-proxy-agent"
                   } and mode == "published":
            continue
        if name not in IMAGES or name in refs:
            raise ValueError(f"Unexpected or duplicate image record: {name}")
        if record["source_sha"] != source_sha:
            raise ValueError(f"Image {name} was not built from {source_sha}")
        platforms = set(record["platforms"])
        if "linux/arm64" not in platforms:
            raise ValueError(f"Image {name} has no ARM64 platform")
        reference = record["reference"]
        if mode == "published":
            if platforms != {"linux/amd64", "linux/arm64"}:
                raise ValueError(
                    f"Image {name} must contain both supported platforms")
            if not re.fullmatch(rf"ghcr\.io/[a-z0-9_.-]+/{re.escape(name)}",
                                image):
                raise ValueError(
                    f"Image {name} must identify its full GHCR repository")
            if not re.fullmatch(rf"{re.escape(image)}@sha256:[0-9a-f]{{64}}",
                                reference):
                raise ValueError(
                    f"Image {name} must use an immutable shared index digest")
        elif image != name or reference != f"kind-registry:5000/{CI_IMAGES[name]}:ci":
            raise ValueError(f"Image {name} must use its expected local CI tag")
        refs[name] = reference
    if set(refs) != IMAGES:
        raise ValueError(f"Missing images: {sorted(IMAGES - set(refs))}")
    return refs


def load_images(archives, cluster, refs):
    paths = sorted(Path(archives).rglob("*.tar")) if archives else []
    if not paths:
        raise ValueError("Local smoke requires image archives")
    for path in paths:
        subprocess.run(["docker", "load", "--input", str(path)], check=True)
    for reference in refs.values():
        raw = subprocess.check_output(["docker", "image", "inspect", reference],
                                      text=True)
        image = json.loads(raw)[0]
        if image["Architecture"] != "arm64" or image["Os"] != "linux":
            raise ValueError(f"Loaded image {reference} is not Linux ARM64")
    subprocess.run(
        ["kind", "load", "docker-image", "--name", cluster, *refs.values()],
        check=True)


def overlay_spec(refs, base, mode="published"):
    images = []
    for name in sorted(CONTROL_IMAGES):
        reference = refs[name]
        entry = {"name": f"ghcr.io/kubeflow/{name}"}
        if "@" in reference:
            entry["newName"], entry["digest"] = reference.split("@")
        else:
            entry["newName"], entry["newTag"] = reference.rsplit(":", 1)
        images.append(entry)
    patch = {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
            "name": "ml-pipeline"
        },
        "spec": {
            "template": {
                "spec": {
                    "containers": [{
                        "name":
                            "ml-pipeline-api-server",
                        "env": [
                            {
                                "name": "V2_DRIVER_IMAGE",
                                "value": refs["kfp-driver"]
                            },
                            {
                                "name": "V2_LAUNCHER_IMAGE",
                                "value": refs["kfp-launcher"]
                            },
                        ],
                    }]
                }
            }
        },
    }
    patches = [{"patch": json.dumps(patch)}]
    if mode == "local":
        patches.append({
            "patch":
                json.dumps({
                    "apiVersion": "apps/v1",
                    "kind": "Deployment",
                    "metadata": {
                        "name": "ml-pipeline-viewer-crd"
                    },
                    "spec": {
                        "template": {
                            "spec": {
                                "containers": [{
                                    "name": "ml-pipeline-viewer-crd",
                                    "imagePullPolicy": "IfNotPresent",
                                }]
                            }
                        }
                    },
                })
        })
    return {
        "apiVersion": "kustomize.config.k8s.io/v1beta1",
        "kind": "Kustomization",
        "resources": [str(base)],
        "images": images,
        "patches": patches,
    }


def pipeline_spec():
    # A container component avoids SDK/protoc bootstrapping on the runner.
    # Caching is off so every run must execute the launcher and user process.
    return {
        "pipelineInfo": {
            "name": "native-arm64-smoke"
        },
        "schemaVersion": "2.1.0",
        "sdkVersion": "kfp-native-arm64-smoke",
        "components": {
            "comp-native": {
                "executorLabel": "exec-native"
            }
        },
        "deploymentSpec": {
            "executors": {
                "exec-native": {
                    "container": {
                        "image":
                            "docker.io/alpine:3.23",
                        "command": ["sh", "-ec"],
                        "args": [
                            f'test "$(uname -m)" = aarch64; echo {MARKER}'
                        ],
                    }
                }
            }
        },
        "root": {
            "dag": {
                "tasks": {
                    "native": {
                        "taskInfo": {
                            "name": "native"
                        },
                        "componentRef": {
                            "name": "comp-native"
                        },
                        "cachingOptions": {
                            "enableCache": False
                        },
                    }
                }
            }
        },
    }


def assert_arm_nodes(nodes):
    items = nodes["items"]
    if not items or any(
            node["status"]["nodeInfo"]["architecture"] != "arm64" or
            node["metadata"]["labels"].get("kubernetes.io/arch") != "arm64"
            for node in items):
        raise ValueError("Smoke cluster must contain only native ARM64 nodes")
    return {node["metadata"]["name"] for node in items}


def executes_launcher(command):
    # Argo's emissary executor wraps the compiled command in the actual Pod.
    if command[:2] == ["/var/run/argo/argoexec", "emissary"]:
        if "--" not in command:
            return False
        command = command[command.index("--") + 1:]
    return command[:1] == ["/kfp-launcher/launch"]


def assert_execution(pods, refs, arm_nodes):
    """Require successful driver and launcher execution on ARM nodes."""
    driver_pods = []
    launcher_pods = []
    for pod in pods["items"]:
        if pod["spec"].get("nodeName") not in arm_nodes:
            raise ValueError(
                "Pipeline pod did not execute on a verified ARM64 node")
        statuses = {
            status["name"]: status
            for key in ("containerStatuses", "initContainerStatuses")
            for status in pod["status"].get(key, [])
        }

        def succeeded(name):
            status = statuses.get(name, {})
            return (status.get("state", {}).get("terminated",
                                                {}).get("exitCode") == 0 and
                    bool(status.get("imageID")))

        for container in pod["spec"]["containers"]:
            if container["image"] == refs["kfp-driver"] and succeeded(
                    container["name"]):
                driver_pods.append(pod["metadata"]["name"])
        for init in pod["spec"].get("initContainers", []):
            if init["name"] != "kfp-launcher" or init["image"] != refs[
                    "kfp-launcher"]:
                continue
            main = next(
                (c for c in pod["spec"]["containers"] if c["name"] == "main"),
                {})
            if (succeeded("kfp-launcher") and succeeded("main") and
                    executes_launcher(main.get("command", []))):
                launcher_pods.append(pod["metadata"]["name"])
    if not driver_pods or not launcher_pods:
        raise ValueError(
            "Missing successful ARM64 driver or launcher execution")
    return {"driver_pods": driver_pods, "launcher_pods": launcher_pods}


def api_request(url, body=None):
    data = None if body is None else json.dumps(body).encode()
    request = urllib.request.Request(
        url, data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=15) as response:
        return json.load(response)


def wait_for_deployments(kubectl):
    deployments = kubectl("-n", "kubeflow", "get", "deployments", "-o",
                          "name").split()
    if not deployments:
        raise ValueError("Standalone installation created no deployments")
    # Unlike kubectl wait, rollout status has no --all flag.
    kubectl("-n", "kubeflow", "rollout", "status", *deployments,
            "--timeout=600s")


def run_smoke(args, refs):
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)

    def kubectl(*command):
        return subprocess.check_output(
            ["kubectl", "--context", args.context, *command],
            text=True,
        )

    nodes = json.loads(kubectl("get", "nodes", "-o", "json"))
    arm_nodes = assert_arm_nodes(nodes)
    (output / "nodes.json").write_text(json.dumps(nodes, indent=2))
    kubectl("apply", "-k", "manifests/kustomize/cluster-scoped-resources")
    kubectl("wait", "--for=condition=Established", "crd", "--all",
            "--timeout=120s")
    rendered = kubectl("kustomize", "--load-restrictor", "LoadRestrictionsNone",
                       args.overlay)
    (output / "installation.yaml").write_text(rendered)
    kubectl("apply", "-f", str(output / "installation.yaml"))
    wait_for_deployments(kubectl)
    with (output / "port-forward.log").open("w") as log:
        forward = subprocess.Popen([
            "kubectl",
            "--context",
            args.context,
            "-n",
            "kubeflow",
            "port-forward",
            "service/ml-pipeline",
            "8888:8888",
            "--address=127.0.0.1",
        ],
                                   stdout=log,
                                   stderr=subprocess.STDOUT)
        try:
            base = "http://127.0.0.1:8888/apis/v2beta1"
            deadline = time.monotonic() + 120
            while True:
                if forward.poll() is not None:
                    raise RuntimeError(
                        "API port-forward exited; see port-forward.log")
                try:
                    api_request(f"{base}/healthz")
                    break
                except (urllib.error.URLError, TimeoutError):
                    if time.monotonic() >= deadline:
                        raise
                    time.sleep(2)
            run = api_request(
                f"{base}/runs", {
                    "display_name": "native-arm64-smoke",
                    "pipeline_spec": pipeline_spec(),
                    "runtime_config": {},
                    "service_account": "pipeline-runner",
                })
            run_id = run["run_id"]
            deadline = time.monotonic() + 600
            while run.get("state") not in {
                    "SUCCEEDED", "FAILED", "CANCELED", "CANCELLED", "SKIPPED"
            }:
                if time.monotonic() >= deadline:
                    raise TimeoutError(
                        f"Run {run_id} did not finish within 600 seconds")
                time.sleep(5)
                run = api_request(f"{base}/runs/{run_id}")
            (output / "run.json").write_text(json.dumps(run, indent=2))
            if run["state"] != "SUCCEEDED":
                raise RuntimeError(f"Pipeline run failed: {run}")
            workflows = json.loads(
                kubectl("-n", "kubeflow", "get", "workflows", "-o", "json"))
            matching = [
                w for w in workflows["items"] if w["metadata"].get(
                    "labels", {}).get("pipeline/runid") == run_id
            ]
            if len(matching) != 1:
                raise ValueError(
                    f"Expected one workflow for run {run_id}, found {len(matching)}"
                )
            workflow = matching[0]
            (output / "workflow.json").write_text(
                json.dumps(workflow, indent=2))
            pods = json.loads(
                kubectl(
                    "-n",
                    "kubeflow",
                    "get",
                    "pods",
                    "-l",
                    f"workflows.argoproj.io/workflow={workflow['metadata']['name']}",
                    "-o",
                    "json",
                ))
            (output / "pipeline-pods.json").write_text(
                json.dumps(pods, indent=2))
            evidence = assert_execution(pods, refs, arm_nodes)
            for pod in evidence["launcher_pods"]:
                logs = kubectl("-n", "kubeflow", "logs", pod, "-c", "main")
                (output / f"{pod}.log").write_text(logs)
                if MARKER not in logs:
                    raise ValueError(
                        "Launched component did not confirm native ARM64 execution"
                    )
            evidence.update(
                source_sha=args.source_sha, run_id=run_id, images=refs)
            (output / "evidence.json").write_text(
                json.dumps(evidence, indent=2))
            print(json.dumps(evidence, indent=2))
        finally:
            forward.terminate()
            forward.wait(timeout=15)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("prepare", "load", "run"))
    parser.add_argument("--records", required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--mode", choices=("published", "local"), required=True)
    parser.add_argument("--overlay")
    parser.add_argument("--archives")
    parser.add_argument("--cluster", default="kfp-arm64")
    parser.add_argument("--context", default="kind-kfp-arm64")
    parser.add_argument("--output", default="arm64-smoke-results")
    args = parser.parse_args()
    refs = image_refs(args.records, args.source_sha, args.mode)
    if args.command == "load":
        load_images(args.archives, args.cluster, refs)
    elif args.command == "prepare":
        overlay = Path(args.overlay)
        overlay.mkdir(parents=True, exist_ok=True)
        base = os.path.relpath(
            Path("manifests/kustomize/env/platform-agnostic").resolve(),
            overlay.resolve())
        (overlay / "kustomization.yaml").write_text(
            json.dumps(overlay_spec(refs, base, args.mode), indent=2))
    else:
        run_smoke(args, refs)


if __name__ == "__main__":
    main()
