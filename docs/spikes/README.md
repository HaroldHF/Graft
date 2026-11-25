# Technical Spikes Log

This directory tracks timeboxed proofs of concept (PoCs) conducted **before** writing production code or freezing [requirements.md](../requirements.md). Each spike resolves an architectural uncertainty identified in [RFC-001 Section 6](../architecture.md#6-open-questions-and-uncertainties-spike-inputs).

## Spike Workflow

1. Review the open spike brief and its timebox.
2. Build a minimal disposable PoC (separate from production packages) to test the hypothesis.
3. Record `Result`, `Key Learnings`, `Impact on Requirements & ADRs`, and `Recommendation & Next Steps` in the spike file, and synthesize cross-spike takeaways in [learnings.md](./learnings.md).
4. Update the status below (`Pending` -> `In Progress` -> `Completed` / `Abandoned`) and promote confirmed choices into the corresponding ADR and [requirements.md](../requirements.md).

To add a new spike, copy [template.md](./template.md) to `docs/spikes/NNN-short-slug.md` and register it in the table below.

## Spike Index

| ID | Title | Status | Date | Timebox | Core Question | Informs / Unblocks |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| Spike-001 | [Docker Network Isolation & SDK Lifecycle](./001-docker-network-isolation.md) | `Completed` | `2026-11-22` | `0.5–1 day` | Dynamic network disconnect vs. `NetworkMode: "none"` + socket permissions & command timeouts | [ADR-002](../adr/002-docker-sdk-sandbox.md), [Threat Model](../threat-model.md), [Requirements](../requirements.md) |
| Spike-002 | [Tree-sitter Go Bindings (CGO vs. WASM/Pure Go)](./002-treesitter-go-bindings.md) | `Completed` | `2026-11-23` | `0.5–1 day` | CGO bindings vs. WASM/pure-Go wrappers for cross-compilation and AST performance | [ADR-003](../adr/003-treesitter-and-mcp.md), [ADR-001](../adr/001-go-control-plane.md), [Requirements](../requirements.md) |
| Spike-003 | [Ollama Local Tool Calling Reliability](./003-ollama-tool-calling.md) | `Completed` | `2026-11-25` | `1 day` | JSON tool-calling accuracy, `num_ctx` scaling, and edit format (`apply_patch` vs. `search_replace`) for Qwen 2.5 Coder 7B vs. 14B under Ollama | [ADR-004](../adr/004-llm-provider-abstraction.md), [ADR-003](../adr/003-treesitter-and-mcp.md), [Benchmarks](../benchmarks.md), [Requirements](../requirements.md), [Threat Model](../threat-model.md) |
