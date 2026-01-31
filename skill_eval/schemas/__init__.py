"""Schemas package for skill-eval."""

from skill_eval.schemas.eval_spec import (
    EvalSpec,
    EvalConfig,
    MetricConfig,
    GraderConfig,
    GraderType,
)
from skill_eval.schemas.task import (
    Task,
    TaskInput,
    TaskExpected,
    TaskGraderConfig,
    TriggerTestCase,
    TriggerTestSuite,
    ToolCallPattern,
    OutcomeExpectation,
    BehaviorExpectation,
)
from skill_eval.schemas.results import (
    EvalResult,
    EvalSummary,
    TaskResult,
    TaskAggregate,
    TrialResult,
    GraderResult,
    MetricResult,
    TranscriptSummary,
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
