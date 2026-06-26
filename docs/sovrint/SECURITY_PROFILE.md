# SOVRINT™ SDK Security Profile

**Author:** Katrina Pietroniro  
**Version:** 1.0  
**Status:** Canonical extension security profile

## Protected Surfaces

- filesystem reads and writes;
- shell execution;
- network and URL access;
- MCP server invocation;
- custom-tool invocation;
- system-message replacement;
- skill loading;
- custom-agent tool access;
- provider credentials;
- session configuration;
- audit event integrity;
- governance and evidence references.

## Default Position

The SOVRINT strict profile is deny-by-default.

A permission is approved only when:

1. its request kind is allowed by the active profile;
2. an application evaluator does not deny it;
3. any pre-existing SDK permission handler also approves it;
4. the audit decision is recorded when fail-closed auditing is enabled.

## Decision Precedence

```text
EXPLICIT DENY
→ APPLICATION EVALUATOR
→ PROFILE ALLOW
→ PROFILE DEFAULT
→ EXISTING HANDLER
→ FINAL DECISION
```

The most restrictive result wins.

## System Message Boundary

The upstream SDK supports append mode and replace mode. Replace mode removes SDK-managed guardrails. The SOVRINT profiles therefore forbid replace mode by default.

Applications may opt out only through an explicit custom profile. Doing so is a deployment decision, not a safe default.

## Tool Surface Reduction

`availableTools` is preferred over a broad tool surface. When both an application config and a SOVRINT profile specify an allowlist, the effective set is the intersection.

`excludedTools` is additive. Profile exclusions and application exclusions are combined.

Custom tools remain caller code. Wrapping a custom tool adds authorization and audit boundaries but does not make the tool intrinsically safe.

## Permission Kinds

The current SDK permission request kinds are:

- `read`
- `write`
- `shell`
- `url`
- `mcp`

The extension treats unknown kinds as denied.

## Audit Events

The extension emits bounded events for:

- permission decision;
- custom-tool start;
- custom-tool approval or rejection;
- custom-tool completion;
- custom-tool failure;
- profile application;
- policy violation.

Audit events must not contain:

- provider API keys;
- bearer tokens;
- unrestricted prompts or responses;
- raw file contents;
- full shell output;
- private witnesses;
- secret environment variables.

## Failure-Safe Rules

```text
UNKNOWN PERMISSION KIND → DENY
AUDIT FAILURE WITH FAIL-CLOSED ENABLED → DENY
SYSTEM MESSAGE REPLACE UNDER FORBIDDEN PROFILE → THROW
EMPTY EFFECTIVE TOOL ALLOWLIST → EXPOSE NO FIRST-PARTY TOOLS
CUSTOM AUTHORIZER ERROR → REJECT TOOL INVOCATION
EXISTING PERMISSION HANDLER DENIAL → PRESERVE DENIAL
```

## Threats Addressed

### Accidental broad permissions

Mitigated through deny-by-default profiles and explicit allowlists.

### Permission-handler bypass

Mitigated by composing profile and application handlers using most-restrictive-wins semantics.

### System prompt replacement

Mitigated by rejecting replace mode unless a custom profile explicitly permits it.

### Tool confusion

Mitigated through exact tool names, wrapped authorization, invocation events, and bounded telemetry.

### Audit exfiltration

Mitigated by bounded event fields and explicit prohibition of secrets and unrestricted content.

### Profile drift

Mitigated through versioned JSON profiles, a common schema, manifest commitments, and CI validation.

## Residual Risks

- the SDK server may introduce new permission fields or kinds;
- an application evaluator may be incorrect;
- a tool may perform broader actions than its name or schema suggests;
- approved reads may still expose sensitive information;
- URL and MCP requests can create external disclosure paths;
- audit sinks can be unavailable, compromised, or incomplete;
- system-message append content is not an enforcement boundary by itself;
- model behavior is not equivalent to policy enforcement;
- technical controls do not establish legal authority or valid consent.

## Canonical Rule

```text
THE PROFILE REDUCES AUTHORITY.
IT DOES NOT CREATE AUTHORITY.
```
