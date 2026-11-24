# Consolidated Spike Learnings

| Field | Value |
| :--- | :--- |
| **Status** | `In Progress (2/3 Spikes Completed)` |
| **Date** | `2026-11-23` |
| **Related Docs** | [Spikes Index](./README.md), [Requirements](../requirements.md), [Threat Model](../threat-model.md), [ADRs](../adr/README.md) |

> [!NOTE]
> Use this file to synthesize empirical findings across [Spikes 001–003](./README.md) into concrete engineering rules before freezing [requirements.md](../requirements.md) and populating [threat-model.md](../threat-model.md). Record only verified measurements and observed behavior.

---

## 1. Consolidated Spike Summary

| Spike | Result | Key Measured Numbers | Chosen Approach | Docs Updated |
| :--- | :--- | :--- | :--- | :--- |
| [Spike-001: Docker Network Isolation](./001-docker-network-isolation.md) | `Completed` | Variant B `347ms` med / `379ms` P95 overhead vs. Variant A `458ms` / `476ms`; `10/10` small + `11/11` large `NetworkMode: "none"` HTTP/DNS blocked; `49ms` (`n=1`) timeout teardown; `22.44s` Windows bind-mount vs. `260ms` native `TreeWalk I/O` (`1,410` files) | `NetworkMode: "none"` + host-side Git; workspace transport abstracted (`ext4` native mount/volume w/o host `.git/` on Linux; buffered `CopyToContainer` w/o `.git/` on Windows / non-Linux hosts) | [ADR-002](../adr/002-docker-sdk-sandbox.md) |
| [Spike-002: Tree-sitter Go Bindings](./002-treesitter-go-bindings.md) | `Completed` | Linux (`1 vCPU / 1 GiB`, `n=5`): Pure Go `942.71ms` med / `74.84 MiB` `VmHWM` (`Brinco`, `476` files), `1,393.79ms` med / `69.69 MiB` (`Celulares_Alex_Frontend`, `1,042` TS/TSX), `81.03ms` med / `30.31 MiB` (`concert-system`, `40 .py`) vs. CGO `356.19ms` / `958.18ms` / `54.72ms` med; `14.21s` `FROM scratch` static build (`18.14 MiB` binary / `27.84 MiB` `containerd` `.Size`); `2,899/2,899` symbol parity (`Celulares_Alex_Frontend`) | Pure Go (`github.com/odvcencio/gotreesitter v0.58.0`) behind `ASTIndexer` interface; `CGO_ENABLED=0` standalone grammar packages | [ADR-003](../adr/003-treesitter-and-mcp.md), [ADR-001](../adr/001-go-control-plane.md) |
| [Spike-003: Ollama Tool Calling](./003-ollama-tool-calling.md) | `Pending` | `TBD` | `TBD` | `TBD` |

---

## 2. Platform & Environment Notes (Windows vs. Linux / Docker)

- **Spike-001 (Docker SDK & Host OS):**
  - **Engine 29+ Go SDK:** `github.com/docker/docker` is deprecated; use `github.com/moby/moby/client` (`v0.6.3`) and `github.com/moby/moby/api` (`v1.57.0`) with `client.New(client.FromEnv)` (automatic API version negotiation `1.40`–`1.57`).
  - **Docker Socket Permissions:** On Windows Docker Desktop, non-admin processes connect directly via `npipe:////./pipe/docker_engine` (`3.25ms` Ping). Inside Linux containers (`alpine:3.21`), `/var/run/docker.sock` has permissions `srw-rw---- (660) 0:0` (`6.183ms` Ping as `root`), rejecting non-root UID `1000:1000` with `permission denied` unless granted socket group access or fronted by a socket proxy.
  - **Cross-VM Bind-Mount I/O vs. Native `ext4`:** On a `~1,410`-file TypeScript repo (`22.9 MB` working tree), reading all files inside the container took `257ms`–`260ms` on native `overlayfs`/`ext4` vs. `22.44s` (and `>30s` on first attempt) across a Windows↔WSL2 NTFS bind-mount (~`87x` slower), while host `os.RemoveAll` scaled from `9ms` (`Graft`) to `139ms`–`244ms`.
  - **Linux Bind-Mount Ownership:** Running a Linux bind-mount test with `--user 1000:1000` created `1000:1000 644` files cleanly; native Linux workers mounting host dirs should set `Config.User` to match the worker `uid:gid`.
- **Spike-002 (CGO Toolchain, `CGO_ENABLED=0` & Linux Container Portability):**
  - **Baseline Windows vs. Zig CGO:** Out-of-the-box Windows Go (`1.26.2`) defaults to `CGO_ENABLED=0` with no C compiler on `PATH`; `go-tree-sitter` fails immediately (`build constraints exclude all Go files`). Installing `zig 0.17.1` (`clang 22.1.8`) enabled native Windows CGO (`48.01s` cold / `18.04s` warm-Zig) and Windows→`linux/amd64` musl (`48.50s`, static) & glibc (`16.25s`, dynamic) builds. Because Go's `-race` detector **always** requires `CGO_ENABLED=1` and a C compiler even for pure-Go code (`bench-purego` fails under `CGO_ENABLED=0` with `go: -race requires cgo`), both `bench-cgo` and `bench-purego` fail Windows `-race` linkage under `zig cc 0.17.1` (`undefined symbol: WaitOnAddress / WakeByAddressSingle / WakeByAddressAll` from Go's ThreadSanitizer runtime missing `-lsynchronization`).
  - **Dynamic Libc Mismatch vs. Static `FROM scratch`:** Multi-stage `spike002-cgo:debian` (`libc.so.6` / `/lib64/ld-linux-x86-64.so.2`) failed inside `alpine:3.21` (`exit=127: not found`), and `spike002-cgo:alpine` (`27.62s` under `--no-cache`) includes `apk add gcc musl-dev` + `go mod download`. `gotreesitter v0.58.0` (`CGO_ENABLED=0`) compiled in `11.33s` `-a` on Windows (`18.33 MiB` initial / `18.34 MiB` final benchmark binary) and `14.21s`–`14.50s` in Docker (`go1.26.11`), producing static binaries (`18.14 MiB` / `19,020,978 B`) confirmed static via `ldd` (`"not a dynamic executable"`) that ran on `edge-cases` on Alpine, Debian, and `FROM scratch` (`27.84 MiB` Docker Engine 29 `containerd` `.Size` including blob + snapshot + OCI metadata).
  - **Linux vs. Windows Concurrency & `-race`:** All three variants passed 4-worker concurrency tests cleanly; `go test -race -v .` failed linkage under Windows Zig for both CGO and Pure Go, but passed `3/3` inside `golang:1.26-bookworm` (`test duration / package total`: `Pure Go` `2.22s` / `3.238s`, `WASM` `0.72s` / `1.735s`, `CGO` `0.14s` / `1.151s` — CGO `15.8x` faster on test time, while Pure Go uniquely gives `-race` visibility inside parser internals).
- *Spike-003 platform notes:* `TBD`

---

## 3. Evidence for Requirements

- **Spike-001 (Sandbox Provisioning, Limits & Cleanup Baselines):**
  - **Container startup (`Create + Start`):** `127ms`–`147ms` med (`147ms`–`163ms` P95) across `NetworkMode: "none"` multi-run suites (`139ms` med / `154ms` P95 in 10-run Variant B; `147ms` med / `163ms` P95 in 5-run B2 Baseline; `127ms` med / `147ms` P95 in 5-run B2 Optimized) on pre-cached `alpine:3.21` fixture; `ContainerRemove(Force: true, RemoveVolumes: true)` completed in `68ms`–`163ms` med (`93ms`–`170ms` P95) across `NetworkMode: "none"` suites (`58ms` med in Variant A; `49ms` `n=1` in forced timeout termination) with `0` leaked containers/volumes across all runs.
  - **Resource enforcement:** `NanoCPUs: 1e9`, `Memory: 1 GiB`, and `PidsLimit: 256` mapped accurately to Linux cgroups v2 (`cpu.max=100000 100000`, `memory.max=1073741824`, `pids.max=256`).
  - **Host-to-container transfer budget:** On Windows hosts, buffered `CopyToContainer` (`256 KiB`, excluding `.git/`) stabilized at `12.48s` med / `12.79s` P95 for `1,410` files (`~22.9 MB`), followed by `737ms` in-container `git init` and `65ms` diff extract + `git apply --check`.
- **Spike-002 (AST Indexing Latency, Memory & Repo Map Token Budgets):**
  - **Indexing Latency (`< 2s` target met):** Under `1 vCPU / 1 GiB` Linux (`n=5`), Pure Go `gotreesitter` indexed `concert-system` (`40 .py`) in `81.03ms` med (`87.78ms` max), `Brinco` (`476` files across Go/Py/TS/TSX) in `942.71ms` med (`959.86ms` max), and `Celulares_Alex_Frontend` (`1,042` TS/TSX files) in `1,393.79ms` med (`1,416.42ms` max) — including `~49ms` one-time grammar blob load. Windows `n=5` warm medians were `112.54ms`, `1,456.06ms`, and `1,545.40ms`.
  - **Memory Headroom (`VmHWM`):** Pure Go peak RSS stayed at `30.31 MiB` (`concert-system`), `74.84 MiB` (`Brinco`), `69.69 MiB` (`Celulares_Alex_Frontend`), and `260.64 MiB` (`n=1` on `2.85 MiB` `minified_2mb.ts` under strict `--memory=1g --memory-swap=1g`), well inside the `1 GiB` budget.
  - **AST Parity & Repo Map Sizing:** `gotreesitter` matched CGO on `2,899/2,899` symbols (`Celulares_Alex_Frontend`), `2,150/2,150` (`Brinco`), and `382/382` (`concert-system`). Estimated full Repo Map footprints (`bytes/4`) reached `6,055` tokens (`concert-system`), `37,244` tokens (`Brinco`), and `62,537` tokens (`Celulares_Alex_Frontend`), proving Phase 2 / Spike-003 must scope or rank Repo Maps to fit `8k`–`32k` local model windows.
- *Spike-003 requirement baselines:* `TBD` (to be consolidated into [requirements.md](../requirements.md) under Issue `#4`).

---

## 4. Evidence for Threat Model

- **Spike-001 (Network & Credential Isolation Verified):**
  - **Zero-network guarantee:** `NetworkMode: "none"` blocked outbound HTTP (`curl` exit `6`) and DNS (`nslookup` exit `1`) across `100%` of tested runs (`10/10` small repo, `11/11` larger repo across Variant B `n=1`, B2 Baseline `n=5`, and B2 Optimized `n=5`; `36/36` total benchmark runs including Variant A's `15/15`), and single-run inspection probes confirmed `/sys/class/net` exposed only `lo` (no `eth0`).
  - **Host-only Git credentials & `.git/` isolation:** Performing `git clone`/`git push` on the worker host (passing tokens ephemerally without writing credentials to `.git/config`, with hooks disabled) and keeping the host `.git/` directory out of the sandbox—both via `CopyToContainer` exclusion and via `--separate-git-dir` / clean working-tree mounts on native Linux—prevents untrusted code from modifying `.git/config` or `.git/hooks/` to trigger host worker RCE.
  - **Dependency-cache hydration boundary (design input):** Running arbitrary repo install scripts (`npm`/`pip`) on the worker host would break isolation; any read-only (`:ro`) dependency caches mounted into `NetworkMode: "none"` sandboxes must be hydrated in a separate credential-less preparer container.
  - **Runaway process termination:** Canceling `context.WithTimeout` on `ExecAttach` left `sleep 60` `Running=true` (`PID=3961`); explicit `ContainerRemove(Force: true, RemoveVolumes: true)` on a fresh cleanup context is mandatory to prevent zombie/forked processes inside containers.
- **Spike-002 (Memory-Safe AST Parsing & DoS / Timeout Controls):**
  - **Eliminating C Parser Attack Surface in Worker Host:** Running Pure Go `gotreesitter` keeps untrusted repository parsing inside Go's bounds-checked runtime rather than calling C parsers (`go-tree-sitter`) on untrusted source files inside the privileged control-plane/worker process.
  - **Pre-Allocation Behavior & Primary File-Size Guardrails (`n=1` probes on Windows):** On `minified_2mb.ts` (`2.85 MiB`), `gotreesitter` pre-allocated `165`–`171 MiB` `HeapInuse` (`110`–`123 MiB` Peak RSS) within `22.44ms` even under a `1ms` `SetTimeoutMicros` budget before checking timeout, and `SetMemoryBudgetBytes` tracks internal arenas rather than OS RSS (`196.27 MiB` Peak RSS under a `16 MiB` budget; `296.68 MiB` Peak RSS over `2,053.67ms` under a `64 MiB` budget due to retry work). Because parsing executes on the Worker Host (outside the `1 GiB` container cgroup), production `ASTIndexer` must enforce strict **pre-parse per-file byte limits (e.g., skip files `> 512 KiB`–`1 MiB`) and worker concurrency/memory caps** as primary OOM defenses—backed by `SetTimeoutMicros` + `SetMemoryBudgetBytes` (`ParseStrict`), empty-root nil checks, and extracting node strings/spans before `tree.Release()` (post-release reads silently return symbol `0` `"end"`). All three variants also handled `2,000` nested parens, `1,000` nested `<div>`, and `64 KiB` binary/NUL inputs with `exit=0`.
- *Spike-003 security findings:* `TBD` (to be consolidated into [threat-model.md](../threat-model.md) under Issue `#4`).

---

## 5. Dead Ends, Surprises & Rejected Approaches

- **Deprecated `github.com/docker/docker` SDK:** Deprecated for Docker Engine 29+ (`client.NewClientWithOpts` deprecated in `moby/client v0.6.3`); replaced with `github.com/moby/moby/client` (`client.New(client.FromEnv)`).
- **Dynamic `NetworkDisconnect` / `NetworkConnect` (Variant A):** Rejected — higher baseline lifecycle overhead (`458ms` vs. `347ms` med), starts containers with outbound access prior to disconnection, and risks exposing Git credentials inside the container for private repositories.
- **Relying on Go `context.WithTimeout` Alone for Exec Termination:** Rejected — closing the hijacked HTTP attach stream does not signal or kill the in-container process (`ExecInspect` confirmed `Running=true` `+100ms` post-cancel).
- **Cross-VM NTFS Bind-Mounts for Multi-File Container I/O:** Rejected as default on Windows (and optionally macOS / non-Linux hosts without native `ext4`) due to `~87x` read degradation measured on Windows↔WSL2 (`22.44s` vs. `257ms` on `1,410` files). Also refuted the assumption that `bufio` + skipping `.git/` would drop Windows `CopyToContainer` to `<2s` on shallow clones: per-file NTFS `open`/`stat` dominates (`12.48s` med), though excluding `.git/` eliminated `20.4s` tail spikes (`P95` `12.79s`) and cut diff extraction `6.2x` (`402ms` -> `65ms`).
- **Pre-packaged WASM Tree-sitter (`github.com/malivvan/tree-sitter v0.0.1`):** Rejected — its embedded `lib/ts.wasm` (`4.02 MiB`) exports only C and C++ grammars; Go, Python, TypeScript, and TSX are unsupported out of the box.
- **Custom `wazero` WASM Reactor (`zig cc --target=wasm32-wasi-musl`):** Rejected — while achieving `0` AST diffs vs. CGO, it requires maintaining a custom C shim and WASI build toolchain (`19.40s` offline WASI build), `WithCloseOnContextDone(true)` destroys the active module instance on timeout (forcing full re-instantiation), and non-shrinking WASM linear memory plus Go tuple decoding spiked RSS to `959.91 MiB` on Windows and `814.94 MiB` `VmHWM` (`Sys=847.17 MiB`) inside a strict `1 GiB` Linux container on `minified_2mb.ts` (`~80%` of container budget on one file).
- **Official CGO Bindings (`github.com/tree-sitter/go-tree-sitter v0.25.0`):** Rejected as default — breaks `CGO_ENABLED=0` dev/prod builds, introduces `glibc` vs. `musl` portability failures (`exit=127` on Alpine), executes C memory allocation in the privileged worker process (opaque to `-race`), requires explicit `.Close()` on every C allocation, and panics in deprecated `Parser.ParseCtx` (`nil pointer dereference` at `parser.go:198`) unless `SetCancellationFlag` is pre-configured (though non-deprecated `ParseWithOptions` `ProgressCallback` canceled cleanly at `1.74ms` / `17.19 MiB` under a `1ms` timeout, and `SetCancellationFlag` + `ParseCtx` canceled cleanly at `5.31ms`). Also observed frozen `tree-sitter-typescript v0.23.2` rejecting Vitest `await importOriginal<typeof import(...)>()` across `5` test files that `gotreesitter v0.58.0` parsed cleanly (`gotreesitter` conversely flagged `1` recoverable false positive in valid `RecruitmentBoard.tsx` `L271–L273`).
