"""
Copilot SDK - Python Client for GitHub Copilot CLI

JSON-RPC based SDK for programmatic control of GitHub Copilot CLI
"""

from .client import CopilotClient
from .session import CopilotSession
from .sovrint_runtime import (
    SOVRINT_READ_ONLY_PROFILE,
    SOVRINT_RESEARCH_PROFILE,
    SOVRINT_STRICT_PROFILE,
    SOVRINT_SYSTEM_APPEND,
    SovrintSecurityProfile,
    apply_sovrint_profile,
    create_sovrint_permission_handler,
    wrap_sovrint_tool,
)
from .tools import define_tool
from .types import (
    AzureProviderOptions,
    ConnectionState,
    CustomAgentConfig,
    GetAuthStatusResponse,
    GetStatusResponse,
    MCPLocalServerConfig,
    MCPRemoteServerConfig,
    MCPServerConfig,
    MessageOptions,
    ModelBilling,
    ModelCapabilities,
    ModelInfo,
    ModelPolicy,
    PermissionHandler,
    PermissionRequest,
    PermissionRequestResult,
    ProviderConfig,
    ResumeSessionConfig,
    SessionConfig,
    SessionEvent,
    Tool,
    ToolHandler,
    ToolInvocation,
    ToolResult,
)

__version__ = "0.1.0"

__all__ = [
    "AzureProviderOptions",
    "CopilotClient",
    "CopilotSession",
    "ConnectionState",
    "CustomAgentConfig",
    "GetAuthStatusResponse",
    "GetStatusResponse",
    "MCPLocalServerConfig",
    "MCPRemoteServerConfig",
    "MCPServerConfig",
    "MessageOptions",
    "ModelBilling",
    "ModelCapabilities",
    "ModelInfo",
    "ModelPolicy",
    "PermissionHandler",
    "PermissionRequest",
    "PermissionRequestResult",
    "ProviderConfig",
    "ResumeSessionConfig",
    "SOVRINT_READ_ONLY_PROFILE",
    "SOVRINT_RESEARCH_PROFILE",
    "SOVRINT_STRICT_PROFILE",
    "SOVRINT_SYSTEM_APPEND",
    "SessionConfig",
    "SessionEvent",
    "SovrintSecurityProfile",
    "Tool",
    "ToolHandler",
    "ToolInvocation",
    "ToolResult",
    "apply_sovrint_profile",
    "create_sovrint_permission_handler",
    "define_tool",
    "wrap_sovrint_tool",
]
