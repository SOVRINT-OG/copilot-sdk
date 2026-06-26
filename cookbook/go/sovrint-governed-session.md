# SOVRINT Governed Session — Go

This recipe creates a Copilot SDK session with a bounded SOVRINT profile, a fail-closed audit sink, and a guarded custom tool.

## Example

```go
package main

import (
	"fmt"
	"log"

	copilot "github.com/github/copilot-sdk/go"
)

func main() {
	auditEvents := make([]copilot.SovrintAuditEvent, 0)
	auditSink := func(event copilot.SovrintAuditEvent) error {
		// Replace this with a bounded append-only sink.
		// Do not add prompts, tool arguments, results, or credentials.
		auditEvents = append(auditEvents, event)
		return nil
	}

	inspectState := copilot.Tool{
		Name:        "inspect_state",
		Description: "Return a bounded health summary for a named component",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"component": map[string]interface{}{"type": "string"},
			},
			"required": []string{"component"},
		},
		Handler: func(invocation copilot.ToolInvocation) (copilot.ToolResult, error) {
			return copilot.ToolResult{
				TextResultForLLM: `{"status":"observation-only"}`,
				ResultType:       "success",
			}, nil
		},
	}

	governedInspectState := copilot.WrapSovrintTool(
		inspectState,
		copilot.SovrintToolGuardOptions{
			Profile:   copilot.SovrintReadOnlyProfile,
			AuditSink: auditSink,
			Authorize: func(invocation copilot.ToolInvocation) (bool, error) {
				return invocation.SessionID != "", nil
			},
		},
	)

	config, err := copilot.ApplySovrintProfile(
		&copilot.SessionConfig{
			Model:     "gpt-5",
			Tools:     []copilot.Tool{governedInspectState},
			Streaming: true,
			SystemMessage: &copilot.SystemMessageConfig{
				Mode: "append",
				Content: "Return observations without representing them as governance decisions.",
			},
		},
		copilot.SovrintReadOnlyProfile,
		copilot.SovrintApplyProfileOptions{AuditSink: auditSink},
	)
	if err != nil {
		log.Fatal(err)
	}

	client := copilot.NewClient(nil)
	if err := client.Start(); err != nil {
		log.Fatal(err)
	}
	defer client.Stop()

	session, err := client.CreateSession(config)
	if err != nil {
		log.Fatal(err)
	}

	response, err := session.SendAndWait(copilot.MessageOptions{
		Prompt: "Use inspect_state for component registry-alpha and summarize the observation.",
	}, 0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(*response.Data.Content)
	fmt.Printf("Recorded %d bounded audit events.\n", len(auditEvents))
}
```

## What the profile enforces

- `read` permission requests may pass.
- `write`, `shell`, `url`, and `mcp` requests are denied.
- system-message replacement returns an error before session creation.
- any pre-existing permission handler is composed using most-restrictive-wins semantics.
- the wrapped tool records start, authorization, completion, rejection, or failure events.
- tool arguments and results are not placed in SOVRINT audit events.

Use `SovrintStrictProfile` to expose no inherited first-party tool surface. Use `SovrintResearchProfile` with `EvaluatePermission` for narrowly scoped application decisions.
