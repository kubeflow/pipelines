"""KFP Plugin SDK for extending pipeline lifecycle with custom hooks."""

from .api import (
    # Request types
    ExecutorHookRequest,
    RunHookRequest,
    TaskHookRequest,
    # Response types
    ExecutorStartResponse,
    RunHookResponse,
    TaskEndResponse,
    TaskStartResponse,
    # Context types
    RunContext,
    TaskContext,
    # Base class
    KfpLifecyclePlugin,
)
from .loader import load_plugins
from .server import create_app

__all__ = [
    # Base class
    "KfpLifecyclePlugin",
    # Context types
    "RunContext",
    "TaskContext",
    # Request types
    "RunHookRequest",
    "TaskHookRequest",
    "ExecutorHookRequest",
    # Response types
    "RunHookResponse",
    "TaskStartResponse",
    "TaskEndResponse",
    "ExecutorStartResponse",
    # Utilities
    "load_plugins",
    "create_app",
]
