"""Project SOVRINT policy onto an SDK session configuration."""

from typing import Optional

from ..types import SessionConfig
from .model import AuditSink, PermissionEvaluator, SovrintSecurityProfile
from .permissions import create_sovrint_permission_handler
from .surface import constrain_surfaces


def apply_sovrint_profile(
    config: SessionConfig,
    profile: SovrintSecurityProfile,
    *,
    audit_sink: Optional[AuditSink] = None,
    evaluate_permission: Optional[PermissionEvaluator] = None,
) -> SessionConfig:
    """Return a profile-constrained copy of an SDK configuration."""

    result: SessionConfig = dict(config)
    current = config.get("system_message")
    if current and current.get("mode") == "replace":
        if profile.forbid_system_message_replace:
            raise ValueError(f"Profile '{profile.profile_id}' forbids the requested system mode")
    else:
        existing = (current or {}).get("content", "").strip()
        appended = profile.system_message_append.strip()
        content = "\n\n".join(value for value in (existing, appended) if value)
        result["system_message"] = {"mode": "append", "content": content}

    constrain_surfaces(config, result, profile)
    result["on_permission_request"] = create_sovrint_permission_handler(
        profile,
        audit_sink=audit_sink,
        evaluate=evaluate_permission,
        downstream=config.get("on_permission_request"),
    )
    return result
