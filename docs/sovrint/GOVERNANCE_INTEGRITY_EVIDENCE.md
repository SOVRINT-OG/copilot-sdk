# SOVRINT™ Governance, Integrity, and Evidence Interfaces

**Author:** Katrina Pietroniro  
**Version:** 1.0  
**Status:** Canonical extension interface guide

## Purpose

This document defines how a governed Copilot SDK session may interact with the SOVRINT Governance Runtime, Integrity Engine, and EvidenceGrid without collapsing their distinct authorities into the SDK process.

## Authority Topology

```text
APPLICATION
→ SOVRINT SDK PROFILE
→ COPILOT SDK SESSION
→ CUSTOM TOOL OR FIRST-PARTY OPERATION
→ GOVERNANCE DECISION, WHEN REQUIRED
→ EXECUTION
→ INTEGRITY REVALIDATION, WHEN REQUIRED
→ AUDIT EVENT
→ EVIDENCE ADAPTER
→ EVIDENCEGRID
```

## Governance Boundary

A governed SDK helper may collect the information required for a governance request, including:

- session identifier;
- tool name;
- permission kind;
- requested action class;
- declared scope;
- target reference;
- reversibility reference;
- intervention estimate;
- consent and review requirements;
- provenance parent.

The helper may not issue an `ALLOW` decision on behalf of the Governance Runtime unless the application has explicitly delegated that narrow decision to a local policy evaluator.

A local approval remains a local approval. It is not automatically a system-wide governance decision.

## Integrity Boundary

The SDK extension may emit integrity observations for:

- permission-profile mismatch;
- unexpected permission kind;
- unavailable audit sink;
- custom-tool authorization failure;
- system-message replacement attempt;
- undeclared tool invocation;
- schema validation failure;
- session configuration drift;
- evidence submission failure.

It may request Integrity Engine classification or correction, but it must not label an observation as verified malicious action without the corresponding evidence and authority.

## Evidence Boundary

Audit events are EvidenceGrid candidates, not EvidenceGrid blocks.

The extension may:

- create a bounded audit event;
- calculate or attach application-provided commitments;
- submit through an evidence adapter;
- retain the returned ledger reference or failure state.

It may not:

- assign EvidenceGrid sequence;
- fabricate a State Root;
- replace Proof Token `π`;
- create continuity receipts;
- represent a pending audit event as accepted evidence;
- erase rejected or quarantined submissions.

## Recommended Decision Classes

| SDK event | Default route |
|---|---|
| Read request under read-only profile | Local profile evaluation |
| Write request | Governance evaluation or deny |
| Shell request | Governance evaluation or deny |
| URL request | Application allowlist plus governance policy |
| MCP request | Named server and tool allowlist plus governance policy |
| Custom tool with no mutation | Local authorization and audit |
| Custom tool with mutation | Governance decision before execution |
| System-message replace attempt | Deny and record |
| Audit sink failure | Deny when fail-closed is enabled |
| Unknown permission kind | Deny, classify, and review |

## Audit Event Lifecycle

```text
CREATED
→ RECORDED_LOCALLY
→ SUBMISSION_PENDING
→ SUBMITTED
→ ACCEPTED OR REJECTED OR QUARANTINED OR RETRYABLE
```

Every status transition must preserve the prior event identifier and provenance parent.

## Evidence Minimization

The event should contain commitments, classifications, and bounded references rather than unrestricted content.

Recommended fields:

- event identifier;
- event class;
- timestamp;
- session reference;
- profile identifier and version;
- permission kind or tool name;
- decision;
- reason code;
- target class;
- scope commitment;
- governance reference;
- integrity reference;
- parent event reference;
- disclosure class;
- evidence status.

## Canonical Separations

```text
MODEL RESPONSE ≠ GOVERNANCE DECISION
LOCAL PROFILE APPROVAL ≠ GLOBAL AUTHORITY
AUDIT EVENT ≠ EVIDENCEGRID BLOCK
ANOMALY ≠ VERIFIED CAUSE
TOOL COMPLETION ≠ RESTORATION
LOG PRESENCE ≠ CONTINUITY PROOF
```
