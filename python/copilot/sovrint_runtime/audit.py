"""Bounded SOVRINT audit events for governed SDK sessions."""

from __future__ import annotations

import inspect
import itertools
from datetime import datetime, timezone
from typing import Any

from .model import AuditEvent, AuditSink, SovrintSecurityProfile

_sequence = itertools.count(1)


async def resolve(value: Any) -> Any:
    """Resolve synchronous and awaitable application callbacks."""

    if inspect.isawaitable(value):
        return await value
    return value


def create_audit_event(
    profile: SovrintSecurityProfile,
    event_class: str,
    session_id: str,
    decision: str,
    reason_code: str,
    *,
    parent_event_id: str | None = None,
    tool_call_id: str | None = None,
    tool_name: str | None = None,
    permission_kind: str | None = None,
) -> AuditEvent:
    """Create a bounded event without prompts, arguments, results, or credentials."""

    now = datetime.now(timezone.utc)
    return {
        "schemaVersion": "1.0",
        "eventId": f"sovrint-{int(now.timestamp() * 1000)}-{next(_sequence)}",
        "parentEventId": parent_event_id,
        "eventClass": event_class,
        "timestampUtc": now.isoformat().replace("+00:00", "Z"),
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


async def emit_audit_event(
    profile: SovrintSecurityProfile,
    sink: AuditSink | None,
    event: AuditEvent,
) -> bool:
    """Record an event and return whether execution may continue."""

    if not profile.audit_enabled:
        return True
    if sink is None:
        return not profile.fail_closed_on_audit_error
    try:
        await resolve(sink(event))
    except Exception:
        return not profile.fail_closed_on_audit_error
    return True
