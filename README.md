# Graft

> **AI Agent Orchestrator for GitHub in Isolated Docker Sandboxes**

**Status:** Pre-implementation (Architecture & Technical Spikes Phase)

## Overview

**Graft** is a distributed platform written in Go that automates issue-to-Pull-Request code patching from GitHub events (such as applying a `graft-fix` label to an issue or responding to CI failures).

Rather than running LLM-generated code directly on the host or treating agents as opaque black boxes, Graft executes every task inside an ephemeral container provisioned via the Docker Engine API—with strict network, CPU, memory, and PID isolation—validating candidate patches against local unit tests before opening a verified Pull Request.

## Core Capabilities (Planned)

- **Idempotent Webhook Ingestion:** Verifies GitHub App webhooks via `HMAC-SHA256` and deduplicates deliveries by `X-GitHub-Delivery` in PostgreSQL.
- **Deterministic Job State Machine:** Persists job state transitions and step-by-step audit trails in PostgreSQL, decoupled from worker execution via Redis Streams.
- **Isolated Sandbox Execution:** Manages ephemeral containers directly through the Go Docker Engine SDK (`cgroups`, PID limits, ephemeral volumes, network isolation).
- **AST-Guided Agent Runtime & MCP:** Indexes repository structure with Tree-sitter to build compact repository maps and exposes structured tools over Model Context Protocol (MCP) JSON-RPC.
- **Provider-Agnostic LLM Client:** Uses an OpenAI-compatible `v1/chat/completions` interface to switch between local Ollama models and cloud LLM APIs.
- **Real-Time Telemetry & Dashboard:** Streams sandbox and LLM logs via Redis Pub/Sub and Server-Sent Events (SSE) to a React dashboard.
- **Evaluation Harness:** Benchmarks `Pass@1`, latency, and token usage on a curated mini SWE-bench suite.

## Tech Stack

| Layer | Technology | Role |
| :--- | :--- | :--- |
| Control Plane & Workers | Go (`chi`), Redis Streams | HTTP webhook ingestion, job orchestration, worker execution |
| Persistence | PostgreSQL (`sqlc`) | Job state machine, audit trail (JSONB), repository settings |
| Sandbox Runtime | Docker Engine API (Go SDK) | Ephemeral isolated containers, resource & network limits |
| Code Intelligence | Tree-sitter (Go), MCP | AST repository mapping and structured agent tool calling |
| LLM Integration | OpenAI-compatible HTTP (`Ollama` / Cloud APIs) | Autonomous reasoning and patch generation loop |
| Real-Time UI | Go SSE Gateway, React + TypeScript + Tailwind | Live console logs, job status, and `git diff` viewer |

## Documentation Map

Start with the [Documentation Hub](./docs/README.md) for the full pre-implementation workflow, or jump directly to a specific artifact:

| Document | Purpose | Status |
| :--- | :--- | :--- |
| [Documentation Hub](./docs/README.md) | Engineering workflow, index of all docs, and maintenance rules | Active |
| [Architecture RFC-001](./docs/architecture.md) | Problem statement, scope, high-level topology, state machine, sequence diagram, and spike inputs | `Draft (In Research)` |
| [Technical Spikes](./docs/spikes/README.md) | Timeboxed PoCs to de-risk open design questions before coding | `3 Pending` |
| [Spike Learnings](./docs/spikes/learnings.md) | Consolidated empirical findings and platform notes across spikes | `Template (Pending Spikes)` |
| [Architecture Decision Records (ADRs)](./docs/adr/README.md) | Key technical choices and trade-offs | `1 Accepted, 4 Proposed` |
| [Requirements](./docs/requirements.md) | Functional and non-functional requirements (frozen post-spikes) | `Template (Pending Evidence)` |
| [Threat Model](./docs/threat-model.md) | Trust boundaries, attack vectors, guardrails, and test verification | `Template (Pending Evidence)` |
| [Benchmarks](./docs/benchmarks.md) | Mini SWE-bench metrics, harness setup, and local vs. cloud comparison | `Template (Pending Evidence)` |

## High-Level Roadmap

1. **Phase 0 — Architecture & Technical Spikes (Current):** Complete [Spikes 001–003](./docs/spikes/README.md), consolidate findings in [learnings.md](./docs/spikes/learnings.md), finalize remaining [ADRs 001–004](./docs/adr/README.md), resolve open security items in the [Threat Model](./docs/threat-model.md), and freeze [Requirements](./docs/requirements.md).
2. **Phase 1 — Control Plane & Sandbox Core:** Implement GitHub webhook ingestion, PostgreSQL schema (`sqlc`), Redis Streams worker loop, reaper crash recovery, and the Go `SandboxManager`.
3. **Phase 2 — Agent Runtime, Tree-sitter & MCP:** Build the AST repository indexer, MCP tool server/client, diff validator guardrails, and OpenAI-compatible LLM loop.
4. **Phase 3 — SSE Telemetry & Dashboard:** Add Redis Pub/Sub log streaming, SSE gateway, and the React/Tailwind live job & diff UI.
5. **Phase 4 — Mini SWE-bench Evaluation:** Run controlled evaluations across local Ollama and cloud models and record findings in [Benchmarks](./docs/benchmarks.md).
