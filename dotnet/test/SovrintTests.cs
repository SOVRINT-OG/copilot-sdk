using GitHub.Copilot.SDK;
using Xunit;

namespace GitHub.Copilot.SDK.Test;

public sealed class SovrintTests
{
    [Fact]
    public async Task StrictProfilePreservesExplicitDenialWhenEvaluatorApproves()
    {
        var events = new List<SovrintAuditEvent>();
        var handler = SovrintProfiles.CreatePermissionHandler(
            SovrintProfiles.Strict,
            new SovrintPermissionHandlerOptions
            {
                AuditSink = events.Add,
                Evaluate = (_, _) => true,
            });

        var result = await handler(
            new PermissionRequest { Kind = "read", ToolCallId = "call-1" },
            new PermissionInvocation { SessionId = "session-1" });

        Assert.Equal("denied-by-rules", result.Kind);
        Assert.Single(events);
        Assert.Equal("PROFILE_EXPLICIT_DENY", result.Rules![0] is Dictionary<string, object> rule
            ? rule["reasonCode"]
            : null);
    }

    [Fact]
    public async Task ReadOnlyProfileApprovesReadAndDeniesWrite()
    {
        var handler = SovrintProfiles.CreatePermissionHandler(SovrintProfiles.ReadOnly);

        var read = await handler(
            new PermissionRequest { Kind = "read" },
            new PermissionInvocation { SessionId = "session-2" });
        var write = await handler(
            new PermissionRequest { Kind = "write" },
            new PermissionInvocation { SessionId = "session-2" });

        Assert.Equal("approved", read.Kind);
        Assert.Equal("denied-by-rules", write.Kind);
    }

    [Fact]
    public async Task UnknownPermissionKindIsDeniedEvenWhenEvaluatorApproves()
    {
        var handler = SovrintProfiles.CreatePermissionHandler(
            SovrintProfiles.Research,
            new SovrintPermissionHandlerOptions { Evaluate = (_, _) => true });

        var result = await handler(
            new PermissionRequest { Kind = "future-kind" },
            new PermissionInvocation { SessionId = "session-3" });

        Assert.Equal("denied-by-rules", result.Kind);
    }

    [Fact]
    public void StrictProfileRemovesInheritedToolsAndExternalSurfacesWithoutMutatingInput()
    {
        var config = new SessionConfig
        {
            AvailableTools = ["read_file", "write_file"],
            McpServers = new Dictionary<string, object>
            {
                ["alpha"] = new object(),
            },
            SkillDirectories = ["./skills"],
            CustomAgents =
            [
                new CustomAgentConfig
                {
                    Name = "worker",
                    Prompt = "work",
                    Tools = null,
                },
            ],
        };

        var projected = SovrintProfiles.ApplyProfile(config, SovrintProfiles.Strict);

        Assert.Empty(projected.AvailableTools!);
        Assert.Empty(projected.McpServers!);
        Assert.Empty(projected.SkillDirectories!);
        Assert.Empty(projected.CustomAgents![0].Tools!);
        Assert.Equal(["read_file", "write_file"], config.AvailableTools);
        Assert.Single(config.McpServers);
        Assert.Single(config.SkillDirectories);
    }

    [Fact]
    public void BuiltInProfileRejectsSystemMessageReplacement()
    {
        var config = new SessionConfig
        {
            SystemMessage = new SystemMessageConfig
            {
                Mode = SystemMessageMode.Replace,
                Content = "replacement",
            },
        };

        var error = Assert.Throws<InvalidOperationException>(
            () => SovrintProfiles.ApplyProfile(config, SovrintProfiles.ReadOnly));

        Assert.Contains("forbids system-message replacement", error.Message);
    }

    [Fact]
    public async Task AuditFailureFailsClosed()
    {
        var handler = SovrintProfiles.CreatePermissionHandler(
            SovrintProfiles.ReadOnly,
            new SovrintPermissionHandlerOptions
            {
                AuditSink = _ => throw new InvalidOperationException("sink unavailable"),
            });

        var result = await handler(
            new PermissionRequest { Kind = "read" },
            new PermissionInvocation { SessionId = "session-4" });

        Assert.Equal("denied-by-rules", result.Kind);
        Assert.Equal("AUDIT_SINK_UNAVAILABLE", ((Dictionary<string, object>)result.Rules![0])["reasonCode"]);
    }
}
