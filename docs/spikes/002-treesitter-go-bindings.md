# Spike-002: Tree-sitter Bindings in Go (CGO vs. WebAssembly / Pure Go)

- **Status:** `Pending`
- **Date:** `2026-11-21`
- **Timebox:** `0.5–1 day`
- **Informs / Blocks:** [ADR-003](../adr/003-treesitter-and-mcp.md), [ADR-001](../adr/001-go-control-plane.md), [Requirements](../requirements.md)

## Objective & Question

**Objective:** Prototype AST symbol extraction for a multi-language sample repository (Go, Python, TypeScript) from Go to generate a compact Repository Map.

**Core Question:** Is it preferable to compile Tree-sitter bindings with CGO in Go, or use WebAssembly (e.g., `wazero`-backed) / pure-Go wrappers to simplify cross-compilation and Docker image builds?

## Context

Graft uses Tree-sitter in the `ANALYZING` stage to build an in-memory repository map ([RFC-001 Section 3](../architecture.md#3-components-and-responsibilities)) so prompts stay compact without stuffing entire files. However, native Tree-sitter bindings require CGO and C/C++ toolchains per target platform, which can complicate Windows development, `CGO_ENABLED=0` static builds, and multi-arch Linux Docker images.

## Experiment Plan

1. Build two minimal Go PoCs that parse a medium-sized repo (~100–300 files across Go and Python) and extract top-level function/struct/class signatures:
   - **Variant A (CGO):** Official/community CGO Tree-sitter Go bindings.
   - **Variant B (WASM / Pure Go):** WASM-hosted Tree-sitter runtime (e.g., `wazero`) or pure-Go parser wrappers.
2. Test local builds on Windows and Linux Docker container builds (`Dockerfile` multi-stage build).
3. Measure cold build time, binary size, runtime indexing latency, memory usage, and grammar availability.

## Evaluation Criteria

- Successful extraction of accurate symbol maps (file path + line ranges + declarations) for at least Go and Python.
- Acceptable indexing latency for typical repositories (target `< 2s` for ~200 files).
- Reproducible build pipeline on both developer workstations (Windows/Linux) and CI/Docker Linux targets.

## Result

`Pending`

## Key Learnings

- `TBD` — Expected observations to capture:
  - Toolchain prerequisites (GCC/Clang/MSYS2 on Windows vs. Alpine/Debian Docker builds).
  - Grammar ecosystem maturity and ease of adding new languages.
  - Runtime latency and memory overhead comparison (CGO vs. WASM).

## Impact on Requirements & ADRs

- **Requirements ([requirements.md](../requirements.md)):** Will set build/portability constraints, AST indexing latency targets, and supported initial languages.
- **ADRs:** Will finalize the Tree-sitter implementation choice in [ADR-003](../adr/003-treesitter-and-mcp.md) and confirm zero-friction Go builds for [ADR-001](../adr/001-go-control-plane.md).

## Recommendation & Next Steps

- `TBD`
