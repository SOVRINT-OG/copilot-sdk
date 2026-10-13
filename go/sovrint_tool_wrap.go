package copilot

// WrapSovrintTool adds application authorization and bounded audit events.
func WrapSovrintTool(tool Tool, options SovrintToolGuardOptions) Tool {
	original := tool.Handler
	tool.Handler = func(invocation ToolInvocation) (ToolResult, error) {
		started := newSovrintAuditEvent(
			options.Profile,
			"TOOL_INVOCATION_STARTED",
			invocation.SessionID,
			"PENDING",
			"TOOL_INVOCATION_RECEIVED",
		)
		started.ToolCallID = invocation.ToolCallID
		started.ToolName = tool.Name
		if !emitSovrintAudit(options.Profile, options.AuditSink, started) {
			return sovrintToolOutcome("AUDIT_SINK_UNAVAILABLE", "rejected"), nil
		}

		if options.Authorize != nil {
			allowed, err := options.Authorize(invocation)
			if err != nil || !allowed {
				recordSovrintToolEvent(
					tool,
					invocation,
					started,
					options,
					"TOOL_INVOCATION_REJECTED",
					"REJECTED",
					"CUSTOM_TOOL_AUTHORIZATION_DENIED",
				)
				return sovrintToolOutcome("CUSTOM_TOOL_AUTHORIZATION_DENIED", "rejected"), nil
			}
		}

		recordSovrintToolEvent(
			tool,
			invocation,
			started,
			options,
			"TOOL_INVOCATION_APPROVED",
			"APPROVED",
			"CUSTOM_TOOL_AUTHORIZED",
		)
		return executeSovrintTool(tool, original, invocation, started, options), nil
	}
	return tool
}
