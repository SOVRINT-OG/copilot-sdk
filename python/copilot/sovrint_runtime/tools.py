"""Authorization and audit guards for caller-defined SDK tools."""

from typing import Optional

from ..types import Tool, ToolInvocation, ToolResult
from .audit import create_audit_event, emit_audit_event, resolve
from .model import AuditSink, SovrintSecurityProfile, ToolAuthorizer


def rejected_tool_result(reason: str) -> ToolResult:
    """Create a bounded rejection result."""

    return ToolResult(
        textResultForLlm="The governed tool invocation was not authorized.",
        resultType="rejected",
        error=reason,
        toolTelemetry={"source": "sovrint", "reasonCode": reason},
    )


def wrap_sovrint_tool(
    tool: Tool,
    profile: SovrintSecurityProfile,
    *,
    audit_sink: Optional[AuditSink] = None,
    authorize: Optional[ToolAuthorizer] = None,
) -> Tool:
    """Wrap a caller-defined tool with authorization and bounded audit events."""

    original = tool.handler

    async def guarded(invocation: ToolInvocation) -> ToolResult:
        session_id = invocation.get("session_id", "unknown")
        call_id = invocation.get("tool_call_id")
        started = create_audit_event(
            profile,
            "TOOL_INVOCATION_STARTED",
            session_id,
            "PENDING",
            "TOOL_INVOCATION_RECEIVED",
            tool_call_id=call_id,
            tool_name=tool.name,
        )
        if not await emit_audit_event(profile, audit_sink, started):
            return rejected_tool_result("AUDIT_SINK_UNAVAILABLE")

        try:
            authorized = authorize is None or bool(await resolve(authorize(invocation)))
        except Exception:
            authorized = False
        if not authorized:
            await emit_audit_event(
                profile,
                audit_sink,
                create_audit_event(
                    profile,
                    "TOOL_INVOCATION_REJECTED",
                    session_id,
                    "REJECTED",
                    "CUSTOM_TOOL_AUTHORIZATION_DENIED",
                    parent_event_id=started["eventId"],
                    tool_call_id=call_id,
                    tool_name=tool.name,
                ),
            )
            return rejected_tool_result("CUSTOM_TOOL_AUTHORIZATION_DENIED")

        await emit_audit_event(
            profile,
            audit_sink,
            create_audit_event(
                profile,
                "TOOL_INVOCATION_APPROVED",
                session_id,
                "APPROVED",
                "CUSTOM_TOOL_AUTHORIZED",
                parent_event_id=started["eventId"],
                tool_call_id=call_id,
                tool_name=tool.name,
            ),
        )
        try:
            result = await resolve(original(invocation))
        except Exception as exc:
            await emit_audit_event(
                profile,
                audit_sink,
                create_audit_event(
                    profile,
                    "TOOL_INVOCATION_FAILED",
                    session_id,
                    "FAILED",
                    "CUSTOM_TOOL_FAILED",
                    parent_event_id=started["eventId"],
                    tool_call_id=call_id,
                    tool_name=tool.name,
                ),
            )
            return ToolResult(
                textResultForLlm="The governed tool invocation failed.",
                resultType="failure",
                error=str(exc),
                toolTelemetry={"source": "sovrint"},
            )

        await emit_audit_event(
            profile,
            audit_sink,
            create_audit_event(
                profile,
                "TOOL_INVOCATION_COMPLETED",
                session_id,
                "RECORDED",
                "CUSTOM_TOOL_COMPLETED",
                parent_event_id=started["eventId"],
                tool_call_id=call_id,
                tool_name=tool.name,
            ),
        )
        return result

    return Tool(
        name=tool.name,
        description=tool.description,
        parameters=tool.parameters,
        handler=guarded,
    )
