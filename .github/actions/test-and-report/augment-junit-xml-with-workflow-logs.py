#!/usr/bin/env python3

import argparse
import base64
import os
import re
import subprocess
import sys
import tempfile
from xml.etree import ElementTree
from typing import Optional


REPORTS_DIRNAME = "reports"
JUNIT_XML_FILENAME = "junit.xml"
MAPPING_FILENAME_ENV = "KFP_TEST_WORKFLOW_MAPPING_FILENAME"

MINIO_BUCKET = "mlpipeline"
LOGS_PREFIX_TEMPLATE = "private-artifacts/{namespace}"
MC_ALIAS = "kfp-minio"

MAX_BYTES_PER_WORKFLOW = 200_000
MAX_BYTES_PER_STEP = 80_000


def _warn(message: str) -> None:
    print(f"WARNING: {message}", file=sys.stderr)


def _run(cmd: list[str], *, check: bool = True, capture: bool = True, text: bool = True) -> subprocess.CompletedProcess:
    return subprocess.run(
        cmd,
        check=check,
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.PIPE if capture else None,
        text=text,
    )


def _get_artifact_repo(namespace: str) -> str:
    cp = _run(
        [
            "kubectl",
            "get",
            "configmap",
            # KFP manifests pin this name. If it changes, update both:
            # - manifests/kustomize/third-party/argo/base/workflow-controller-configmap-patch.yaml
            # - this script (CI report augmentation)
            "workflow-controller-configmap",
            "-n",
            namespace,
            "-o",
            "jsonpath={.data.artifactRepository}",
        ]
    )
    return cp.stdout or ""


def _get_minio_creds(namespace: str) -> tuple[str, str]:
    access = _run(
        [
            "kubectl",
            "get",
            "secret",
            # KFP manifests pin this Secret name. If it changes, update both:
            # - manifests/kustomize/third-party/minio/base/mlpipeline-minio-artifact-secret.yaml
            # - manifests/kustomize/third-party/seaweedfs/base/seaweedfs/mlpipeline-minio-artifact-secret.yaml
            # - this script (CI report augmentation)
            "mlpipeline-minio-artifact",
            "-n",
            namespace,
            "-o",
            "jsonpath={.data.accesskey}",
        ]
    ).stdout.strip()
    secret = _run(
        [
            "kubectl",
            "get",
            "secret",
            # Keep in sync with manifests (see comment above).
            "mlpipeline-minio-artifact",
            "-n",
            namespace,
            "-o",
            "jsonpath={.data.secretkey}",
        ]
    ).stdout.strip()
    access_key = base64.b64decode(access.encode("utf-8")).decode("utf-8").strip()
    secret_key = base64.b64decode(secret.encode("utf-8")).decode("utf-8").strip()
    return access_key, secret_key


def _mc_alias_set(mc: str, endpoint: str, access_key: str, secret_key: str) -> None:
    _run([mc, "alias", "set", MC_ALIAS, endpoint, access_key, secret_key, "--api", "S3v4"])


def _mc_find_logs(mc: str, bucket: str, prefix: str) -> list[str]:
    cp = _run([mc, "find", f"{MC_ALIAS}/{bucket}/{prefix}", "--name", "*.log"], check=False)
    paths = []
    if cp.returncode == 0 and cp.stdout:
        for line in cp.stdout.splitlines():
            line = line.strip()
            if line:
                paths.append(line)
    if paths:
        return paths
    cp = _run([mc, "ls", "--recursive", f"{MC_ALIAS}/{bucket}/{prefix}"], check=False)
    if cp.returncode != 0 or not cp.stdout:
        return []
    for line in cp.stdout.splitlines():
        parts = line.split()
        if not parts:
            continue
        p = parts[-1]
        if p.endswith(".log"):
            paths.append(p)
    return paths


def _tail_bytes(s: str, max_bytes: int) -> str:
    b = s.encode("utf-8", errors="replace")
    if len(b) <= max_bytes:
        return s
    tail = b[-max_bytes:]
    return f"[truncated: showing last {max_bytes} bytes of {len(b)}]\n" + tail.decode("utf-8", errors="replace")


def _mc_cat(mc: str, path: str) -> str:
    cp = _run([mc, "cat", path], check=False)
    if cp.returncode != 0:
        err = (cp.stderr or "").strip()
        return f"[mc cat failed for {path}] {err}\n"
    return cp.stdout or ""


def _parse_test_to_workflows_line(line: str) -> Optional[tuple[str, list[str]]]:
    stripped = line.strip()
    if not stripped:
        return None
    parts = stripped.split("|", 1)
    if len(parts) != 2:
        return None
    test_name, workflows_csv = parts[0].strip(), parts[1].strip()
    if not test_name or not workflows_csv:
        return None
    workflows = [w.strip() for w in workflows_csv.split(",") if w.strip()]
    if not workflows:
        return None
    return test_name, workflows


def map_test_to_workflows(mapping_file: str) -> dict[str, list[str]]:
    """Parse a test-to-workflow mapping file into a dictionary.

    The mapping file is expected to contain one mapping per non-empty line, in the form:

        <test_name>|<workflow_name_1>,<workflow_name_2>,...

    Lines that are empty or cannot be parsed by ``_parse_test_to_workflows_line`` are ignored.

    Args:
        mapping_file: Path to the text file containing test-to-workflow mappings.

    Returns:
        A dictionary mapping each test name to a list of associated workflow names.
        If a test name appears on multiple lines, all workflows from those lines are
        accumulated in the list for that test.
    """
    test_to_workflows_map: dict[str, list[str]] = {}
    with open(mapping_file, "r", encoding="utf-8", errors="replace") as f:
        for line in f:
            parsed = _parse_test_to_workflows_line(line)
            if not parsed:
                continue
            test_name, workflows = parsed
            test_to_workflows_map.setdefault(test_name, []).extend(workflows)
    return test_to_workflows_map


def _is_failed_testcase(tc: ElementTree.Element) -> bool:
    return tc.find("failure") is not None or tc.find("error") is not None


def _match_test_name(testcase_name: str, test_to_workflows_map: dict[str, list[str]]) -> Optional[str]:
    if testcase_name in test_to_workflows_map:
        return testcase_name
    norm = re.sub(r"\s+", " ", testcase_name).strip()
    if norm in test_to_workflows_map:
        return norm
    for k in test_to_workflows_map.keys():
        if k in testcase_name or testcase_name in k:
            return k
    return None


def _append_system_out(tc: ElementTree.Element, text_to_append: str) -> None:
    so = tc.find("system-out")
    if so is None:
        so = ElementTree.SubElement(tc, "system-out")
        so.text = ""
    if so.text is None:
        so.text = ""
    if so.text and not so.text.endswith("\n"):
        so.text += "\n"
    so.text += text_to_append


def _require_mc_path(mc_path: str) -> str:
    if not mc_path:
        _warn("MinIO client path is required; skipping workflow log augmentation.")
        return ""
    mc = str(os.fspath(mc_path))
    if not os.path.isfile(mc):
        _warn(f"MinIO client not found at: {mc}; skipping workflow log augmentation.")
        return ""
    if not os.access(mc, os.X_OK):
        _warn(f"MinIO client is not executable: {mc}; skipping workflow log augmentation.")
        return ""
    return mc


def _build_workflow_logs_text(
    *,
    mc: str,
    bucket: str,
    logs_prefix: str,
    workflows: list[str],
    max_bytes_per_workflow: int,
    max_bytes_per_step: int,
) -> str:
    out_lines: list[str] = ["===== Argo Workflows archived logs (tailed) ====="]
    for wf in workflows:
        out_lines.append(f"--- Workflow: {wf} ---")
        wf_prefix = f"{logs_prefix}/{wf}/"
        log_paths = _mc_find_logs(mc, bucket, wf_prefix)
        if not log_paths:
            out_lines.append(f"[no *.log files found under s3://{bucket}/{wf_prefix}]")
            continue

        bytes_budget = max_bytes_per_workflow
        for p in log_paths:
            step_name = os.path.basename(os.path.dirname(p))
            out_lines.append(f"[step: {step_name}]")
            content = _mc_cat(mc, p)
            content = _tail_bytes(content, min(max_bytes_per_step, bytes_budget))
            out_lines.append(content.rstrip("\n"))
            out_lines.append("")
            bytes_budget -= len(content.encode("utf-8", errors="replace"))
            if bytes_budget <= 0:
                out_lines.append(f"[truncated: workflow {wf} exceeded max bytes budget]")
                break

    out_lines.append("===== End Argo Workflows logs =====")
    return "\n".join(out_lines) + "\n"


def _augment_junit_xml(
    *,
    junit_xml_path: str,
    test_to_workflows_map: dict[str, list[str]],
    mc: str,
    bucket: str,
    logs_prefix: str,
    max_bytes_per_workflow: int,
    max_bytes_per_step: int,
) -> int:
    tree = ElementTree.parse(junit_xml_path)
    root = tree.getroot()
    testcases = root.findall(".//testcase")

    modified = 0
    for tc in testcases:
        name = tc.get("name") or ""
        if not name or not _is_failed_testcase(tc):
            continue

        key = _match_test_name(name, test_to_workflows_map)
        if not key:
            continue

        workflows = test_to_workflows_map.get(key, [])
        if not workflows:
            continue

        out = _build_workflow_logs_text(
            mc=mc,
            bucket=bucket,
            logs_prefix=logs_prefix,
            workflows=workflows,
            max_bytes_per_workflow=max_bytes_per_workflow,
            max_bytes_per_step=max_bytes_per_step,
        )
        _append_system_out(tc, out)
        modified += 1

    if modified == 0:
        return 0

    with tempfile.NamedTemporaryFile("wb", delete=False, dir=os.path.dirname(junit_xml_path) or None) as tmp:
        tmp_path = tmp.name
        tree.write(tmp, encoding="utf-8", xml_declaration=True)
    os.replace(tmp_path, junit_xml_path)
    return modified


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--test-directory", required=True)
    parser.add_argument("--namespace", required=True)
    parser.add_argument("--mc-path", required=True)
    args = parser.parse_args()

    test_directory = str(args.test_directory)
    namespace = str(args.namespace)
    mc_path = str(args.mc_path)

    junit_xml = os.path.join(test_directory, REPORTS_DIRNAME, JUNIT_XML_FILENAME)
    mapping_filename = os.environ.get(MAPPING_FILENAME_ENV)
    if not mapping_filename:
        _warn(f"{MAPPING_FILENAME_ENV} is not set; skipping workflow log augmentation.")
        return
    mapping_filename = str(mapping_filename)
    mapping_file = os.path.join(test_directory, REPORTS_DIRNAME, mapping_filename)
    mc = _require_mc_path(mc_path)
    if not mc:
        return

    if not (os.path.isfile(mapping_file) and os.path.isfile(junit_xml)):
        _warn(f"mapping file or junit.xml not found (mapping={mapping_file}, junit={junit_xml}); skipping.")
        return

    test_to_workflows_map = map_test_to_workflows(mapping_file)
    if not test_to_workflows_map:
        _warn("No test->workflows entries found; skipping workflow log augmentation.")
        return

    try:
        artifact_repo = _get_artifact_repo(namespace)
    except subprocess.CalledProcessError as e:
        _warn(f"Failed to read Argo artifact repository config: {e}; skipping.")
        return
    if not re.search(r'^\s*archiveLogs\s*:\s*true\s*(?:#.*)?$', artifact_repo, re.MULTILINE):
        _warn("Argo Workflows log archiving is not enabled (archiveLogs: true not found); skipping.")
        return

    try:
        access_key, secret_key = _get_minio_creds(namespace)
    except subprocess.CalledProcessError as e:
        _warn(f"Failed to read artifact credentials: {e}; skipping.")
        return
    try:
        _mc_alias_set(mc, "http://localhost:9000", access_key, secret_key)
    except subprocess.CalledProcessError as e:
        _warn(f"Failed to configure mc alias: {e}; skipping.")
        return
    modified = _augment_junit_xml(
        junit_xml_path=junit_xml,
        test_to_workflows_map=test_to_workflows_map,
        mc=mc,
        bucket=MINIO_BUCKET,
        logs_prefix=LOGS_PREFIX_TEMPLATE.format(namespace=namespace),
        max_bytes_per_workflow=MAX_BYTES_PER_WORKFLOW,
        max_bytes_per_step=MAX_BYTES_PER_STEP,
    )

    if modified:
        print(f"Updated junit.xml: appended workflow logs to {modified} failing testcase(s).")
    return


if __name__ == "__main__":
    main()

