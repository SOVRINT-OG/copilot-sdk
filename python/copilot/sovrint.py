"""SOVRINT governed-session helpers for the Copilot SDK.

This module is additive. It composes the existing SDK SessionConfig and Tool
interfaces without changing the Copilot CLI JSON-RPC protocol.
"""

from __future__ import annotations

import inspect
import itertools
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Awaitable, Callable, Dict, Optional, Sequence, Tuple, Union

from .types import (
    PermissionHandler,
    PermissionRequest,
    PermissionRequestResult,
    SessionConfig,
    Tool,
    ToolInvocation,
    ToolResult,
)

PermissionDecision = str
AuditSink = Callable[[Dict[str, Any]], Union[None, Awaitable[None]]]
PermissionEvaluator = Callable[
    [PermissionRequest, Dict[str, str]],
    Union[Optional[PermissionDecision], Awaitable[Optional[PermissionDecision]]],
]
ToolAuthorizer = Callable[[ToolInvocation], Union[bool, Awaitable[bool]]]

SOVRINT_SYSTEM_APPEND = (
    "Operate under a bounded SOVRINT governed-session profile. "
    "Use only explicitly exposed tools and declared authority. "
    "Treat observations, inferences, recommendations, governance decisions, "
    "integrity findings, and accepted evidence as distinct classes. "
    "Do not claim approval, verification, restoration, or EvidenceGrid acceptance "
    "without an explicit external result."
)


@dataclass(frozen=True)
class SovrintSecurityProfile:
    """Versioned governed-session profile."""

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
    default_decision="deny",
    deny_kinds=("read", "write", "shell", "url", "mcp"),
    tool_surface_mode="none",
    allowed_mcp_servers=(),
    allowed_skill_directories=(),
    fail_closed_on_audit_error=True,
)

SOVRINT_READ_ONLY_PROFILE = SovrintSecurityProfile(
    profile_id="sovrint.read-only",
    version="1.0",
    description="Permit reads while denying mutating and external permission kinds.",
    default_decision="deny",
    allow_kinds=("read",),
    deny_kinds=("write", "shell", "url", "mcp"),
    tool_surface_mode="inherit",
    allowed_mcp_servers=(),
    allowed_skill_directories=(),
    fail_closed_on_audit_error=True,
    system_message_append=f"{SOVRINT_SYSTEM_APPEND} Operate in read-only mode.",
)

SOVRINT_RESEARCH_PROFILE = SovrintSecurityProfile(
    profile_id="sovrint.research",
    version="1.0",
    description="Permit reads and require an evaluator for every other permission kind.",
    default_decision="deny",
    allow_kinds=("read",),
    deny_kinds=(),
    tool_surface_mode="inherit",
    allowed_mcp_servers=(),
    allowed_skill_directories=(),
    fail_closed_on_audit_error=False,
    system_message_append=(
        f"{SOVRINT_SYSTEM_APPEND} Separate research observations from verified findings."
    ),
)

_audit_sequence = itertools.count(1)


async def _resolve(value: Any) -> Any:
    if inspect.isawaitable(value):
        return await value
    return value


def _audit_event(
    profile: SovrintSecurityProfile,
    *,
    event_class: str,
    session_id: str,
    decision: str,
    reason_code: str,
    tool_call_id: Optional[str] = None,
    tool_name: Optional[str] = None,
    permission_kind: Optional[str] = None,
    parent_event_id: Optional[str] = None,
) -> Dict[str, Any]:
    sequence = next(_audit_sequence)
    return {
        "schemaVersion": "1.0",
        "eventId": "sovrint-{0}-{1}".format(int(datetime.now(timezone.utc).timestamp() * 1000), sequence),
        "parentEventId": parent_event_id,
        "eventClass": event_class,
        "timestampUtc": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "profileId": profile.profile_id,
        "profileVersion": profile.version,
        "sessionId": session_id,
        "toolCallId": tool_call_id,
        "toolName": tool_name,
        "permissionKind": permission_kind,
        "decision": decision,
        "reasonCode": reason_code,
        "disclosureClass": "INTERNAL",
        "evidenceStatus": "NOT_SUBMITTED",
    }


async def _emit_audit(
    profile: SovrintSecurityProfile,
    sink: Optional[AuditSink],
    event: Dict[str, Any],
) -> bool:
    if not profile.audit_enabled:
        return True
    if sink is None:
        return not profile.fail_closed_on_audit_error
    try:
        await _resolve(sink(event))
        return True
    except Exception:
        return not profile.fail_closed_on_audit_error


def _permission_result(
    approved: bool,
    profile: SovrintSecurityProfile,
    reason_code: str,
) -> PermissionRequestResult:
    return PermissionRequestResult(
        kind="approved" if approved else "denied-by-rules",
        rules=[
            {
                "source": "sovrint",
                "profileId": profile.profile_id,
                "profileVersion": profile.version,
                "reasonCode": reason_code,
            }
        ],
    )


async def _evaluate_profile_permission(
    profile: SovrintSecurityProfile,
    request: PermissionRequest,
    invocation: Dict[str, str],
    evaluator: Optional[PermissionEvaluator],
) -> Tuple[bool, str]:
    kind = request.get("kind")
    if not kind:
        return False, "UNKNOWN_PERMISSION_KIND"

    if kind in profile.deny_kinds:
        return False, "PROFILE_EXPLICIT_DENY"

    if evaluator is not None:
        try:
            evaluated = await _resolve(evaluator(request, invocation))
        except Exception:
            return False, "APPLICATION_EVALUATOR_FAILED"
        if evaluated == "approve":
            return True, "APPLICATION_EVALUATOR_APPROVED"
        if evaluated == "deny":
            return False, "APPLICATION_EVALUATOR_DENIED"

    if kind in profile.allow_kinds:
        return True, "PROFILE_EXPLICIT_ALLOW"

    if profile.default_decision == "approve":
        return True, "PROFILE_DEFAULT_ALLOW"
    return False, "PROFILE_DEFAULT_DENY"


def create_sovrint_permission_handler(
    profile: SovrintSecurityProfile,
    *,
    audit_sink: Optional[AuditSink] = None,
    evaluate: Optional[PermissionEvaluator] = None,
    downstream: Optional[PermissionHandler] = None,
) -> PermissionHandler:
    """Create a most-restrictive-wins SDK permission handler."""

    async def handler(
        request: PermissionRequest,
        invocation: Dict[str, str],
    ) -> PermissionRequestResult:
        approved, reason_code = await _evaluate_profile_permission(
            profile, request, invocation, evaluate
        )

        if approved and downstream is not None:
            try:
                downstream_result = await _resolve(downstream(request, invocation))
                if downstream_result.get("kind") != "approved":
                    approved = False
                    reason_code = "DOWNSTREAM_HANDLER_DENIED"
            except Exception:
                approved = False
                reason_code = "DOWNSTREAM_HANDLER_FAILED"

        event = _audit_event(
            profile,
            event_class="PERMISSION_DECISION",
            session_id=invocation.get("session_id", invocation.get("sessionId", "unknown")),
            tool_call_id=request.get("toolCallId"),
            permission_kind=request.get("kind", "unknown"),
            decision="APPROVED" if approved else "DENIED",
            reason_code=reason_code,
        )
        audit_recorded = await _emit_audit(profile, audit_sink, event)
        if not audit_recorded and approved:
            return _permission_result(False, profile, "AUDIT_SINK_UNAVAILABLE")

        return _permission_result(approved, profile, reason_code)

    return handler


def _intersection(left: Sequence[str], right: Sequence[str]) -> list:
    allowed = set(right)
    return [value for value in left if value in allowed]


def _merge_unique(left: Sequence[str], right: Sequence[str]) -> list:
    return list(dict.fromkeys(list(left) + list(right)))


def _apply_system_message(
    config: SessionConfig,
    profile: SovrintSecurityProfile,
) -> Optional[Dict[str, str]]:
    current = config.get("system_message")
    if current and current.get("mode") == "replace":
        if profile.forbid_system_message_replace:
            raise ValueError(
                "SOVRINT profile '{0}' forbids system-message replacement".format(
                    profile.profile_id
                )
            )
        return current

    existing = (current or {}).get("content", "").strip()
    appended = profile.system_message_append.strip()
    content = "\n\n".join(value for value in (existing, appended) if value)
    if not content:
        return current
    return {"mode": "append", "content": content}


def apply_sovrint_profile(
    config: SessionConfig,
    profile: SovrintSecurityProfile,
    *,
    audit_sink: Optional[AuditSink] = None,
    evaluate_permission: Optional[PermissionEvaluator] = None,
) -> SessionConfig:
    """Return a copied SessionConfig constrained by a SOVRINT profile."""

    result: SessionConfig = dict(config)

    if profile.tool_surface_mode == "none":
        result["available_tools"] = []
    elif profile.tool_surface_mode == "allowlist":
        configured = config.get("available_tools")
        profile_tools = list(profile.available_tools)
        result["available_tools"] = (
            _intersection(configured, profile_tools) if configured is not None else profile_tools
        )

    result["excluded_tools"] = _merge_unique(
        config.get("excluded_tools", []), profile.excluded_tools
    )
    result["system_message"] = _apply_system_message(config, profile)
    result["on_permission_request"] = create_sovrint_permission_handler(
        profile,
        audit_sink=audit_sink,
        evaluate=evaluate_permission,
        downstream=config.get("on_permission_request"),
    )

    custom_agents = config.get("custom_agents")
    if custom_agents is not None and profile.tool_surface_mode != "inherit":
        constrained_agents = []
        for agent in custom_agents:
            constrained = dict(agent)
            if profile.tool_surface_mode == "none":
                constrained["tools"] = []
            else:
                agent_tools = agent.get("tools") or list(profile.available_tools)
                constrained["tools"] = _intersection(agent_tools, profile.available_tools)
            constrained_agents.append(constrained)
        result["custom_agents"] = constrained_agents

    mcp_servers = config.get("mcp_servers")
    if mcp_servers is not None:
        allowed_servers = set(profile.allowed_mcp_servers)
        result["mcp_servers"] = {
            name: server for name, server in mcp_servers.items() if name in allowed_servers
        }

    skill_directories = config.get("skill_directories")
    if skill_directories is not None:
        allowed_directories = set(profile.allowed_skill_directories)
        result["skill_directories"] = [
            directory for directory in skill_directories if directory in allowed_directories
        ]

    result["disabled_skills"] = _merge_unique(
        config.get("disabled_skills", []), profile.disabled_skills
    )
    return result


def _rejected_tool_result(reason_code: str) -> ToolResult:
    return ToolResult(
        textResultForLlm="The governed tool invocation was not authorized.",
        resultType="rejected",
        error=reason_code,
        toolTelemetry={"source": "sovrint", "reasonCode": reason_code},
    )


def wrap_sovrint_tool(
    tool: Tool,
    profile: SovrintSecurityProfile,
    *,
    audit_sink: Optional[AuditSink] = None,
    authorize: Optional[ToolAuthorizer] = None,
) -> Tool:
    """Wrap an SDK Tool with authorization and bounded audit events."""

    original_handler = tool.handler

    async def guarded_handler(invocation: ToolInvocation) -> ToolResult:
        session_id = invocation.get("session_id", "unknown")
        tool_call_id = invocation.get("tool_call_id")
        started = _audit_event(
            profile,
            event_class="TOOL_INVOCATION_STARTED",
            session_id=session_id,
            tool_call_id=tool_call_id,
            tool_name=tool.name,
            decision="PENDING",
            reason_code="TOOL_INVOCATION_RECEIVED",
        )

        if not await _emit_audit(profile, audit_sink, started):
            return _rejected_tool_result("AUDIT_SINK_UNAVAILABLE")

        if authorize is not None:
            try:
                authorized = bool(await _resolve(authorize(invocation)))
            except Exception:
                authorized = False
            if not authorized:
                await _emit_audit(
                    profile,
                    audit_sink,
                    _audit_event(
                        profile,
                        event_class="TOOL_INVOCATION_REJECTED",
                        session_id=session_id,
                        tool_call_id=tool_call_id,
                        tool_name=tool.name,
                        decision="REJECTED",
                        reason_code="CUSTOM_TOOL_AUTHORIZATION_DENIED",
                        parent_event_id=started["eventId"],
                    ),
                )
                return _rejected_tool_result("CUSTOM_TOOL_AUTHORIZATION_DENIED")

        await _emit_audit(
            profile,
            audit_sink,
            _audit_event(
                profile,
                event_class="TOOL_INVOCATION_APPROVED",
                session_id=session_id,
                tool_call_id=tool_call_id,
                tool_name=tool.name,
                decision="APPROVED",
                reason_code="CUSTOM_TOOL_AUTHORIZED",
                parent_event_id=started["eventId"],
            ),
        )

        try:
            result = await _resolve(original_handler(invocation))
        except Exception as exc:
            await _emit_audit(
                profile,
                audit_sink,
                _audit_event(
                    profile,
                    event_class="TOOL_INVOCATION_FAILED",
                    session_id=session_id,
                    tool_call_id=tool_call_id,
                    tool_name=tool.name,
                    decision="FAILED",
                    reason_code="CUSTOM_TOOL_FAILED",
                    parent_event_id=started["eventId"],
                ),
            )
            return ToolResult(
                textResultForLlm="The governed tool invocation failed.",
                resultType="failure",
                error=str(exc),
                toolTelemetry={"source": "sovrint"},
            )

        await _emit_audit(
            profile,
            audit_sink,
            _audit_event(
                profile,
                event_class="TOOL_INVOCATION_COMPLETED",
                session_id=session_id,
                tool_call_id=tool_call_id,
                tool_name=tool.name,
                decision="RECORDED",
                reason_code="CUSTOM_TOOL_COMPLETED",
                parent_event_id=started["eventId"],
            ),
        )
        return result

    return Tool(
        name=tool.name,
        description=tool.description,
        parameters=tool.parameters,
        handler=guarded_handler,
    )
