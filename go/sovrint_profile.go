package copilot

// SovrintPermissionDecision is a bounded application decision.
type SovrintPermissionDecision string

const (
	SovrintApprove SovrintPermissionDecision = "approve"
	SovrintDeny    SovrintPermissionDecision = "deny"
)

// SovrintSecurityProfile defines a versioned governed-session policy.
type SovrintSecurityProfile struct {
	ProfileID                  string
	Version                    string
	Description                string
	DefaultDecision            SovrintPermissionDecision
	AllowKinds                 []string
	DenyKinds                  []string
	ToolSurfaceMode            string
	AvailableTools             []string
	ExcludedTools              []string
	AllowedMCPServers          []string
	AllowedSkillDirectories    []string
	DisabledSkills             []string
	ForbidSystemMessageReplace bool
	FailClosedOnAuditError     bool
	AuditEnabled               bool
	SystemMessageAppend        string
}

// SovrintSystemAppend is the canonical governed-session instruction layer.
const SovrintSystemAppend = "Operate under a bounded SOVRINT governed-session profile. Use only explicitly exposed tools and declared authority. Keep observations, inferences, recommendations, governance decisions, integrity findings, and accepted evidence distinct. Do not claim approval, verification, restoration, or EvidenceGrid acceptance without an explicit external result."

var SovrintStrictProfile = SovrintSecurityProfile{
	ProfileID:                  "sovrint.strict",
	Version:                    "1.0",
	Description:                "Deny every permission and expose no inherited first-party tools.",
	DefaultDecision:            SovrintDeny,
	DenyKinds:                  []string{"read", "write", "shell", "url", "mcp"},
	ToolSurfaceMode:            "none",
	ForbidSystemMessageReplace: true,
	FailClosedOnAuditError:     true,
	AuditEnabled:               true,
	SystemMessageAppend:        SovrintSystemAppend,
}

var SovrintReadOnlyProfile = SovrintSecurityProfile{
	ProfileID:                  "sovrint.read-only",
	Version:                    "1.0",
	Description:                "Permit reads while denying mutating and external permission kinds.",
	DefaultDecision:            SovrintDeny,
	AllowKinds:                 []string{"read"},
	DenyKinds:                  []string{"write", "shell", "url", "mcp"},
	ToolSurfaceMode:            "inherit",
	ForbidSystemMessageReplace: true,
	FailClosedOnAuditError:     true,
	AuditEnabled:               true,
	SystemMessageAppend:        SovrintSystemAppend + " Operate in read-only mode.",
}

var SovrintResearchProfile = SovrintSecurityProfile{
	ProfileID:                  "sovrint.research",
	Version:                    "1.0",
	Description:                "Permit reads and defer other permission kinds to an application evaluator.",
	DefaultDecision:            SovrintDeny,
	AllowKinds:                 []string{"read"},
	ToolSurfaceMode:            "inherit",
	ForbidSystemMessageReplace: true,
	AuditEnabled:               true,
	SystemMessageAppend:        SovrintSystemAppend + " Separate research observations from verified findings.",
}
