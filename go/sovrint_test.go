package copilot

import "testing"

func auditOK(event SovrintAuditEvent) error { return nil }

func TestSovrintPermissionProfiles(t *testing.T) {
	strict := CreateSovrintPermissionHandler(
		SovrintStrictProfile,
		SovrintPermissionHandlerOptions{AuditSink: auditOK},
	)
	result, err := strict(
		PermissionRequest{Kind: "read"},
		PermissionInvocation{SessionID: "session-1"},
	)
	if err != nil || result.Kind != "denied-by-rules" {
		t.Fatalf("strict profile should deny read: %#v %v", result, err)
	}

	readOnly := CreateSovrintPermissionHandler(
		SovrintReadOnlyProfile,
		SovrintPermissionHandlerOptions{AuditSink: auditOK},
	)
	read, _ := readOnly(
		PermissionRequest{Kind: "read"},
		PermissionInvocation{SessionID: "session-2"},
	)
	write, _ := readOnly(
		PermissionRequest{Kind: "write"},
		PermissionInvocation{SessionID: "session-2"},
	)
	if read.Kind != "approved" || write.Kind != "denied-by-rules" {
		t.Fatalf("unexpected read-only decisions: read=%#v write=%#v", read, write)
	}
}

func TestSovrintResearchEvaluator(t *testing.T) {
	handler := CreateSovrintPermissionHandler(
		SovrintResearchProfile,
		SovrintPermissionHandlerOptions{
			AuditSink: auditOK,
			Evaluate: func(
				request PermissionRequest,
				invocation PermissionInvocation,
			) (SovrintPermissionDecision, error) {
				if request.Kind == "url" {
					return SovrintApprove, nil
				}
				return "", nil
			},
		},
	)
	result, err := handler(
		PermissionRequest{Kind: "url"},
		PermissionInvocation{SessionID: "session-3"},
	)
	if err != nil || result.Kind != "approved" {
		t.Fatalf("research evaluator should approve: %#v %v", result, err)
	}
}

func TestApplySovrintProfile(t *testing.T) {
	_, err := ApplySovrintProfile(
		&SessionConfig{
			SystemMessage: &SystemMessageConfig{Mode: "replace", Content: "replacement"},
		},
		SovrintReadOnlyProfile,
		SovrintApplyProfileOptions{AuditSink: auditOK},
	)
	if err == nil {
		t.Fatal("expected replacement rejection")
	}

	config, err := ApplySovrintProfile(
		&SessionConfig{
			AvailableTools:   []string{"read_file", "write_file"},
			SkillDirectories: []string{"skills"},
			CustomAgents: []CustomAgentConfig{
				{Name: "worker", Prompt: "work"},
			},
		},
		SovrintStrictProfile,
		SovrintApplyProfileOptions{AuditSink: auditOK},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.AvailableTools == nil || len(config.AvailableTools) != 0 {
		t.Fatalf("expected explicit empty tool surface: %#v", config.AvailableTools)
	}
	if len(config.SkillDirectories) != 0 {
		t.Fatalf("expected no skill directories: %#v", config.SkillDirectories)
	}
	if len(config.CustomAgents) != 1 || config.CustomAgents[0].Tools == nil {
		t.Fatalf("expected constrained custom agent: %#v", config.CustomAgents)
	}
}

func TestWrapSovrintTool(t *testing.T) {
	called := false
	tool := Tool{
		Name: "inspect_state",
		Handler: func(invocation ToolInvocation) (ToolResult, error) {
			called = true
			return ToolResult{TextResultForLLM: "ok", ResultType: "success"}, nil
		},
	}
	guarded := WrapSovrintTool(
		tool,
		SovrintToolGuardOptions{
			Profile:   SovrintReadOnlyProfile,
			AuditSink: auditOK,
			Authorize: func(invocation ToolInvocation) (bool, error) { return false, nil },
		},
	)
	result, err := guarded.Handler(ToolInvocation{
		SessionID:  "session-4",
		ToolCallID: "call-1",
		ToolName:   "inspect_state",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called || result.ResultType != "rejected" {
		t.Fatalf("expected guarded rejection: called=%v result=%#v", called, result)
	}
}
