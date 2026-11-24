# ADR-001: Go Modular Monolith for Control Plane and Worker Runtime

- **Status:** `Accepted`
- **Date:** `2026-11-23`
- **Related Spikes:** [Spike 002: Tree-sitter Go Bindings](../spikes/002-treesitter-go-bindings.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Requirements](../requirements.md)

## Context

Graft requires high-concurrency webhook ingestion, transactional PostgreSQL state tracking, Redis Streams orchestration, direct Docker Engine API lifecycle control, and real-time SSE log streaming. A primary early decision is whether to build Graft as a unified Go modular monolith or split the control plane (Go) from an agent runtime service written in Python (e.g., using LangChain).

## Options Considered

1. **Option A (`Go Modular Monolith`):** Build both the Control Plane HTTP API (`chi`), PostgreSQL persistence (`sqlc`), Redis Streams/PubSub workers, Docker SDK `SandboxManager`, MCP client/server, and LLM HTTP loop in a single Go codebase separated into clean internal modules.
2. **Option B (`Split Go Control Plane + Python/LangChain Agent Service`):** Keep webhook ingestion and Docker orchestration in Go while delegating LLM reasoning and tool loops to a separate Python microservice using LangChain.

## Decision

We adopt **Option A (`Go Modular Monolith`)** because:

- **Deterministic State & Transparency:** Avoiding heavy black-box agent frameworks gives us full control over every prompt, tool call, retry transition, and PostgreSQL audit-trail entry.
- **Native Concurrency & Systems Tooling:** Go integrates natively with the official Docker Engine SDK, Redis Streams, `sqlc`, and SSE streaming without cross-language IPC serialization overhead.
- **Single Deployment Footprint:** Fewer moving parts, simpler local development, and cleaner worker crash recovery (`reaper` + container UUID lifecycle) in early phases.
- **Zero-CGO Static Runtime Packaging Validated ([Spike 002](../spikes/002-treesitter-go-bindings.md)):** Using pure-Go Tree-sitter (`gotreesitter`) keeps development and production runtime builds at `CGO_ENABLED=0`, compiling without a C toolchain on Windows (`11.33s` `-a` on initial `18.33 MiB`; `18.34 MiB` final benchmark binary) and producing portable static Linux binaries (`18.14 MiB` / `19,020,978 B`) for Alpine, Debian, and `FROM scratch` (`14.21s`, `27.84 MiB` Docker Engine 29 `containerd` `.Size`). Running `go test -race` (required by `lefthook.yml` and [ADR-005](./005-github-app-token-manager.md)) still requires `CGO_ENABLED=1` with a compatible C toolchain on Windows (e.g., MinGW-w64 rather than `zig cc 0.17.1`, which failed TSan linkage) or execution in Linux/Docker/CI.

## Consequences

- **Positive:** Strong static typing, `CGO_ENABLED=0` cross-compilation and static container builds for runtime binaries (while running `-race` tests in Linux/CI or with a compatible C toolchain), low memory overhead per worker, unified observability, and full ownership of the Plan & Execute agent loop.
- **Negative / Trade-offs:** We implement our own lightweight agent loop and OpenAI-compatible client rather than reusing off-the-shelf Python agent ecosystems, and local Windows `-race` test runs still require a compatible C toolchain despite zero-CGO runtime builds.

## References

- [RFC-001: Graft Architecture — Sections 2 & 3](../architecture.md)
