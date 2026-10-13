package copilot

func containsSovrintValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	if containsSovrintValue(profile.DenyKinds, request.Kind) {
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
	if containsSovrintValue(profile.AllowKinds, request.Kind) {
		return true, "PROFILE_EXPLICIT_ALLOW"
	}
	if profile.DefaultDecision == SovrintApprove {
		return true, "PROFILE_DEFAULT_ALLOW"
	}
	return false, "PROFILE_DEFAULT_DENY"
}
