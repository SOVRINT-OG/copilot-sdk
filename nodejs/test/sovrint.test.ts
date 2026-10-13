import { describe, expect, it, vi } from "vitest";
import {
    SOVRINT_READ_ONLY_PROFILE,
    SOVRINT_RESEARCH_PROFILE,
    SOVRINT_STRICT_PROFILE,
    applySovrintProfile,
    createSovrintPermissionHandler,
    wrapSovrintTool,
    type SovrintAuditEvent,
    type Tool,
} from "../src/index.js";

describe("SOVRINT governed SDK profile", () => {
    it("preserves strict explicit denial even when an evaluator approves", async () => {
        const events: SovrintAuditEvent[] = [];
        const handler = createSovrintPermissionHandler(SOVRINT_STRICT_PROFILE, {
            auditSink: (event) => events.push(event),
            evaluate: () => "approve",
        });

        const result = await handler(
            { kind: "read", toolCallId: "call-1" },
            { sessionId: "session-1" }
        );

        expect(result.kind).toBe("denied-by-rules");
        expect(events).toHaveLength(1);
        expect(events[0]).toMatchObject({
            eventClass: "PERMISSION_DECISION",
            decision: "DENIED",
            reasonCode: "PROFILE_EXPLICIT_DENY",
        });
    });

    it("approves reads and denies writes under the read-only profile", async () => {
        const events: SovrintAuditEvent[] = [];
        const handler = createSovrintPermissionHandler(SOVRINT_READ_ONLY_PROFILE, {
            auditSink: (event) => events.push(event),
        });

        const read = await handler({ kind: "read" }, { sessionId: "session-2" });
        const write = await handler({ kind: "write" }, { sessionId: "session-2" });

        expect(read.kind).toBe("approved");
        expect(write.kind).toBe("denied-by-rules");
        expect(events.map((event) => event.decision)).toEqual(["APPROVED", "DENIED"]);
    });

    it("allows the research evaluator to approve a request not explicitly allowed", async () => {
        const handler = createSovrintPermissionHandler(SOVRINT_RESEARCH_PROFILE, {
            auditSink: () => undefined,
            evaluate: (request) => (request.kind === "url" ? "approve" : undefined),
        });

        const result = await handler({ kind: "url" }, { sessionId: "session-3" });
        expect(result.kind).toBe("approved");
    });

    it("preserves a downstream permission denial", async () => {
        const handler = createSovrintPermissionHandler(SOVRINT_READ_ONLY_PROFILE, {
            auditSink: () => undefined,
            downstream: () => ({ kind: "denied-interactively-by-user" }),
        });

        const result = await handler({ kind: "read" }, { sessionId: "session-4" });
        expect(result.kind).toBe("denied-by-rules");
        expect(result.rules).toEqual(
            expect.arrayContaining([
                expect.objectContaining({ reasonCode: "DOWNSTREAM_HANDLER_DENIED" }),
            ])
        );
    });

    it("fails closed when an audit sink fails", async () => {
        const handler = createSovrintPermissionHandler(SOVRINT_READ_ONLY_PROFILE, {
            auditSink: () => {
                throw new Error("sink unavailable");
            },
        });

        const result = await handler({ kind: "read" }, { sessionId: "session-5" });
        expect(result.kind).toBe("denied-by-rules");
        expect(result.rules).toEqual(
            expect.arrayContaining([
                expect.objectContaining({ reasonCode: "AUDIT_SINK_UNAVAILABLE" }),
            ])
        );
    });

    it("rejects system-message replacement by default", () => {
        expect(() =>
            applySovrintProfile(
                {
                    systemMessage: { mode: "replace", content: "replacement" },
                },
                SOVRINT_READ_ONLY_PROFILE,
                { auditSink: () => undefined }
            )
        ).toThrow(/forbids system-message replacement/);
    });

    it("removes inherited tools, MCP servers, skills, and custom-agent tools in strict mode", () => {
        const config = applySovrintProfile(
            {
                availableTools: ["read_file", "write_file"],
                mcpServers: {
                    alpha: { type: "http", url: "https://example.invalid", tools: ["*"] },
                },
                skillDirectories: ["./skills"],
                customAgents: [
                    {
                        name: "worker",
                        prompt: "work",
                        tools: null,
                    },
                ],
            },
            SOVRINT_STRICT_PROFILE,
            { auditSink: () => undefined }
        );

        expect(config.availableTools).toEqual([]);
        expect(config.mcpServers).toEqual({});
        expect(config.skillDirectories).toEqual([]);
        expect(config.customAgents?.[0].tools).toEqual([]);
        expect(config.systemMessage).toMatchObject({ mode: "append" });
    });

    it("rejects an unauthorized custom tool without calling its handler", async () => {
        const originalHandler = vi.fn(() => "ok");
        const tool: Tool<{ value: string }> = {
            name: "mutate_state",
            description: "Test tool",
            parameters: { type: "object" },
            handler: originalHandler,
        };
        const guarded = wrapSovrintTool(tool, {
            profile: SOVRINT_READ_ONLY_PROFILE,
            auditSink: () => undefined,
            authorize: () => false,
        });

        const result = await guarded.handler(
            { value: "x" },
            {
                sessionId: "session-6",
                toolCallId: "tool-call-1",
                toolName: "mutate_state",
                arguments: { value: "x" },
            }
        );

        expect(originalHandler).not.toHaveBeenCalled();
        expect(result).toMatchObject({
            resultType: "rejected",
            error: "CUSTOM_TOOL_AUTHORIZATION_DENIED",
        });
    });

    it("records a completed authorized custom tool invocation", async () => {
        const events: SovrintAuditEvent[] = [];
        const tool: Tool<{ value: string }> = {
            name: "inspect_state",
            description: "Test tool",
            parameters: { type: "object" },
            handler: ({ value }) => ({ value }),
        };
        const guarded = wrapSovrintTool(tool, {
            profile: SOVRINT_READ_ONLY_PROFILE,
            auditSink: (event) => events.push(event),
            authorize: () => true,
        });

        const result = await guarded.handler(
            { value: "ok" },
            {
                sessionId: "session-7",
                toolCallId: "tool-call-2",
                toolName: "inspect_state",
                arguments: { value: "ok" },
            }
        );

        expect(result).toEqual({ value: "ok" });
        expect(events.map((event) => event.eventClass)).toEqual([
            "TOOL_INVOCATION_STARTED",
            "TOOL_INVOCATION_APPROVED",
            "TOOL_INVOCATION_COMPLETED",
        ]);
        expect(events.every((event) => event.metadata === undefined)).toBe(true);
    });
});
