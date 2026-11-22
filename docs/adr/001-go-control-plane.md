# ADR-001: Go Modular Monolith for Control Plane and Worker Runtime

- **Status:** `Proposed`
- **Date:** `2026-11-21`
- **Related Spikes:** None (validated through [architecture.md](../architecture.md) design review; build/packaging details depend on [Spike 002](../spikes/002-treesitter-go-bindings.md))
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Requirements](../requirements.md)

## Context

Graft requires high-concurrency webhook ingestion, transactional PostgreSQL state tracking, Redis Streams orchestration, direct Docker Engine API lifecycle control, and real-time SSE log streaming. A primary early decision is whether to build Graft as a unified Go modular monolith or split the control plane (Go) from an agent runtime service written in Python (e.g., using LangChain).

## Options Considered

1. **Option A (`Go Modular Monolith`):** Build both the Control Plane HTTP API (`chi`), PostgreSQL persistence (`sqlc`), Redis Streams/PubSub workers, Docker SDK `SandboxManager`, MCP client/server, and LLM HTTP loop in a single Go codebase separated into clean internal modules.
2. **Option B (`Split Go Control Plane + Python/LangChain Agent Service`):** Keep webhook ingestion and Docker orchestration in Go while delegating LLM reasoning and tool loops to a separate Python microservice using LangChain.

## Decision

We propose **Option A (`Go Modular Monolith`)** because:

- **Deterministic State & Transparency:** Avoiding heavy black-box agent frameworks gives us full control over every prompt, tool call, retry transition, and PostgreSQL audit-trail entry.
- **Native Concurrency & Systems Tooling:** Go integrates natively with the official Docker Engine SDK, Redis Streams, `sqlc`, and SSE streaming without cross-language IPC serialization overhead.
- **Single Deployment Footprint:** Fewer moving parts, simpler local development, and cleaner worker crash recovery (`reaper` + container UUID lifecycle) in early phases.

*Final acceptance is pending review of Tree-sitter packaging in [Spike 002](../spikes/002-treesitter-go-bindings.md) to ensure no unexpected CGO build friction reverses this trade-off.*

## Consequences

- **Positive:** Strong static typing, low memory overhead per worker, unified observability, and full ownership of the Plan & Execute agent loop.
- **Negative / Trade-offs:** We implement our own lightweight agent loop and OpenAI-compatible client rather than reusing off-the-shelf Python agent ecosystems, and Tree-sitter Go bindings must be evaluated carefully for cross-compilation ([Spike 002](../spikes/002-treesitter-go-bindings.md)).

## References

- [RFC-001: Graft Architecture — Sections 2 & 3](../architecture.md)
