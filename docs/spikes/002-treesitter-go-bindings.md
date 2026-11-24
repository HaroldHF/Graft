# Spike-002: Tree-sitter Bindings in Go (CGO vs. WebAssembly / Pure Go)

- **Status:** `Completed`
- **Date:** `2026-11-23`
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

**Completed (with empirical recommendation: Variant B2 Pure Go via `github.com/odvcencio/gotreesitter v0.58.0` behind an `ASTIndexer` interface).**

### Environment & Toolchain

- **Windows Host (`windows/amd64`):** Go `1.26.2`; baseline environment had `CGO_ENABLED=0` with no C/C++ compiler on `PATH`. Installed `zig 0.17.1` (`clang 22.1.8`, target `x86_64-unknown-windows-gnu`) via `winget` for native and cross-compiled CGO testing.
- **Linux Containers (`linux/amd64`):** Docker Desktop running Linux Engine `29.9.3`; build images `golang:1.26-alpine` and `golang:1.26-bookworm` (`go1.26.11 linux/amd64`); runtime targets `alpine:3.21` (`musl`), `debian:bookworm-slim` (`glibc`), and `scratch`.
- **Evaluated Candidates (disposable PoCs in `tmp/spike-002/`):**
  - **Variant A (Official CGO):** `github.com/tree-sitter/go-tree-sitter v0.25.0`, `github.com/tree-sitter/tree-sitter-go v0.25.0`, `github.com/tree-sitter/tree-sitter-python v0.25.0`, and `github.com/tree-sitter/tree-sitter-typescript v0.23.2` (`LanguageTypescript()` + `LanguageTSX()`).
  - **Variant B1a (Pre-packaged WASM / `wazero` — Rejected in Probe):** `github.com/malivvan/tree-sitter v0.0.1` + `github.com/tetratelabs/wazero v1.8.2` (`lib/ts.wasm`, `4,217,596 B` / `4.02 MiB`). Out of the box, it compiles and exports only `LanguageC` and `LanguageCpp` (`Go`, `Python`, `TypeScript`, and `TSX` are unsupported). Windows `CGO_ENABLED=0` cold `-a` build took `7.70s` (`11.24 MiB` / `11,782,656 B`); C++ smoke check measured `150.32ms` `wazero` init + `1.00ms` parse (`n=1`).
  - **Variant B1b (Custom WASM Reactor + `wazero`):** Official C sources (`go-tree-sitter v0.25.0` + 4 grammars) + a custom C11 shim compiled via `zig cc --target=wasm32-wasi-musl -mexec-model=reactor -Os -Wl,-z,stack-size=1048576` into `ts.wasm` (`3,603,387 B` / `3.44 MiB`, `19.40s` compile time) and embedded in Go with `github.com/tetratelabs/wazero v1.8.2`.
  - **Variant B2 (Pure Go Runtime — Recommended):** `github.com/odvcencio/gotreesitter v0.58.0` importing standalone grammar packages (`github.com/odvcencio/gotreesitter/grammars/{go,python,typescript,tsx}`).

### 1. Fixtures & Repository Map Footprint

All three variants executed the exact same declaration-extraction specification across four fixture suites:

| Fixture | Total Non-`.git` Files / Size | Indexable Files (`.go`/`.py`/`.ts`/`.tsx`) | Extracted Symbols | Repo Map Size / Lines | Est. Tokens (`bytes/4`) | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`HaroldHF/concert-system`** | `48` / `0.39 MiB` (`413,903 B`) | `40` (`40 .py`) | `382` | `24,218 B` / `422` | `6,055` | Real multi-file Python repository |
| **`HaroldHF/Brinco`** | `811` / `9.25 MiB` (`9,697,451 B`) | `476` (`148 .go`, `1 .py`, `193 .ts`, `134 .tsx`) | `2,150` | `148,974 B` / `2,626` | `37,244` | Real Go + TypeScript + TSX repository |
| **`2834kendall/Celulares_Alex_Frontend`** | `1,410` / `21.87 MiB` (`22,933,160 B`) | `1,042` (`630 .ts`, `412 .tsx`) | `2,899` | `250,203 B` *(CGO/WASM)* / `250,147 B` *(Pure Go)* / `3,941` | `62,551` *(CGO/WASM)* / `62,537` *(Pure Go)* | Real `1,000+` file TS/TSX codebase; confirms token-budget filtering is required for small local contexts |
| **`edge-cases`** *(synthetic)* | `6` / `~0.21 MiB` | `6` (`2 .go`, `1 .py`, `2 .ts`, `1 .tsx`) | `810` | `62,327 B` / `816` | `15,582` | `empty.go` (`0 B`), `broken.{go,py,ts,tsx}` (`3` error files detected; `broken.py` indentation issue parsed with `hasError=false` across all three variants), `large.ts` (`220,076 B`, `6,000` lines) |

### 2. Build Speed, Docker Multi-Stage Images & Cross-Compilation (`n=1`)

- **Windows Local Builds (`windows/amd64`, Go `1.26.2`):**
  - **Variant A (`CGO_ENABLED=1` + `zig cc`):** Cold `-a` probe build took `48.01s` (`6.98 MiB`); `bench-cgo.exe` warm-Zig `-a` build took `18.04s` (`7.53 MiB` / `7,898,624 B`), and the final instrumented benchmark binary (with `peak_rss_windows.go` and `-timeout-ms`) weighed `7.62 MiB` (`7,988,224 B`). Fails out of the box when `CGO_ENABLED=0` (`build constraints exclude all Go files`).
  - **Variant B1b (`bench-wasm.exe`, `CGO_ENABLED=0`):** Go `-a` build took `6.18s` (`10.77 MiB` / `11,293,696 B`, embedding `3.44 MiB` `ts.wasm` compiled in `19.40s`); the final instrumented benchmark binary weighed `10.78 MiB` (`11,299,328 B`).
  - **Variant B2 (`bench-purego.exe`, `CGO_ENABLED=0`):** Go `-a` build took `11.33s` (`18.33 MiB` / `19,222,016 B`), zero C toolchain required; the final instrumented benchmark binary weighed `18.34 MiB` (`19,231,744 B`).
- **Windows -> `linux/amd64` Cross-Compilation:**
  - `bench-purego` (`CGO_ENABLED=0`): `9.65s`, `18.14 MiB` (`19,018,194 B`) static ELF — executed with `exit=0` on `edge-cases` on both `alpine:3.21` and `debian:bookworm-slim`.
  - `bench-wasm` (`CGO_ENABLED=0`): `13.58s`, `10.61 MiB` (`11,122,082 B`) static ELF — executed with `exit=0` on `edge-cases` on both `alpine:3.21` and `debian:bookworm-slim`.
  - `bench-cgo` (`CGO_ENABLED=0`): failed (`build constraints exclude all Go files`). With `CGO_ENABLED=1` + `zig cc -target x86_64-linux-musl`: `48.50s`, `9.90 MiB` (`10,384,672 B`) static ELF (ran `edge-cases` `n=1` on Alpine in `38.68ms` / `17.19 MiB` and Debian in `40.09ms` / `17.17 MiB`); with `zig cc -target x86_64-linux-gnu`: `16.25s`, `8.45 MiB` (`8,861,512 B`) dynamic ELF (`libc.so.6`, `libpthread.so.0`; ran `edge-cases` `n=1` on Debian in `36.54ms` / `16.85 MiB`).
- **Multi-Stage Linux Docker Builds (`--no-cache`, `go1.26.11`):**

| Image | Build Time (`n=1`) | Docker `.Size` (`containerd`)\* | Binary Size | `ldd` & Cross-Distro Portability |
| :--- | :--- | :--- | :--- | :--- |
| **`spike002-cgo:alpine`** *(includes `apk add gcc musl-dev` + `go mod download`)* | `27.62s` | `20.90 MiB` | `7,165,224 B` (`6.83 MiB`) | Dynamic (`ld-musl-x86_64.so.1`, `libc.musl-x86_64.so.1`) |
| **`spike002-cgo:debian`** | `15.38s` | `117.65 MiB` | `7,153,424 B` (`6.82 MiB`) | Dynamic (`libc.so.6`, `ld-linux-x86-64.so.2`); **fails on `alpine:3.21` (`exit=127: not found`)** |
| **`spike002-purego:alpine`** | `14.26s` | `39.37 MiB` | `19,020,978 B` (`18.14 MiB`) | Static (`CGO_ENABLED=0`; confirmed static via `ldd`: `"not a dynamic executable"`) |
| **`spike002-purego:debian`** | `14.50s` | `136.14 MiB` | `19,020,978 B` (`18.14 MiB`) | Static (`CGO_ENABLED=0`) |
| **`spike002-purego:scratch`** | `14.21s` | `27.84 MiB` | `19,020,978 B` (`18.14 MiB`) | Static; **runs on `FROM scratch` on `edge-cases` (`exit=0`, `VmHWM=41.88 MiB`)** |
| **`spike002-wasm:alpine`** *(copies prebuilt `3.44 MiB` `ts.wasm`)* | `7.30s` | `26.41 MiB` | `11,119,298 B` (`10.60 MiB`) | Static (`CGO_ENABLED=0`; confirmed static via `ldd`: `"not a dynamic executable"`) |
| **`spike002-wasm:debian`** *(copies prebuilt `3.44 MiB` `ts.wasm`)* | `7.03s` | `123.17 MiB` | `11,119,298 B` (`10.60 MiB`) | Static (`CGO_ENABLED=0`) |
| **`spike002-wasm:scratch`** *(copies prebuilt `3.44 MiB` `ts.wasm`)* | `7.22s` | `14.88 MiB` | `11,119,298 B` (`10.60 MiB`) | Static; **runs on `FROM scratch` on `edge-cases` (`exit=0`, `VmHWM=76.89 MiB`)** |

- *Docker accounting & WASM prebuild note:* Under Docker Engine 29's `containerd` image store, `docker image inspect --format '{{.Size}}'` sums compressed layer blobs, unpacked snapshots, and OCI metadata—so `spike002-purego:scratch` contains only the single `18.14 MiB` (`19,020,978 B`) static binary despite `.Size` reporting `27.84 MiB`, and `spike002-wasm:scratch` contains only the single `10.60 MiB` (`11,119,298 B`) binary despite `.Size` reporting `14.88 MiB`. Likewise, `spike002-wasm:*` Docker build times (`7.03s`–`7.30s`) compile only the Go wrapper copying `ts.wasm` (`19.40s` offline WASI compile excluded from Docker build time).

### 3. Warm Interleaved Indexing Latency & OS Peak RSS (`n=5` After 1 Warmup)

All `n=5` runs rotated variant order per round after 1 discarded OS-cache warmup pass. On Windows (`windows/amd64`), peak RSS was measured via `PeakWorkingSetSize`; inside Linux (`alpine:3.21` capped at `--cpus=1 --memory=1g --memory-swap=1g --pids-limit=256 --network=none`), peak RSS was measured via kernel `VmHWM`. Values are **`Min / Med / Max` (`n=5`)**:

| Platform & Fixture | Variant | `init_ms` (`Min / Med / Max`) | `index_ms` (`Min / Med / Max`) | `total_ms` (`Min / Med / Max`) | OS Peak RSS (`MiB`, `Min / Med / Max`) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Windows — `concert-system`** (`40 .py`) | `cgo` | `0.00 / 0.00 / 0.00` | `71.68 / 86.50 / 93.18` | `71.68 / 86.50 / 93.18` | `9.86 / 9.93 / 9.96` |
| | `purego` | `55.56 / 61.04 / 67.59` | `40.06 / 50.45 / 55.30` | `100.61 / 112.54 / 118.32` | `32.75 / 34.22 / 34.45` |
| | `wasm` | `80.38 / 87.15 / 89.67` | `113.05 / 125.06 / 130.78` | `193.43 / 208.66 / 218.75` | `51.50 / 51.62 / 51.70` |
| **Windows — `Brinco`** (`476` files) | `cgo` | `0.00 / 0.00 / 0.00` | `629.50 / 679.06 / 769.49` | `629.50 / 679.06 / 769.49` | `16.29 / 16.69 / 16.95` |
| | `wasm` | `80.52 / 87.43 / 98.03` | `792.98 / 866.67 / 948.45` | `880.40 / 948.83 / 1,043.15` | `45.03 / 51.75 / 51.91` |
| | `purego` | `53.36 / 60.43 / 67.60` | `1,229.79 / 1,388.46 / 1,678.56` | `1,283.15 / 1,456.06 / 1,738.99` | `89.04 / 107.36 / 131.24` |
| **Windows — `Celulares_Alex_Frontend`** (`1,042`) | `cgo` | `0.00 / 0.00 / 0.57` | `1,175.26 / 1,204.21 / 1,313.96` | `1,175.26 / 1,204.21 / 1,313.96` | `18.55 / 18.68 / 18.87` |
| | `purego` | `44.92 / 47.28 / 47.84` | `1,477.11 / 1,498.11 / 1,583.51` | `1,522.03 / 1,545.40 / 1,631.35` | `93.30 / 100.34 / 106.48` |
| | `wasm` | `64.98 / 67.96 / 76.37` | `1,733.01 / 1,774.07 / 1,795.16` | `1,800.97 / 1,839.05 / 1,871.53` | `60.13 / 60.21 / 60.82` |
| **Linux (`1 vCPU / 1 GiB`) — `concert-system`** | `cgo` | `0.03 / 0.03 / 0.03` | `53.44 / 54.69 / 55.66` | `53.47 / 54.72 / 55.69` | `6.04 / 6.08 / 6.08` |
| | `purego` | `48.77 / 48.84 / 51.54` | `31.03 / 32.19 / 36.24` | `79.82 / 81.03 / 87.78` | `30.16 / 30.31 / 30.62` |
| | `wasm` | `72.79 / 73.24 / 80.10` | `86.85 / 92.97 / 97.52` | `160.09 / 170.18 / 174.69` | `43.39 / 45.62 / 45.78` |
| **Linux (`1 vCPU / 1 GiB`) — `Brinco`** | `cgo` | `0.03 / 0.03 / 0.06` | `345.53 / 356.16 / 382.21` | `345.58 / 356.19 / 382.24` | `12.02 / 12.08 / 12.28` |
| | `wasm` | `71.66 / 74.04 / 77.35` | `555.25 / 570.91 / 579.30` | `628.31 / 644.24 / 656.65` | `38.75 / 39.22 / 42.76` |
| | `purego` | `47.82 / 49.30 / 50.50` | `875.51 / 893.66 / 910.56` | `924.33 / 942.71 / 959.86` | `72.34 / 74.84 / 79.53` |
| **Linux (`1 vCPU / 1 GiB`) — `Celulares_Alex_Frontend`** | `cgo` | `0.03 / 0.03 / 0.03` | `947.49 / 958.15 / 968.23` | `947.52 / 958.18 / 968.26` | `14.30 / 14.49 / 14.68` |
| | `purego` | `48.47 / 48.99 / 50.07` | `1,331.96 / 1,345.32 / 1,367.58` | `1,381.31 / 1,393.79 / 1,416.42` | `68.91 / 69.69 / 73.91` |
| | `wasm` | `69.67 / 73.27 / 76.30` | `1,622.54 / 1,643.24 / 1,671.25` | `1,697.59 / 1,716.51 / 1,740.92` | `49.15 / 49.76 / 50.39` |

- *Additional single-run observations (`n=1`):* Debian `glibc` `spike002-cgo:debian` on `Brinco` (`1 vCPU / 1 GiB`) completed in `312.79ms` with `13.91 MiB` `VmHWM`. First cold-NTFS/Defender reads on Windows before warmup were substantially slower (`9,855.72ms` on `Brinco`, `21,527.17ms` on `Celulares_Alex_Frontend` for the first cold runner).

### 4. AST Parity, Adversarial Stress, Timeouts & Concurrency (`n=1` per probe)

- **AST Symbol Parity:**
  - `cgo` vs. custom `wasm`: **`0` diff lines** across all four fixture sets (`1,564` files, `6,241` symbols).
  - `cgo` vs. `purego`: **`0` diff lines** on `edge-cases`, `concert-system`, and `Brinco`. On `Celulares_Alex_Frontend`, all `2,899/2,899` extracted symbols and line ranges matched identically; only `6` file headers differed on `[parse-error]` flags: `5` Vitest test files (`estadoPeriodoData.test.ts`, `RecruitmentBoardDrag.test.tsx`, `week.test.ts`, `BrandsPanel.test.tsx`, `AppearanceForm.test.tsx`) failed in official `tree-sitter-typescript v0.23.2` on `await importOriginal<typeof import(...)>()` while passing in `gotreesitter v0.58.0`, and `1` file (`RecruitmentBoard.tsx`) parsed cleanly in official C v0.23.2 while `gotreesitter v0.58.0` reported a recoverable syntax error around `L271–L273`.
- **Adversarial Inputs & Strict `1 GiB` OOM Stress Test:**
  - `deep_parens.ts` (`4,065 B`, `2,000` nested `(`) and `deep_jsx.tsx` (`11,067 B`, `1,000` nested `<div>`): all three variants exited `0` in `3.27ms`–`15.27ms` (`error_files=0`).
  - `binary_nul.ts` (`65,536 B` random + `\x00` + invalid UTF-8): all three exited `0` (`error_files=1`, `symbols=0`; Windows `704.10ms` / `23.29 MiB` CGO, `294.72ms` / `45.00 MiB` Pure Go, `1,912.37ms` / `107.80 MiB` WASM).
  - `minified_2mb.ts` (`2,992,780 B`, single-line `45,000` funcs):
    - Windows (`n=1`): CGO `1,160.91ms` / `197.55 MiB`; Pure Go `597.79ms` (`550.29ms` index) / `255.80 MiB`; WASM `1,722.65ms` (`1,656.95ms` index) / **`959.91 MiB`**.
    - Linux strict `1 GiB` (`--memory=1g --memory-swap=1g`, `n=1`): all `ExitCode=0`, `OOMKilled=false`; CGO `1,038.99ms` / `198.07 MiB` `VmHWM`; Pure Go `738.55ms` / `260.64 MiB` `VmHWM`; WASM `2,378.29ms` / **`814.94 MiB` `VmHWM`** (`Sys=847.17 MiB`, ~`80%` of container budget on one file).
- **Timeout Cancellation & Memory Budget (`minified_2mb.ts` `2,992,780 B`, `n=1`):**
  - **CGO (`go-tree-sitter v0.25.0`):**
    - Deprecated `ParseCtx` without `SetCancellationFlag`: **panicked (`exit=2`, nil pointer dereference at `parser.go:198`)** because `ParseCtx` calls `atomic.StoreUintptr(p.CancellationFlag(), 1)` without initializing the flag (`ParseCtx` is marked deprecated in `v0.25.0` for removal in `v0.26` in favor of `ParseWithOptions`).
    - Non-deprecated `ParseWithOptions` (`ProgressCallback`, `cmd/probe-cancel` on Windows): canceled cleanly with `tree=nil` at **`1.74ms` / `17.19 MiB` Peak RSS** (`1ms` timeout, `callbacks=1`, `last_offset=112`), **`5.38ms` / `17.36 MiB` RSS** (`5ms`, `callbacks=97`, `last_offset=9,647`), and **`20.31ms` / `21.29 MiB` RSS** (`20ms`, `callbacks=716`, `last_offset=72,853`). Initializing `SetCancellationFlag(&flag)` + `ParseCtx` (`5ms`) also canceled cleanly (`5.31ms`, `tree=nil`, `flag_val=1`, `17.63 MiB` RSS).
  - **WASM (`wazero` `WithCloseOnContextDone(true)`, `bench-wasm -timeout-ms 5`):** Exited `0`, `timed_out=1`, `index_ms=9.16`, `48.75 MiB` RSS. Because `WithCloseOnContextDone(true)` closes/destroys the active WASM module instance on timeout (`break` out of indexing loop), indexing further files requires full module re-instantiation (`~73–87ms`).
  - **Pure Go (`gotreesitter` `SetTimeoutMicros` & `SetMemoryBudgetBytes`):**
    - `bench-purego -timeout-ms 5` exited `0`, `timed_out=1`, `index_ms=24.39`, `145.07 MiB` RSS.
    - **Pre-allocation before timeout polling (`cmd/probe-budget`, `-mem-budget-mb 0` on Windows):** Even at `-timeout-ms 1`, `gotreesitter` spent **`22.44ms`** (`stop_reason=timeout`, `err=parse stopped before accepting input: timeout`) and reserved **`165.14 MiB` `HeapInuse` / `181.16 MiB` `Sys` (`112.09 MiB` Peak RSS)** before polling the timeout; `-timeout-ms 5` measured `22.56ms` / `165.16 MiB` `HeapInuse` (`123.11 MiB` RSS) and `-timeout-ms 20` measured `27.28ms` / `170.59 MiB` `HeapInuse` (`110.46 MiB` RSS) — `~55x–60x` file size pre-allocated up front.
    - **Direct `SetMemoryBudgetBytes` without timeout (`timeout_ms=0` on Windows):** At `16 MiB` (`ParseStrict`), `gotreesitter` stopped cleanly (`stop_reason=memory_budget`, `err=parse stopped before accepting input: memory_budget`, `stopped_early=true`, `264.68ms`), yet OS Peak RSS still reached **`196.27 MiB`** (`199.49 MiB` `HeapInuse`); non-strict `Parse` returned `err=<nil>` (`has_root=true`) with `stop_reason=memory_budget` (`264.60ms`, `197.28 MiB` RSS). At `64 MiB` (`ParseStrict`), budget-starved retry work ran for **`2,053.67ms`** and spiked Peak RSS to **`296.68 MiB` (`301.48 MiB` `HeapInuse`)** (`4.6x` the `64 MiB` budget) before halting (`stop_reason=memory_budget`). At `256 MiB`, parsing accepted cleanly (`469.62ms`, `169.98 MiB` RSS). Control `edge-cases/large.ts` (`220,076 B`, `64 MiB` budget) accepted in `54.31ms` at `26.36 MiB` RSS (`15.02 MiB` `HeapInuse`).
    - **Worker Host architectural implication:** Because `gotreesitter` runs on the privileged Worker Host process (outside the `1 GiB` container cgroup) and pre-allocates memory proportional to input bytes before timeout checks—while `SetMemoryBudgetBytes` tracks internal arenas rather than OS RSS—**strict pre-parse per-file byte limits (e.g., skipping files `> 512 KiB`–`1 MiB` before `Parse`) and worker concurrency/memory limits are mandatory primary guardrails**, with `SetTimeoutMicros` and `SetMemoryBudgetBytes` as secondary defense-in-depth.
- **Concurrency & `-race`:**
  - `4`-worker parallel parsing over `80` `Brinco` files passed `go test -v` for all three variants (`Pure Go` `0.24s`, `WASM` `0.13s`, `CGO` `0.08s`).
  - `go test -race` **always requires `CGO_ENABLED=1` and a C compiler** even for `100%` pure-Go code (`bench-purego` under `CGO_ENABLED=0` fails immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`). Under `zig cc 0.17.1` on Windows, **both** `bench-cgo` and `bench-purego` fail linkage (`lld-link: error: undefined symbol: WaitOnAddress / WakeByAddressSingle / WakeByAddressAll` from Go's ThreadSanitizer runtime missing `-lsynchronization`), confirming a Go TSan + Zig Windows toolchain incompatibility rather than a `go-tree-sitter` defect.
  - Inside `golang:1.26-bookworm` (`linux/amd64`), `go test -race -v .` passed `3/3` (`test duration / package total`): `Pure Go` **`2.22s` test / `3.238s` pkg**, `WASM` **`0.72s` test / `1.735s` pkg**, `CGO` **`0.14s` test / `1.151s` pkg** (CGO `15.8x` faster on test time). Pure Go's actual `-race` advantage is visibility: Go's race detector instruments parser internals directly, whereas data races in C Tree-sitter or WASM linear memory are invisible to `-race`.

## Key Learnings

1. **Pure Go (`gotreesitter`) Eliminates Runtime C Toolchain & Coupling:** `github.com/odvcencio/gotreesitter v0.58.0` compiles with `CGO_ENABLED=0` across Windows (`11.33s` `-a`, `18.34 MiB` final binary) and Linux (`14.21s` `FROM scratch`, `18.14 MiB` static binary / `27.84 MiB` `containerd` `.Size`), indexes `1,042` TS/TSX files in `1.39s` median under `1 vCPU / 1 GiB` Linux (`69.69 MiB` `VmHWM`), instruments parser internals under `-race` on Linux/CI, and avoids C memory-safety exposure when parsing untrusted repositories in the worker process.
2. **`malivvan/tree-sitter` Is Not Production-Ready for Multi-Language Use; Custom WASM Approaches `1 GiB` Ceiling:** `malivvan/tree-sitter v0.0.1` only exports C and C++ grammars. Building a custom `zig cc --target=wasm32-wasi-musl` reactor works (`0` diffs vs. CGO), but requires maintaining a C shim + WASI toolchain (`19.40s` offline WASI build), destroys the module instance on timeout under `WithCloseOnContextDone(true)`, and spiked RSS to `814.94 MiB` `VmHWM` on Linux (`959.91 MiB` on Windows) on `minified_2mb.ts` (`~80%` of the `1 GiB` budget on one file).
3. **CGO Trade-offs (`go-tree-sitter v0.25.0`):** Official CGO yields the smallest binaries (`6.82`–`6.83 MiB` on Linux), lowest steady RSS (`14.49 MiB` `VmHWM` on `1,042` files), fastest `Brinco` indexing (`356.19ms` vs. Pure Go `942.71ms`, `2.65x` faster total / `2.51x` faster `index_ms`), and crisp cancellation under non-deprecated `ParseWithOptions` (`1.74ms` / `17.19 MiB` at `1ms` timeout). However, it breaks `CGO_ENABLED=0` builds, ties binaries to `glibc` vs. `musl` (`exit=127` on Alpine), executes C memory management inside the privileged worker process (opaque to `-race`), requires explicit `.Close()` calls, freezes `tree-sitter-typescript` at `v0.23.2`, and panics in deprecated `ParseCtx` if `SetCancellationFlag` is omitted.
4. **API, Pre-Allocation & Pre-1.0 Hygiene Rules for `gotreesitter`:**
   - **Primary Worker Host OOM defense:** Enforce strict **pre-parse per-file byte caps** (e.g., skip files `> 512 KiB`–`1 MiB` before `Parse`) and **worker concurrency/memory caps**; `SetTimeoutMicros` does not stop `~55x–60x` file-size pre-allocation (`165`–`171 MiB` `HeapInuse` on `2.85 MiB`), and `SetMemoryBudgetBytes` tracks internal arenas rather than OS RSS (`296.68 MiB` RSS observed under a `64 MiB` budget). Use `SetTimeoutMicros` and `SetMemoryBudgetBytes` as secondary controls alongside `ParseStrict`.
   - Always extract all node metadata/text **before** calling `tree.Release()`, because `Release()` returns arenas to a pool and resets node symbols (`root.Type(lang)` silently returns `"end"` post-release).
   - Always guard `tree.RootNode() == nil` (e.g., on `0`-byte files like `empty.go`), and enforce one `*gotreesitter.Parser` per goroutine (or `ParserPool`).
   - Because `gotreesitter v0.58.0` is a pre-1.0 MIT third-party reimplementation with its own grammar blobs/scanner VM (explaining the `1/1,042` `RecruitmentBoard.tsx` `L271–L273` false positive vs. C), isolate it strictly behind an `ASTIndexer` interface so reverting to CGO remains a localized swap.
5. **Repo Map Token Sizing Matters for Local Models:** Even concise declaration maps reach `37,244` estimated tokens on `Brinco` (`476` files) and `62,537` (`~62.5k`, Pure Go; `62,551` CGO/WASM) estimated tokens on `Celulares_Alex_Frontend` (`1,042` files), meaning [Spike-003](./003-ollama-tool-calling.md) / Phase 2 must support ranking/filtering or directory scoping rather than injecting an unbounded full-repo map into small `8k`–`32k` local LLM context windows.

## Impact on Requirements & ADRs

- **ADRs:** Empirically validates the Pure Go Tree-sitter AST engine (`gotreesitter` behind `ASTIndexer`) for [ADR-003](../adr/003-treesitter-and-mcp.md) (`Accepted`, with MCP JSON-RPC contracts adopted by design pending [Spike-003](./003-ollama-tool-calling.md) / Phase 2), and unblocks/accepts [ADR-001](../adr/001-go-control-plane.md) (`Accepted`) for zero-CGO (`CGO_ENABLED=0`) dev/production static builds across Windows and Linux (`scratch` / `alpine`), noting that `go test -race` still requires a compatible C toolchain on Windows or Linux/CI.
- **Requirements ([requirements.md](../requirements.md)):** Establishes verified pre-implementation baselines for AST indexing latency (`942.71ms` median on `476` multi-language files and `1,393.79ms` median on `1,042` TS/TSX files under `1 vCPU / 1 GiB`), memory footprint (`~70–75 MiB` `VmHWM` steady, `260.64 MiB` on `2.85 MiB` single-line stress input), `CGO_ENABLED=0` runtime builds, pre-parse file-size / concurrency limits, and Repo Map token-budget scoping (to be consolidated under Issue `#4`).
- **Threat Model ([threat-model.md](../threat-model.md)):** Establishes memory-safe AST parsing of untrusted repositories in pure Go (no C heap corruption/segfault surface in worker processes), mandatory **pre-parse per-file size limits** + worker concurrency/memory bounds against pre-allocation DoS, plus secondary `SetTimeoutMicros` / `SetMemoryBudgetBytes` parse controls.

## Recommendation & Next Steps

1. **Adopt `github.com/odvcencio/gotreesitter` (`CGO_ENABLED=0`) behind a focused `ASTIndexer` interface** in [ADR-003](../adr/003-treesitter-and-mcp.md), importing only required standalone grammar packages (`grammars/go`, `grammars/python`, `grammars/typescript`, `grammars/tsx`) and enforcing strict **pre-parse per-file byte limits + worker concurrency caps** alongside `SetTimeoutMicros` and `SetMemoryBudgetBytes`.
2. **Promote [ADR-001](../adr/001-go-control-plane.md) and [ADR-003](../adr/003-treesitter-and-mcp.md) to `Accepted`** and consolidate takeaways into [learnings.md](./learnings.md) and [architecture.md](../architecture.md).
3. **Keep all disposable Spike-002 PoCs and cloned fixtures strictly in `tmp/spike-002/` (out of `main`)**, and proceed to [Spike-003](./003-ollama-tool-calling.md) using real Spike-002 Repo Maps + token-budget scoping to validate local Ollama tool-calling (`num_ctx` `8k`/`16k`/`32k`).
