# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Portable component context annotation; not a driver integration."""

import dataclasses
from typing import Any, Dict, List, Optional


@dataclasses.dataclass
class TaskConfig:
    """Configuration annotation retained for typed component definitions.

    The native compiler currently rejects automatic TaskConfig injection.
    Pass explicit component parameters or Kubernetes settings instead; there
    is no KFP driver that populates this value.
    """

    affinity: Optional[Dict[str, Any]] = None
    tolerations: Optional[List[Dict[str, Any]]] = None
    node_selector: Optional[Dict[str, str]] = None
    env: Optional[List[Dict[str, Any]]] = None
    volumes: Optional[List[Dict[str, Any]]] = None
    volume_mounts: Optional[List[Dict[str, Any]]] = None
    resources: Optional[Dict[str, Any]] = None
    resource_claims: Optional[List[Dict[str, Any]]] = None
