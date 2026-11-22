# Benchmarks — Mini SWE-bench Evaluation (Local vs. Cloud)

| Field | Value |
| :--- | :--- |
| **Status** | `Template (Pending Implementation & Evaluation)` |
| **Date** | `2026-11-21` |
| **Related Docs** | [architecture.md](./architecture.md), [Spike 003](./spikes/003-ollama-tool-calling.md) |

> [!NOTE]
> This document is intentionally kept as a template. Metrics, test environment details, dataset composition, and local vs. cloud comparison results will be populated once the agent pipeline and evaluation harness are tested.

---

## 1. Evaluation Objective

`[State what configurations this benchmark compares and what decision it informs.]`

---

## 2. Metrics

| Metric | Definition | Unit |
| :--- | :--- | :--- |
| `[e.g., Pass@1, Latency, Token Usage, Cost]` | `[How it is measured]` | `[%, s, tokens, USD]` |

---

## 3. Environment & Dataset

| Parameter | Specification |
| :--- | :--- |
| **Host OS / Hardware / GPU** | `[TBD]` |
| **Docker Engine Version** | `[TBD]` |
| **Sandbox Limits** | `[TBD]` |
| **Dataset & Task Count** | `[TBD]` |
| **Pinned Models** | `[TBD]` |

---

## 4. Results

| Model / Provider | Deployment | Pass@1 | Median Latency | P95 Latency | Avg Iterations | Avg Tokens | Cost / Issue |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `[Model]` | `[Local / Cloud]` | `[TBD]` | `[TBD]` | `[TBD]` | `[TBD]` | `[TBD]` | `[TBD]` |

---

## 5. Failure Analysis & Takeaways

- `[Observed failure mode or takeaway backed by run logs]`
