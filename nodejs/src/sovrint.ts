import type {
    PermissionHandler,
    PermissionRequest,
    PermissionRequestResult,
    SessionConfig,
    Tool,
    ToolInvocation,
    ToolResultObject,
} from "./types.js";

export type SovrintPermissionDecision = "approve" | "deny";
export type SovrintToolSurfaceMode = "inherit" | "allowlist" | "none";
export type SovrintAuditDecision =
    | "APPROVED"
    | "DENIED"
    | "REJECTED"
    | "FAILED"
    | "RECORDED"
    | "PENDING";

export type SovrintAuditEventClass =
    | "PROFILE_APPLIED"
    | "PERMISSION_DECISION"
    | "TOOL_INVOCATION_STARTED"
    | "TOOL_INVOCATION_APPROVED"
    | "TOOL_INVOCATION_REJECTED"
    | "TOOL_INVOCATION_COMPLETED"
    | "TOOL_INVOCATION_FAILED"
    | "SYSTEM_MESSAGE_REPLACE_REJECTED"
    | "POLICY_VIOLATION"
    | "AUDIT_SINK_FAILURE";

export interface SovrintSecurityProfile {
    profileId: string;
    version: string;
    description?: string;
    defaultDecision: SovrintPermissionDecision;
    allowKinds?: PermissionRequest["kind"][];
    denyKinds?: PermissionRequest["kind"][];
    toolSurfaceMode?: SovrintToolSurfaceMode;
    availableTools?: string[];
    excludedTools?: string[];
    allowedMcpServers?: string[];
    allowedSkillDirectories?: string[];
    disabledSkills?: string[];
    forbidSystemMessageReplace?: boolean;
    failClosedOnAuditError?: boolean;
    auditEnabled?: boolean;
    systemMessageAppend?: string;
}

export interface SovrintAuditEvent {
    schemaVersion: "1.0";
    eventId: string;
    parentEventId?: string | null;
    eventClass: SovrintAuditEventClass;
    timestampUtc: string;
    profileId: string;
    profileVersion: string;
    sessionId: string;
    toolCallId?: string | null;
    toolName?: string | null;
    permissionKind?: PermissionRequest["kind"] | "unknown" | null;
    decision: SovrintAuditDecision;
    reasonCode: string;
    targetClass?: string | null;
    disclosureClass: "PUBLIC" | "INTERNAL" | "PROTECTED" | "RESTRICTED";
    evidenceStatus:
        | "NOT_SUBMITTED"
        | "SUBMISSION_PENDING"
        | "SUBMITTED"
        | "ACCEPTED"
        | "REJECTED"
        | "QUARANTINED"
        | "RETRYABLE";
    metadata?: Record<string, unknown>;
}

export type SovrintAuditSink = (event: SovrintAuditEvent) => Promise<void> | void;

export type SovrintPermissionEvaluator = (
    request: PermissionRequest,
    invocation: { sessionId: string }
) => Promise<SovrintPermissionDecision | undefined> | SovrintPermissionDecision | undefined;

export interface SovrintPermissionHandlerOptions {
    auditSink?: SovrintAuditSink;
    evaluate?: SovrintPermissionEvaluator;
    downstream?: PermissionHandler;
}

export interface SovrintApplyProfileOptions {
    auditSink?: SovrintAuditSink;
    evaluatePermission?: SovrintPermissionEvaluator;
}

export interface SovrintToolGuardOptions<TArgs = unknown> {
    profile: SovrintSecurityProfile;
    auditSink?: SovrintAuditSink;
    authorize?: (
        args: TArgs,
        invocation: ToolInvocation
    ) => Promise<boolean> | boolean;
    failureMode?: "result" | "rethrow";
}

export const SOVRINT_SYSTEM_APPEND = [
    "Operate under a bounded SOVRINT governed-session profile.",
    "Use only explicitly exposed tools and declared authority.",
    "Treat observations, inferences, recommendations, governance decisions, integrity findings, and accepted evidence as distinct classes.",
    "Do not claim approval, verification, restoration, or EvidenceGrid acceptance without an explicit external result.",
].join(" ");

export const SOVRINT_STRICT_PROFILE: Readonly<SovrintSecurityProfile> = {
    profileId: "sovrint.strict",
    version: "1.0",
    description: "Deny every permission and expose no inherited first-party tools.",
    defaultDecision: "deny",
    allowKinds: [],
    denyKinds: ["read", "write", "shell", "url", "mcp"],
    toolSurfaceMode: "none",
    availableTools: [],
    excludedTools: [],
    allowedMcpServers: [],
    allowedSkillDirectories: [],
    disabledSkills: [],
    forbidSystemMessageReplace: true,
    failClosedOnAuditError: true,
    auditEnabled: true,
    systemMessageAppend: SOVRINT_SYSTEM_APPEND,
};

export const SOVRINT_READ_ONLY_PROFILE: Readonly<SovrintSecurityProfile> = {
    profileId: "sovrint.read-only",
    version: "1.0",
    description: "Permit read requests while denying mutating and external permission kinds.",
    defaultDecision: "deny",
    allowKinds: ["read"],
    denyKinds: ["write", "shell", "url", "mcp"],
    toolSurfaceMode: "inherit",
    excludedTools: [],
    allowedMcpServers: [],
    allowedSkillDirectories: [],
    disabledSkills: [],
    forbidSystemMessageReplace: true,
    failClosedOnAuditError: true,
    auditEnabled: true,
    systemMessageAppend: `${SOVRINT_SYSTEM_APPEND} Operate in read-only mode.`,
};

export const SOVRINT_RESEARCH_PROFILE: Readonly<SovrintSecurityProfile> = {
    profileId: "sovrint.research",
    version: "1.0",
    description: "Permit reads and require an application evaluator for every other permission kind.",
    defaultDecision: "deny",
    allowKinds: ["read"],
    denyKinds: [],
    toolSurfaceMode: "inherit",
    excludedTools: [],
    allowedMcpServers: [],
    allowedSkillDirectories: [],
    disabledSkills: [],
    forbidSystemMessageReplace: true,
    failClosedOnAuditError: false,
    auditEnabled: true,
    systemMessageAppend: `${SOVRINT_SYSTEM_APPEND} Separate research observations from verified findings.`,
};

let auditSequence = 0;

function createAuditEvent(
    profile: SovrintSecurityProfile,
    event: Omit<
        SovrintAuditEvent,
        "schemaVersion" | "eventId" | "timestampUtc" | "profileId" | "profileVersion"
    >
): SovrintAuditEvent {
    auditSequence += 1;
    return {
        schemaVersion: "1.0",
        eventId: `sovrint-${Date.now()}-${auditSequence}`,
        timestampUtc: new Date().toISOString(),
        profileId: profile.profileId,
        profileVersion: profile.version,
        ...event,
    };
}

async function emitAuditEvent(
    profile: SovrintSecurityProfile,
    sink: SovrintAuditSink | undefined,
    event: SovrintAuditEvent
): Promise<boolean> {
    if (!profile.auditEnabled) {
        return true;
    }
    if (!sink) {
        return !profile.failClosedOnAuditError;
    }
    try {
        await sink(event);
        return true;
    } catch {
        return !profile.failClosedOnAuditError;
    }
}

function permissionResult(
    approved: boolean,
    profile: SovrintSecurityProfile,
    reasonCode: string
): PermissionRequestResult {
    return {
        kind: approved ? "approved" : "denied-by-rules",
        rules: [
            {
                source: "sovrint",
                profileId: profile.profileId,
                profileVersion: profile.version,
                reasonCode,
            },
        ],
    };
}

async function evaluateProfilePermission(
    profile: SovrintSecurityProfile,
    request: PermissionRequest,
    invocation: { sessionId: string },
    evaluator?: SovrintPermissionEvaluator
): Promise<{ approved: boolean; reasonCode: string }> {
    if (!request.kind) {
        return { approved: false, reasonCode: "UNKNOWN_PERMISSION_KIND" };
    }

    if (profile.denyKinds?.includes(request.kind)) {
        return { approved: false, reasonCode: "PROFILE_EXPLICIT_DENY" };
    }

    if (evaluator) {
        try {
            const result = await evaluator(request, invocation);
            if (result === "approve") {
                return { approved: true, reasonCode: "APPLICATION_EVALUATOR_APPROVED" };
            }
            if (result === "deny") {
                return { approved: false, reasonCode: "APPLICATION_EVALUATOR_DENIED" };
            }
        } catch {
            return { approved: false, reasonCode: "APPLICATION_EVALUATOR_FAILED" };
        }
    }

    if (profile.allowKinds?.includes(request.kind)) {
        return { approved: true, reasonCode: "PROFILE_EXPLICIT_ALLOW" };
    }

    return profile.defaultDecision === "approve"
        ? { approved: true, reasonCode: "PROFILE_DEFAULT_ALLOW" }
        : { approved: false, reasonCode: "PROFILE_DEFAULT_DENY" };
}

export function createSovrintPermissionHandler(
    profile: SovrintSecurityProfile,
    options: SovrintPermissionHandlerOptions = {}
): PermissionHandler {
    return async (request, invocation) => {
        const profileDecision = await evaluateProfilePermission(
            profile,
            request,
            invocation,
            options.evaluate
        );

        let approved = profileDecision.approved;
        let reasonCode = profileDecision.reasonCode;

        if (approved && options.downstream) {
            try {
                const downstreamResult = await options.downstream(request, invocation);
                if (downstreamResult.kind !== "approved") {
                    approved = false;
                    reasonCode = "DOWNSTREAM_HANDLER_DENIED";
                }
            } catch {
                approved = false;
                reasonCode = "DOWNSTREAM_HANDLER_FAILED";
            }
        }

        const event = createAuditEvent(profile, {
            eventClass: "PERMISSION_DECISION",
            sessionId: invocation.sessionId,
            toolCallId: request.toolCallId ?? null,
            toolName: null,
            permissionKind: request.kind ?? "unknown",
            decision: approved ? "APPROVED" : "DENIED",
            reasonCode,
            disclosureClass: "INTERNAL",
            evidenceStatus: "NOT_SUBMITTED",
        });

        const auditRecorded = await emitAuditEvent(profile, options.auditSink, event);
        if (!auditRecorded && approved) {
            return permissionResult(false, profile, "AUDIT_SINK_UNAVAILABLE");
        }

        return permissionResult(approved, profile, reasonCode);
    };
}

function intersectTools(left: string[], right: string[]): string[] {
    const allowed = new Set(right);
    return left.filter((tool) => allowed.has(tool));
}

function mergeUnique(left: string[] = [], right: string[] = []): string[] {
    return [...new Set([...left, ...right])];
}

function applySystemMessage(
    config: SessionConfig,
    profile: SovrintSecurityProfile
): SessionConfig["systemMessage"] {
    if (config.systemMessage?.mode === "replace") {
        if (profile.forbidSystemMessageReplace !== false) {
            throw new Error(
                `SOVRINT profile '${profile.profileId}' forbids system-message replacement`
            );
        }
        return config.systemMessage;
    }

    const existing = config.systemMessage?.content?.trim();
    const appended = profile.systemMessageAppend?.trim();
    const content = [existing, appended].filter(Boolean).join("\n\n");
    return content ? { mode: "append", content } : config.systemMessage;
}

export function applySovrintProfile(
    config: SessionConfig,
    profile: SovrintSecurityProfile,
    options: SovrintApplyProfileOptions = {}
): SessionConfig {
    let availableTools = config.availableTools;

    if (profile.toolSurfaceMode === "none") {
        availableTools = [];
    } else if (profile.toolSurfaceMode === "allowlist") {
        const profileTools = profile.availableTools ?? [];
        availableTools = config.availableTools
            ? intersectTools(config.availableTools, profileTools)
            : [...profileTools];
    }

    let customAgents = config.customAgents;
    if (customAgents && profile.toolSurfaceMode !== "inherit") {
        customAgents = customAgents.map((agent) => {
            if (profile.toolSurfaceMode === "none") {
                return { ...agent, tools: [] };
            }
            const profileTools = profile.availableTools ?? [];
            const agentTools = agent.tools ?? profileTools;
            return { ...agent, tools: intersectTools(agentTools, profileTools) };
        });
    }

    let mcpServers = config.mcpServers;
    if (mcpServers && profile.allowedMcpServers) {
        const allowed = new Set(profile.allowedMcpServers);
        mcpServers = Object.fromEntries(
            Object.entries(mcpServers).filter(([name]) => allowed.has(name))
        );
    }

    let skillDirectories = config.skillDirectories;
    if (skillDirectories && profile.allowedSkillDirectories) {
        const allowed = new Set(profile.allowedSkillDirectories);
        skillDirectories = skillDirectories.filter((directory) => allowed.has(directory));
    }

    return {
        ...config,
        availableTools,
        excludedTools: mergeUnique(config.excludedTools, profile.excludedTools),
        systemMessage: applySystemMessage(config, profile),
        onPermissionRequest: createSovrintPermissionHandler(profile, {
            auditSink: options.auditSink,
            evaluate: options.evaluatePermission,
            downstream: config.onPermissionRequest,
        }),
        customAgents,
        mcpServers,
        skillDirectories,
        disabledSkills: mergeUnique(config.disabledSkills, profile.disabledSkills),
    };
}

function rejectedToolResult(reasonCode: string): ToolResultObject {
    return {
        textResultForLlm: "The governed tool invocation was not authorized.",
        resultType: "rejected",
        error: reasonCode,
        toolTelemetry: { source: "sovrint", reasonCode },
    };
}

export function wrapSovrintTool<TArgs>(
    tool: Tool<TArgs>,
    options: SovrintToolGuardOptions<TArgs>
): Tool<TArgs> {
    const { profile } = options;

    return {
        ...tool,
        handler: async (args: TArgs, invocation: ToolInvocation) => {
            const started = createAuditEvent(profile, {
                eventClass: "TOOL_INVOCATION_STARTED",
                sessionId: invocation.sessionId,
                toolCallId: invocation.toolCallId,
                toolName: tool.name,
                permissionKind: null,
                decision: "PENDING",
                reasonCode: "TOOL_INVOCATION_RECEIVED",
                disclosureClass: "INTERNAL",
                evidenceStatus: "NOT_SUBMITTED",
            });

            const startRecorded = await emitAuditEvent(profile, options.auditSink, started);
            if (!startRecorded) {
                return rejectedToolResult("AUDIT_SINK_UNAVAILABLE");
            }

            if (options.authorize) {
                let authorized = false;
                try {
                    authorized = await options.authorize(args, invocation);
                } catch {
                    authorized = false;
                }

                if (!authorized) {
                    await emitAuditEvent(
                        profile,
                        options.auditSink,
                        createAuditEvent(profile, {
                            parentEventId: started.eventId,
                            eventClass: "TOOL_INVOCATION_REJECTED",
                            sessionId: invocation.sessionId,
                            toolCallId: invocation.toolCallId,
                            toolName: tool.name,
                            permissionKind: null,
                            decision: "REJECTED",
                            reasonCode: "CUSTOM_TOOL_AUTHORIZATION_DENIED",
                            disclosureClass: "INTERNAL",
                            evidenceStatus: "NOT_SUBMITTED",
                        })
                    );
                    return rejectedToolResult("CUSTOM_TOOL_AUTHORIZATION_DENIED");
                }
            }

            await emitAuditEvent(
                profile,
                options.auditSink,
                createAuditEvent(profile, {
                    parentEventId: started.eventId,
                    eventClass: "TOOL_INVOCATION_APPROVED",
                    sessionId: invocation.sessionId,
                    toolCallId: invocation.toolCallId,
                    toolName: tool.name,
                    permissionKind: null,
                    decision: "APPROVED",
                    reasonCode: "CUSTOM_TOOL_AUTHORIZED",
                    disclosureClass: "INTERNAL",
                    evidenceStatus: "NOT_SUBMITTED",
                })
            );

            try {
                const result = await tool.handler(args, invocation);
                await emitAuditEvent(
                    profile,
                    options.auditSink,
                    createAuditEvent(profile, {
                        parentEventId: started.eventId,
                        eventClass: "TOOL_INVOCATION_COMPLETED",
                        sessionId: invocation.sessionId,
                        toolCallId: invocation.toolCallId,
                        toolName: tool.name,
                        permissionKind: null,
                        decision: "RECORDED",
                        reasonCode: "CUSTOM_TOOL_COMPLETED",
                        disclosureClass: "INTERNAL",
                        evidenceStatus: "NOT_SUBMITTED",
                    })
                );
                return result;
            } catch (error) {
                await emitAuditEvent(
                    profile,
                    options.auditSink,
                    createAuditEvent(profile, {
                        parentEventId: started.eventId,
                        eventClass: "TOOL_INVOCATION_FAILED",
                        sessionId: invocation.sessionId,
                        toolCallId: invocation.toolCallId,
                        toolName: tool.name,
                        permissionKind: null,
                        decision: "FAILED",
                        reasonCode: "CUSTOM_TOOL_FAILED",
                        disclosureClass: "INTERNAL",
                        evidenceStatus: "NOT_SUBMITTED",
                    })
                );

                if (options.failureMode === "result") {
                    return {
                        textResultForLlm: "The governed tool invocation failed.",
                        resultType: "failure",
                        error: error instanceof Error ? error.message : "unknown error",
                        toolTelemetry: { source: "sovrint" },
                    } satisfies ToolResultObject;
                }
                throw error;
            }
        },
    };
}
