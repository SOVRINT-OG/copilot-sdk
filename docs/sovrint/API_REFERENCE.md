# SOVRINT™ Extension API Reference

**Author:** Katrina Pietroniro  
**Version:** 1.0

## Cross-Language Mapping

| Capability | TypeScript | Python | Go |
|---|---|---|---|
| Strict profile | `SOVRINT_STRICT_PROFILE` | `SOVRINT_STRICT_PROFILE` | `SovrintStrictProfile` |
| Read-only profile | `SOVRINT_READ_ONLY_PROFILE` | `SOVRINT_READ_ONLY_PROFILE` | `SovrintReadOnlyProfile` |
| Research profile | `SOVRINT_RESEARCH_PROFILE` | `SOVRINT_RESEARCH_PROFILE` | `SovrintResearchProfile` |
| Apply profile | `applySovrintProfile` | `apply_sovrint_profile` | `ApplySovrintProfile` |
| Permission handler | `createSovrintPermissionHandler` | `create_sovrint_permission_handler` | `CreateSovrintPermissionHandler` |
| Guard custom tool | `wrapSovrintTool` | `wrap_sovrint_tool` | `WrapSovrintTool` |

## Permission Evaluation

The handler applies explicit denial, application evaluation, explicit allowance, profile default, any existing handler, and bounded audit recording. The most restrictive result wins.

## Tool-Surface Modes

- `inherit` keeps the application surface.
- `allowlist` intersects application and profile lists.
- `none` creates an explicit empty first-party tool list and clears custom-agent tool lists.

## Custom Tools

The wrapper records receipt, approval or rejection, and completion or failure. It excludes arguments and results from the bounded audit event.

## Compatibility

The helpers return normal SDK session configurations, permission handlers, and tools. Client methods and the JSON-RPC protocol remain unchanged.
