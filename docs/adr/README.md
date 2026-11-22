# Architecture Decision Records (ADRs)

This directory captures key architectural decisions for **Graft**, documenting the context, evaluated alternatives, chosen approach, and trade-offs.

## Status Lifecycle

- **`Proposed`:** Initial draft aligned with [RFC-001](../architecture.md), awaiting spike validation or team sign-off.
- **`Accepted`:** Validated and adopted for implementation.
- **`Superseded`:** Replaced by a newer ADR (link to the successor ADR rather than deleting history).
- **`Deprecated`:** No longer applicable due to scope or system changes.

## Creating a New ADR

1. Copy [template.md](./template.md) to `docs/adr/NNN-short-slug.md`.
2. Fill in `Context`, `Options Considered`, `Decision`, and `Consequences`.
3. Add an entry to the index table below.

## ADR Index

| ID | Title | Status | Date | Blocking / Related Spike | Summary |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ADR-001 | [Go Modular Monolith Control Plane](./001-go-control-plane.md) | `Proposed` | `2026-11-21` | None | Use a Go modular monolith (`chi`, `sqlc`, Docker SDK) over splitting into a Python/LangChain service |
| ADR-002 | [Docker Engine SDK Sandbox Manager](./002-docker-sdk-sandbox.md) | `Proposed` | `2026-11-21` | [Spike 001](../spikes/001-docker-network-isolation.md) | Manage ephemeral isolated containers directly via the Go Docker Engine API instead of shell commands or persistent containers |
| ADR-003 | [Tree-sitter AST Mapping & MCP Tools](./003-treesitter-and-mcp.md) | `Proposed` | `2026-11-21` | [Spike 002](../spikes/002-treesitter-go-bindings.md) | Index repositories with Tree-sitter and expose tools over MCP JSON-RPC instead of dumping raw files into prompts |
| ADR-004 | [OpenAI-Compatible LLM Provider Abstraction](./004-llm-provider-abstraction.md) | `Proposed` | `2026-11-21` | [Spike 003](../spikes/003-ollama-tool-calling.md) | Target the standard OpenAI `v1/chat/completions` contract to switch seamlessly between local Ollama and cloud APIs |
| ADR-005 | [GitHub App Installation Token Manager](./005-github-app-token-manager.md) | `Accepted` | `2026-11-21` | None | Use a per-worker in-memory cache with `singleflight` refresh and `401` retry, verified via Phase 1 `go test -race` unit tests |
