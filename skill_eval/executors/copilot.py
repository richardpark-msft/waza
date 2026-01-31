"""Copilot SDK executor for real integration testing.

This executor uses the @github/copilot-sdk to run actual Copilot agent sessions,
providing real LLM responses for integration testing.

Prerequisites:
- Install: pip install skill-eval[copilot]
- Authenticate: Run `copilot` CLI and follow prompts

Usage:
    config:
      executor: copilot-sdk
      model: claude-sonnet-4-20250514
      skill_directories:
        - ./skills
      mcp_servers:
        azure:
          type: stdio
          command: npx
          args: ["-y", "@azure/mcp", "server", "start"]
"""

from __future__ import annotations

import asyncio
import os
import shutil
import tempfile
import time
from typing import Any

from skill_eval.executors.base import BaseExecutor, ExecutionResult, SessionEvent

# Lazy import for optional dependency
CopilotClient = None


def _get_copilot_client():
    """Lazy load the Copilot SDK client."""
    global CopilotClient
    if CopilotClient is None:
        try:
            from copilot_sdk import CopilotClient as _CopilotClient
            CopilotClient = _CopilotClient
        except ImportError:
            raise ImportError(
                "Copilot SDK not installed. Install with: pip install skill-eval[copilot]\n"
                "Or: pip install copilot-sdk"
            )
    return CopilotClient


class CopilotExecutor(BaseExecutor):
    """Executor using GitHub Copilot SDK for real agent sessions.
    
    This provides actual LLM responses and skill invocations for integration testing.
    """
    
    def __init__(
        self,
        model: str = "claude-sonnet-4-20250514",
        skill_directories: list[str] | None = None,
        mcp_servers: dict[str, Any] | None = None,
        timeout_seconds: int = 300,
        yolo_mode: bool = True,
        non_interactive: bool = True,
        **kwargs: Any,
    ):
        """Initialize Copilot executor.
        
        Args:
            model: Model to use for responses
            skill_directories: Directories containing SKILL.md files
            mcp_servers: MCP server configurations
            timeout_seconds: Session timeout
            yolo_mode: Enable yolo mode (auto-approve actions)
            non_interactive: Run in non-interactive mode
        """
        super().__init__(model=model, **kwargs)
        self.skill_directories = skill_directories or []
        self.mcp_servers = mcp_servers or {}
        self.timeout_seconds = timeout_seconds
        self.yolo_mode = yolo_mode
        self.non_interactive = non_interactive
        
        self._client = None
        self._workspace: str | None = None
    
    async def setup(self) -> None:
        """Initialize Copilot client."""
        ClientClass = _get_copilot_client()
        
        # Create temp workspace
        self._workspace = tempfile.mkdtemp(prefix="skill-eval-")
        
        # Build CLI args
        cli_args = []
        if self.yolo_mode:
            cli_args.append("--yolo")
        if self.non_interactive:
            cli_args.append("-p")  # non-interactive/pipe mode
        
        self._client = ClientClass(
            logLevel="error",
            cwd=self._workspace,
            cliArgs=cli_args,
        )
    
    async def teardown(self) -> None:
        """Clean up resources."""
        if self._client:
            try:
                await asyncio.to_thread(self._client.stop)
            except Exception:
                pass
            self._client = None
        
        if self._workspace and os.path.exists(self._workspace):
            try:
                shutil.rmtree(self._workspace)
            except Exception:
                pass
            self._workspace = None
    
    async def execute(
        self,
        prompt: str,
        context: dict[str, Any] | None = None,
        skill_name: str | None = None,
    ) -> ExecutionResult:
        """Execute a prompt using Copilot SDK."""
        if not self._client:
            await self.setup()
        
        start_time = time.time()
        events: list[SessionEvent] = []
        output_parts: list[str] = []
        error: str | None = None
        
        try:
            # Set up workspace context if provided
            if context:
                await self._setup_context(context)
            
            # Create session
            session = await asyncio.to_thread(
                self._client.createSession,
                {
                    "model": self.model,
                    "skillDirectories": self.skill_directories,
                    "mcpServers": self.mcp_servers,
                }
            )
            
            # Set up event collection
            done_event = asyncio.Event()
            
            def handle_event(raw_event: dict[str, Any]) -> None:
                event = SessionEvent(
                    type=raw_event.get("type", "unknown"),
                    data=raw_event.get("data", {}),
                )
                events.append(event)
                
                # Collect assistant messages
                if event.type == "assistant.message" and event.content:
                    output_parts.append(event.content)
                elif event.type == "assistant.message_delta" and event.delta_content:
                    output_parts.append(event.delta_content)
                
                # Check for completion
                if event.type == "session.idle":
                    done_event.set()
                elif event.type == "session.error":
                    nonlocal error
                    error = event.data.get("message", "Unknown error")
                    done_event.set()
            
            # Register event handler
            session.on(handle_event)
            
            # Send prompt
            await asyncio.to_thread(session.send, {"prompt": prompt})
            
            # Wait for completion with timeout
            try:
                await asyncio.wait_for(
                    done_event.wait(),
                    timeout=self.timeout_seconds,
                )
            except asyncio.TimeoutError:
                error = f"Session timed out after {self.timeout_seconds}s"
            
            # Cleanup session
            try:
                await asyncio.to_thread(session.destroy)
            except Exception:
                pass
            
        except Exception as e:
            error = str(e)
        
        duration_ms = int((time.time() - start_time) * 1000)
        output = "".join(output_parts)
        
        # Extract tool calls
        tool_calls = [
            {"name": e.tool_name, "arguments": e.arguments}
            for e in events
            if e.type == "tool.execution_start"
        ]
        
        return ExecutionResult(
            output=output,
            events=events,
            model=self.model,
            skill_name=skill_name,
            duration_ms=duration_ms,
            tool_calls=tool_calls,
            error=error,
            success=error is None,
        )
    
    async def _setup_context(self, context: dict[str, Any]) -> None:
        """Set up workspace with context files."""
        if not self._workspace:
            return
        
        files = context.get("files", [])
        for file_info in files:
            path = file_info.get("path", "")
            content = file_info.get("content", "")
            
            if path and content:
                full_path = os.path.join(self._workspace, path)
                os.makedirs(os.path.dirname(full_path), exist_ok=True)
                with open(full_path, "w") as f:
                    f.write(content)


def is_copilot_sdk_available() -> bool:
    """Check if Copilot SDK is installed and available."""
    try:
        _get_copilot_client()
        return True
    except ImportError:
        return False


def get_sdk_skip_reason() -> str | None:
    """Get reason why SDK tests should be skipped, or None if they can run."""
    if os.environ.get("CI") == "true":
        return "Running in CI environment"
    
    if os.environ.get("SKIP_INTEGRATION_TESTS") == "true":
        return "SKIP_INTEGRATION_TESTS=true"
    
    if not is_copilot_sdk_available():
        return "copilot-sdk not installed"
    
    return None
