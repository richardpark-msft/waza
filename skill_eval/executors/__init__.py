"""Executors for running skill evaluations."""

from skill_eval.executors.base import BaseExecutor, ExecutionResult, SessionEvent
from skill_eval.executors.mock import MockExecutor

__all__ = [
    "BaseExecutor",
    "ExecutionResult",
    "SessionEvent",
    "MockExecutor",
]

# Lazy import for optional Copilot SDK dependency
def get_copilot_executor():
    """Get CopilotExecutor if SDK is available."""
    try:
        from skill_eval.executors.copilot import CopilotExecutor
        return CopilotExecutor
    except ImportError:
        return None
