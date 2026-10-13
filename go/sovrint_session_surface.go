package copilot

func constrainSovrintSessionSurfaces(
	config *SessionConfig,
	result *SessionConfig,
	profile SovrintSecurityProfile,
) {
	if config.CustomAgents != nil {
		result.CustomAgents = append([]CustomAgentConfig{}, config.CustomAgents...)
		if profile.ToolSurfaceMode != "inherit" {
			for index := range result.CustomAgents {
				if profile.ToolSurfaceMode == "none" {
					result.CustomAgents[index].Tools = []string{}
					continue
				}
				tools := result.CustomAgents[index].Tools
				if tools == nil {
					tools = profile.AvailableTools
				}
				result.CustomAgents[index].Tools = intersectSovrintStrings(
					tools,
					profile.AvailableTools,
				)
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
		result.SkillDirectories = intersectSovrintStrings(
			config.SkillDirectories,
			profile.AllowedSkillDirectories,
		)
	}
	result.DisabledSkills = mergeSovrintStrings(config.DisabledSkills, profile.DisabledSkills)
}
