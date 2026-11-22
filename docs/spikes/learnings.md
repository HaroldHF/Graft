# Consolidated Spike Learnings

| Field | Value |
| :--- | :--- |
| **Status** | `Template (Pending Spikes)` |
| **Date** | `2026-11-21` |
| **Related Docs** | [Spikes Index](./README.md), [Requirements](../requirements.md), [Threat Model](../threat-model.md), [ADRs](../adr/README.md) |

> [!NOTE]
> Use this file to synthesize empirical findings across [Spikes 001–003](./README.md) into concrete engineering rules before freezing [requirements.md](../requirements.md) and populating [threat-model.md](../threat-model.md). Record only verified measurements and observed behavior.

---

## 1. Consolidated Spike Summary

| Spike | Result | Key Measured Numbers | Chosen Approach | Docs Updated |
| :--- | :--- | :--- | :--- | :--- |
| [Spike-001: Docker Network Isolation](./001-docker-network-isolation.md) | `Pending` | `TBD` | `TBD` | `TBD` |
| [Spike-002: Tree-sitter Go Bindings](./002-treesitter-go-bindings.md) | `Pending` | `TBD` | `TBD` | `TBD` |
| [Spike-003: Ollama Tool Calling](./003-ollama-tool-calling.md) | `Pending` | `TBD` | `TBD` | `TBD` |

---

## 2. Platform & Environment Notes (Windows vs. Linux / Docker)

- `TBD` — Record Docker socket paths/permissions, CGO/WASM toolchain differences, and container volume caveats observed across host OS targets.

---

## 3. Evidence for Requirements

- `TBD` — Translate measured startup latencies, AST indexing times, execution timeouts, and tool-call thresholds into verifiable `FR`/`NFR` rows in [requirements.md](../requirements.md).

---

## 4. Evidence for Threat Model

- `TBD` — Record verified network isolation checks, credential-boundary decisions (e.g., host-side `git` vs. sandbox), and resource-cap termination behavior to populate [threat-model.md](../threat-model.md).

---

## 5. Dead Ends, Surprises & Rejected Approaches

- `TBD` — Document approaches tested and discarded during spikes so future contributors do not repeat failed experiments.
