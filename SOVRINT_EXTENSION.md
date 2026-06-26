# SOVRINT™ Governed SDK Extension

**Author:** Katrina Pietroniro  
**Version:** 1.0  
**Status:** Additive and opt-in

This repository includes a governed-session extension around the existing Copilot SDK interfaces. The upstream JSON-RPC protocol remains unchanged.

The extension includes permission profiles, session-configuration helpers, custom-tool guards, bounded audit events, shared schemas, tests, and cross-language recipes.

## Documentation

- [Architecture](docs/sovrint/README.md)
- [Security profile](docs/sovrint/SECURITY_PROFILE.md)
- [Governance, integrity, and evidence interfaces](docs/sovrint/GOVERNANCE_INTEGRITY_EVIDENCE.md)

## Recipes

- [TypeScript](cookbook/nodejs/sovrint-governed-session.md)
- [Python](cookbook/python/sovrint-governed-session.md)
- [Go](cookbook/go/sovrint-governed-session.md)
- [.NET](cookbook/dotnet/sovrint-governed-session.md)

Existing SDK behavior remains unchanged unless an application applies the extension helpers.
