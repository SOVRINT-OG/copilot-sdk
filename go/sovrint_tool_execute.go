package copilot

func executeSovrintTool(
	tool Tool,
	handler ToolHandler,
	invocation ToolInvocation,
	started SovrintAuditEvent,
	options SovrintToolGuardOptions,
) ToolResult {
	if handler == nil {
		return finishSovrintTool(
			tool,
			invocation,
			started,
			options,
			"CUSTOM_TOOL_HANDLER_MISSING",
			nil,
		)
	}
	result, err := handler(invocation)
	if err != nil {
		return finishSovrintTool(
			tool,
			invocation,
			started,
			options,
			err.Error(),
			nil,
		)
	}
	return finishSovrintTool(tool, invocation, started, options, "", &result)
}
