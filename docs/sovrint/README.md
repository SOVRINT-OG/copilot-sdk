# SOVRINT™ Governed Copilot SDK Profile

**Author:** Katrina Pietroniro  
**Framework:** SOVRINT™  
**Repository:** `SOVRINT-OG/copilot-sdk`  
**Extension Version:** 1.0  
**Status:** Additive SDK extension; upstream Copilot SDK interfaces preserved

## Purpose

The SOVRINT™ governed profile adds an explicit control plane around Copilot SDK sessions without replacing the upstream JSON-RPC client or changing the upstream SDK protocol.

It provides:

- deny-by-default permission profiles;
- least-authority tool exposure;
- append-only system-message governance by default;
- custom-tool authorization wrappers;
- bounded audit events;
- governance, integrity, and EvidenceGrid interface contracts;
- simulation-safe examples;
- language-specific helpers for TypeScript, Python, and Go;
- recipes for TypeScript, Python, Go, and .NET;
- schema and CI validation.

## Runtime Position

```text
APPLICATION
→ SOVRINT SESSION PROFILE
→ COPILOT SDK CLIENT
→ JSON-RPC
→ COPILOT CLI SERVER
```

The extension does not represent itself as the GitHub Copilot service, does not replace GitHub authentication or billing, and does not modify the Copilot CLI protocol.

## Canonical Control Sequence

```text
DECLARE PROFILE
→ REDUCE TOOL SURFACE
→ INSTALL PERMISSION HANDLER
→ APPEND GOVERNANCE INSTRUCTIONS
→ WRAP CUSTOM TOOLS
→ CREATE SESSION
→ OBSERVE EVENTS
→ RECORD BOUNDED AUDIT EVENTS
→ ESCALATE OR CONTINUE
```

## Security Profiles

### `strict`

- denies every first-party permission unless an application-specific evaluator explicitly approves it;
- exposes no first-party tool unless the application supplies an allowlist;
- forbids system-message replacement;
- fails closed when the audit sink cannot record a permission decision.

### `read-only`

- permits `read` requests;
- denies `shell`, `write`, `mcp`, and `url` by default;
- forbids system-message replacement;
- supports additional application-specific restrictions.

### `research`

- permits `read` requests by default;
- requires an application evaluator for network, MCP, shell, and write requests;
- exposes only explicitly named tools;
- preserves audit events for decisions and custom-tool invocations.

Profiles are templates, not universal safety guarantees. The application remains responsible for request-field interpretation, tool definitions, deployment boundaries, and audit storage.

## Language Support

| Surface | Support |
|---|---|
| TypeScript / Node.js | First-class helper module and unit tests |
| Python | First-class helper module and unit tests |
| Go | First-class helper module and unit tests |
| .NET | Governed-session recipe using native SDK interfaces |
| Common | JSON schemas, profiles, policies, examples, CI |

## Repository Additions

```text
docs/sovrint/
  README.md
  SECURITY_PROFILE.md
  GOVERNANCE_INTEGRITY_EVIDENCE.md
sovrint/
  manifest.yaml
  profiles/
  policy/
  schemas/
  examples/
nodejs/src/sovrint.ts
python/copilot/sovrint.py
go/sovrint.go
cookbook/*/sovrint-governed-session.md
```

## Authority Separation

The SDK helper may:

- restrict session tools;
- evaluate permission requests;
- append governance instructions;
- wrap caller-defined tools;
- emit bounded audit events;
- refuse unsafe profile combinations.

It may not:

- grant permissions beyond application authority;
- treat a model response as a governance decision;
- treat an audit event as EvidenceGrid acceptance;
- fabricate continuity receipts;
- replace the SOVRINT Governance Runtime, Integrity Engine, or EvidenceGrid;
- claim that deny-by-default configuration eliminates all application risk.

## Upstream Compatibility

The extension is additive. Existing `CopilotClient`, `SessionConfig`, custom tools, MCP configuration, provider configuration, and session APIs remain unchanged.

Applications opt in by importing the SOVRINT helpers and applying a profile before session creation.

## Canonical Doctrine

```text
NO TOOL WITHOUT DECLARED PURPOSE.
NO PERMISSION WITHOUT AN EXPLICIT DECISION.
NO SYSTEM OVERRIDE WITHOUT AN EXPLICIT POLICY.
NO MUTATION WITHOUT SCOPE.
NO AUDIT CLAIM BEYOND THE EVIDENCE ACTUALLY RECORDED.
```
