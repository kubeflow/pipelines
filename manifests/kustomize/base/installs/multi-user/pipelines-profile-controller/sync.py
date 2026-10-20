# Copyright 2020-2021 The Kubeflow Authors
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

import base64
import copy
from datetime import datetime
import hashlib
from http.server import BaseHTTPRequestHandler
from http.server import ThreadingHTTPServer
import ipaddress
import json
import os
import posixpath
import re
from urllib.parse import unquote
from urllib.parse import urlsplit

# From awscli installed in alpine/k8s image
import botocore.session

S3_BUCKET_NAME = 'mlpipeline'
HEALTHZ_PATH = '/healthz'
# Metacontroller records each returned attachment in this annotation and
# three-way merges against it on the next sync.
LAST_APPLIED_ANNOTATION = 'metacontroller.k8s.io/last-applied-configuration'
# Metadata worth echoing back for a Secret this controller does not rewrite.
# Volatile server metadata such as resourceVersion and last-applied state would
# alter the annotation each cycle and reconcile InPlace Secrets forever.
# Stable, verified Namespace ownership is preserved separately where needed.
PRESERVED_SECRET_METADATA = ('name', 'namespace', 'labels', 'annotations')

session = botocore.session.get_session()
# S3 client for lifecycle policy and bucket policy management
s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL",
                                 "http://seaweedfs.kubeflow:8333")
s3 = session.create_client(
    's3', region_name='foobar', endpoint_url=s3_endpoint_url)


def preserved_secret(observed):
    """Desired state for an observed Secret that only external controllers
    populate: its data and intentional metadata, without volatile fields."""
    secret = copy.deepcopy(observed)
    metadata = secret.get('metadata', {})
    secret['metadata'] = {
        key: metadata[key]
        for key in PRESERVED_SECRET_METADATA
        if key in metadata
    }
    annotations = secret['metadata'].get('annotations')
    if annotations is not None:
        annotations.pop(LAST_APPLIED_ANNOTATION, None)
        if not annotations:
            del secret['metadata']['annotations']
    return secret


EXTERNAL_STORAGE_CONFIG_NAME = "kfp-artifact-storage-config"
EXTERNAL_STORAGE_ENV = "KFP_ARTIFACT_EXTERNAL_STORAGE_CONFIG"
POLICY_SOURCE_LABEL = "pipelines.kubeflow.org/artifact-storage-policy-source"
EXTERNAL_STORAGE_PROVISIONAL = "pipelines.kubeflow.org/artifact-storage-provisional"


def external_credential_binding(namespace, namespace_uid, storage, identity,
                                configuration_revision):
    values = [
        "external-file-v1", namespace, namespace_uid, storage["source"],
        storage["bucket"], storage["endpoint"], storage["region"],
        identity["principal"], identity["credentialFile"], identity["profile"],
        configuration_revision
    ]
    return hashlib.sha256("\0".join(values).encode("utf-8")).hexdigest()


def preserved_attachment(observed):
    resource = preserved_secret(observed)
    resource.pop("status", None)
    owners = observed.get("metadata", {}).get("ownerReferences")
    if owners:
        resource["metadata"]["ownerReferences"] = copy.deepcopy(owners)
    return resource


def preserve_namespace_ownership(parent, attachments, resources):
    """Retain verified attachment ownership across pending/managed updates."""
    namespace = parent.get('metadata', {}).get('name')
    namespace_uid = parent.get('metadata', {}).get('uid')
    if not namespace or not namespace_uid:
        return resources
    for resource in resources:
        metadata = resource['metadata']
        # The central authorization record never enters attachment preservation.
        # Keep its ownership outside last-applied state on both paths.
        if (metadata.get('namespace') != namespace or
            (resource['kind'] == 'Secret' and
             metadata.get('name') == artifact_storage_record_name(namespace))):
            continue
        observed = attachments.get(
            f"{resource['kind']}.{resource['apiVersion']}",
            {}).get(f"{metadata['namespace']}/{metadata['name']}", {})
        observed_metadata = observed.get('metadata', {})
        owners = observed_metadata.get('ownerReferences')
        if (observed.get('apiVersion') != resource['apiVersion'] or
                observed.get('kind') != resource['kind'] or any(
                    observed_metadata.get(key) != metadata[key]
                    for key in ['namespace', 'name']) or
                not isinstance(owners, list) or not owners or not all(
                    isinstance(owner, dict) and owner.get('apiVersion') == 'v1'
                    and owner.get('kind') == 'Namespace' and owner.get('name')
                    == namespace and owner.get('uid') == namespace_uid
                    for owner in owners)):
            continue
        # Pending preservation puts these references in last-applied state.
        # Omitting them on recovery would detach our own objects. Never invent
        # references or adopt an orphan/foreign object to repair that symptom.
        metadata['ownerReferences'] = copy.deepcopy(owners)
    return resources


def external_approval_valid(approval, namespace_uid):
    """The installation administrator approves the broker and its policy.

    Tenant annotations and credential contents cannot create this
    approval.
    """
    if not isinstance(approval,
                      dict) or approval.get("namespaceUID") != namespace_uid:
        return False
    storage, identity = approval.get("storage",
                                     {}), approval.get("identity", {})
    if not isinstance(storage, dict) or not isinstance(identity, dict):
        return False
    fields = [
        (namespace_uid, r"[a-zA-Z0-9-]{1,128}"),
        (storage.get("bucket"), r"[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]"),
        (storage.get("region"), r"[a-zA-Z0-9-]{1,64}"),
        (identity.get("principal"), r"[a-zA-Z0-9][a-zA-Z0-9_.:@/+=,-]{0,511}"),
        (identity.get("profile"), r"[a-zA-Z0-9_+=,.@-]{1,128}"),
        (identity.get("credentialFile"), r"/[a-zA-Z0-9_./-]{1,1023}"),
    ]
    if not all(
            isinstance(value, str) and re.fullmatch(pattern, value)
            for value, pattern in fields):
        return False
    credential_file = identity["credentialFile"]
    if (storage.get("source") != "s3" or
            identity.get("kind") != "shared-credentials-file" or
            posixpath.normpath(credential_file) != credential_file or
            credential_file.endswith("/") or credential_file.startswith("//")):
        return False
    endpoint_text = storage.get("endpoint")
    if not isinstance(endpoint_text, str) or not 0 < len(endpoint_text) <= 2048:
        return False
    try:
        endpoint = urlsplit(endpoint_text)
        host, port = endpoint.hostname, endpoint.port
        if (endpoint.scheme not in ["http", "https"] or not host or
                endpoint.username is not None or
                endpoint.password is not None or endpoint.path or
                endpoint.query or endpoint.fragment):
            return False
        if "%" in host:
            return False
        if ":" in host:
            host = "[" + ipaddress.IPv6Address(host).compressed + "]"
        elif not re.fullmatch(r"[a-z0-9.-]+", host):
            return False
        elif re.fullmatch(r"(?:[0-9]+|0x[0-9a-f]+)",
                          host.rstrip(".").split(".")[-1]):
            host = str(ipaddress.IPv4Address(host))
        canonical = f"{endpoint.scheme}://{host}"
        if port is not None and port != (443
                                         if endpoint.scheme == "https" else 80):
            canonical += f":{port}"
        if canonical != endpoint_text:
            return False
    except ValueError:
        return False
    template = approval.get("podTemplate", {})
    if not isinstance(template, dict) or not isinstance(
            template.get("spec"), dict):
        return False
    if any(template["spec"].get(field)
           for field in ["hostNetwork", "hostPID", "hostIPC"]):
        return False
    containers = template["spec"].get("containers", [])
    if (not isinstance(containers, list) or
            not all(isinstance(container, dict) for container in containers) or
            sum(
                container.get("name") == "ml-pipeline-ui-artifact"
                for container in containers) != 1):
        return False
    if not isinstance(template.get("metadata", {}), dict):
        return False
    metadata = template.get("metadata", {})
    if not all(
            isinstance(metadata.get(key, {}), dict)
            for key in ["labels", "annotations"]):
        return False
    volumes = template["spec"].get("volumes", [])
    if not isinstance(volumes, list) or not all(
            isinstance(volume, dict) and isinstance(volume.get("name"), str)
            for volume in volumes):
        return False
    for field in ["containers", "initContainers"]:
        items = template["spec"].get(field, [])
        if not isinstance(items, list) or not all(
                isinstance(item, dict) and isinstance(item.get("name"), str)
                for item in items):
            return False
        if len({item["name"] for item in items}) != len(items):
            return False
        for item in items:
            for key in ["envFrom", "volumeMounts", "volumeDevices", "ports"]:
                if not isinstance(item.get(key, []), list) or not all(
                        isinstance(value, dict) for value in item.get(key, [])):
                    return False
            for key in ["command", "args"]:
                if not isinstance(item.get(key, []), list) or not all(
                        isinstance(value, str) for value in item.get(key, [])):
                    return False
            environment = item.get("env", [])
            if not isinstance(environment, list) or not all(
                    isinstance(variable, dict) and
                    isinstance(variable.get("name"), str)
                    for variable in environment):
                return False
    return True


class ExternalStoragePolicyError(ValueError):
    """An authored diagnostic safe to log without configuration contents."""


def external_json(value):
    """Reject duplicate security settings instead of taking the last value."""

    def unique_object(pairs):
        result = {}
        for key, item in pairs:
            if key in result:
                raise ExternalStoragePolicyError(
                    "Duplicate external storage configuration key")
            result[key] = item
        return result

    return json.loads(value, object_pairs_hook=unique_object)


def exact_fields(value, required, optional=()):
    return (isinstance(value, dict) and set(required) <= value.keys() and
            value.keys() <= set(required) | set(optional))


def enrollment_timestamp(value):
    if not isinstance(value, str) or not re.fullmatch(
            r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,6})?Z",
            value):
        raise ExternalStoragePolicyError(
            "Enrollment requires a UTC timestamp ending in Z")
    return datetime.fromisoformat(value[:-1] + "+00:00")


def policy_source_reference_valid(reference, kfp_namespace):
    return (exact_fields(reference, ["namespace", "configMap", "key"]) and
            reference["namespace"] == kfp_namespace and
            isinstance(reference["configMap"], str) and
            re.fullmatch(r"[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?",
                         reference["configMap"]) and
            isinstance(reference["key"], str) and
            re.fullmatch(r"[a-zA-Z0-9_.-]{1,253}", reference["key"]))


def policy_source(reference, related, kfp_namespace, namespace):
    """Only administrator-published ConfigMaps can supply identity or code."""
    if (not policy_source_reference_valid(reference, kfp_namespace) or
            namespace == kfp_namespace):
        raise ExternalStoragePolicyError(
            "Storage policy sources must be in the installation namespace")
    source = related.get("ConfigMap.v1",
                         {}).get(f"{kfp_namespace}/{reference['configMap']}",
                                 {})
    metadata = source.get("metadata", {})
    if (metadata.get("name") != reference["configMap"] or
            metadata.get("namespace") != kfp_namespace or
            metadata.get("labels", {}).get(POLICY_SOURCE_LABEL) != "true"):
        raise ExternalStoragePolicyError(
            "Storage policy source is missing or lacks its discovery label")
    return external_json(source.get("data", {}).get(reference["key"]))


def external_storage_approval(parent, config, related, kfp_namespace):
    """Resolve one explicit approval or one matching administrator policy.

    Labels select a policy; only its protected UID-bound source supplies
    the principal. Matching policies never fall back to stock credential
    creation.
    """
    namespace = parent["metadata"]["name"]
    namespace_uid = parent["metadata"].get("uid", "")
    if config and (config.get(
            "metadata", {}).get("namespace") != kfp_namespace or config.get(
                "metadata", {}).get("name") != EXTERNAL_STORAGE_CONFIG_NAME):
        raise ExternalStoragePolicyError(
            "Storage configuration must come from the installation namespace")
    data = config.get("data", {})
    approvals = external_json(data.get("approvals.json", "{}"))
    document = external_json(
        data.get("adoption-policies.json", '{"version":1,"policies":[]}'))
    if (not isinstance(approvals, dict) or
            not exact_fields(document, ["version", "policies"]) or
            type(document["version"]) is not int or document["version"] != 1 or
            not isinstance(document["policies"], list)):
        raise ExternalStoragePolicyError(
            "Invalid external storage adoption policy document")
    matching, names = [], set()
    for policy in document["policies"]:
        # A valid nonmatching selector proves this policy cannot authorize this
        # namespace. Its unrelated configuration errors must not stop storage.
        selector = policy.get("namespaceSelector") if isinstance(policy,
                                                                 dict) else None
        if (not exact_fields(selector, ["matchLabels"]) or
                not isinstance(selector["matchLabels"], dict) or not all(
                    isinstance(key, str) and key and isinstance(value, str)
                    for key, value in selector["matchLabels"].items())):
            raise ExternalStoragePolicyError(
                "Invalid external storage namespace selector")
        labels = parent["metadata"].get("labels", {})
        if not all(
                labels.get(key) == value
                for key, value in selector["matchLabels"].items()):
            continue
        if (not exact_fields(policy, [
                "name", "namespaceSelector", "bindingsSource", "allowedStorage",
                "receiverTemplateRef"
        ], [
                "adoptExistingNamespaces", "enrollNewNamespaces",
                "enrollmentCutoff"
        ]) or not isinstance(policy["name"], str) or not re.fullmatch(
                r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", policy["name"]) or
                policy["name"] in names or not all(
                    policy_source_reference_valid(policy[key], kfp_namespace)
                    for key in ["bindingsSource", "receiverTemplateRef"]) or
                not exact_fields(policy["allowedStorage"],
                                 ["source", "endpoint", "bucket", "region"]) or
                not all(
                    type(policy.get(key, True)) is bool for key in
                    ["adoptExistingNamespaces", "enrollNewNamespaces"])):
            raise ExternalStoragePolicyError(
                "Invalid or duplicate external storage adoption policy")
        names.add(policy["name"])
        existing = policy.get("adoptExistingNamespaces", True)
        new = policy.get("enrollNewNamespaces", True)
        if existing != new or "enrollmentCutoff" in policy:
            enrollment_timestamp(policy.get("enrollmentCutoff"))
        matching.append(policy)
    if len(matching) > 1 or (matching and namespace in approvals):
        raise ExternalStoragePolicyError(
            "Overlapping external storage approvals or adoption policies")
    if not matching:
        return approvals.get(namespace), namespace in approvals, None

    policy = matching[0]
    existing = policy.get("adoptExistingNamespaces", True)
    new = policy.get("enrollNewNamespaces", True)
    eligible = existing
    if existing != new:
        created = enrollment_timestamp(
            parent["metadata"].get("creationTimestamp"))
        cutoff = enrollment_timestamp(policy["enrollmentCutoff"])
        eligible = existing if created <= cutoff else new
    if not eligible:
        raise ExternalStoragePolicyError(
            "Matching storage policy excludes this namespace from enrollment")
    bindings = policy_source(policy["bindingsSource"], related, kfp_namespace,
                             namespace)
    template = policy_source(policy["receiverTemplateRef"], related,
                             kfp_namespace, namespace)
    binding = bindings.get(namespace) if isinstance(bindings, dict) else None
    if (not exact_fields(binding, ["namespaceUID", "principal"]) or
            binding["namespaceUID"] != namespace_uid or
            not exact_fields(template, ["identity", "podTemplate"]) or
            not exact_fields(template["identity"],
                             ["kind", "credentialFile", "profile"])):
        raise ExternalStoragePolicyError(
            "Storage policy needs a current UID-bound identity and trusted receiver template"
        )
    approval = {
        "namespaceUID": namespace_uid,
        "storage": policy["allowedStorage"],
        "identity": {
            **template["identity"], "principal": binding["principal"]
        },
        "podTemplate": template["podTemplate"],
    }
    provenance = {"name": policy["name"]}
    if "enrollmentCutoff" in policy:
        provenance["enrollmentCutoff"] = policy["enrollmentCutoff"]
    return approval, True, provenance


def normalized_credential_wiring(pod_spec):
    """Keep source precedence and argument order while normalizing API
    defaults."""
    result = {"volumes": copy.deepcopy(pod_spec.get("volumes", []))}
    defaults = {
        "env": [],
        "envFrom": [],
        "volumeMounts": [],
        "volumeDevices": [],
        "command": [],
        "args": [],
        "workingDir": "",
        "lifecycle": {},
        "restartPolicy": None,
        "securityContext": {}
    }
    for field in ["containers", "initContainers"]:
        result[field] = [{
            key: copy.deepcopy(container.get(key, default))
            for key, default in defaults.items()
        }
                         for container in pod_spec.get(field, [])]
        for container, normalized in zip(
                pod_spec.get(field, []), result[field]):
            normalized["name"] = container["name"]
            security = normalized["securityContext"]
            if isinstance(security, dict):
                for key, default in [("privileged", False),
                                     ("readOnlyRootFilesystem", False),
                                     ("procMount", "Default")]:
                    security.setdefault(key, default)
            for variable in normalized["env"]:
                if "valueFrom" not in variable:
                    variable.setdefault("value", "")
                else:
                    normalize_field_reference(variable["valueFrom"])
                    for key in ["secretKeyRef", "configMapKeyRef"]:
                        if isinstance(variable["valueFrom"].get(key), dict):
                            variable["valueFrom"][key].setdefault(
                                "optional", False)
            for source in normalized["envFrom"]:
                source.setdefault("prefix", "")
                for key in ["secretRef", "configMapRef"]:
                    if isinstance(source.get(key), dict):
                        source[key].setdefault("optional", False)
            for mount in normalized["volumeMounts"]:
                for key, default in [("readOnly", False), ("subPath", ""),
                                     ("subPathExpr", ""),
                                     ("mountPropagation", "None")]:
                    mount.setdefault(key, default)
    for volume in result["volumes"]:
        for key in ["secret", "configMap", "downwardAPI", "projected"]:
            source = volume.get(key)
            if isinstance(source, dict):
                source.setdefault("defaultMode", 420)
        sources = [volume]
        if isinstance(volume.get("projected"), dict):
            sources += volume["projected"].get("sources", [])
        for source in sources:
            for key in ["secret", "configMap"]:
                if isinstance(source.get(key), dict):
                    source[key].setdefault("optional", False)
                    source[key].setdefault("items", [])
            downward = source.get("downwardAPI")
            if isinstance(downward, dict):
                for item in downward.get("items", []):
                    normalize_field_reference(item)
            token = source.get("serviceAccountToken")
            if isinstance(token, dict):
                token.setdefault("expirationSeconds", 3600)
    return result


def normalize_field_reference(source):
    if isinstance(source.get("fieldRef"), dict):
        source["fieldRef"].setdefault("apiVersion", "v1")
    if isinstance(source.get("resourceFieldRef"), dict):
        source["resourceFieldRef"].setdefault("divisor", "1")


def normalized_credential_template(template):
    """Normalize credential fields in a copy, retaining unrelated template
    fields."""
    result = copy.deepcopy(template)
    spec = result.setdefault("spec", {})
    wiring = normalized_credential_wiring(spec)
    spec["volumes"] = wiring["volumes"]
    for field in ["containers", "initContainers"]:
        for container, normalized in zip(
                spec.setdefault(field, []), wiring[field]):
            container.update(normalized)
    return result


# Only authored Kubernetes field names enter logs; arbitrary map keys and all
# field values stay private. Positional indexes locate ordered wiring changes.
READINESS_PATH_FIELDS = frozenset("""
metadata spec template containers initContainers volumes name image imagePullPolicy
labels annotations env envFrom command args value valueFrom workingDir lifecycle
postStart preStop exec httpGet tcpSocket restartPolicy securityContext runAsUser
runAsGroup runAsNonRoot fsGroup privileged readOnlyRootFilesystem procMount
volumeMounts volumeDevices mountPath readOnly subPath subPathExpr mountPropagation
emptyDir projected sources secret configMap downwardAPI serviceAccountToken items
key path defaultMode optional expirationSeconds audience fieldRef fieldPath
apiVersion resourceFieldRef containerName resource divisor secretKeyRef configMapKeyRef
secretRef configMapRef prefix serviceAccountName automountServiceAccountToken
hostNetwork hostPID hostIPC ports port protocol targetPort selector matchLabels
namespaceSelector podSelector policyTypes ingress egress from to type externalName
readinessProbe livenessProbe startupProbe periodSeconds timeoutSeconds replicas
""".split())


def readiness_mismatch_paths(actual, desired, path, subset=False):
    """Describe at most five structural differences, without field values."""
    paths = []

    def append(value):
        if len(paths) < 5:
            paths.append(value[:192])

    def field_path(parent, key):
        if (key in READINESS_PATH_FIELDS and not parent.endswith(
            (".labels", ".annotations", ".matchLabels"))):
            return parent + "." + key
        return parent + "[key]"

    def visit(found, wanted, current):
        if len(paths) >= 5 or found == wanted:
            return
        if len(current) >= 180:
            append(current)
        elif isinstance(wanted, dict) and isinstance(found, dict):
            keys = (["name"] if "name" in wanted else
                    []) + [key for key in wanted if key != "name"]
            for key in keys:
                value = wanted[key]
                child = field_path(current, key)
                if key not in found:
                    append(child)
                else:
                    visit(found[key], value, child)
            if not subset:
                for key in found.keys() - wanted.keys():
                    append(field_path(current, key))
        elif isinstance(wanted, list) and isinstance(found, list):
            if not subset and len(found) != len(wanted):
                append(current)
            for index, value in enumerate(wanted):
                if subset and any(
                        contains_desired(item, value) for item in found):
                    continue
                child = f"{current}[{index}]"
                candidates = [
                    item for item in found
                    if isinstance(item, dict) and isinstance(value, dict) and
                    "name" in value and item.get("name") == value["name"]
                ] if subset else []
                if len(candidates) == 1:
                    visit(candidates[0], value, child)
                elif index < len(found):
                    visit(found[index], value, child)
                else:
                    append(child)
        else:
            append(current)

    visit(actual, desired, path)
    return paths


def readiness_failed(details, stage, paths):
    if details is not None:
        details.update(stage=stage, paths=paths[:5])
    return False


def external_receiver_ready(resources,
                            observed,
                            namespace,
                            failure_details=None):
    if not receiver_rollout_ready(
            resources,
            observed,
            namespace,
            normalize_credential_fields=True,
            failure_details=failure_details):
        return False
    desired = next(
        resource for resource in resources if resource["kind"] == "Deployment")
    actual = observed["Deployment.apps/v1"][
        f"{namespace}/ml-pipeline-ui-artifact"]
    wanted, found = desired["spec"]["template"]["spec"], actual["spec"][
        "template"]["spec"]
    # Defaulted Kubernetes fields are harmless; additional credential sources or
    # broker processes are not. The approved template is the source of truth.
    for field in [
            "serviceAccountName", "automountServiceAccountToken", "hostNetwork",
            "hostPID", "hostIPC"
    ]:
        if found.get(field) != wanted.get(field) and found.get(field) not in [
                None, False
        ]:
            return readiness_failed(failure_details, "pod-identity",
                                    ["Deployment.spec.template.spec." + field])
    actual_wiring = normalized_credential_wiring(found)
    expected_wiring = normalized_credential_wiring(wanted)
    if actual_wiring != expected_wiring:
        return readiness_failed(
            failure_details, "credential-wiring",
            readiness_mismatch_paths(actual_wiring, expected_wiring,
                                     "Deployment.spec.template.spec"))
    return True


def _normalize_domain(domain):
    return domain if domain.startswith('.') else '.' + domain


def artifact_storage_record_name(namespace):
    return f"kfp-artifact-storage-{namespace}"


def artifact_credential_binding(namespace, namespace_uid, endpoint, access_key,
                                secret_key):
    values = [namespace, namespace_uid, endpoint, access_key, secret_key]
    return hashlib.sha256("\0".join(values).encode("utf-8")).hexdigest()


def managed_object_policy(namespace):
    return {
        "Version":
            "2012-10-17",
        "Statement": [{
            "Effect":
                "Allow",
            "Action": ["s3:Put*", "s3:Get*", "s3:List*", "s3:DeleteObject"],
            "Resource": [
                f"arn:aws:s3:::{S3_BUCKET_NAME}/artifacts/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/private-artifacts/{namespace}/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/private/{namespace}/*",
                f"arn:aws:s3:::{S3_BUCKET_NAME}/shared/*",
            ],
        }],
    }


def policy_document(value):
    return json.loads(unquote(value)) if isinstance(value, str) else value


def has_managed_object_policy(document, namespace):
    """Recognize the upstream grant without changing storage-admin
    additions."""
    expected = managed_object_policy(namespace)["Statement"][0]
    for statement in document.get("Statement", []):
        if (statement.get("Effect") == expected["Effect"] and
                not statement.get("Condition") and
                set(statement.get("Action", [])) == set(expected["Action"]) and
                set(statement.get("Resource", [])) == set(
                    expected["Resource"])):
            return True
    return False


def contains_managed_bucket_policy(document):
    statements = document.get("Statement", [])
    return all(
        statement in statements for statement in managed_bucket_list_policy(
            S3_BUCKET_NAME)["Statement"])


def receiver_rollout_ready(desired_resources,
                           attachments,
                           namespace,
                           normalize_credential_fields=False,
                           failure_details=None):
    """Only observed current-template replicas activate the new request
    path."""
    desired = {
        resource["kind"]: resource
        for resource in desired_resources
        if resource["metadata"]["name"] == "ml-pipeline-ui-artifact"
    }
    for kind, version in [("Deployment", "apps/v1"), ("Service", "v1"),
                          ("NetworkPolicy", "networking.k8s.io/v1")]:
        actual = attachments.get(f"{kind}.{version}",
                                 {}).get(f"{namespace}/ml-pipeline-ui-artifact",
                                         {})
        if actual.get("metadata", {}).get("deletionTimestamp"):
            return readiness_failed(failure_details, "resource-deleting",
                                    [kind + ".metadata.deletionTimestamp"])
        actual_spec = actual.get("spec", {})
        expected_spec = desired[kind]["spec"]
        if kind == "Deployment":
            actual_template = copy.deepcopy(actual_spec.get("template", {}))
            # Kubernetes omits empty literal EnvVar.value fields on readback.
            # Preserve nonempty values and valueFrom so changed settings fail.
            for field in ["containers", "initContainers"]:
                for container in actual_template.get("spec", {}).get(field, []):
                    for variable in container.get("env", []):
                        if "valueFrom" not in variable:
                            variable.setdefault("value", "")
            expected_template = expected_spec["template"]
            if normalize_credential_fields:
                actual_template = normalized_credential_template(
                    actual_template)
                expected_template = normalized_credential_template(
                    expected_template)
            # API defaulting adds fields, so compare only fields we generate.
            if not contains_desired(actual_template, expected_template):
                return readiness_failed(
                    failure_details, "deployment-template",
                    readiness_mismatch_paths(
                        actual_template,
                        expected_template,
                        "Deployment.spec.template",
                        subset=True))
            generation = actual.get("metadata", {}).get("generation", 0)
            status = actual.get("status", {})
            replicas = actual_spec.get("replicas", 1)
            if (generation < 1 or replicas < 1 or
                    status.get("observedGeneration", 0) < generation or any(
                        status.get(field, 0) != replicas for field in [
                            "replicas", "updatedReplicas", "availableReplicas",
                            "readyReplicas"
                        ]) or status.get("unavailableReplicas", 0) != 0):
                paths = []
                if generation < 1:
                    paths.append("Deployment.metadata.generation")
                if replicas < 1:
                    paths.append("Deployment.spec.replicas")
                if status.get("observedGeneration", 0) < generation:
                    paths.append("Deployment.status.observedGeneration")
                paths += [
                    "Deployment.status." + field for field in [
                        "replicas", "updatedReplicas", "availableReplicas",
                        "readyReplicas"
                    ] if status.get(field, 0) != replicas
                ]
                if status.get("unavailableReplicas", 0) != 0:
                    paths.append("Deployment.status.unavailableReplicas")
                return readiness_failed(failure_details, "deployment-rollout",
                                        paths)
        elif kind == "NetworkPolicy" and actual_spec != expected_spec:
            return readiness_failed(
                failure_details, "network-policy",
                readiness_mismatch_paths(actual_spec, expected_spec,
                                         "NetworkPolicy.spec"))
        elif kind == "Service" and (actual_spec.get(
                "type", "ClusterIP") != "ClusterIP" or
                                    actual_spec.get("externalName")):
            return readiness_failed(
                failure_details, "service-type",
                ["Service.spec.type", "Service.spec.externalName"])
        elif not contains_desired(actual_spec, expected_spec):
            return readiness_failed(
                failure_details, "service-spec",
                readiness_mismatch_paths(
                    actual_spec, expected_spec, "Service.spec", subset=True))
    return True


def contains_desired(actual, desired):
    if isinstance(desired, dict):
        return isinstance(actual, dict) and all(
            key in actual and contains_desired(actual[key], value)
            for key, value in desired.items())
    if isinstance(desired, list):
        return isinstance(actual, list) and len(actual) >= len(desired) and all(
            any(contains_desired(item, expected)
                for item in actual)
            for expected in desired)
    return actual == desired


def artifact_server_environment(namespace,
                                cluster_domain,
                                allowed_artifact_endpoints,
                                http_base_url="",
                                object_store_host="seaweedfs",
                                kfp_namespace="kubeflow",
                                namespace_uid=""):
    return [
        {
            "name": "KFP_ARTIFACT_NAMESPACE_UID",
            "value": namespace_uid
        },
        {
            "name": "MINIO_HOST",
            "value": object_store_host
        },
        {
            "name": "MINIO_NAMESPACE",
            "value": kfp_namespace
        },
        {
            "name": "MINIO_PORT",
            "value": "8333"
        },
        {
            "name": "MINIO_SSL",
            "value": "false"
        },
        {
            "name": "ARTIFACTS_REQUIRE_PROXY_NAMESPACE",
            "value": "false",
        },
        {
            "name": "MINIO_ACCESS_KEY",
            "valueFrom": {
                "secretKeyRef": {
                    "key": "accesskey",
                    "name": "mlpipeline-minio-artifact"
                }
            }
        },
        {
            "name": "MINIO_SECRET_KEY",
            "valueFrom": {
                "secretKeyRef": {
                    "key": "secretkey",
                    "name": "mlpipeline-minio-artifact"
                }
            }
        },
        {
            "name": "ML_PIPELINE_SERVICE_HOST",
            "value": f"ml-pipeline.kubeflow{_normalize_domain(cluster_domain)}"
        },
        {
            "name": "ML_PIPELINE_SERVICE_PORT",
            "value": "8888"
        },
        {
            "name": "FRONTEND_SERVER_NAMESPACE",
            "value": namespace,
        },
        {
            "name": "CLUSTER_DOMAIN",
            "value": cluster_domain,
        },
        {
            "name": "HTTP_BASE_URL",
            "value": http_base_url,
        },
        {
            "name": "ALLOWED_ARTIFACT_ENDPOINTS",
            "value": allowed_artifact_endpoints,
        },
    ]


def managed_bucket_list_policy(bucket_name):
    bucket_arn = f"arn:aws:s3:::{bucket_name}"
    desired = {
        "Version":
            "2012-10-17",
        "Statement": [
            {
                "Sid": "AllowOwnDeepList",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringLike": {
                        "s3:prefix": [
                            "private-artifacts/${aws:username}/*",
                            "private/${aws:username}/*",
                            "artifacts/*",
                            "shared/*",
                        ]
                    }
                },
            },
            {
                "Sid": "AllowHeadBucket",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringEquals": {
                        "s3:RequestMethod": "HEAD"
                    }
                },
            },
            {
                "Sid": "AllowTopLevelDelimitedBrowse",
                "Effect": "Allow",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "Null": {
                        "s3:prefix": "true"
                    },
                    "StringEquals": {
                        "s3:delimiter": "/"
                    },
                },
            },
            {
                "Sid": "DenyOtherDeepList",
                "Effect": "Deny",
                "Principal": "*",
                "Action": "s3:ListBucket",
                "Resource": bucket_arn,
                "Condition": {
                    "StringLike": {
                        "s3:prefix": ["private-artifacts/*", "private/*"]
                    },
                    "StringNotLike": {
                        "s3:prefix": [
                            "private-artifacts/${aws:username}/*",
                            "private/${aws:username}/*",
                        ]
                    },
                },
            },
        ],
    }
    return desired


def create_iam_client():
    # To interact with SeaweedFS user management. Region does not matter.
    endpoint_url = os.environ.get("AWS_ENDPOINT_URL")
    if endpoint_url:
        return session.create_client(
            'iam', region_name='foobar', endpoint_url=endpoint_url)
    return session.create_client('iam', region_name='foobar')


iam = create_iam_client()


def main():
    settings = get_settings_from_env()
    server = server_factory(**settings)
    server.serve_forever()


def get_settings_from_env(controller_port=None,
                          frontend_image=None,
                          frontend_tag=None,
                          disable_istio_sidecar=None,
                          artifacts_proxy_enabled=None,
                          allowed_artifact_endpoints=None,
                          artifact_retention_days=None,
                          cluster_domain=None,
                          object_store_host=None,
                          http_base_url=None,
                          kfp_namespace=None):
    """Returns a dict of settings from environment variables relevant to the
    controller.

    Environment settings can be overridden by passing them here as arguments.

    Settings are pulled from the all-caps version of the setting name.  The
    following defaults are used if those environment variables are not set
    to enable backwards compatibility with previous versions of this script:
        frontend_image: ghcr.io/kubeflow/kfp-frontend
        frontend_tag: value of KFP_VERSION environment variable
        disable_istio_sidecar: Required (no default)
    """
    settings = dict()
    settings["controller_port"] = \
        controller_port or \
        os.environ.get("CONTROLLER_PORT", "8080")

    settings["frontend_image"] = \
        frontend_image or \
        os.environ.get("FRONTEND_IMAGE", "ghcr.io/kubeflow/kfp-frontend")

    settings["artifacts_proxy_enabled"] = \
        artifacts_proxy_enabled or \
        os.environ.get("ARTIFACTS_PROXY_ENABLED", "false")

    settings[
        "http_base_url"] = http_base_url if http_base_url is not None else os.environ.get(
            "HTTP_BASE_URL", "")

    settings["allowed_artifact_endpoints"] = \
        allowed_artifact_endpoints if allowed_artifact_endpoints is not None \
            else os.environ.get("ALLOWED_ARTIFACT_ENDPOINTS", "")

    settings["artifact_retention_days"] = \
        artifact_retention_days or \
        os.environ.get("ARTIFACT_RETENTION_DAYS", -1)

    settings["cluster_domain"] = \
        cluster_domain or \
        os.environ.get("CLUSTER_DOMAIN", ".svc.cluster.local")

    settings["kfp_namespace"] = \
        kfp_namespace or os.environ.get("KFP_NAMESPACE", "kubeflow")

    settings["object_store_host"] = \
        object_store_host or \
        os.environ.get("OBJECT_STORE_HOST", "seaweedfs")

    # Look for specific tags for each image first, falling back to
    # previously used KFP_VERSION environment variable for backwards
    # compatibility
    settings["frontend_tag"] = \
        frontend_tag or \
        os.environ.get("FRONTEND_TAG") or \
        os.environ["KFP_VERSION"]

    settings["disable_istio_sidecar"] = \
        disable_istio_sidecar if disable_istio_sidecar is not None \
            else os.environ.get("DISABLE_ISTIO_SIDECAR") == "true"

    return settings


def server_factory(frontend_image,
                   frontend_tag,
                   disable_istio_sidecar,
                   artifacts_proxy_enabled,
                   artifact_retention_days,
                   allowed_artifact_endpoints="",
                   cluster_domain=".svc.cluster.local",
                   object_store_host="seaweedfs",
                   url="",
                   controller_port=8080,
                   http_base_url="",
                   kfp_namespace="kubeflow"):
    """Returns a ThreadingHTTPServer populated with Handler with customized
    settings."""

    class Controller(BaseHTTPRequestHandler):

        def ensure_namespace_object_policy(self, namespace):
            policy_name = f"KubeflowProject{namespace}"
            try:
                iam.get_user_policy(UserName=namespace, PolicyName=policy_name)
                return
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error", {}).get("Code") != "NoSuchEntity":
                    raise
            # Establish the user and policy before minting credentials: a failed
            # IAM request can then retry without creating orphan access keys.
            try:
                iam.create_user(UserName=namespace)
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error",
                                   {}).get("Code") != "EntityAlreadyExists":
                    raise
            # Another provisioner or storage administrator may have supplied a
            # policy while the user was created. Never replace that policy.
            try:
                iam.get_user_policy(UserName=namespace, PolicyName=policy_name)
                return
            except Exception as exception:
                if getattr(exception, "response",
                           {}).get("Error", {}).get("Code") != "NoSuchEntity":
                    raise
            iam.put_user_policy(
                UserName=namespace,
                PolicyName=policy_name,
                PolicyDocument=json.dumps(managed_object_policy(namespace)))

        def artifact_storage_record(self, parent, attachments, resources):
            namespace = parent["metadata"]["name"]
            namespace_uid = parent["metadata"].get("uid", "")
            record = {
                "version": 1,
                "namespace": namespace,
                "namespaceUID": namespace_uid,
                "state": "unsupported",
                "bucket": S3_BUCKET_NAME,
            }
            secret = next(
                (resource for resource in resources
                 if resource["kind"] == "Secret" and
                 resource["metadata"]["name"] == "mlpipeline-minio-artifact"),
                {})
            try:
                data = secret.get("data", {})
                access_key = base64.b64decode(
                    data["accesskey"], validate=True).decode("utf-8")
                secret_key = base64.b64decode(
                    data["secretkey"], validate=True).decode("utf-8")
                if not access_key or not secret_key or not namespace_uid:
                    raise ValueError("Incomplete namespace storage credentials")
            except (KeyError, ValueError, UnicodeError):
                return self.artifact_storage_record_secret(namespace, record)

            record["credentialBinding"] = artifact_credential_binding(
                namespace, namespace_uid,
                f"http://{object_store_host}.{kfp_namespace}:8333", access_key,
                secret_key)
            # Environment Secret references refresh only when the Pod restarts.
            # A separate digest triggers rollout without exposing the request binding.
            receiver = next(resource for resource in resources
                            if resource["kind"] == "Deployment")
            receiver["spec"]["template"]["metadata"]["annotations"][
                "kubeflow-pipelines/artifact-storage-revision"] = hashlib.sha256(
                    ("receiver-rollout\0" + record["credentialBinding"]
                    ).encode("utf-8")).hexdigest()[:16]
            try:
                # IAM is the admin-controlled authority; a tenant Secret's name,
                # labels and readiness never establish its storage identity.
                keys = iam.list_access_keys(UserName=namespace)
                markers = set()
                while not any(
                        key.get("AccessKeyId") == access_key and
                        key.get("Status") == "Active"
                        for key in keys.get("AccessKeyMetadata", [])):
                    if not keys.get("IsTruncated"):
                        return self.artifact_storage_record_secret(
                            namespace, record)
                    marker = keys.get("Marker")
                    if not marker or marker in markers:
                        raise ValueError("Incomplete IAM access-key pagination")
                    markers.add(marker)
                    keys = iam.list_access_keys(
                        UserName=namespace, Marker=marker)
                policy = policy_document(
                    iam.get_user_policy(
                        UserName=namespace,
                        PolicyName=f"KubeflowProject{namespace}")
                    ["PolicyDocument"])
                if not has_managed_object_policy(policy, namespace):
                    return self.artifact_storage_record_secret(
                        namespace, record)
                # Initialization can fail after credentials have been persisted.
                # Retry only for a verified namespace identity; add missing managed
                # statements without replacing administrator or conflicting policy.
                self.ensure_bucket_list_policy(S3_BUCKET_NAME)
                bucket_policy = policy_document(
                    s3.get_bucket_policy(Bucket=S3_BUCKET_NAME)["Policy"])
                record["state"] = "pending"
                if contains_managed_bucket_policy(
                        bucket_policy) and receiver_rollout_ready(
                            resources, attachments, namespace):
                    record["state"] = "ready"
            except Exception as exception:
                error_code = getattr(exception, "response",
                                     {}).get("Error", {}).get("Code")
                record[
                    "state"] = "unsupported" if error_code == "NoSuchEntity" else "pending"
                # Exceptions can carry request credentials; log only the class.
                print(
                    f"Artifact storage authorization for {namespace} is {record['state']}; "
                    f"verify namespace IAM and the bucket list policy ({type(exception).__name__})."
                )
            return self.artifact_storage_record_secret(namespace, record)

        def artifact_storage_record_secret(self, namespace, record):
            return {
                "apiVersion": "v1",
                "kind": "Secret",
                "metadata": {
                    "name": artifact_storage_record_name(namespace),
                    "namespace": kfp_namespace,
                },
                "type": "Opaque",
                "data": {
                    "authorization.json":
                        base64.b64encode(
                            json.dumps(record, sort_keys=True).encode("utf-8")
                        ).decode("ascii")
                },
            }

        def upsert_lifecycle_policy(self, bucket_name, artifact_retention_days):
            """Configures or deletes the lifecycle policy based on the
            artifact_retention_days string."""
            try:
                retention_days = int(artifact_retention_days)
            except ValueError:
                print(
                    f"ERROR: ARTIFACT_RETENTION_DAYS value '{artifact_retention_days}' is not a valid integer. Aborting policy update."
                )
                return

            # To disable lifecycle policy we need to delete it
            if retention_days <= 0:
                print(
                    f"ARTIFACT_RETENTION_DAYS is non-positive ({retention_days} days). Attempting to delete lifecycle policy."
                )
                try:
                    response = s3.get_bucket_lifecycle_configuration(
                        Bucket=bucket_name)
                    # Check if there are any enabled rules
                    has_enabled_rules = any(
                        rule.get('Status') == 'Enabled'
                        for rule in response.get('Rules', []))

                    if has_enabled_rules:
                        s3.delete_bucket_lifecycle(Bucket=bucket_name)
                        print("Successfully deleted lifecycle policy.")
                    else:
                        print("No enabled lifecycle rules found to delete.")
                except Exception:
                    print(f"Warning: No lifecycle policy exists")
                return

            # Create/update lifecycle policy
            life_cycle_policy = {
                "Rules": [{
                    "Status": "Enabled",
                    "Filter": {
                        "Prefix": "private-artifacts"
                    },
                    "Expiration": {
                        "Days": retention_days
                    },
                    "ID": "private-artifacts",
                },]
            }
            print('upsert_lifecycle_policy:', life_cycle_policy)

            try:
                api_response = s3.put_bucket_lifecycle_configuration(
                    Bucket=bucket_name,
                    LifecycleConfiguration=life_cycle_policy)
                print('Lifecycle policy configured successfully:', api_response)
            except Exception as exception:
                if hasattr(exception,
                           'response') and 'Error' in exception.response:
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception.response['Error']['Code']} - {exception}"
                    )
                else:
                    print(
                        f"ERROR: Failed to configure lifecycle policy: {exception}"
                    )

        def ensure_bucket_list_policy(self, bucket_name):
            """Bucket-wide ListBucket isolation, keyed on ${aws:username}.

            Top-level folder browsing is allowed for everyone; listing
            deeper inside another namespace's private prefix is denied.
            HeadBucket (KServe storage-initializer) is allowed. Set as a
            bucket policy because the embedded user-policy engine is
            Allow-only and cannot express Deny; SeaweedFS evaluates the
            bucket policy before it.
            """
            desired = managed_bucket_list_policy(bucket_name)
            try:
                current = policy_document(
                    s3.get_bucket_policy(Bucket=bucket_name)["Policy"])
            except Exception as exception:
                code = getattr(exception, "response", {}).get("Error",
                                                              {}).get("Code")
                if code != "NoSuchBucketPolicy":
                    print(
                        f"Cannot inspect bucket list policy for {bucket_name}; "
                        "leaving existing policy unchanged.")
                    return
                current = {"Version": "2012-10-17", "Statement": []}
            statements = current.get("Statement", [])
            for required in desired["Statement"]:
                same_sid = [
                    statement for statement in statements
                    if statement.get("Sid") == required["Sid"]
                ]
                if same_sid and same_sid != [required]:
                    print(
                        f"Bucket list policy {bucket_name} has a conflicting managed "
                        "statement; preserve it and review the storage configuration."
                    )
                    return
            missing = [
                required for required in desired["Statement"]
                if required not in statements
            ]
            if not missing:
                return
            # Existing administrator statements remain intact.
            current["Statement"] = statements + missing
            try:
                s3.put_bucket_policy(
                    Bucket=bucket_name, Policy=json.dumps(current))
                print(f"Added managed ListBucket statements on {bucket_name}")
            except Exception:
                print(f"Cannot update bucket list policy for {bucket_name}; "
                      "storage authorization remains pending.")

        def customize(self, parent):
            namespace = parent["metadata"]["name"]
            return {
                "relatedResources": [
                    {
                        "apiVersion": "v1",
                        "resource": "configmaps",
                        "namespace": kfp_namespace,
                        "names": [EXTERNAL_STORAGE_CONFIG_NAME]
                    },
                    {
                        "apiVersion": "v1",
                        "resource": "configmaps",
                        "namespace": kfp_namespace,
                        "labelSelector": {
                            "matchLabels": {
                                POLICY_SOURCE_LABEL: "true"
                            }
                        }
                    },
                    *[{
                        "apiVersion": version,
                        "resource": resource,
                        "namespace": namespace,
                        "names": ["ml-pipeline-ui-artifact"]
                    } for version, resource in [(
                        "apps/v1", "deployments"), ("v1", "services"),
                                                ("networking.k8s.io/v1",
                                                 "networkpolicies")]],
                ]
            }

        def external_storage_sync(self, parent, attachments, related):
            namespace = parent["metadata"]["name"]
            namespace_uid = parent["metadata"].get("uid", "")
            config = related.get("ConfigMap.v1", {}).get(
                f"{kfp_namespace}/{EXTERNAL_STORAGE_CONFIG_NAME}", {})
            prior = attachments.get("Secret.v1", {}).get(
                f"{kfp_namespace}/{artifact_storage_record_name(namespace)}",
                {})
            previous_external = f"{kfp_namespace}/{artifact_storage_record_name(namespace)}" in attachments.get(
                "Secret.v1", {})
            try:
                previous_record = json.loads(
                    base64.b64decode(
                        prior.get("data", {}).get("authorization.json", "")))
                # Only our installation-namespace record can mark a transient
                # config error as unadopted. Unmarked legacy v2 records retain
                # their conservative revocation tombstone, even without a Pod.
                provisional = (
                    prior.get("metadata", {}).get("namespace") == kfp_namespace
                    and prior.get("metadata", {}).get("name")
                    == artifact_storage_record_name(namespace) and
                    prior.get("metadata", {}).get(
                        "annotations", {}).get(EXTERNAL_STORAGE_PROVISIONAL)
                    == "true" and previous_record.get("version") == 2 and
                    previous_record.get("namespace") == namespace and
                    previous_record.get("namespaceUID") == namespace_uid and
                    previous_record.get("state") == "pending" and
                    not any(key in previous_record for key in [
                        "storage", "identity", "configurationRevision",
                        "credentialBinding"
                    ]))
                managed = (
                    previous_record.get("version") == 1 and
                    previous_record.get("namespace") == namespace and
                    previous_record.get("namespaceUID") == namespace_uid and
                    previous_record.get("state")
                    in ["ready", "pending", "unsupported"])
                previous_external = not (managed or provisional)
            except (ValueError, TypeError, UnicodeError, AttributeError):
                pass
            existing = attachments.get("Deployment.apps/v1", {}).get(
                f"{namespace}/ml-pipeline-ui-artifact", {})
            previous_external = previous_external or any(
                variable.get("name") == EXTERNAL_STORAGE_ENV
                for receiver in [
                    existing,
                    related.get("Deployment.apps/v1", {}).get(
                        f"{namespace}/ml-pipeline-ui-artifact", {})
                ]
                for container in receiver.get("spec", {}).get(
                    "template", {}).get("spec", {}).get("containers", [])
                for variable in container.get("env", []))
            invalid = False
            policy_error = None
            provenance = None
            try:
                approval, approval_present, provenance = external_storage_approval(
                    parent, config, related, kfp_namespace)
            except ExternalStoragePolicyError as error:
                invalid, approval, approval_present = True, None, True
                policy_error = str(error)
            except (ValueError, TypeError, AttributeError):
                invalid, approval, approval_present = True, None, True
            if not approval_present and not invalid and not previous_external:
                return None

            # Preserve all previously owned resources while an approval is
            # unavailable or revoked. Never fall back to minting stock credentials.
            resources = [
                preserved_attachment(resource)
                for group in attachments.values()
                for resource in group.values()
                if not (resource.get("kind") == "Secret" and resource.get(
                    "metadata", {}).get("namespace") == kfp_namespace and
                        resource.get("metadata", {}).get(
                            "name") == artifact_storage_record_name(namespace))
            ]
            record = {
                "version": 2,
                "namespace": namespace,
                "namespaceUID": namespace_uid,
                "state": "pending"
            }

            if provenance:
                record["adoptionPolicy"] = provenance
            approved = not invalid and external_approval_valid(
                approval, namespace_uid)

            def record_secret():
                secret = self.artifact_storage_record_secret(namespace, record)
                # Validation alone commits the external identity: revocation
                # before its first successful rollout must still fail closed.
                if not previous_external and not approved:
                    secret["metadata"]["annotations"] = {
                        EXTERNAL_STORAGE_PROVISIONAL: "true"
                    }
                return secret

            def pending(reason):
                print(
                    f"External artifact storage authorization for {namespace} is pending: {reason}.",
                    flush=True)
                return {
                    "status": {
                        "kubeflow-pipelines-ready": "False"
                    },
                    "attachments": resources + [record_secret()],
                    "resyncAfterSeconds": 10
                }

            if not approved:
                return pending(
                    policy_error or
                    "verify the installation administrator's storage approval, adoption policy, protected sources, and namespace UID"
                )
            observed = copy.deepcopy(attachments)
            for kind, version in [("Deployment", "apps/v1"), ("Service", "v1"),
                                  ("NetworkPolicy", "networking.k8s.io/v1")]:
                key = f"{kind}.{version}"
                name = f"{namespace}/ml-pipeline-ui-artifact"
                candidate = related.get(key, {}).get(name)
                attached = attachments.get(key, {}).get(name, {})
                foreign_owner = any(
                    owner.get("kind") != "Namespace" or
                    owner.get("uid") != namespace_uid for owner in attached.get(
                        "metadata", {}).get("ownerReferences", []))
                if foreign_owner or (candidate and
                                     name not in attachments.get(key, {})):
                    return pending(
                        "the existing artifact receiver resources belong to another controller; retain that owner and configure a compatible managed receiver"
                    )
                if candidate:
                    observed.setdefault(key, {})[name] = candidate

            configuration_revision = hashlib.sha256(
                json.dumps(
                    approval["podTemplate"],
                    sort_keys=True,
                    separators=(",", ":"),
                    ensure_ascii=True).encode("utf-8")).hexdigest()
            template = copy.deepcopy(approval["podTemplate"])
            metadata = template.setdefault("metadata", {})
            metadata.setdefault("labels", {})["app"] = "ml-pipeline-ui-artifact"
            container = next(item for item in template["spec"]["containers"]
                             if item["name"] == "ml-pipeline-ui-artifact")
            container["image"] = f"{frontend_image}:{frontend_tag}"
            container.pop("command", None)
            container.pop("args", None)
            receiver_config = {
                key: approval[key]
                for key in ["namespaceUID", "storage", "identity"]
            }
            receiver_config["configurationRevision"] = configuration_revision
            environment = artifact_server_environment(
                namespace=namespace,
                cluster_domain=cluster_domain,
                allowed_artifact_endpoints=allowed_artifact_endpoints,
                http_base_url=http_base_url,
                object_store_host=object_store_host,
                kfp_namespace=kfp_namespace,
                namespace_uid=namespace_uid)
            # External storage never depends on the stock SeaweedFS Secret.
            environment = [
                item for item in environment
                if not item["name"].startswith("MINIO_")
            ]
            environment += [{
                "name": "ENABLE_AUTHZ",
                "value": "false"
            }, {
                "name": "KFP_SECURITY_ARTIFACT_AUTHORIZATION_MODE",
                "value": "auto"
            }, {
                "name": "ARTIFACTS_SERVICE_PROXY_ENABLED",
                "value": "false"
            }, {
                "name":
                    EXTERNAL_STORAGE_ENV,
                "value":
                    json.dumps(
                        receiver_config, sort_keys=True, separators=(",", ":"))
            }]
            protected = {item["name"] for item in environment}
            container["env"] = [
                item for item in container.get("env", [])
                if item["name"] not in protected
            ] + environment
            container["readinessProbe"] = {
                "tcpSocket": {
                    "port": 3000
                },
                "periodSeconds": 2,
                "timeoutSeconds": 1
            }
            template["spec"].setdefault("serviceAccountName", "default-editor")
            receiver_resources = [{
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {
                    "name": "ml-pipeline-ui-artifact",
                    "namespace": namespace
                },
                "spec": {
                    "selector": {
                        "matchLabels": {
                            "app": "ml-pipeline-ui-artifact"
                        }
                    },
                    "template": template
                }
            }, {
                "apiVersion": "v1",
                "kind": "Service",
                "metadata": {
                    "name": "ml-pipeline-ui-artifact",
                    "namespace": namespace
                },
                "spec": {
                    "ports": [{
                        "name": "http",
                        "port": 80,
                        "protocol": "TCP",
                        "targetPort": 3000
                    }],
                    "selector": {
                        "app": "ml-pipeline-ui-artifact"
                    }
                }
            }, {
                "apiVersion": "networking.k8s.io/v1",
                "kind": "NetworkPolicy",
                "metadata": {
                    "name": "ml-pipeline-ui-artifact",
                    "namespace": namespace
                },
                "spec": {
                    "podSelector": {
                        "matchLabels": {
                            "app": "ml-pipeline-ui-artifact"
                        }
                    },
                    "policyTypes": ["Ingress"],
                    "ingress": [{
                        "from": [{
                            "namespaceSelector": {
                                "matchLabels": {
                                    "kubernetes.io/metadata.name": kfp_namespace
                                }
                            },
                            "podSelector": {
                                "matchLabels": {
                                    "app": "ml-pipeline-ui"
                                }
                            }
                        }],
                        "ports": [{
                            "protocol": "TCP",
                            "port": 3000
                        }]
                    }]
                }
            }]
            # A pending sync preserves these references in last-applied state.
            # Keep the already validated ownership when regenerating the receiver,
            # or Metacontroller's three-way update would detach our own objects.
            for resource in receiver_resources:
                attached = attachments.get(
                    f"{resource['kind']}.{resource['apiVersion']}",
                    {}).get(f"{namespace}/ml-pipeline-ui-artifact", {})
                owners = attached.get("metadata", {}).get("ownerReferences")
                if owners:
                    resource["metadata"]["ownerReferences"] = copy.deepcopy(
                        owners)
            resources = [
                resource for resource in resources if not (
                    resource.get("kind") in
                    ["Deployment", "Service", "NetworkPolicy"] and
                    resource["metadata"].get("namespace") == namespace and
                    resource["metadata"]["name"] == "ml-pipeline-ui-artifact")
            ]
            record.update({
                "storage":
                    approval["storage"],
                "identity":
                    approval["identity"],
                "configurationRevision":
                    configuration_revision,
                "credentialBinding":
                    external_credential_binding(namespace, namespace_uid,
                                                approval["storage"],
                                                approval["identity"],
                                                configuration_revision)
            })
            failure_details = {}
            if external_receiver_ready(receiver_resources, observed, namespace,
                                       failure_details):
                record["state"] = "ready"
            else:
                print(
                    f"External artifact storage readiness for {namespace} is pending: "
                    + json.dumps(
                        failure_details, sort_keys=True, separators=(",", ":")),
                    flush=True)
            policy = receiver_resources[2]
            observed_policy = observed.get(
                "NetworkPolicy.networking.k8s.io/v1",
                {}).get(f"{namespace}/ml-pipeline-ui-artifact", {})
            if not existing and observed_policy.get("spec") != policy["spec"]:
                receiver_resources = [
                    resource for resource in receiver_resources
                    if resource["kind"] != "Deployment"
                ]
            resources += receiver_resources
            result = {
                "status": {
                    "kubeflow-pipelines-ready":
                        "True" if record["state"] == "ready" else "False"
                },
                "attachments": resources + [record_secret()]
            }
            if record["state"] == "pending":
                result["resyncAfterSeconds"] = 10
            return result

        def sync(self, parent, attachments, related=None):
            # parent is a namespace
            namespace = parent.get("metadata", {}).get("name")

            pipeline_enabled = parent.get("metadata", {}).get(
                "labels", {}).get("pipelines.kubeflow.org/enabled")

            if pipeline_enabled != "true":
                return {"status": {}, "attachments": []}

            if artifacts_proxy_enabled.lower() == "true":
                external = self.external_storage_sync(parent, attachments,
                                                      related or {})
                if external is not None:
                    return external

            # Compute status based on observed state.
            desired_status = {
                "kubeflow-pipelines-ready":
                    len([
                        key for key in attachments["Secret.v1"]
                        if key.startswith(f"{namespace}/")
                    ]) == 1 and len(attachments["ConfigMap.v1"]) == 3 and
                    len(attachments["Deployment.apps/v1"]) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    len(attachments["Service.v1"]) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    len(
                        attachments.get("NetworkPolicy.networking.k8s.io/v1",
                                        {})) ==
                    (1 if artifacts_proxy_enabled.lower() == "true" else 0) and
                    "True" or "False"
            }

            # Generate the desired attachment object(s).
            desired_resources = [
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "kfp-launcher",
                        "namespace": namespace,
                    },
                    "data": {
                        "defaultPipelineRoot":
                            f"minio://{S3_BUCKET_NAME}/private-artifacts/{namespace}/v2/artifacts",
                        "clusterDomain":
                            cluster_domain,
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "metadata-grpc-configmap",
                        "namespace": namespace,
                    },
                    "data": {
                        "METADATA_GRPC_SERVICE_HOST":
                            "metadata-grpc-service.kubeflow",
                        "METADATA_GRPC_SERVICE_PORT":
                            "8080",
                    },
                },
                {
                    "apiVersion": "v1",
                    "kind": "ConfigMap",
                    "metadata": {
                        "name": "artifact-repositories",
                        "namespace": namespace,
                        "annotations": {
                            "workflows.argoproj.io/default-artifact-repository":
                                "default-namespaced"
                        }
                    },
                    "data": {
                        "default-namespaced":
                            json.dumps({
                                "archiveLogs": True,
                                "s3": {
                                    "endpoint":
                                        f"{object_store_host}.kubeflow{_normalize_domain(cluster_domain)}:9000",
                                    "bucket":
                                        S3_BUCKET_NAME,
                                    "keyFormat":
                                        f"private-artifacts/{namespace}/{{{{workflow.name}}}}/{{{{workflow.creationTimestamp.Y}}}}/{{{{workflow.creationTimestamp.m}}}}/{{{{workflow.creationTimestamp.d}}}}/{{{{pod.name}}}}",
                                    "insecure":
                                        True,
                                    "accessKeySecret": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "accesskey",
                                    },
                                    "secretKeySecret": {
                                        "name": "mlpipeline-minio-artifact",
                                        "key": "secretkey",
                                    }
                                }
                            })
                    }
                },
            ]

            # Add artifact fetcher related resources if enabled
            if artifacts_proxy_enabled.lower() == "true":
                desired_resources.extend([
                    {
                        "apiVersion": "networking.k8s.io/v1",
                        "kind": "NetworkPolicy",
                        "metadata": {
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                        },
                        "spec": {
                            "podSelector": {
                                "matchLabels": {
                                    "app": "ml-pipeline-ui-artifact"
                                }
                            },
                            "policyTypes": ["Ingress"],
                            # Both selectors must match: a tenant must not gain access by
                            # labeling its own pod as the shared UI. Other NetworkPolicies
                            # are additive and must not grant cross-tenant proxy access.
                            "ingress": [{
                                "from": [{
                                    "namespaceSelector": {
                                        "matchLabels": {
                                            "kubernetes.io/metadata.name":
                                                kfp_namespace
                                        }
                                    },
                                    "podSelector": {
                                        "matchLabels": {
                                            "app": "ml-pipeline-ui"
                                        }
                                    },
                                }],
                                "ports": [{
                                    "protocol": "TCP",
                                    "port": 3000
                                }],
                            }],
                        },
                    },
                    {
                        "apiVersion": "apps/v1",
                        "kind": "Deployment",
                        "metadata": {
                            "labels": {
                                "app": "ml-pipeline-ui-artifact"
                            },
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                        },
                        "spec": {
                            "selector": {
                                "matchLabels": {
                                    "app": "ml-pipeline-ui-artifact"
                                }
                            },
                            "template": {
                                "metadata": {
                                    "labels": {
                                        "app": "ml-pipeline-ui-artifact"
                                    },
                                    "annotations": {
                                        **({
                                            "sidecar.istio.io/inject": "false"
                                        } if disable_istio_sidecar else {}),
                                        "kubeflow-pipelines/image-spec-hash":
                                            hashlib.sha256(
                                                f"{frontend_image}:{frontend_tag}"
                                                .encode()).hexdigest()[:16],
                                    },
                                },
                                "spec": {
                                    "containers": [{
                                        "name":
                                            "ml-pipeline-ui-artifact",
                                        "image":
                                            f"{frontend_image}:{frontend_tag}",
                                        "imagePullPolicy":
                                            "IfNotPresent",
                                        "ports": [{
                                            "containerPort": 3000
                                        }],
                                        "readinessProbe": {
                                            "tcpSocket": {
                                                "port": 3000
                                            },
                                            "periodSeconds": 2,
                                            "timeoutSeconds": 1,
                                        },
                                        "env":
                                            artifact_server_environment(
                                                namespace,
                                                cluster_domain,
                                                allowed_artifact_endpoints,
                                                http_base_url,
                                                object_store_host,
                                                kfp_namespace,
                                                parent.get("metadata",
                                                           {}).get("uid", ""),
                                            ),
                                        "resources": {
                                            "requests": {
                                                "cpu": "10m",
                                                "memory": "70Mi"
                                            },
                                            "limits": {
                                                "cpu": "100m",
                                                "memory": "500Mi"
                                            },
                                        }
                                    }],
                                    "serviceAccountName": "default-editor"
                                }
                            }
                        }
                    },
                    {
                        "apiVersion": "v1",
                        "kind": "Service",
                        "metadata": {
                            "name": "ml-pipeline-ui-artifact",
                            "namespace": namespace,
                            "labels": {
                                "app": "ml-pipeline-ui-artifact"
                            }
                        },
                        "spec": {
                            "ports": [{
                                "name":
                                    "http",  # name is required to let istio understand request protocol
                                "port": 80,
                                "protocol": "TCP",
                                "targetPort": 3000
                            }],
                            "selector": {
                                "app": "ml-pipeline-ui-artifact"
                            }
                        }
                    },
                ])

            print('Received request:\n', json.dumps(parent, sort_keys=True))
            print('Desired resources except secrets:\n',
                  json.dumps(desired_resources, sort_keys=True))

            # Moved after the print argument because this is sensitive data.

            # Check if secret is already there when the controller made the request. If yes, then
            # use it. Else create a new credentials on seaweedfs for the namespace.
            if s3_secret := attachments["Secret.v1"].get(
                    f"{namespace}/mlpipeline-minio-artifact"):
                desired_resources.append(preserved_secret(s3_secret))
                print('Using existing secret')
            else:
                self.ensure_namespace_object_policy(namespace)
                print('Creating new access key.')
                s3_access_key = iam.create_access_key(UserName=namespace)

                self.ensure_bucket_list_policy(S3_BUCKET_NAME)

                self.upsert_lifecycle_policy(S3_BUCKET_NAME,
                                             artifact_retention_days)

                desired_resources.insert(
                    0, {
                        "apiVersion": "v1",
                        "kind": "Secret",
                        "metadata": {
                            "name": "mlpipeline-minio-artifact",
                            "namespace": namespace,
                        },
                        "data": {
                            "accesskey":
                                base64.b64encode(s3_access_key["AccessKey"]
                                                 ["AccessKeyId"].encode('utf-8')
                                                ).decode("utf-8"),
                            "secretkey":
                                base64.b64encode(
                                    s3_access_key["AccessKey"]
                                    ["SecretAccessKey"].encode('utf-8')
                                ).decode("utf-8"),
                        },
                    })

            if artifacts_proxy_enabled.lower() == "true":
                storage_record_secret = self.artifact_storage_record(
                    parent, attachments, desired_resources)
                desired_resources.append(storage_record_secret)
                policy = next(resource for resource in desired_resources
                              if resource["kind"] == "NetworkPolicy")
                observed_policy = attachments.get(
                    "NetworkPolicy.networking.k8s.io/v1",
                    {}).get(f"{namespace}/ml-pipeline-ui-artifact", {})
                existing_receiver = attachments.get(
                    "Deployment.apps/v1",
                    {}).get(f"{namespace}/ml-pipeline-ui-artifact")
                # Establish ingress isolation before a new receiver can listen.
                # Retain existing receivers while their policy converges.
                if existing_receiver is None and observed_policy.get(
                        "spec") != policy["spec"]:
                    desired_resources = [
                        resource for resource in desired_resources
                        if resource["kind"] != "Deployment"
                    ]
                result = {
                    "status":
                        desired_status,
                    "attachments":
                        preserve_namespace_ownership(parent, attachments,
                                                     desired_resources)
                }
                storage_state = json.loads(
                    base64.b64decode(storage_record_secret["data"]
                                     ["authorization.json"]))["state"]
                if storage_state == "pending":
                    result["resyncAfterSeconds"] = 10
                return result
            return {
                "status":
                    desired_status,
                "attachments":
                    preserve_namespace_ownership(parent, attachments,
                                                 desired_resources)
            }

        def do_GET(self):
            # Liveness/readiness endpoint; must not call external services.
            if self.path.split("?", 1)[0] == HEALTHZ_PATH:
                self.send_response(200)
                self.send_header("Content-type", "text/plain")
                self.end_headers()
                self.wfile.write(b"ok")
            else:
                self.send_error(404)

        def log_request(self, code="-", size="-"):
            # Skip probe requests so they don't flood the controller logs.
            # Malformed requests are logged before `command`/`path` are set.
            is_probe = (
                getattr(self, "command", None) == "GET" and
                getattr(self, "path", "").split("?", 1)[0] == HEALTHZ_PATH)
            if not is_probe:
                super().log_request(code, size)

        def do_POST(self):
            # Serve the sync() function as a JSON webhook.
            observed = json.loads(
                self.rfile.read(int(self.headers.get("content-length"))))
            if self.path.split("?", 1)[0] == "/customize":
                desired = self.customize(observed["parent"])
            else:
                desired = self.sync(observed["object"], observed["attachments"],
                                    observed.get("related", {}))

            self.send_response(200)
            self.send_header("Content-type", "application/json")
            self.end_headers()
            self.wfile.write(bytes(json.dumps(desired), 'utf-8'))

    return ThreadingHTTPServer((url, int(controller_port)), Controller)


if __name__ == "__main__":
    main()
