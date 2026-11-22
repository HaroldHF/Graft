# Requirements Specification

| Field | Value |
| :--- | :--- |
| **Status** | `Template (Pending Spikes)` |
| **Date** | `2026-11-21` |
| **Source Architecture** | [RFC-001: Graft Architecture](./architecture.md) |
| **Blocking Spikes** | [Technical Spikes Log](./spikes/README.md) |

> [!NOTE]
> This document is intentionally kept as a template until [Spikes 001–003](./spikes/README.md) and prototype tests produce empirical evidence. Add requirements here only after validating feasibility, timeouts, and security boundaries.

---

## 1. Scope

### In-Scope

- `[TBD — Confirm scope items validated from RFC-001]`

### Out-of-Scope

- `[TBD — Confirm explicit exclusions]`

---

## 2. Functional Requirements (FR)

| ID | Subsystem | Requirement | Acceptance Criteria | Evidence / Spike |
| :--- | :--- | :--- | :--- | :--- |
| `FR-01` | `[Component]` | `[Expected behavior]` | `[Testable condition]` | `[Spike / ADR]` |

---

## 3. Non-Functional Requirements (NFR)

| ID | Category | Measurable Target | Verification Method | Evidence / Spike |
| :--- | :--- | :--- | :--- | :--- |
| `NFR-01` | `[Security / Latency / Limits / Build]` | `[Measured threshold]` | `[Test / Benchmark]` | `[Spike / ADR]` |

---

## 4. Constraints & Assumptions

- `[Constraint or runtime assumption verified during spikes]`

---

## 5. Freeze Criteria

- [ ] Spikes `001`–`003` completed with recorded findings and consolidated in [spikes/learnings.md](./spikes/learnings.md)
- [ ] Remaining core ADRs (`001`–`004`) promoted to `Accepted` ([ADR-005](./adr/005-github-app-token-manager.md) already `Accepted`)
- [ ] Threat model mitigations backed by concrete verification tests
