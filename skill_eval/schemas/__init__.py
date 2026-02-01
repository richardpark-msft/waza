"""Schemas package for skill-eval."""

from skill_eval.schemas.eval_spec import (
    EvalConfig,
    EvalSpec,
    GraderConfig,
    GraderType,
    MetricConfig,
)
from skill_eval.schemas.results import (
    EvalResult,
    EvalSummary,
    GraderResult,
    MetricResult,
    TaskAggregate,
    TaskResult,
    TranscriptSummary,
    TrialResult,
)
from skill_eval.schemas.task import (
    BehaviorExpectation,
    OutcomeExpectation,
    Task,
    TaskExpected,
    TaskGraderConfig,
    TaskInput,
    ToolCallPattern,
    TriggerTestCase,
    TriggerTestSuite,
)

__all__ = [
    # Eval spec
    "EvalSpec",
    "EvalConfig",
    "MetricConfig",
    "GraderConfig",
    "GraderType",
    # Tasks
    "Task",
    "TaskInput",
    "TaskExpected",
    "TaskGraderConfig",
    "TriggerTestCase",
    "TriggerTestSuite",
    "ToolCallPattern",
    "OutcomeExpectation",
    "BehaviorExpectation",
    # Results
    "EvalResult",
    "EvalSummary",
    "TaskResult",
    "TaskAggregate",
    "TrialResult",
    "GraderResult",
    "MetricResult",
    "TranscriptSummary",
]
