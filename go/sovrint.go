package copilot

import (
	"fmt"
	"sync/atomic"
	"time"
)

// SovrintPermissionDecision is a bounded permission outcome used by an
// application evaluator before the SDK server receives a final decision.
type SovrintPermissionDecision string

const (
	SovrintApprove SovrintPermissionDecision = "approve"
	SovrintDeny    SovrintPermissionDecision = "deny"
)

// SovrintSecurityProfile defines a versioned governed-session profile.
type SovrintSecurityProfile struct {
	ProfileID                  string
	Version                    string
	Description                string
	DefaultDecision            SovrintPermissionDecision
	AllowKinds                 []string
	DenyKinds                  []string
	ToolSurfaceMode            string // inherit, allowlist, none
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

// SovrintAuditEvent is a bounded event. It intentionally excludes tool
// arguments, model responses, file contents, provider credentials, and raw
// command output.
type SovrintAuditEvent struct {
	SchemaVersion   string                 `json:"schemaVersion"`
	EventID         string                 `json:"eventId"`
	ParentEventID   string                 `json:"parentEventId,omitempty"`
	EventClass      string                 `json:"eventClass"`
	TimestampUTC    string                 `json:"timestampUtc"`
	ProfileID       string                 `json:"profileId"`
	ProfileVersion  string                 `json:"profileVersion"`
	SessionID       string                 `json:"sessionId"`
	ToolCallID      string                 `json:"toolCallId,omitempty"`
	ToolName        string                 `json:"toolName,omitempty"`
	PermissionKind  string                 `json:"permissionKind,omitempty"`
	Decision        string                 `json:"decision"`
	ReasonCode      string                 `json:"reasonCode"`
	DisclosureClass string                 `json:"disclosureClass"`
	EvidenceStatus  string                 `json:"evidenceStatus"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// SovrintAuditSink records a bounded audit event.
type SovrintAuditSink func(event SovrintAuditEvent) error

// SovrintPermissionEvaluator returns a decision or an empty decision to defer
// to the profile.
type SovrintPermissionEvaluator func(
	request PermissionRequest,
	invocation PermissionInvocation,
) (SovrintPermissionDecision, error)

// SovrintPermissionHandlerOptions configures permission composition.
type SovrintPermissionHandlerOptions struct {
	AuditSink  SovrintAuditSink
	Evaluate   SovrintPermissionEvaluator
	Downstream PermissionHandler
}

// SovrintApplyProfileOptions configures profile application.
type SovrintApplyProfileOptions struct {
	AuditSink         SovrintAuditSink
	EvaluatePermission SovrintPermissionEvaluator
}

// SovrintToolAuthorizer approves a caller-defined tool invocation.
type SovrintToolAuthorizer func(invocation ToolInvocation) (bool, error)

// SovrintToolGuardOptions configures a guarded custom tool.
type SovrintToolGuardOptions struct {
	Profile   SovrintSecurityProfile
	AuditSink SovrintAuditSink
	Authorize SovrintToolAuthorizer
}

const SovrintSystemAppend = "Operate under a bounded SOVRINT governed-session profile. Use only explicitly exposed tools and declared authority. Treat observations, inferences, recommendations, governance decisions, integrity findings, and accepted evidence as distinct classes. Do not claim approval, verification, restoration, or EvidenceGrid acceptance without an explicit external result."

var SovrintStrictProfile = SovrintSecurityProfile{
	ProfileID:                  "sovrint.strict",
	Version:                    "1.0",
	Description:                "Deny every permission and expose no inherited first-party tools.",
	DefaultDecision:            SovrintDeny,
	DenyKinds:                  []string{"read", "write", "shell", "url", "mcp"},
	ToolSurfaceMode:            "none",
	AllowedMCPServers:          []string{},
	AllowedSkillDirectories:    []string{},
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
	AllowedMCPServers:          []string{},
	AllowedSkillDirectories:    []string{},
	ForbidSystemMessageReplace: true,
	FailClosedOnAuditError:     true,
	AuditEnabled:               true,
	SystemMessageAppend:        SovrintSystemAppend + " Operate in read-only mode.",
}

var SovrintResearchProfile = SovrintSecurityProfile{
	ProfileID:                  "sovrint.research",
	Version:                    "1.0",
	Description:                "Permit reads and require an evaluator for every other permission kind.",
	DefaultDecision:            SovrintDeny,
	AllowKinds:                 []string{"read"},
	DenyKinds:                  []string{},
	ToolSurfaceMode:            "inherit",
	AllowedMCPServers:          []string{},
	AllowedSkillDirectories:    []string{},
	ForbidSystemMessageReplace: true,
	FailClosedOnAuditError:     false,
	AuditEnabled:               true,
	SystemMessageAppend:        SovrintSystemAppend + " Separate research observations from verified findings.",
}

var sovrintAuditSequence uint64

func newSovrintAuditEvent(
	profile SovrintSecurityProfile,
	eventClass string,
	sessionID string,
	decision string,
	reasonCode string,
) SovrintAuditEvent {
	sequence := atomic.AddUint64(&sovrintAuditSequence, 1)
	return SovrintAuditEvent{
		SchemaVersion:   "1.0",
		EventID:         fmt.Sprintf("sovrint-%d-%d", time.Now().UnixMilli(), sequence),
		EventClass:      eventClass,
		TimestampUTC:    time.Now().UTC().Format(time.RFC3339Nano),
		ProfileID:       profile.ProfileID,
		ProfileVersion:  profile.Version,
		SessionID:       sessionID,
		Decision:        decision,
		ReasonCode:      reasonCode,
		DisclosureClass: "INTERNAL",
		EvidenceStatus:  "NOT_SUBMITTED",
	}
}

func emitSovrintAudit(
	profile SovrintSecurityProfile,
	sink SovrintAuditSink,
	event SovrintAuditEvent,
) bool {
	if !profile.AuditEnabled {
		return true
	}
	if sink == nil {
		return !profile.FailClosedOnAuditError
	}
	if err := sink(event); err != nil {
		return !profile.FailClosedOnAuditError
	}
	return true
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sovrintPermissionResult(
	approved bool,
	profile SovrintSecurityProfile,
	reasonCode string,
) PermissionRequestResult {
	kind := "denied-by-rules"
	if approved {
		kind = "approved"
	}
	return PermissionRequestResult{
		Kind: kind,
		Rules: []interface{}{
			map[string]interface{}{
				"source":         "sovrint",
				"profileId":      profile.ProfileID,
				"profileVersion": profile.Version,
				"reasonCode":     reasonCode,
			},
		},
	}
}

func evaluateSovrintPermission(
	profile SovrintSecurityProfile,
	request PermissionRequest,
	invocation PermissionInvocation,
	evaluator SovrintPermissionEvaluator,
) (bool, string) {
	if request.Kind == "" {
		return false, "UNKNOWN_PERMISSION_KIND"
	}
	if containsString(profile.DenyKinds, request.Kind) {
		return false, "PROFILE_EXPLICIT_DENY"
	}
	if evaluator != nil {
		decision, err := evaluator(request, invocation)
		if err != nil {
			return false, "APPLICATION_EVALUATOR_FAILED"
		}
		if decision == SovrintApprove {
			return true, "APPLICATION_EVALUATOR_APPROVED"
		}
		if decision == SovrintDeny {
			return false, "APPLICATION_EVALUATOR_DENIED"
		}
	}
	if containsString(profile.AllowKinds, request.Kind) {
		return true, "PROFILE_EXPLICIT_ALLOW"
	}
	if profile.DefaultDecision == SovrintApprove {
		return true, "PROFILE_DEFAULT_ALLOW"
	}
	return false, "PROFILE_DEFAULT_DENY"
}

// CreateSovrintPermissionHandler composes the profile, application evaluator,
// and any existing handler using most-restrictive-wins semantics.
func CreateSovrintPermissionHandler(
	profile SovrintSecurityProfile,
	options SovrintPermissionHandlerOptions,
) PermissionHandler {
	return func(
		request PermissionRequest,
		invocation PermissionInvocation,
	) (PermissionRequestResult, error) {
		approved, reasonCode := evaluateSovrintPermission(
			profile,
			request,
			invocation,
			options.Evaluate,
		)

		if approved && options.Downstream != nil {
			downstreamResult, err := options.Downstream(request, invocation)
			if err != nil {
				approved = false
				reasonCode = "DOWNSTREAM_HANDLER_FAILED"
			} else if downstreamResult.Kind != "approved" {
				approved = false
				reasonCode = "DOWNSTREAM_HANDLER_DENIED"
			}
		}

		event := newSovrintAuditEvent(
			profile,
			"PERMISSION_DECISION",
			invocation.SessionID,
			map[bool]string{true: "APPROVED", false: "DENIED"}[approved],
			reasonCode,
		)
		event.ToolCallID = request.ToolCallID
		event.PermissionKind = request.Kind

		if !emitSovrintAudit(profile, options.AuditSink, event) && approved {
			return sovrintPermissionResult(false, profile, "AUDIT_SINK_UNAVAILABLE"), nil
		}
		return sovrintPermissionResult(approved, profile, reasonCode), nil
	}
}

func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

func intersectStrings(left []string, right []string) []string {
	allowed := make(map[string]struct{}, len(right))
	for _, value := range right {
		allowed[value] = struct{}{}
	}
	result := make([]string, 0)
	for _, value := range left {
		if _, ok := allowed[value]; ok {
			result = append(result, value)
		}
	}
	return result
}

func mergeUniqueStrings(left []string, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	result := make([]string, 0, len(left)+len(right))
	for _, value := range append(copyStrings(left), right...) {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// ApplySovrintProfile returns a copied SessionConfig constrained by the profile.
func ApplySovrintProfile(
	config *SessionConfig,
	profile SovrintSecurityProfile,
	options SovrintApplyProfileOptions,
) (*SessionConfig, error) {
	if config == nil {
		config = &SessionConfig{}
	}
	result := *config
	result.AvailableTools = copyStrings(config.AvailableTools)
	result.ExcludedTools = copyStrings(config.ExcludedTools)
	result.SkillDirectories = copyStrings(config.SkillDirectories)
	result.DisabledSkills = copyStrings(config.DisabledSkills)

	if profile.ToolSurfaceMode == "none" {
		result.AvailableTools = []string{}
	} else if profile.ToolSurfaceMode == "allowlist" {
		if config.AvailableTools == nil {
			result.AvailableTools = copyStrings(profile.AvailableTools)
		} else {
			result.AvailableTools = intersectStrings(config.AvailableTools, profile.AvailableTools)
		}
	}
	result.ExcludedTools = mergeUniqueStrings(config.ExcludedTools, profile.ExcludedTools)

	if config.SystemMessage != nil && config.SystemMessage.Mode == "replace" {
		if profile.ForbidSystemMessageReplace {
			return nil, fmt.Errorf(
				"SOVRINT profile %q forbids system-message replacement",
				profile.ProfileID,
			)
		}
		copied := *config.SystemMessage
		result.SystemMessage = &copied
	} else {
		existing := ""
		if config.SystemMessage != nil {
			existing = config.SystemMessage.Content
		}
		content := profile.SystemMessageAppend
		if existing != "" && content != "" {
			content = existing + "\n\n" + content
		} else if existing != "" {
			content = existing
		}
		if content != "" {
			result.SystemMessage = &SystemMessageConfig{Mode: "append", Content: content}
		}
	}

	result.OnPermissionRequest = CreateSovrintPermissionHandler(
		profile,
		SovrintPermissionHandlerOptions{
			AuditSink:  options.AuditSink,
			Evaluate:   options.EvaluatePermission,
			Downstream: config.OnPermissionRequest,
		},
	)

	if config.CustomAgents != nil {
		result.CustomAgents = append([]CustomAgentConfig{}, config.CustomAgents...)
		if profile.ToolSurfaceMode != "inherit" {
			for index := range result.CustomAgents {
				if profile.ToolSurfaceMode == "none" {
					result.CustomAgents[index].Tools = []string{}
				} else {
					tools := result.CustomAgents[index].Tools
					if tools == nil {
						tools = profile.AvailableTools
					}
					result.CustomAgents[index].Tools = intersectStrings(tools, profile.AvailableTools)
				}
			}
		}
	}

	if config.MCPServers != nil {
		allowed := make(map[string]struct{}, len(profile.AllowedMCPServers))
		for _, name := range profile.AllowedMCPServers {
			allowed[name] = struct{}{}
		}
		result.MCPServers = make(map[string]MCPServerConfig)
		for name, server := range config.MCPServers {
			if _, ok := allowed[name]; ok {
				result.MCPServers[name] = server
			}
		}
	}

	if config.SkillDirectories != nil {
		result.SkillDirectories = intersectStrings(
			config.SkillDirectories,
			profile.AllowedSkillDirectories,
		)
	}
	result.DisabledSkills = mergeUniqueStrings(config.DisabledSkills, profile.DisabledSkills)
	return &result, nil
}

func rejectedSovrintToolResult(reasonCode string) ToolResult {
	return ToolResult{
		TextResultForLLM: "The governed tool invocation was not authorized.",
		ResultType:       "rejected",
		Error:            reasonCode,
		ToolTelemetry: map[string]interface{}{
			"source":     "sovrint",
			"reasonCode": reasonCode,
		},
	}
}

// WrapSovrintTool adds an application authorization gate and bounded audit
// events around a caller-defined tool.
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
			return rejectedSovrintToolResult("AUDIT_SINK_UNAVAILABLE"), nil
		}

		if options.Authorize != nil {
			authorized, err := options.Authorize(invocation)
			if err != nil || !authorized {
				rejected := newSovrintAuditEvent(
					options.Profile,
					"TOOL_INVOCATION_REJECTED",
					invocation.SessionID,
					"REJECTED",
					"CUSTOM_TOOL_AUTHORIZATION_DENIED",
				)
				rejected.ParentEventID = started.EventID
				rejected.ToolCallID = invocation.ToolCallID
				rejected.ToolName = tool.Name
				emitSovrintAudit(options.Profile, options.AuditSink, rejected)
				return rejectedSovrintToolResult("CUSTOM_TOOL_AUTHORIZATION_DENIED"), nil
			}
		}

		approved := newSovrintAuditEvent(
			options.Profile,
			"TOOL_INVOCATION_APPROVED",
			invocation.SessionID,
			"APPROVED",
			"CUSTOM_TOOL_AUTHORIZED",
		)
		approved.ParentEventID = started.EventID
		approved.ToolCallID = invocation.ToolCallID
		approved.ToolName = tool.Name
		emitSovrintAudit(options.Profile, options.AuditSink, approved)

		result, err := original(invocation)
		if err != nil {
			failed := newSovrintAuditEvent(
				options.Profile,
				"TOOL_INVOCATION_FAILED",
				invocation.SessionID,
				"FAILED",
				"CUSTOM_TOOL_FAILED",
			)
			failed.ParentEventID = started.EventID
			failed.ToolCallID = invocation.ToolCallID
			failed.ToolName = tool.Name
			emitSovrintAudit(options.Profile, options.AuditSink, failed)
			return ToolResult{
				TextResultForLLM: "The governed tool invocation failed.",
				ResultType:       "failure",
				Error:            err.Error(),
				ToolTelemetry:    map[string]interface{}{"source": "sovrint"},
			}, nil
		}

		completed := newSovrintAuditEvent(
			options.Profile,
			"TOOL_INVOCATION_COMPLETED",
			invocation.SessionID,
			"RECORDED",
			"CUSTOM_TOOL_COMPLETED",
		)
		completed.ParentEventID = started.EventID
		completed.ToolCallID = invocation.ToolCallID
		completed.ToolName = tool.Name
		emitSovrintAudit(options.Profile, options.AuditSink, completed)
		return result, nil
	}
	return tool
}
