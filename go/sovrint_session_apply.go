package copilot

import "fmt"

type SovrintApplyProfileOptions struct {
	AuditSink          SovrintAuditSink
	EvaluatePermission SovrintPermissionEvaluator
}

// ApplySovrintProfile projects a governed profile onto a copied SDK configuration.
func ApplySovrintProfile(
	config *SessionConfig,
	profile SovrintSecurityProfile,
	options SovrintApplyProfileOptions,
) (*SessionConfig, error) {
	if config == nil {
		config = &SessionConfig{}
	}
	result := *config
	result.AvailableTools = copySovrintStrings(config.AvailableTools)
	result.ExcludedTools = copySovrintStrings(config.ExcludedTools)
	result.SkillDirectories = copySovrintStrings(config.SkillDirectories)
	result.DisabledSkills = copySovrintStrings(config.DisabledSkills)

	if profile.ToolSurfaceMode == "none" {
		result.AvailableTools = []string{}
	} else if profile.ToolSurfaceMode == "allowlist" {
		if config.AvailableTools == nil {
			result.AvailableTools = copySovrintStrings(profile.AvailableTools)
		} else {
			result.AvailableTools = intersectSovrintStrings(
				config.AvailableTools,
				profile.AvailableTools,
			)
		}
	}
	result.ExcludedTools = mergeSovrintStrings(config.ExcludedTools, profile.ExcludedTools)

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
		appendSovrintSystemMessage(config, &result, profile)
	}

	result.OnPermissionRequest = CreateSovrintPermissionHandler(
		profile,
		SovrintPermissionHandlerOptions{
			AuditSink:  options.AuditSink,
			Evaluate:   options.EvaluatePermission,
			Downstream: config.OnPermissionRequest,
		},
	)
	constrainSovrintSessionSurfaces(config, &result, profile)
	return &result, nil
}

func appendSovrintSystemMessage(
	config *SessionConfig,
	result *SessionConfig,
	profile SovrintSecurityProfile,
) {
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
