package copilot

import (
	"fmt"
	"sync/atomic"
	"time"
)

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
