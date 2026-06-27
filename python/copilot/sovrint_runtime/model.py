"""SOVRINT governed-session models and canonical built-in profiles."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Awaitable, Callable, Dict, Optional, Tuple, Union

from ..types import PermissionRequest, ToolInvocation

PermissionDecision = str
AuditEvent = Dict[str, Any]
AuditSink = Callable[[AuditEvent], Union[None, Awaitable[None]]]
PermissionEvaluator = Callable[
    [PermissionRequest, Dict[str, str]],
    Union[Optional[PermissionDecision], Awaitable[Optional[PermissionDecision]]],
]
ToolAuthorizer = Callable[[ToolInvocation], Union[bool, Awaitable[bool]]]

SOVRINT_SYSTEM_APPEND = (
    "Operate under a bounded SOVRINT governed-session profile. "
    "Use only explicitly exposed tools and declared authority. "
    "Keep observations, inferences, recommendations, governance decisions, "
    "integrity findings, and accepted evidence distinct. "
    "Do not claim approval, verification, restoration, or EvidenceGrid acceptance "
    "without an explicit external result."
)


@dataclass(frozen=True)
class SovrintSecurityProfile:
    """Versioned governed-session policy."""

    profile_id: str
    version: str
    default_decision: PermissionDecision = "deny"
    allow_kinds: Tuple[str, ...] = ()
    deny_kinds: Tuple[str, ...] = ()
    tool_surface_mode: str = "inherit"
    available_tools: Tuple[str, ...] = ()
    excluded_tools: Tuple[str, ...] = ()
    allowed_mcp_servers: Tuple[str, ...] = ()
    allowed_skill_directories: Tuple[str, ...] = ()
    disabled_skills: Tuple[str, ...] = ()
    forbid_system_message_replace: bool = True
    fail_closed_on_audit_error: bool = False
    audit_enabled: bool = True
    system_message_append: str = SOVRINT_SYSTEM_APPEND
    description: str = ""


SOVRINT_STRICT_PROFILE = SovrintSecurityProfile(
    profile_id="sovrint.strict",
    version="1.0",
    description="Deny every permission and expose no inherited first-party tools.",
    deny_kinds=("read", "write", "shell", "url", "mcp"),
    tool_surface_mode="none",
    fail_closed_on_audit_error=True,
)

SOVRINT_READ_ONLY_PROFILE = SovrintSecurityProfile(
    profile_id="sovrint.read-only",
    version="1.0",
    description="Permit reads while denying mutating and external permission kinds.",
    allow_kinds=("read",),
    deny_kinds=("write", "shell", "url", "mcp"),
    fail_closed_on_audit_error=True,
    system_message_append=f"{SOVRINT_SYSTEM_APPEND} Operate in read-only mode.",
)

SOVRINT_RESEARCH_PROFILE = SovrintSecurityProfile(
    profile_id="sovrint.research",
    version="1.0",
    description="Permit reads and defer other permission kinds to an application evaluator.",
    allow_kinds=("read",),
    system_message_append=(
        f"{SOVRINT_SYSTEM_APPEND} Separate research observations from verified findings."
    ),
)
