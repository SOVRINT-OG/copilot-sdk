# GitHub Copilot SDK Cookbook — Node.js / TypeScript

This folder hosts short, practical recipes for using the GitHub Copilot SDK with Node.js/TypeScript. Each recipe is concise, copy‑pasteable, and points to fuller examples and tests.

## Recipes

- [SOVRINT Governed Session](sovrint-governed-session.md): Apply deny-by-default profiles, bounded permission handling, custom-tool guards, and audit events.
- [Error Handling](error-handling.md): Handle errors gracefully including connection failures, timeouts, and cleanup.
- [Multiple Sessions](multiple-sessions.md): Manage multiple independent conversations simultaneously.
- [Managing Local Files](managing-local-files.md): Organize files by metadata using AI-powered grouping strategies.
- [PR Visualization](pr-visualization.md): Generate interactive PR age charts using GitHub MCP Server.
- [Persisting Sessions](persisting-sessions.md): Save and resume sessions across restarts.

## Contributing

Add a new recipe by creating a markdown file in this folder and linking it above. Follow repository guidance in [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Status

The SOVRINT governed-session recipe is implemented and backed by SDK unit tests. Other listed recipes may remain scaffolds until populated.
