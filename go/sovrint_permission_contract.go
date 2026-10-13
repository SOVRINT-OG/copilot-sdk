package copilot

type SovrintPermissionEvaluator func(
	PermissionRequest,
	PermissionInvocation,
) (SovrintPermissionDecision, error)

type SovrintPermissionHandlerOptions struct {
	AuditSink  SovrintAuditSink
	Evaluate   SovrintPermissionEvaluator
	Downstream PermissionHandler
}
