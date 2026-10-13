package copilot

func recordSovrintToolEvent(
	tool Tool,
	invocation ToolInvocation,
	parent SovrintAuditEvent,
	options SovrintToolGuardOptions,
	eventClass string,
	decision string,
	reasonCode string,
) {
	event := newSovrintAuditEvent(
		options.Profile,
		eventClass,
		invocation.SessionID,
		decision,
		reasonCode,
	)
	event.ParentEventID = parent.EventID
	event.ToolCallID = invocation.ToolCallID
	event.ToolName = tool.Name
	emitSovrintAudit(options.Profile, options.AuditSink, event)
}
