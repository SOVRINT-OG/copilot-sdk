import pytest

from copilot import (
    SOVRINT_READ_ONLY_PROFILE,
    SOVRINT_RESEARCH_PROFILE,
    SOVRINT_STRICT_PROFILE,
    Tool,
    apply_sovrint_profile,
    create_sovrint_permission_handler,
    wrap_sovrint_tool,
)


@pytest.mark.asyncio
async def test_strict_profile_preserves_explicit_denial():
    events = []
    handler = create_sovrint_permission_handler(
        SOVRINT_STRICT_PROFILE,
        audit_sink=events.append,
        evaluate=lambda request, invocation: "approve",
    )

    result = await handler(
        {"kind": "read", "toolCallId": "call-1"},
        {"session_id": "session-1"},
    )

    assert result["kind"] == "denied-by-rules"
    assert events[0]["reasonCode"] == "PROFILE_EXPLICIT_DENY"


@pytest.mark.asyncio
async def test_read_only_profile_approves_read_and_denies_write():
    events = []
    handler = create_sovrint_permission_handler(
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=events.append,
    )

    read = await handler({"kind": "read"}, {"session_id": "session-2"})
    write = await handler({"kind": "write"}, {"session_id": "session-2"})

    assert read["kind"] == "approved"
    assert write["kind"] == "denied-by-rules"
    assert [event["decision"] for event in events] == ["APPROVED", "DENIED"]


@pytest.mark.asyncio
async def test_research_evaluator_can_approve_non_default_request():
    handler = create_sovrint_permission_handler(
        SOVRINT_RESEARCH_PROFILE,
        audit_sink=lambda event: None,
        evaluate=lambda request, invocation: (
            "approve" if request.get("kind") == "url" else None
        ),
    )

    result = await handler({"kind": "url"}, {"session_id": "session-3"})
    assert result["kind"] == "approved"


@pytest.mark.asyncio
async def test_downstream_denial_is_preserved():
    async def downstream(request, invocation):
        return {"kind": "denied-interactively-by-user"}

    handler = create_sovrint_permission_handler(
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=lambda event: None,
        downstream=downstream,
    )

    result = await handler({"kind": "read"}, {"session_id": "session-4"})
    assert result["kind"] == "denied-by-rules"
    assert result["rules"][0]["reasonCode"] == "DOWNSTREAM_HANDLER_DENIED"


@pytest.mark.asyncio
async def test_audit_failure_fails_closed():
    def failing_sink(event):
        raise RuntimeError("sink unavailable")

    handler = create_sovrint_permission_handler(
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=failing_sink,
    )

    result = await handler({"kind": "read"}, {"session_id": "session-5"})
    assert result["kind"] == "denied-by-rules"
    assert result["rules"][0]["reasonCode"] == "AUDIT_SINK_UNAVAILABLE"


def test_system_message_replacement_is_rejected():
    with pytest.raises(ValueError, match="forbids system-message replacement"):
        apply_sovrint_profile(
            {"system_message": {"mode": "replace", "content": "replacement"}},
            SOVRINT_READ_ONLY_PROFILE,
            audit_sink=lambda event: None,
        )


def test_strict_profile_removes_inherited_surfaces():
    config = apply_sovrint_profile(
        {
            "available_tools": ["read_file", "write_file"],
            "mcp_servers": {
                "alpha": {
                    "type": "http",
                    "url": "https://example.invalid",
                    "tools": ["*"],
                }
            },
            "skill_directories": ["./skills"],
            "custom_agents": [
                {
                    "name": "worker",
                    "prompt": "work",
                    "tools": None,
                }
            ],
        },
        SOVRINT_STRICT_PROFILE,
        audit_sink=lambda event: None,
    )

    assert config["available_tools"] == []
    assert config["mcp_servers"] == {}
    assert config["skill_directories"] == []
    assert config["custom_agents"][0]["tools"] == []
    assert config["system_message"]["mode"] == "append"


@pytest.mark.asyncio
async def test_guarded_tool_rejects_without_calling_original_handler():
    calls = []

    async def original(invocation):
        calls.append(invocation)
        return {
            "textResultForLlm": "ok",
            "resultType": "success",
        }

    tool = Tool(
        name="mutate_state",
        description="Test tool",
        parameters={"type": "object"},
        handler=original,
    )
    guarded = wrap_sovrint_tool(
        tool,
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=lambda event: None,
        authorize=lambda invocation: False,
    )

    result = await guarded.handler(
        {
            "session_id": "session-6",
            "tool_call_id": "tool-call-1",
            "tool_name": "mutate_state",
            "arguments": {"value": "x"},
        }
    )

    assert calls == []
    assert result["resultType"] == "rejected"
    assert result["error"] == "CUSTOM_TOOL_AUTHORIZATION_DENIED"


@pytest.mark.asyncio
async def test_guarded_tool_records_completion():
    events = []

    async def original(invocation):
        return {
            "textResultForLlm": "ok",
            "resultType": "success",
        }

    tool = Tool(
        name="inspect_state",
        description="Test tool",
        parameters={"type": "object"},
        handler=original,
    )
    guarded = wrap_sovrint_tool(
        tool,
        SOVRINT_READ_ONLY_PROFILE,
        audit_sink=events.append,
        authorize=lambda invocation: True,
    )

    result = await guarded.handler(
        {
            "session_id": "session-7",
            "tool_call_id": "tool-call-2",
            "tool_name": "inspect_state",
            "arguments": {},
        }
    )

    assert result["resultType"] == "success"
    assert [event["eventClass"] for event in events] == [
        "TOOL_INVOCATION_STARTED",
        "TOOL_INVOCATION_APPROVED",
        "TOOL_INVOCATION_COMPLETED",
    ]
    assert all("arguments" not in event and "result" not in event for event in events)
