# ADR-003: Tree-sitter AST Repository Mapping and MCP Tooling

- **Status:** `Proposed`
- **Date:** `2026-11-21`
- **Related Spikes:** [Spike 002: Tree-sitter Go Bindings](../spikes/002-treesitter-go-bindings.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Requirements](../requirements.md)

## Context

Passing entire source files or whole repositories into LLM prompts quickly exhausts context windows—especially on local models—inflates token costs, and degrades fault localization. Simultaneously, the agent needs a structured, auditable, and guardrail-enforced way to read files and apply unified diffs inside the sandbox.

## Options Considered

1. **Option A (`Tree-sitter AST Repo Map + MCP JSON-RPC Tools`):** Parse repository ASTs with Tree-sitter to build a compact in-memory repository map (symbols, signatures, file structure) and expose explicit operations (`read_file`, `apply_patch`, etc.) through the Model Context Protocol (MCP).
2. **Option B (`Full-File Context Stuffing + Ad-hoc Tool Schemas`):** Pass raw repository files or regex-grepped chunks directly into prompts and define custom non-standard tool-invocation parsers.

## Decision

We propose **Option A (`Tree-sitter AST Repo Map + MCP JSON-RPC Tools`)** because:

- **Context Efficiency:** Syntactic AST maps give the LLM high-signal architectural awareness of files and symbols before selectively reading targeted files, saving tokens and improving localization on local and cloud models alike.
- **Standardized Guardrail Boundary:** Exposing tools over MCP JSON-RPC centralizes path validation (blocking `.github/`, `.env`, and path traversal) and produces clean structured audit logs for PostgreSQL and SSE.

*Pending [Spike 002](../spikes/002-treesitter-go-bindings.md):* The exact Go Tree-sitter implementation (CGO bindings vs. WebAssembly / pure-Go wrappers) remains open until cross-compilation and performance benchmarks are measured.

## Consequences

- **Positive:** Smaller prompt context footprints, cleaner separation between reasoning and sandbox filesystem mutation, and standard MCP tooling interoperability.
- **Negative / Trade-offs:** Requires maintaining language grammars (Go, Python, TypeScript, etc.) and resolving CGO vs. WASM build complexity in Go.

## References

- [Spike 002: Tree-sitter Go Bindings](../spikes/002-treesitter-go-bindings.md)
- [RFC-001: Graft Architecture — Section 3](../architecture.md)
