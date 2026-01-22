from __future__ import annotations

import os
from importlib.metadata import entry_points
from typing import List, Set

from .api import KfpLifecyclePlugin

CUSTOM_ENTRYPOINT_GROUP = "kfp.sdk.plugins"
OOB_ENTRYPOINT_GROUP = "kfp.sdk.plugins.oob"


def _get_enabled_oob_plugins() -> Set[str] | str | None:
    """
    Parse the OOB_PLUGINS environment variable to determine which OOB plugins to load.

    Returns:
        - None if OOB plugins are disabled (empty string or "none")
        - "all" if all OOB plugins should be loaded
        - A set of plugin names to enable
    """
    val = os.environ.get("OOB_PLUGINS", "").strip()
    if not val or val.lower() == "none":
        return None
    if val.lower() == "all":
        return "all"
    return set(p.strip() for p in val.split(",") if p.strip())


def _load_from_entrypoint(group_name: str) -> List[KfpLifecyclePlugin]:
    """Load plugins from a specific entry point group."""
    eps = entry_points()
    group = eps.select(group=group_name) if hasattr(eps, "select") else eps.get(group_name, [])

    plugins: list[KfpLifecyclePlugin] = []
    for ep in group:
        factory_or_obj = ep.load()
        plugin = factory_or_obj() if callable(factory_or_obj) else factory_or_obj

        if not isinstance(plugin, KfpLifecyclePlugin):
            raise TypeError(
                f"Entry point '{ep.name}' must return a KfpLifecyclePlugin instance; got {type(plugin)!r}"
            )
        plugins.append(plugin)

    return plugins


def load_plugins() -> List[KfpLifecyclePlugin]:
    """
    Load all enabled plugins from entry points.

    Custom plugins from 'kfp.sdk.plugins' are always loaded.
    OOB plugins from 'kfp.sdk.plugins.oob' are loaded based on the OOB_PLUGINS
    environment variable:
        - Not set or "none": No OOB plugins loaded
        - "all": All OOB plugins loaded
        - "plugin1,plugin2": Only specified plugins loaded
    """
    plugins: list[KfpLifecyclePlugin] = []

    # 1. Load all custom plugins (unconditional)
    plugins.extend(_load_from_entrypoint(CUSTOM_ENTRYPOINT_GROUP))

    # 2. Load OOB plugins based on config
    enabled_oob = _get_enabled_oob_plugins()
    if enabled_oob is not None:
        oob_plugins = _load_from_entrypoint(OOB_ENTRYPOINT_GROUP)
        for plugin in oob_plugins:
            if enabled_oob == "all" or plugin.name in enabled_oob:
                plugins.append(plugin)

    # stable ordering by plugin name
    plugins.sort(key=lambda p: (p.name.lower()))
    return plugins
