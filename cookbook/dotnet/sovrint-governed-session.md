# SOVRINT Governed Session — .NET

The .NET SDK includes a first-class `SovrintProfiles` helper for permission composition, tool-surface projection, MCP and skill filtering, system-message append protection, and bounded audit events. This recipe applies the helper through the native `SessionConfig` surface; it does not alter the SDK protocol or create global governance authority.

## Conservative profile

```csharp
using GitHub.Copilot.SDK;

const string SovrintSystemAppend = @"
Operate under a bounded SOVRINT governed-session profile.
Use only explicitly exposed tools and declared authority.
Treat observations, inferences, recommendations, governance decisions,
integrity findings, and accepted evidence as distinct classes.
Do not claim approval, verification, restoration, or EvidenceGrid acceptance
without an explicit external result.
";

static SessionConfig ApplySovrintStrictProfile(SessionConfig config)
{
    if (config.SystemMessage?.Mode == SystemMessageMode.Replace)
    {
        throw new InvalidOperationException(
            "The SOVRINT strict profile forbids system-message replacement."
        );
    }

    var existing = config.SystemMessage?.Content;
    config.SystemMessage = new SystemMessageConfig
    {
        Mode = SystemMessageMode.Append,
        Content = string.Join(
            Environment.NewLine + Environment.NewLine,
            new[] { existing, SovrintSystemAppend }.Where(value => !string.IsNullOrWhiteSpace(value))
        )
    };

    // An explicit empty allowlist exposes no inherited first-party tools.
    config.AvailableTools = [];

    // Caller-defined tools must be explicitly supplied after review.
    config.Tools = [];

    return config;
}

await using var client = new CopilotClient();
await client.StartAsync();

var config = SovrintProfiles.ApplyProfile(new SessionConfig
{
    Model = "gpt-5",
    Streaming = true,
}, SovrintProfiles.Strict);

await using var session = await client.CreateSessionAsync(config);
await session.SendAsync(new MessageOptions
{
    Prompt = "Summarize the supplied text without using tools."
});
```

## Read-oriented profile

A read-oriented deployment should provide an explicit `AvailableTools` allowlist containing only reviewed read operations. Do not infer tool safety from names alone; validate the actual SDK and CLI tool identifiers available in the deployed version.

```csharp
var config = new SessionConfig
{
    Model = "gpt-5",
    Streaming = true,
    AvailableTools =
    [
        // Add only reviewed read-tool identifiers for the deployed CLI version.
    ],
    SystemMessage = new SystemMessageConfig
    {
        Mode = SystemMessageMode.Append,
        Content = SovrintSystemAppend + Environment.NewLine +
                  "Operate in read-only mode."
    }
};
```

## Custom tools

Custom tools are application code. Wrap their handlers with application authorization and bounded audit recording before placing them in `SessionConfig.Tools`.

The wrapper should record only:

- session reference;
- tool name;
- invocation reference;
- decision;
- reason code;
- timestamp;
- evidence status.

It should not place arguments, results, credentials, prompts, or raw file contents into the audit event.

## Boundaries

- system-message append content is guidance, not a complete enforcement boundary;
- an empty tool allowlist is safer than inheriting an unknown tool surface;
- local application approval is not a global governance decision;
- an audit event is not EvidenceGrid acceptance;
- unknown or unsupported permission surfaces should fail closed.
