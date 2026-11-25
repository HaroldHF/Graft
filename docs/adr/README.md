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
| ADR-001 | [Go Modular Monolith Control Plane](./001-go-control-plane.md) | `Accepted` | `2026-11-23` | [Spike 002](../spikes/002-treesitter-go-bindings.md) | Use a `CGO_ENABLED=0` Go modular monolith (`chi`, `sqlc`, Docker SDK, pure-Go AST) over splitting into a Python/LangChain service |
| ADR-002 | [Docker Engine SDK Sandbox Manager](./002-docker-sdk-sandbox.md) | `Accepted` | `2026-11-22` | [Spike 001](../spikes/001-docker-network-isolation.md) | Manage ephemeral `NetworkMode: "none"` containers via `github.com/moby/moby/client` with host-side Git instead of CLI shell-outs or dynamic network toggles |
| ADR-003 | [Tree-sitter AST Mapping & MCP Tools](./003-treesitter-and-mcp.md) | `Accepted` | `2026-11-25` | [Spike 002](../spikes/002-treesitter-go-bindings.md), [Spike 003](../spikes/003-ollama-tool-calling.md) | Index repositories with pure-Go `gotreesitter` behind `ASTIndexer` and expose guarded `search_replace` / discovery tools over MCP JSON-RPC |
| ADR-004 | [LLM Provider Abstraction (`LLMClient`)](./004-llm-provider-abstraction.md) | `Accepted` | `2026-11-25` | [Spike 003](../spikes/003-ollama-tool-calling.md) | Expose one Go `LLMClient` interface over cloud `v1/chat/completions` and Ollama `/api/chat` (`options.num_ctx`), with `content` JSON fallback + sanitizer and 1-turn repair |
| ADR-005 | [GitHub App Installation Token Manager](./005-github-app-token-manager.md) | `Accepted` | `2026-11-21` | None | Use a per-worker in-memory cache with `singleflight` refresh and `401` retry, verified via Phase 1 `go test -race` unit tests |
