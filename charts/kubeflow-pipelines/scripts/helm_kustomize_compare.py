#!/usr/bin/env python3

import yaml
import sys
import re
from typing import Dict, List, Any


# ----------------------------
# YAML Loading (Defensive)
# ----------------------------

def load_manifests(file_path: str) -> List[Dict]:
    """Load YAML manifests from file with safe fallback parsing."""
    with open(file_path, "r") as f:
        content = f.read()

    docs: List[Dict] = []

    try:
        for doc in yaml.safe_load_all(content):
            if doc:
                docs.append(doc)
    except yaml.YAMLError:
        # Fallback manual split
        for doc_str in content.split("---"):
            doc_str = doc_str.strip()
            if not doc_str:
                continue
            try:
                doc = yaml.safe_load(doc_str)
                if doc:
                    docs.append(doc)
            except yaml.YAMLError:
                continue

    return docs


# ----------------------------
# Metadata Normalization
# ----------------------------

def clean_helm_metadata(obj: Any) -> Any:
    """Remove Helm-injected metadata fields while preserving canonical metadata."""
    if isinstance(obj, dict):
        cleaned = {}

        for key, value in obj.items():
            if key == "metadata" and isinstance(value, dict):
                cleaned_metadata = {}

                for meta_key, meta_value in value.items():

                    # Strip runtime-managed fields
                    if meta_key in [
                        "generation",
                        "resourceVersion",
                        "uid",
                        "creationTimestamp",
                        "managedFields",
                    ]:
                        continue

                    if meta_key == "labels" and isinstance(meta_value, dict):
                        cleaned_labels = {}

                        for label_key, label_value in meta_value.items():
                            if (
                                not label_key.startswith("helm.sh/")
                                and label_key != "app.kubernetes.io/managed-by"
                                and label_key != "app.kubernetes.io/instance"
                                and label_key != "app.kubernetes.io/version"
                                and label_key != "app.kubernetes.io/name"
                                and label_key != "app.kubernetes.io/component"
                            ):
                                cleaned_labels[label_key] = label_value

                        if cleaned_labels:
                            cleaned_metadata["labels"] = cleaned_labels
                        continue

                    if meta_key == "annotations" and isinstance(meta_value, dict):
                        cleaned_annotations = {}

                        for ann_key, ann_value in meta_value.items():
                            if not ann_key.startswith(
                                ("helm.sh/", "meta.helm.sh/")
                            ):
                                cleaned_annotations[ann_key] = ann_value

                        if cleaned_annotations:
                            cleaned_metadata["annotations"] = cleaned_annotations
                        continue

                    cleaned_metadata[meta_key] = meta_value

                cleaned[key] = cleaned_metadata
            else:
                cleaned[key] = clean_helm_metadata(value)

        return cleaned

    if isinstance(obj, list):
        return [clean_helm_metadata(item) for item in obj]

    return obj


# ----------------------------
# Hash Normalization
# ----------------------------

def normalize_hash_suffix(name: str) -> str:
    """Remove Kustomize hash suffix from resource names."""
    return re.sub(r"-[a-z0-9]{10}$", "", name)


def normalize_manifest(manifest: Dict) -> Dict:
    """Normalize manifest for deterministic comparison."""
    normalized = manifest.copy()

    normalized = clean_helm_metadata(normalized)

    # Remove status block
    normalized.pop("status", None)

    if "metadata" in normalized and "name" in normalized["metadata"]:
        kind = normalized.get("kind", "")
        if kind in ["Secret", "ConfigMap"]:
            normalized["metadata"]["name"] = normalize_hash_suffix(
                normalized["metadata"]["name"]
            )

    def remove_empty_values(obj):
        if isinstance(obj, dict):
            return {
                k: remove_empty_values(v)
                for k, v in obj.items()
                if v is not None and v != {} and v != []
            }
        if isinstance(obj, list):
            return [remove_empty_values(item) for item in obj if item is not None]
        return obj

    return remove_empty_values(normalized)


# ----------------------------
# Resource Keying
# ----------------------------

def get_resource_key(manifest: Dict) -> str:
    """Generate stable key for resource comparison."""
    kind = manifest.get("kind", "Unknown")
    metadata = manifest.get("metadata", {})
    name = metadata.get("name", "unknown")
    namespace = metadata.get("namespace", "")

    if kind in ["Secret", "ConfigMap"]:
        name = normalize_hash_suffix(name)

    if namespace:
        return f"{kind}/{namespace}/{name}"

    return f"{kind}/{name}"


# ----------------------------
# Deep Structural Diff
# ----------------------------

def deep_diff(obj1: Any, obj2: Any, path: str = "") -> List[str]:
    differences: List[str] = []

    if type(obj1) != type(obj2):
        differences.append(
            f"{path}: type mismatch ({type(obj1).__name__} vs {type(obj2).__name__})"
        )
        return differences

    if isinstance(obj1, dict):
        all_keys = set(obj1.keys()) | set(obj2.keys())

        for key in sorted(all_keys):
            key_path = f"{path}.{key}" if path else key

            if key not in obj1:
                differences.append(f"{key_path}: missing in kustomize")
            elif key not in obj2:
                differences.append(f"{key_path}: missing in helm")
            else:
                differences.extend(
                    deep_diff(obj1[key], obj2[key], key_path)
                )

    elif isinstance(obj1, list):
        if len(obj1) != len(obj2):
            differences.append(
                f"{path}: list length mismatch ({len(obj1)} vs {len(obj2)})"
            )
        else:
            for i, (item1, item2) in enumerate(zip(obj1, obj2)):
                differences.extend(
                    deep_diff(item1, item2, f"{path}[{i}]")
                )

    elif obj1 != obj2:
        differences.append(f"{path}: '{obj1}' != '{obj2}'")

    return differences


# ----------------------------
# Comparison Logic
# ----------------------------

def compare_manifests(
    kustomize_file: str,
    helm_file: str,
) -> bool:
    """Compare Kustomize and Helm manifests with strict parity."""

    kustomize_manifests = load_manifests(kustomize_file)
    helm_manifests = load_manifests(helm_file)

    kustomize_resources: Dict[str, Dict] = {}
    helm_resources: Dict[str, Dict] = {}

    for manifest in kustomize_manifests:
        normalized = normalize_manifest(manifest)
        key = get_resource_key(normalized)
        kustomize_resources[key] = normalized

    for manifest in helm_manifests:
        normalized = normalize_manifest(manifest)
        key = get_resource_key(normalized)
        helm_resources[key] = normalized

    kustomize_keys = set(kustomize_resources.keys())
    helm_keys = set(helm_resources.keys())

    only_in_kustomize = kustomize_keys - helm_keys
    only_in_helm = helm_keys - kustomize_keys

    success = True

    if only_in_kustomize:
        print(f"Resources only in Kustomize: {len(only_in_kustomize)}")
        for key in sorted(only_in_kustomize):
            print(f"  {key}")
        success = False

    if only_in_helm:
        print(f"Unexpected resources only in Helm: {len(only_in_helm)}")
        for key in sorted(only_in_helm):
            print(f"  {key}")
        success = False

    for key in sorted(kustomize_keys & helm_keys):
        differences = deep_diff(
            kustomize_resources[key],
            helm_resources[key],
        )

        if differences:
            print(f"Differences in {key}: {len(differences)} fields")
            for diff in differences:
                print(f"  {diff}")
            success = False

    return success


# ----------------------------
# Entry Point
# ----------------------------

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(
            "Usage: python helm_kustomize_compare.py <kustomize_file> <helm_file>"
        )
        sys.exit(1)

    kustomize_file = sys.argv[1]
    helm_file = sys.argv[2]

    result = compare_manifests(kustomize_file, helm_file)
    sys.exit(0 if result else 1)
