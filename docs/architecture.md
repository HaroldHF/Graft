# RFC-001: Graft Architecture

| Field | Value |
| :--- | :--- |
| **System** | Graft — AI Agent Orchestrator for GitHub in Isolated Sandboxes |
| **Status** | `Draft (In Research)` |
| **Author** | `<Author / GitHub Handle>` |
| **Date** | `2026-11-21` |

---

## 1. Problem and Objective

### 1.1 Context

Automated issue resolution using Large Language Models (LLMs) introduces two major risks when executed without adequate isolation and control:

- **Security Risks:** Running arbitrary LLM-generated code or scripts directly on the host machine can lead to secret exfiltration or malicious execution (e.g., via Prompt Injection).
- **Lack of Determinism and Visibility:** Many agent frameworks operate as opaque black boxes without transactional state persistence or step-by-step auditability.

### 1.2 Purpose of Graft

**Graft** is a distributed platform written in Go that automates the code-patching lifecycle triggered by GitHub events (such as issue labeling or CI failures). It executes every task inside an ephemeral container provisioned via the Docker Engine API, isolating network access and compute resources. Candidate patches are validated against local unit tests before opening a verified Pull Request.

### 1.3 Scope

#### In-Scope

- Idempotent ingestion of webhooks signed by GitHub Apps (`HMAC-SHA256`).
- Persistent job state machine in PostgreSQL paired with a Redis task queue.
- Sandbox manager in Go using the Docker Engine API directly (network isolation, CPU/RAM/PID limits, and ephemeral volumes).
- Provider-agnostic autonomous agent engine (supporting local Ollama and cloud APIs via an OpenAI-compatible client).
- Repository indexing via AST parsing with Tree-sitter and structured tool exposure via the Model Context Protocol (MCP).
- Real-time log and telemetry streaming to a web dashboard via Server-Sent Events (SSE).
- Benchmark evaluation suite (mini SWE-bench) to measure `Pass@1`, latency, and token usage.

#### Out-of-Scope

- Training or fine-tuning language models.
- Multi-cluster Kubernetes orchestration (focused instead on a high-performance Docker runtime).
- Real-time multi-user collaborative editing inside the sandbox.

---

## 2. High-Level Architecture

The system is structured as a modular monolith in Go decoupled by events and task queues.

```mermaid
flowchart TD
    GH["GitHub App / Webhooks"] -->|"POST HMAC-SHA256"| API["Graft Control Plane - HTTP API"]
    API -->|"Persist Job & HMAC Check"| DB[("PostgreSQL")]
    API -->|"Enqueue Task"| REDIS[("Redis Streams")]

    subgraph worker_cluster["Worker Cluster / Job Execution"]
        W["Graft Worker Node"] -->|"Dequeue Task"| REDIS
        W -->|"State Transition"| DB
        W -->|"Provision"| DOCKER["Docker Engine API"]
        DOCKER -->|"Ephemeral Container"| SBX["Sandbox Environment"]

        subgraph sandbox_container["Sandbox Isolated Container"]
            SBX -->|"Repo Code + AST"| TS["Tree-sitter Engine"]
            SBX -->|"Isolated Execution"| TEST["Test Runner (go test / pytest)"]
        end

        W -->|"MCP Protocol"| AGENT["Agent Runtime Loop"]
        AGENT -->|"v1/chat/completions"| LLM["LLM Provider (Ollama or Cloud API)"]
    end

    W -->|"SSE Events / Stream Logs"| REDIS_PUB["Redis Pub/Sub"]
    REDIS_PUB -->|"Push Real-Time Data"| SSE["SSE Gateway"]
    SSE -->|"Stream Events"| DASH["React Dashboard"]
    W -->|"Push Branch & Open PR"| GH
```

---

## 3. Components and Responsibilities

| Component | Technology | Primary Responsibility | Interface / Contract |
| :--- | :--- | :--- | :--- |
| **Control Plane API** | Go (`chi`) | Webhook reception, HMAC signature verification, idempotency control, and job status queries | `HTTP POST /api/v1/webhooks/github` |
| **Persistence** | PostgreSQL (`sqlc`) | Persist job state machine, step-by-step history (audit trail), and repository configurations | SQL tables + `JSONB` columns |
| **Job Queue & Bus** | Redis | Decouple asynchronous tasks between API and Workers, plus a Pub/Sub channel for live events | Redis Streams (`graft:jobs`) & Pub/Sub (`graft:telemetry`) |
| **Sandbox Manager** | Go + Docker SDK | Full container lifecycle: creation, clean workspace provisioning, `NetworkMode: "none"` isolation, cgroup limits, and forced teardown | Go interface `SandboxManager` |
| **AST Indexer** | Tree-sitter (Go) | Generate syntactic repository maps to reduce prompt context window usage | In-memory Repository Map |
| **Agent Runtime** | Go + MCP | Reasoning loop (Plan & Execute), tool-call validation, and security guardrails | MCP JSON-RPC Client/Server |
| **SSE Gateway** | Go | Relay container and LLM logs in real time to the frontend | `HTTP GET /api/v1/jobs/{id}/stream` |
| **Dashboard** | React + TS + Tailwind | Display current job status, `git diff` viewer, and live sandbox console | SPA consuming SSE & REST APIs |

---

## 4. Job State Machine

Each **Job** represents the resolution attempt for a specific issue. State transitions are deterministic and persisted transactionally in PostgreSQL.

```mermaid
stateDiagram-v2
    [*] --> QUEUED: Webhook received and validated
    QUEUED --> PROVISIONING: Worker claims job
    PROVISIONING --> ANALYZING: Container ready and repo cloned
    ANALYZING --> PATCHING: AST map and diagnosis ready
    PATCHING --> TESTING: Patch applied in sandbox

    TESTING --> PR_OPENED: Tests passed
    TESTING --> PATCHING: Tests failed (retries <= N)

    TESTING --> FAILED: Retries exhausted or critical error
    PROVISIONING --> FAILED: Docker error or network timeout
    ANALYZING --> FAILED: AST parse error or LLM unreachable

    QUEUED --> TIMEOUT: Max queue wait time exceeded
    PATCHING --> TIMEOUT: Max execution time exceeded

    PR_OPENED --> [*]
    FAILED --> [*]
    TIMEOUT --> [*]
```

### Transition and Recovery Rules

- **Idempotency:** If a webhook arrives with an `X-GitHub-Delivery` key that has already been processed or is currently running, the system ignores the duplicate request.
- **Worker Crash Recovery:** If a worker disconnects without updating state, a background **reaper** process detects jobs stuck in `PROVISIONING` or `PATCHING` for more than **10 minutes**, destroys orphaned containers by UUID, and re-enqueues the task.
- **Retry Limit:** The `TESTING -> PATCHING` transition allows at most `N = 5` iterations. Once exceeded, the job transitions to `FAILED`.

---

## 5. Sequence Diagram (Main Flow)

```mermaid
sequenceDiagram
    autonumber
    actor User as User (GitHub)
    participant GH as GitHub API
    participant CP as Control Plane (Go)
    participant DB as PostgreSQL
    participant Q as Redis Queue
    participant W as Worker Node
    participant DK as Docker Engine API
    participant SBX as Sandbox Container
    participant LLM as LLM (Ollama/Cloud)

    User->>GH: Apply 'graft-fix' label to an Issue
    GH->>CP: POST /api/v1/webhooks/github (HMAC Header)
    CP->>CP: Validate HMAC-SHA256 signature
    CP->>DB: Register Job (State: QUEUED)
    CP->>Q: Enqueue Job ID
    CP-->>GH: 202 Accepted

    Q->>W: Dequeue Job Task
    W->>DB: Update state (PROVISIONING)
    W->>GH: Clone repo on host (hooks disabled, ephemeral token)
    W->>DK: Create Container (NetworkMode none, 1 vCPU, 1 GiB RAM, PidsLimit)
    DK-->>SBX: Container initialized (offline from creation)
    W->>SBX: Provision clean workspace (host .git excluded)
    W->>DB: Update state (ANALYZING)

    loop Reasoning Loop (Max N iterations)
        W->>SBX: Run Tree-sitter (Repo Map)
        W->>LLM: Prompt + AST Context + Issue
        LLM-->>W: Tool Call (read_file / apply_patch)
        W->>SBX: Apply unified diff
        W->>DB: Update state (TESTING)
        W->>SBX: Exec 'go test' / 'pytest' (No network)
        SBX-->>W: Exit Code (0 = Pass, >0 = Fail)
    end

    alt Tests Passed
        W->>SBX: Extract unified diff
        W->>W: Validate & apply diff to host clone
        W->>GH: Git push branch from host & open PR
        W->>DB: Update state (PR_OPENED)
    else Tests Failed / Max Iterations Exceeded
        W->>GH: Post comment with failure diagnostics
        W->>DB: Update state (FAILED)
    end

    W->>DK: Force-Remove Container & Temp Workspace
```

> [!NOTE]
> **Sandbox Isolation Decided ([ADR-002](./adr/002-docker-sdk-sandbox.md) `Accepted`):** Per [Spike 001](./spikes/001-docker-network-isolation.md), sandboxes always start with `NetworkMode: "none"` and never receive GitHub credentials or writable host `.git/` directories; `git clone` and `git push` run exclusively on the worker host. See [Threat Model](./threat-model.md).

---

## 6. Open Questions and Uncertainties (Spike Inputs)

The following design questions remain to be resolved through technical spikes before freezing [requirements.md](./requirements.md):

1. **[Spike 002 — Tree-sitter Go Bindings](./spikes/002-treesitter-go-bindings.md):** Is it preferable to compile Tree-sitter bindings with CGO in Go or use WebAssembly / pure-Go wrappers to simplify cross-compilation?
2. **[Spike 003 — Ollama Tool Calling](./spikes/003-ollama-tool-calling.md):** What JSON tool-calling formatting success rate do Qwen 2.5 Coder 7B vs. 14B achieve when running locally under Ollama?

> [!NOTE]
> **Resolved Pre-Implementation Decisions:**
> - **Docker SDK & Network Isolation:** Decided in [ADR-002](./adr/002-docker-sdk-sandbox.md) (`Accepted`) via [Spike 001](./spikes/001-docker-network-isolation.md) — `github.com/moby/moby/client`, `NetworkMode: "none"` from creation, host-side Git with isolated host `.git/`, abstracted workspace transport, and mandatory forced `ContainerRemove` on timeout.
> - **GitHub App Installation Token Caching:** Decided in [ADR-005](./adr/005-github-app-token-manager.md) (`Accepted`) — minting and refreshing 1-hour Installation Access Tokens uses a per-worker in-memory `TokenManager` with `singleflight` deduplication (validated directly via Phase 1 `go test -race` unit tests rather than a disposable spike).

---

## 7. Related Documents

- [Documentation Hub](./README.md)
- [Technical Spikes](./spikes/README.md)
- [Architecture Decision Records (ADRs)](./adr/README.md)
- [Requirements](./requirements.md)
- [Threat Model](./threat-model.md)
- [Benchmarks](./benchmarks.md)
