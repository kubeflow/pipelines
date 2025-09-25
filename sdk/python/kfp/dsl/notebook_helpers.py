"""Notebook-related helper APIs for KFP DSL.

This module provides a stub for `dsl.run_notebook(**kwargs)`. At runtime
inside `@dsl.notebook_component`, the SDK binds this symbol to a helper
that executes the embedded notebook with the provided parameters.
"""

from __future__ import annotations

from typing import Any


def run_notebook(**kwargs: Any) -> None:
    """Execute the component's embedded Jupyter notebook with injected
    parameters.

    This is a stub placeholder. Inside a function decorated with
    `@dsl.notebook_component`, the SDK binds this symbol at runtime to a helper
    that materializes the embedded notebook, injects a parameters cell from the
    provided `**kwargs`, and executes the notebook via nbclient.

    Calling this function outside of a notebook component context will raise
    NotImplementedError.

    Args:
        **kwargs: Parameter names and values to inject into the notebook's
            parameters cell.

    Raises:
        NotImplementedError: Always, unless overridden at runtime inside a
            notebook component.
    """
    raise NotImplementedError(
        'dsl.run_notebook is only available inside a @dsl.notebook_component at runtime.'
    )
