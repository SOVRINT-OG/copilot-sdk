# SOVRINT Governed Session — Python

This recipe creates a Copilot SDK session with a bounded SOVRINT profile, a fail-closed audit sink, and a guarded custom tool.

## Example

```python
import asyncio

from copilot import (
    CopilotClient,
    SOVRINT_READ_ONLY_PROFILE,
    apply_sovrint_profile,
    define_tool,
    wrap_sovrint_tool,
)


audit_events = []


def audit_sink(event):
    # Replace this with a bounded append-only sink.
    # Do not add prompts, tool arguments, results, or credentials.
    audit_events.append(event)


@define_tool(description="Return a bounded health summary for a named component")
async def inspect_state(params: dict) -> dict:
    return {
        "component": params["component"],
        "status": "observation-only",
    }


governed_inspect_state = wrap_sovrint_tool(
    inspect_state,
    SOVRINT_READ_ONLY_PROFILE,
    audit_sink=audit_sink,
    authorize=lambda invocation: bool(invocation.get("session_id")),
)


async def main():
    client = CopilotClient()
    await client.start()

    config = apply_sovrint_profile(
        {
            "model": "gpt-5",
            "tools": [governed_inspect_state],
            "streaming": True,
            "system_message": {
                "mode": "append",
                "content": (
                    "Return observations without representing them as governance decisions."
                ),
            },
        },
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=audit_sink,
    )

    session = await client.create_session(config)
    response = await session.send_and_wait(
        {
            "prompt": (
                "Use inspect_state for component registry-alpha and summarize the observation."
            )
        }
    )

    print(response.data.content)
    print(f"Recorded {len(audit_events)} bounded audit events.")
    await client.stop()


asyncio.run(main())
```

## What the profile enforces

- `read` permission requests may pass.
- `write`, `shell`, `url`, and `mcp` requests are denied.
- system-message replacement raises `ValueError` before session creation.
- any pre-existing permission handler is composed using most-restrictive-wins semantics.
- the wrapped tool records start, authorization, completion, rejection, or failure events.
- tool arguments and results are not placed in SOVRINT audit events.

## Strict mode

Use `SOVRINT_STRICT_PROFILE` to expose no inherited first-party tool surface and deny every permission kind.

## Research mode

Use `SOVRINT_RESEARCH_PROFILE` with `evaluate_permission=` to approve a narrowly scoped request after application-specific validation. Returning `None` defers to the profile, which denies by default.
