package copilot

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
