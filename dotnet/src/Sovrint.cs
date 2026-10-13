using System.Collections.ObjectModel;
using System.Threading;
using GitHub.Copilot.SDK;

namespace GitHub.Copilot.SDK;

/// <summary>Defines the tool exposure behavior applied by a governed session profile.</summary>
public enum SovrintToolSurfaceMode
{
    Inherit,
    Allowlist,
    None,
}

/// <summary>Versioned, opt-in policy used to constrain a Copilot session.</summary>
public sealed record SovrintSecurityProfile
{
    public required string ProfileId { get; init; }
    public required string Version { get; init; }
    public string Description { get; init; } = "";
    public IReadOnlySet<string> AllowKinds { get; init; } = new HashSet<string>(StringComparer.Ordinal);
    public IReadOnlySet<string> DenyKinds { get; init; } = new HashSet<string>(StringComparer.Ordinal);
    public SovrintToolSurfaceMode ToolSurfaceMode { get; init; } = SovrintToolSurfaceMode.Inherit;
    public IReadOnlyList<string> AvailableTools { get; init; } = Array.Empty<string>();
    public IReadOnlyList<string> ExcludedTools { get; init; } = Array.Empty<string>();
    public IReadOnlySet<string>? AllowedMcpServers { get; init; }
    public IReadOnlySet<string>? AllowedSkillDirectories { get; init; }
    public IReadOnlySet<string> DisabledSkills { get; init; } = new HashSet<string>(StringComparer.Ordinal);
    public bool ForbidSystemMessageReplace { get; init; } = true;
    public bool FailClosedOnAuditError { get; init; }
    public bool AuditEnabled { get; init; } = true;
    public string SystemMessageAppend { get; init; } = SovrintProfiles.SystemAppend;
}

/// <summary>A bounded audit event. It intentionally has no prompt, argument, or result fields.</summary>
public sealed record SovrintAuditEvent(
    string EventId,
    string EventClass,
    DateTimeOffset TimestampUtc,
    string ProfileId,
    string ProfileVersion,
    string SessionId,
    string Decision,
    string ReasonCode,
    string DisclosureClass = "INTERNAL",
    string EvidenceStatus = "NOT_SUBMITTED");

public sealed class SovrintPermissionHandlerOptions
{
    public Action<SovrintAuditEvent>? AuditSink { get; init; }
    public Func<PermissionRequest, PermissionInvocation, bool?>? Evaluate { get; init; }
    public PermissionHandler? Downstream { get; init; }
}

public sealed class SovrintApplyProfileOptions
{
    public Action<SovrintAuditEvent>? AuditSink { get; init; }
    public Func<PermissionRequest, PermissionInvocation, bool?>? EvaluatePermission { get; init; }
}

public static class SovrintProfiles
{
    public const string SystemAppend =
        "Operate under a bounded SOVRINT governed-session profile. " +
        "Use only explicitly exposed tools and declared authority. " +
        "Keep observations, inferences, recommendations, governance decisions, integrity findings, " +
        "and accepted evidence distinct. Do not claim approval, verification, restoration, " +
        "or EvidenceGrid acceptance without an explicit external result.";

    private static readonly IReadOnlySet<string> KnownPermissionKinds =
        new HashSet<string>(["read", "write", "shell", "url", "mcp"], StringComparer.Ordinal);

    public static SovrintSecurityProfile Strict { get; } = new()
    {
        ProfileId = "sovrint.strict",
        Version = "1.0",
        Description = "Deny every permission and expose no inherited tools or external surfaces.",
        DenyKinds = new HashSet<string>(KnownPermissionKinds, StringComparer.Ordinal),
        ToolSurfaceMode = SovrintToolSurfaceMode.None,
        AllowedMcpServers = new HashSet<string>(StringComparer.Ordinal),
        AllowedSkillDirectories = new HashSet<string>(StringComparer.Ordinal),
        FailClosedOnAuditError = true,
    };

    public static SovrintSecurityProfile ReadOnly { get; } = new()
    {
        ProfileId = "sovrint.read-only",
        Version = "1.0",
        Description = "Permit reads while denying mutating and external permission kinds.",
        AllowKinds = new HashSet<string>(["read"], StringComparer.Ordinal),
        DenyKinds = new HashSet<string>(["write", "shell", "url", "mcp"], StringComparer.Ordinal),
        AllowedMcpServers = new HashSet<string>(StringComparer.Ordinal),
        AllowedSkillDirectories = new HashSet<string>(StringComparer.Ordinal),
        FailClosedOnAuditError = true,
        SystemMessageAppend = SystemAppend + " Operate in read-only mode.",
    };

    public static SovrintSecurityProfile Research { get; } = new()
    {
        ProfileId = "sovrint.research",
        Version = "1.0",
        Description = "Permit reads and defer other known permission kinds to an application evaluator.",
        AllowKinds = new HashSet<string>(["read"], StringComparer.Ordinal),
        SystemMessageAppend = SystemAppend + " Separate research observations from verified findings.",
    };

    public static PermissionHandler CreatePermissionHandler(
        SovrintSecurityProfile profile,
        SovrintPermissionHandlerOptions? options = null)
    {
        ArgumentNullException.ThrowIfNull(profile);
        options ??= new SovrintPermissionHandlerOptions();

        return async (request, invocation) =>
        {
            ArgumentNullException.ThrowIfNull(request);
            ArgumentNullException.ThrowIfNull(invocation);

            var kind = request.Kind;
            var approved = false;
            var reason = "PROFILE_DEFAULT_DENY";

            if (string.IsNullOrWhiteSpace(kind) || !KnownPermissionKinds.Contains(kind))
            {
                reason = "UNKNOWN_PERMISSION_KIND";
            }
            else if (profile.DenyKinds.Contains(kind))
            {
                reason = "PROFILE_EXPLICIT_DENY";
            }
            else if (profile.AllowKinds.Contains(kind))
            {
                approved = true;
                reason = "PROFILE_EXPLICIT_ALLOW";
            }
            else if (options.Evaluate is not null)
            {
                try
                {
                    var evaluated = options.Evaluate(request, invocation);
                    if (evaluated.HasValue)
                    {
                        approved = evaluated.Value;
                        reason = approved ? "APPLICATION_EVALUATOR_APPROVED" : "APPLICATION_EVALUATOR_DENIED";
                    }
                }
                catch
                {
                    reason = "APPLICATION_EVALUATOR_FAILED";
                }
            }

            if (approved && options.Downstream is not null)
            {
                try
                {
                    var downstream = await options.Downstream(request, invocation).ConfigureAwait(false);
                    if (!string.Equals(downstream.Kind, "approved", StringComparison.Ordinal))
                    {
                        approved = false;
                        reason = "DOWNSTREAM_HANDLER_DENIED";
                    }
                }
                catch
                {
                    approved = false;
                    reason = "DOWNSTREAM_HANDLER_FAILED";
                }
            }

            var auditRecorded = Emit(options.AuditSink, profile, new SovrintAuditEvent(
                CreateEventId(),
                "PERMISSION_DECISION",
                DateTimeOffset.UtcNow,
                profile.ProfileId,
                profile.Version,
                invocation.SessionId,
                approved ? "APPROVED" : "DENIED",
                reason));

            if (!auditRecorded && approved && profile.FailClosedOnAuditError)
            {
                approved = false;
                reason = "AUDIT_SINK_UNAVAILABLE";
            }

            return new PermissionRequestResult
            {
                Kind = approved ? "approved" : "denied-by-rules",
                Rules =
                [
                    new Dictionary<string, object>
                    {
                        ["source"] = "sovrint",
                        ["profileId"] = profile.ProfileId,
                        ["profileVersion"] = profile.Version,
                        ["reasonCode"] = reason,
                    },
                ],
            };
        };
    }

    public static SessionConfig ApplyProfile(
        SessionConfig config,
        SovrintSecurityProfile profile,
        SovrintApplyProfileOptions? options = null)
    {
        ArgumentNullException.ThrowIfNull(config);
        ArgumentNullException.ThrowIfNull(profile);
        options ??= new SovrintApplyProfileOptions();

        var systemMessage = config.SystemMessage;
        if (systemMessage?.Mode == SystemMessageMode.Replace && profile.ForbidSystemMessageReplace)
        {
            throw new InvalidOperationException(
                $"SOVRINT profile '{profile.ProfileId}' forbids system-message replacement");
        }

        var result = new SessionConfig
        {
            SessionId = config.SessionId,
            Model = config.Model,
            ConfigDir = config.ConfigDir,
            Tools = profile.ToolSurfaceMode == SovrintToolSurfaceMode.None
                ? Array.Empty<Microsoft.Extensions.AI.AIFunction>()
                : config.Tools is null ? null : new List<Microsoft.Extensions.AI.AIFunction>(config.Tools),
            SystemMessage = systemMessage?.Mode == SystemMessageMode.Replace
                ? new SystemMessageConfig { Mode = systemMessage.Mode, Content = systemMessage.Content }
                : new SystemMessageConfig
                {
                    Mode = SystemMessageMode.Append,
                    Content = JoinText(systemMessage?.Content, profile.SystemMessageAppend),
                },
            AvailableTools = ApplyToolSurface(config.AvailableTools, profile),
            ExcludedTools = MergeUnique(config.ExcludedTools, profile.ExcludedTools),
            Provider = config.Provider,
            OnPermissionRequest = CreatePermissionHandler(profile, new SovrintPermissionHandlerOptions
            {
                AuditSink = options.AuditSink,
                Evaluate = options.EvaluatePermission,
                Downstream = config.OnPermissionRequest,
            }),
            Streaming = config.Streaming,
            McpServers = ConstrainDictionary(config.McpServers, profile.AllowedMcpServers),
            CustomAgents = ConstrainAgents(config.CustomAgents, profile),
            SkillDirectories = ConstrainList(config.SkillDirectories, profile.AllowedSkillDirectories),
            DisabledSkills = MergeUnique(config.DisabledSkills, profile.DisabledSkills),
        };

        Emit(options.AuditSink, profile, new SovrintAuditEvent(
            CreateEventId(),
            "PROFILE_APPLIED",
            DateTimeOffset.UtcNow,
            profile.ProfileId,
            profile.Version,
            config.SessionId ?? "",
            "RECORDED",
            "PROFILE_APPLIED"));
        return result;
    }

    private static List<string>? ApplyToolSurface(
        List<string>? configured,
        SovrintSecurityProfile profile)
    {
        if (profile.ToolSurfaceMode == SovrintToolSurfaceMode.None)
        {
            return [];
        }

        if (profile.ToolSurfaceMode != SovrintToolSurfaceMode.Allowlist)
        {
            return configured is null ? null : new List<string>(configured);
        }

        var allowed = new HashSet<string>(profile.AvailableTools, StringComparer.Ordinal);
        return configured is null
            ? new List<string>(profile.AvailableTools)
            : configured.Where(allowed.Contains).Distinct(StringComparer.Ordinal).ToList();
    }

    private static Dictionary<string, object>? ConstrainDictionary(
        Dictionary<string, object>? configured,
        IReadOnlySet<string>? allowed)
    {
        if (configured is null)
        {
            return null;
        }

        return allowed is null
            ? new Dictionary<string, object>(configured, StringComparer.Ordinal)
            : configured.Where(pair => allowed.Contains(pair.Key))
                .ToDictionary(pair => pair.Key, pair => pair.Value, StringComparer.Ordinal);
    }

    private static List<string>? ConstrainList(List<string>? configured, IReadOnlySet<string>? allowed)
    {
        if (configured is null)
        {
            return null;
        }

        return allowed is null
            ? new List<string>(configured)
            : configured.Where(allowed.Contains).Distinct(StringComparer.Ordinal).ToList();
    }

    private static List<CustomAgentConfig>? ConstrainAgents(
        List<CustomAgentConfig>? configured,
        SovrintSecurityProfile profile)
    {
        if (configured is null)
        {
            return null;
        }

        return configured.Select(agent =>
        {
            var tools = profile.ToolSurfaceMode switch
            {
                SovrintToolSurfaceMode.None => [],
                SovrintToolSurfaceMode.Allowlist => (agent.Tools ?? profile.AvailableTools.ToList())
                    .Where(new HashSet<string>(profile.AvailableTools, StringComparer.Ordinal).Contains)
                    .Distinct(StringComparer.Ordinal)
                    .ToList(),
                _ => agent.Tools is null ? null : new List<string>(agent.Tools),
            };
            return new CustomAgentConfig
            {
                Name = agent.Name,
                DisplayName = agent.DisplayName,
                Description = agent.Description,
                Tools = tools,
                Prompt = agent.Prompt,
                McpServers = agent.McpServers is null
                    ? null
                    : new Dictionary<string, object>(agent.McpServers, StringComparer.Ordinal),
                Infer = agent.Infer,
            };
        }).ToList();
    }

    private static List<string> MergeUnique(
        IEnumerable<string>? left,
        IEnumerable<string> right) =>
        (left ?? []).Concat(right).Distinct(StringComparer.Ordinal).ToList();

    private static string JoinText(string? existing, string appended) =>
        string.Join("\n\n", new[] { existing?.Trim(), appended.Trim() }
            .Where(value => !string.IsNullOrWhiteSpace(value)));

    private static bool Emit(
        Action<SovrintAuditEvent>? sink,
        SovrintSecurityProfile profile,
        SovrintAuditEvent auditEvent)
    {
        if (!profile.AuditEnabled)
        {
            return true;
        }

        if (sink is null)
        {
            return !profile.FailClosedOnAuditError;
        }

        try
        {
            sink(auditEvent);
            return true;
        }
        catch
        {
            return false;
        }
    }

    private static long _eventSequence;

    private static string CreateEventId() =>
        $"sovrint-{DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()}-{Interlocked.Increment(ref _eventSequence)}";
}
