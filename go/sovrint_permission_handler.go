package copilot

// CreateSovrintPermissionHandler composes permission authorities conservatively.
func CreateSovrintPermissionHandler(
	profile SovrintSecurityProfile,
	options SovrintPermissionHandlerOptions,
) PermissionHandler {
	return func(
		request PermissionRequest,
		invocation PermissionInvocation,
	) (PermissionRequestResult, error) {
		approved, reason := evaluateSovrintPermission(
			profile,
			request,
			invocation,
			options.Evaluate,
		)
		if approved && options.Downstream != nil {
			downstream, err := options.Downstream(request, invocation)
			if err != nil {
				approved, reason = false, "DOWNSTREAM_HANDLER_FAILED"
			} else if downstream.Kind != "approved" {
				approved, reason = false, "DOWNSTREAM_HANDLER_DENIED"
			}
		}

		decision := "DENIED"
		if approved {
			decision = "APPROVED"
		}
		event := newSovrintAuditEvent(
			profile,
			"PERMISSION_DECISION",
			invocation.SessionID,
			decision,
			reason,
		)
		event.ToolCallID = request.ToolCallID
		event.PermissionKind = request.Kind
		if !emitSovrintAudit(profile, options.AuditSink, event) && approved {
			return sovrintPermissionResult(false, profile, "AUDIT_SINK_UNAVAILABLE"), nil
		}
		return sovrintPermissionResult(approved, profile, reason), nil
	}
}
