package copilot

func finishSovrintTool(
	tool Tool,
	invocation ToolInvocation,
	started SovrintAuditEvent,
	options SovrintToolGuardOptions,
	reason string,
	result *ToolResult,
) ToolResult {
	if result == nil {
		recordSovrintToolEvent(
			tool,
			invocation,
			started,
			options,
			"TOOL_INVOCATION_FAILED",
			"FAILED",
			"CUSTOM_TOOL_FAILED",
		)
		return sovrintToolOutcome(reason, "failure")
	}
	recordSovrintToolEvent(
		tool,
		invocation,
		started,
		options,
		"TOOL_INVOCATION_COMPLETED",
		"RECORDED",
		"CUSTOM_TOOL_COMPLETED",
	)
	return *result
}
