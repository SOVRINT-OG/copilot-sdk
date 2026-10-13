"""Most-restrictive SOVRINT permission composition."""

from __future__ import annotations

from typing import Dict, Optional, Tuple

from ..types import PermissionHandler, PermissionRequest, PermissionRequestResult
from .audit import create_audit_event, emit_audit_event, resolve
from .model import AuditSink, PermissionEvaluator, SovrintSecurityProfile


def permission_result(
    approved: bool,
    profile: SovrintSecurityProfile,
    reason_code: str,
) -> PermissionRequestResult:
    """Create a native SDK permission result carrying SOVRINT lineage."""

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


async def evaluate_permission(
    profile: SovrintSecurityProfile,
    request: PermissionRequest,
    invocation: Dict[str, str],
    evaluator: Optional[PermissionEvaluator],
) -> Tuple[bool, str]:
    """Evaluate explicit denial, application policy, allowance, then default."""

    kind = request.get("kind")
    if not kind:
        return False, "UNKNOWN_PERMISSION_KIND"
    if kind in profile.deny_kinds:
        return False, "PROFILE_EXPLICIT_DENY"
    if evaluator is not None:
        try:
            decision = await resolve(evaluator(request, invocation))
        except Exception:
            return False, "APPLICATION_EVALUATOR_FAILED"
        if decision == "approve":
            return True, "APPLICATION_EVALUATOR_APPROVED"
        if decision == "deny":
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
    """Compose profile, application, and existing SDK decisions."""

    async def handler(
        request: PermissionRequest,
        invocation: Dict[str, str],
    ) -> PermissionRequestResult:
        approved, reason = await evaluate_permission(
            profile, request, invocation, evaluate
        )
        if approved and downstream is not None:
            try:
                downstream_result = await resolve(downstream(request, invocation))
                if downstream_result.get("kind") != "approved":
                    approved, reason = False, "DOWNSTREAM_HANDLER_DENIED"
            except Exception:
                approved, reason = False, "DOWNSTREAM_HANDLER_FAILED"

        event = create_audit_event(
            profile,
            "PERMISSION_DECISION",
            invocation.get("session_id", invocation.get("sessionId", "unknown")),
            "APPROVED" if approved else "DENIED",
            reason,
            tool_call_id=request.get("toolCallId"),
            permission_kind=request.get("kind", "unknown"),
        )
        if not await emit_audit_event(profile, audit_sink, event) and approved:
            return permission_result(False, profile, "AUDIT_SINK_UNAVAILABLE")
        return permission_result(approved, profile, reason)

    return handler
