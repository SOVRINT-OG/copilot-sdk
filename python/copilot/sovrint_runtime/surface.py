"""Tool, agent, MCP, and skill surface constraints."""

from typing import List, Sequence

from ..types import SessionConfig
from .model import SovrintSecurityProfile


def intersect(left: Sequence[str], right: Sequence[str]) -> List[str]:
    allowed = set(right)
    return [value for value in left if value in allowed]


def merge(left: Sequence[str], right: Sequence[str]) -> List[str]:
    return list(dict.fromkeys([*left, *right]))


def constrain_surfaces(
    source: SessionConfig,
    result: SessionConfig,
    profile: SovrintSecurityProfile,
) -> None:
    """Project a profile onto SDK capability surfaces."""

    if profile.tool_surface_mode == "none":
        result["available_tools"] = []
    elif profile.tool_surface_mode == "allowlist":
        configured = source.get("available_tools")
        result["available_tools"] = (
            intersect(configured, profile.available_tools)
            if configured is not None
            else list(profile.available_tools)
        )

    result["excluded_tools"] = merge(
        source.get("excluded_tools", []), profile.excluded_tools
    )

    agents = source.get("custom_agents")
    if agents is not None and profile.tool_surface_mode != "inherit":
        constrained = []
        for agent_source in agents:
            agent = dict(agent_source)
            agent["tools"] = (
                []
                if profile.tool_surface_mode == "none"
                else intersect(
                    agent_source.get("tools") or profile.available_tools,
                    profile.available_tools,
                )
            )
            constrained.append(agent)
        result["custom_agents"] = constrained

    servers = source.get("mcp_servers")
    if servers is not None:
        allowed_servers = set(profile.allowed_mcp_servers)
        result["mcp_servers"] = {
            name: value for name, value in servers.items() if name in allowed_servers
        }

    directories = source.get("skill_directories")
    if directories is not None:
        result["skill_directories"] = intersect(
            directories, profile.allowed_skill_directories
        )
    result["disabled_skills"] = merge(
        source.get("disabled_skills", []), profile.disabled_skills
    )
