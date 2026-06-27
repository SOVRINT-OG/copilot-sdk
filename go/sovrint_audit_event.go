package copilot

// SovrintAuditEvent is the bounded event envelope for governed SDK activity.
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

type SovrintAuditSink func(SovrintAuditEvent) error
