# ADR-003: Tree-sitter AST Repository Mapping and MCP Tooling

- **Status:** `Accepted`
- **Date:** `2026-11-23`
- **Related Spikes:** [Spike 002: Tree-sitter Go Bindings](../spikes/002-treesitter-go-bindings.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Requirements](../requirements.md)

## Context

Passing entire source files or whole repositories into LLM prompts quickly exhausts context windows—especially on local models—inflates token costs, and degrades fault localization. Simultaneously, the agent needs a structured, auditable, and guardrail-enforced way to read files and apply unified diffs inside the sandbox, while keeping Go builds portable (`CGO_ENABLED=0`) across Windows developer workstations and Linux containers without exposing worker processes to C parser memory-safety bugs.

## Options Considered

1. **Option A (`Pure Go Tree-sitter AST Repo Map + MCP JSON-RPC Tools`):** Parse repository ASTs using the pure-Go Tree-sitter runtime (`github.com/odvcencio/gotreesitter`, `CGO_ENABLED=0`) behind an `ASTIndexer` interface to build a compact in-memory repository map (symbols, signatures, file structure) and expose explicit operations (`read_file`, `apply_patch`, etc.) through the Model Context Protocol (MCP).
2. **Option B (`Official CGO Tree-sitter Bindings`):** Wrap C Tree-sitter via `github.com/tree-sitter/go-tree-sitter` and per-language CGO grammar bindings.
3. **Option C (`WebAssembly Tree-sitter via wazero`):** Use pre-packaged (`github.com/malivvan/tree-sitter`) or custom-compiled WASM modules executed via `wazero`.
4. **Option D (`Full-File Context Stuffing + Ad-hoc Tool Schemas`):** Pass raw repository files or regex-grepped chunks directly into prompts and define custom non-standard tool-invocation parsers.

## Decision

We adopt **Option A (`Pure Go Tree-sitter AST Repo Map + MCP JSON-RPC Tools`)**—with the AST runtime empirically validated in [Spike 002](../spikes/002-treesitter-go-bindings.md), while the MCP JSON-RPC tool contract and schemas remain adopted by design pending evaluation in [Spike 003](../spikes/003-ollama-tool-calling.md) and Phase 2 implementation:

- **Pure-Go AST Runtime (`CGO_ENABLED=0`):** Use `github.com/odvcencio/gotreesitter` (`v0.58.0` tested) importing only standalone required grammar packages (`grammars/go`, `grammars/python`, `grammars/typescript`, `grammars/tsx`, etc.) behind a focused Go `ASTIndexer` interface. This compiles cleanly without a C/C++ toolchain on Windows (`11.33s` `-a` on initial `18.33 MiB`; `18.34 MiB` final benchmark binary) and builds static `FROM scratch` Linux containers (`14.21s`, `18.14 MiB` static binary / `27.84 MiB` Docker Engine 29 `containerd` `.Size`), indexing `1,042` TS/TSX files in `1.39s` median (`69.69 MiB` `VmHWM`) under `1 vCPU / 1 GiB` Linux with `2,899/2,899` symbol parity against official C Tree-sitter.
- **Safety & Lifecycle Guardrails:** Enforce **pre-parse per-file byte limits** (e.g., skipping files `> 512 KiB`–`1 MiB` before `Parse`) and **worker concurrency/memory caps** as primary Worker Host OOM defenses—because [Spike 002](../spikes/002-treesitter-go-bindings.md) probes on `minified_2mb.ts` (`2.85 MiB`) showed `gotreesitter` pre-allocates `165`–`171 MiB` `HeapInuse` (`110`–`123 MiB` Peak RSS) even under a `1ms` timeout (`22.44ms`) before checking `SetTimeoutMicros`, and `SetMemoryBudgetBytes` tracks internal arenas rather than OS RSS (`196.27 MiB` RSS under `16 MiB` budget; `296.68 MiB` RSS over `2.05s` under `64 MiB` budget). Pair those primary caps with secondary `SetTimeoutMicros` + `SetMemoryBudgetBytes` (`ParseStrict`) controls, per-goroutine parsers (`ParserPool`), `tree.RootNode() == nil` guards on empty files, pre-`Release()` metadata extraction, and Repo Map token-budget scoping (`~37.2k`–`62.5k` full Pure Go tokens on `476`–`1,042` file repos).
- **Standardized MCP Guardrail Boundary:** Expose sandbox file/patch operations over MCP JSON-RPC (to be validated in [Spike 003](../spikes/003-ollama-tool-calling.md) and implemented in Phase 2) to centralize path validation (blocking `.github/`, `.env*`, and path traversal) and produce structured audit logs for PostgreSQL and SSE.
- **Rejected Alternatives:**
  - *Option B (CGO):* Requires per-target C compilers (breaking `CGO_ENABLED=0` dev/production builds), ties binaries to `glibc` vs. `musl` (`debian` CGO binary failed on `alpine:3.21` with `exit=127`), executes C parsers inside privileged worker memory (opaque to `-race`), requires explicit `.Close()`, and panics (`nil pointer dereference`) in deprecated `ParseCtx` when `SetCancellationFlag` is omitted (though non-deprecated `ParseWithOptions` `ProgressCallback` cancels cleanly at `1.74ms` / `17.19 MiB` on `1ms` timeout).
  - *Option C (WASM):* `malivvan/tree-sitter v0.0.1` supports only C/C++; a custom `zig cc` WASI reactor required maintaining a C shim, destroys the active WASM module instance on timeout under `WithCloseOnContextDone(true)`, and reached `814.94 MiB` `VmHWM` on Linux (`959.91 MiB` on Windows) on a `2.85 MiB` minified file due to non-shrinking linear memory.
  - *Option D (Context Stuffing):* Exhausts local context windows and bypasses structured auditability.

## Consequences

- **Positive:** Zero-CGO (`CGO_ENABLED=0`) static builds across Windows and Linux, memory-safe parsing of untrusted repositories in Go, `go test -race` visibility into parser internals (on Linux/CI or compatible C toolchains; `-race` itself still requires `CGO_ENABLED=1`), secondary cooperative parse timeouts/memory budgets, smaller prompt footprints, and standardized MCP tool boundaries.
- **Negative / Trade-offs:**
  - Larger Linux static binary (`18.14 MiB` vs. CGO's `6.82`–`6.83 MiB`) and higher steady RSS (`~70–75 MiB` `VmHWM` vs. `~12–15 MiB` CGO) when indexing `500`–`1,000+` files.
  - Slower indexing on mixed Go/TS/TSX `Brinco` (`942.71ms` median vs. CGO `356.19ms` = `2.65x` slower total / `2.51x` slower `index_ms`, and slower than WASM's `644.24ms`, though still under the `< 2s` target).
  - Pre-1.0 third-party dependency risk (`gotreesitter v0.58.0`, MIT) maintaining its own grammar blobs and external-scanner VM (producing `1/1,042` recoverable syntax false positive in `RecruitmentBoard.tsx` `L271–L273` where official C parsed cleanly, though symbol parity was `100%`), making `ASTIndexer` interface isolation mandatory.
  - Pre-allocation proportional to file size before timeout checks requires enforcing strict pre-parse per-file size caps on the Worker Host.

## References

- [Spike 002: Tree-sitter Go Bindings](../spikes/002-treesitter-go-bindings.md)
- [RFC-001: Graft Architecture — Section 3](../architecture.md)
