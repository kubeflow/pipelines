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
"""Exercise rendered storage/profile manifests across a persistent-volume
upgrade."""

import argparse
import base64
from contextlib import contextmanager
import hashlib
import json
from pathlib import Path
import subprocess
import time
import urllib.request

import yaml

ROOT = Path(__file__).resolve().parents[3]
SEAWEED = ROOT / "manifests/kustomize/third-party/seaweedfs/base"
PROFILE = (
    ROOT /
    "manifests/kustomize/base/installs/multi-user/pipelines-profile-controller")
CLUSTER = "third-party-upgrade"
BUCKET = "mlpipeline"
PAYLOAD = b"KFP persistent upgrade compatibility\x00\xff\n" * 128


def run(*args, data=None):
    return subprocess.run(
        args, input=data, text=True, capture_output=True, check=True).stdout


def kubectl(*args, data=None):
    return run(
        "kubectl",
        "--context",
        f"kind-{CLUSTER}",
        "-n",
        "kubeflow",
        *args,
        data=data)


def rendered(path):
    return list(yaml.safe_load_all(run("kustomize", "build", str(path))))


def apply(objects):
    kubectl("apply", "-f", "-", data=yaml.safe_dump_all(objects))


def wait_ready(name):
    kubectl("rollout", "status", f"deployment/{name}", "--timeout=360s")


@contextmanager
def forward(deployment, local, remote, evidence):
    with (evidence / f"port-forward-{deployment}-{local}.log").open("a") as log:
        process = subprocess.Popen(
            [
                "kubectl",
                "--context",
                f"kind-{CLUSTER}",
                "-n",
                "kubeflow",
                "port-forward",
                f"deployment/{deployment}",
                f"{local}:{remote}",
            ],
            stdout=log,
            stderr=log,
        )
        try:
            url = f"http://127.0.0.1:{local}"
            health = "/healthz" if remote == 8080 else "/status"
            for _ in range(60):
                if process.poll() is not None:
                    raise RuntimeError(
                        f"Port-forward for {deployment} stopped; inspect its log"
                    )
                try:
                    with urllib.request.urlopen(url + health, timeout=2):
                        break
                except OSError:
                    time.sleep(1)
            else:
                raise RuntimeError(f"{deployment} did not become reachable")
            yield url
        finally:
            process.terminate()
            process.wait(timeout=10)


def secret_credentials(secret):
    """Read Kubernetes Secret values using stringData-over-data precedence."""
    values = []
    for key in ("accesskey", "secretkey"):
        try:
            if key in secret.get("stringData", {}):
                value = secret["stringData"][key]
            else:
                value = base64.b64decode(
                    secret["data"][key], validate=True).decode("utf-8")
            if not isinstance(value, str) or not value:
                raise ValueError("Empty or non-string credential")
        except (KeyError, TypeError, ValueError) as error:
            raise ValueError(f"Secret has invalid or missing {key}") from error
        values.append(value)
    return tuple(values)


def client(endpoint, access, secret):
    import boto3
    from botocore.config import Config

    return boto3.client(
        "s3",
        endpoint_url=endpoint,
        aws_access_key_id=access,
        aws_secret_access_key=secret,
        region_name="us-east-1",
        config=Config(
            s3={"addressing_style": "path"},
            retries={"max_attempts": 4},
            request_checksum_calculation="when_required",
            response_checksum_validation="when_required",
        ),
    )


def sync_profile(endpoint, namespace, secret=None):
    attachments = {
        kind: {} for kind in ("Secret.v1", "ConfigMap.v1", "Deployment.apps/v1",
                             "Service.v1")
    }
    if secret:
        attachments["Secret.v1"][
            f"{namespace}/mlpipeline-minio-artifact"] = secret
    body = {
        "object": {
            "metadata": {
                "name": namespace,
                "labels": {
                    "pipelines.kubeflow.org/enabled": "true"
                },
            }
        },
        "attachments": attachments,
    }
    request = urllib.request.Request(
        endpoint + "/sync",
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        result = json.load(response)
    return next(
        item for item in result["attachments"] if item["kind"] == "Secret")


def verify_payload(s3, key):
    actual = s3.get_object(Bucket=BUCKET, Key=key)["Body"].read()
    if actual != PAYLOAD:
        raise AssertionError(f"Persistent object {key} changed during upgrade")


def verify_state(s3):
    verify_payload(s3, "shared/before-upgrade")
    rules = s3.get_bucket_lifecycle_configuration(Bucket=BUCKET)["Rules"]
    if not any(rule["Status"] == "Enabled" and rule["Expiration"]["Days"] == 30
               for rule in rules):
        raise AssertionError("Lifecycle configuration did not survive upgrade")


def exercise(evidence):
    storage = rendered(SEAWEED)
    credentials = [
        obj for obj in storage if obj.get("kind") == "Secret" and
        obj.get("metadata", {}).get("name") == "mlpipeline-minio-artifact"
    ]
    if len(credentials) != 1:
        raise ValueError(
            "Expected exactly one rendered mlpipeline-minio-artifact Secret")
    admin_credentials = secret_credentials(credentials[0])
    deployment = next(obj for obj in storage if obj["kind"] == "Deployment")
    target = deployment["spec"]["template"]["spec"]["containers"][0]["image"]
    deployment["spec"]["template"]["spec"]["containers"][0]["image"] = (
        "ghcr.io/chrislusf/seaweedfs:4.34")
    apply([
        {
            "apiVersion": "v1",
            "kind": "Namespace",
            "metadata": {
                "name": "kubeflow"
            }
        },
        {
            "apiVersion": "v1",
            "kind": "ConfigMap",
            "metadata": {
                "name": "pipeline-install-config"
            },
            "data": {
                "appVersion": "compatibility",
                "ARTIFACT_RETENTION_DAYS": "30",
            },
        },
    ])
    apply(storage)
    wait_ready("seaweedfs")
    (evidence / "before-upgrade-pods.json").write_text(
        kubectl("get", "pods", "-o", "json"))
    pvc_uid = json.loads(kubectl("get", "pvc", "seaweedfs-pvc", "-o",
                                 "json"))["metadata"]["uid"]
    with forward("seaweedfs", 18333, 8333, evidence) as endpoint:
        s3 = client(endpoint, *admin_credentials)
        s3.put_object(Bucket=BUCKET, Key="shared/before-upgrade", Body=PAYLOAD)
        s3.put_bucket_lifecycle_configuration(
            Bucket=BUCKET,
            LifecycleConfiguration={
                "Rules": [{
                    "ID": "private-artifacts",
                    "Status": "Enabled",
                    "Filter": {
                        "Prefix": "private-artifacts"
                    },
                    "Expiration": {
                        "Days": 30
                    },
                }]
            },
        )
        verify_state(s3)
    kubectl("set", "image", "deployment/seaweedfs", f"seaweedfs={target}")
    wait_ready("seaweedfs")
    # The webhook is invoked directly; installing Metacontroller is unnecessary for this contract test.
    profile = [
        obj for obj in rendered(PROFILE) if obj["kind"] != "DecoratorController"
    ]
    apply(profile)
    wait_ready("kubeflow-pipelines-profile-controller")
    with forward("seaweedfs", 18333, 8333, evidence) as endpoint:
        s3 = client(endpoint, *admin_credentials)
        verify_state(s3)
        with forward("kubeflow-pipelines-profile-controller", 18080, 8080,
                     evidence) as webhook:
            secret = sync_profile(webhook, "compat-user")
            repeated = sync_profile(webhook, "compat-user", secret)
            if repeated["data"] != secret["data"]:
                raise AssertionError(
                    "Profile reconciliation replaced existing credentials")
        user = client(
            endpoint,
            *secret_credentials(secret),
        )
        user.put_object(
            Bucket=BUCKET,
            Key="private-artifacts/compat-user/after-upgrade",
            Body=PAYLOAD,
        )
        verify_payload(user, "private-artifacts/compat-user/after-upgrade")
        verify_state(s3)
    kubectl("rollout", "restart", "deployment/seaweedfs")
    wait_ready("seaweedfs")
    with forward("seaweedfs", 18333, 8333, evidence) as endpoint:
        verify_state(client(endpoint, *admin_credentials))
        user = client(
            endpoint,
            *secret_credentials(secret),
        )
        verify_payload(user, "private-artifacts/compat-user/after-upgrade")
    final_uid = json.loads(
        kubectl("get", "pvc", "seaweedfs-pvc", "-o", "json"))["metadata"]["uid"]
    if final_uid != pvc_uid:
        raise AssertionError("Upgrade replaced the PVC")
    (evidence / "result.json").write_text(
        json.dumps(
            {
                "result":
                    "passed",
                "old_image":
                    "ghcr.io/chrislusf/seaweedfs:4.34",
                "new_image":
                    target,
                "pvc_uid":
                    pvc_uid,
                "payload_sha256":
                    hashlib.sha256(PAYLOAD).hexdigest(),
                "checks": [
                    "old-image-write",
                    "upgrade-read",
                    "lifecycle-persistence",
                    "profile-health",
                    "profile-iam-credentials",
                    "profile-idempotence",
                    "new-image-write",
                    "restart-read",
                    "restart-credentials",
                    "same-pvc",
                ],
            },
            indent=2,
        ) + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--evidence", type=Path, required=True)
    args = parser.parse_args()
    args.evidence.mkdir(parents=True, exist_ok=True)
    run("kind", "create", "cluster", "--name", CLUSTER, "--image",
        "kindest/node:v1.36.1", "--wait", "120s")
    try:
        exercise(args.evidence)
    finally:
        for name, command in {
                "pods.json": ("get", "pods", "-o", "json"),
                "events.txt":
                    ("get", "events", "--sort-by=.metadata.creationTimestamp"),
                "seaweedfs.log":
                    ("logs", "deployment/seaweedfs", "--all-containers=true"),
                "profile.log":
                    ("logs", "deployment/kubeflow-pipelines-profile-controller"
                    ),
        }.items():
            try:
                (args.evidence / name).write_text(kubectl(*command))
            except subprocess.CalledProcessError as error:
                (args.evidence / name).write_text(error.stderr)
        run("kind", "delete", "cluster", "--name", CLUSTER)


if __name__ == "__main__":
    main()
