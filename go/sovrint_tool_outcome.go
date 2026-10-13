package copilot

func sovrintToolOutcome(reasonCode string, resultType string) ToolResult {
	return ToolResult{
		TextResultForLLM: "The governed tool invocation did not proceed.",
		ResultType:       resultType,
		Error:            reasonCode,
		ToolTelemetry: map[string]interface{}{
			"source":     "sovrint",
			"reasonCode": reasonCode,
		},
	}
}
