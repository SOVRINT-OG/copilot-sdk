# SOVRINT Governed Session — Node.js / TypeScript

This recipe creates a Copilot SDK session with a bounded SOVRINT profile, a fail-closed audit sink, and a guarded custom tool.

## Example

```typescript
import {
    CopilotClient,
    SOVRINT_READ_ONLY_PROFILE,
    applySovrintProfile,
    defineTool,
    wrapSovrintTool,
    type SovrintAuditEvent,
} from "@github/copilot-sdk";

const auditEvents: SovrintAuditEvent[] = [];
const auditSink = async (event: SovrintAuditEvent) => {
    // Replace this with a bounded append-only sink.
    // Do not add prompts, tool arguments, results, or credentials.
    auditEvents.push(event);
};

const inspectState = defineTool("inspect_state", {
    description: "Return a bounded health summary for a named component",
    parameters: {
        type: "object",
        properties: {
            component: { type: "string" },
        },
        required: ["component"],
    },
    handler: async ({ component }: { component: string }) => ({
        component,
        status: "observation-only",
    }),
});

const governedInspectState = wrapSovrintTool(inspectState, {
    profile: SOVRINT_READ_ONLY_PROFILE,
    auditSink,
    authorize: async (_args, invocation) => invocation.sessionId.length > 0,
});

const sessionConfig = applySovrintProfile(
    {
        model: "gpt-5",
        tools: [governedInspectState],
        streaming: true,
        systemMessage: {
            mode: "append",
            content: "Return observations without representing them as governance decisions.",
        },
    },
    SOVRINT_READ_ONLY_PROFILE,
    {
        auditSink,
    }
);

const client = new CopilotClient();
const session = await client.createSession(sessionConfig);

const response = await session.sendAndWait({
    prompt: "Use inspect_state for component registry-alpha and summarize the observation.",
});

console.log(response?.data.content);
console.log(`Recorded ${auditEvents.length} bounded audit events.`);

await client.stop();
```

## What the profile enforces

- `read` permission requests may pass.
- `write`, `shell`, `url`, and `mcp` requests are denied.
- system-message replacement throws before session creation.
- any pre-existing permission handler is composed using most-restrictive-wins semantics.
- the wrapped tool records start, authorization, completion, rejection, or failure events.
- tool arguments and results are not placed in SOVRINT audit events.

## Strict mode

Replace `SOVRINT_READ_ONLY_PROFILE` with `SOVRINT_STRICT_PROFILE` to expose no inherited first-party tool surface and deny every permission kind. Caller-defined tools must still be explicitly supplied and guarded.

## Research mode

`SOVRINT_RESEARCH_PROFILE` permits reads and defers all other permission kinds to `evaluatePermission`. The evaluator should use deployment-specific allowlists and must return no decision when it cannot establish scope.
