package copilot

type SovrintToolAuthorizer func(ToolInvocation) (bool, error)

type SovrintToolGuardOptions struct {
	Profile   SovrintSecurityProfile
	AuditSink SovrintAuditSink
	Authorize SovrintToolAuthorizer
}
