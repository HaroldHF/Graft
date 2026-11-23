# Spike-001: Container Isolation and Network Control with Docker SDK (Go)

- **Status:** `Completed`
- **Date:** `2026-11-22`
- **Timebox:** `0.5–1 day`
- **Informs / Blocks:** [ADR-002](../adr/002-docker-sdk-sandbox.md), [Threat Model](../threat-model.md), [Requirements](../requirements.md)

## Objective & Question

**Objective:** Using the Docker Engine SDK in Go, create an ephemeral container, clone a repository, isolate its network, and execute a command under a strict timeout (`context.WithTimeout`).

**Core Question:** What is the latency and operational impact of dynamically disconnecting a container's network interface via the Docker Engine API after cloning, compared to creating the container with `NetworkMode: "none"` from the start (and running `git clone`/`git push` on the worker host via a bind-mounted ephemeral workspace)?

## Context

[RFC-001](../architecture.md) specifies isolating the sandbox during analysis and test execution (`1 vCPU`, `1 GiB RAM`, `PidsLimit`, no network), and we need to evaluate whether running `git clone` or `git push` inside the sandbox risks exposing GitHub credentials (feeding into [Threat Model](../threat-model.md)). Additionally, we need hands-on clarity on Docker socket permissions across Windows (Docker Desktop named pipe) and Linux (`/var/run/docker.sock`) before finalizing `SandboxManager`.

## Experiment Plan

1. Write a small Go PoC using the official Docker Engine SDK (`client.NewClientWithOpts`).
2. Benchmark **Variant A (Dynamic Network Cut-off):**
   - Start container with default bridge network -> clone sample repo inside container -> call `NetworkDisconnect` -> run test command with `context.WithTimeout` -> verify `curl` fails -> reconnect network or copy diff -> remove container and volumes.
3. Benchmark **Variant B (`NetworkMode: "none"` from Creation):**
   - Clone sample repo into an ephemeral temp directory on the host -> create container with `HostConfig.NetworkMode = "none"`, `NanoCPUs`, `Memory`, `PidsLimit` -> bind-mount workspace -> run test command with `context.WithTimeout` -> remove container and temp dir.
4. Test timeout termination (e.g., an infinite loop / `sleep 60` under a `5s` `context.WithTimeout`) and confirm the container process is killed and zero orphan containers/volumes remain.
5. Document socket permission and path differences observed on Windows vs. Linux.

## Evaluation Criteria

- Measure container startup + isolation latency for Variant A vs. Variant B (10-run median and P95).
- Verify outbound HTTP/DNS fails completely once isolated.
- Confirm `context.WithTimeout` reliably interrupts runaway processes and tears down the container within bounded cleanup time.
- Identify exact Docker socket configuration/permissions required on Windows and Linux.

## Result

**Completed (with empirical recommendation: Variant B `NetworkMode: "none"` + host-side Git, with abstracted workspace transport).**

### Environment & Toolchain

- **Host & Go Runtime:** Windows (`windows/amd64`), Go `1.26.2` (+ cross-compiled static `linux/amd64` probe binary)
- **Docker Engine:** Docker Desktop `4.99.0 (245822)` running Linux Engine `29.9.3` (API `1.56`, MinAPI `1.40`)
- **Go SDK:** `github.com/moby/moby/client v0.6.3` and `github.com/moby/moby/api v1.57.0` (isolated disposable PoC in `tmp/spike-001/`)
- **Fixture Image:** `graft-spike001-sandbox:latest` (`alpine:3.21` + `git` + `curl`)

### 1. Primary 10-Run Benchmark (`HaroldHF/Graft` — Small Repository)

| Metric | Variant A (`NetworkDisconnect`) Min / Med / P95 / Max | Variant B (`NetworkMode: "none"` + Host Clone) Min / Med / P95 / Max |
| :--- | :--- | :--- |
| `Create + Start` | `154ms` / `188ms` / `204ms` / `206ms` | `127ms` / `139ms` / `154ms` / `162ms` |
| `Git Clone` (`--depth 1`) | `599ms` / `623ms` / `697ms` / `717ms` *(inside)* | `690ms` / `707ms` / `754ms` / `775ms` *(host)* |
| `NetworkDisconnect` | `81ms` / `119ms` / `142ms` / `143ms` | `0ms` *(not applicable)* |
| `Isolated Exec + Probes` | `92ms` / `94ms` / `95ms` / `95ms` | `93ms` / `94ms` / `99ms` / `100ms` |
| `Host git status` | `N/A` | `35ms` / `39ms` / `41ms` / `41ms` |
| `ContainerRemove` (`Force: true`) | `44ms` / `58ms` / `65ms` / `66ms` | `45ms` / `68ms` / `93ms` / `95ms` |
| `Host Dir Remove` (`os.RemoveAll`) | `0ms` | `7ms` / `9ms` / `12ms` / `13ms` |
| **Lifecycle Overhead (excl. clone)** | `406ms` / **`458ms`** / **`476ms`** / `479ms` | `329ms` / **`347ms`** / **`379ms`** / `385ms` |
| **Total End-to-End** | `1.027s` / **`1.077s`** / **`1.17s`** / `1.196s` | `1.035s` / **`1.061s`** / **`1.101s`** / `1.114s` |

- **Additional checks:** `NetworkConnect` reconnect overhead measured separately in single-run Variant A was `47ms`. Both variants achieved `10/10` outbound HTTP block (`curl` exit `6`: `Could not resolve host: example.com`), `10/10` DNS block (`nslookup` exit `1`: `Network unreachable`), and `0` leftover containers. Variant B passed `10/10` host `git status` detection (`.spike001-marker`) and `os.RemoveAll` cleanup without file-lock errors.

### 2. Larger-Repository Sensitivity Benchmark (`2834kendall/Celulares_Alex_Frontend`)

Measured repo footprint across tested clones: `1,392`–`1,410` non-`.git` working-tree files (`22.68`–`22.93 MB`), `37.0`–`44.0 MB` total on disk. To measure cross-VM NTFS bind-mount I/O degradation vs. native container `overlayfs`/`ext4`, a full non-`.git` file read (`find` + `cat` -> `TreeWalk I/O`) was timed inside the isolated container:

| Variant | Runs | `Git Clone` | `Create + Start` | Workspace Transfer / Init | `TreeWalk I/O` | Diff / Status Round-Trip | `ContainerRemove` | `Host Dir Remove` | Total End-to-End |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **A (`NetworkDisconnect`, native `ext4`)** | `5` | `2.483s` med *(inside)* | `274ms` *(n=1 insp.)* | `110ms` med *(disconnect)* | **`257ms`** med (`241ms–260ms`) | `N/A` | `246ms` *(n=1 insp.)* | `0ms` | `3.261s` med (`3.206s–3.362s`) |
| **B (`NetworkMode: "none"` + NTFS bind-mount)** | `1` *(n=1, 180s cap)* | `2.481s` *(host, n=1)* | `218ms` *(n=1)* | `0ms` | **`22.44s`** *(n=1; prior run >30s timeout)* | `209ms` (`git status`, `n=1`) | `62ms` *(n=1)* | `244ms` *(n=1)* | `27.978s` *(n=1)* |
| **B2 Baseline (`CopyToContainer`, `.git/` incl., unbuffered)** | `5` | `2.546s` med *(host)* | `147ms` med (`163ms` P95) | `12.9s` med / `18.946s` P95 (`20.391s` max) | **`266ms`** med (`246ms–273ms`) | `402ms` med (`git apply --check` `5/5` OK) | `119ms` med (`144ms` P95) | `203ms` med | `16.572s` med / `22.779s` P95 |
| **B2 Optimized (`CopyToContainer`, `bufio 256 KiB`, no `.git/` + `git init`)** | `5` | `2.446s` med *(host)* | `127ms` med (`147ms` P95) | `12.479s` med / `12.785s` P95 + `737ms` `git init` | **`260ms`** med (`244ms–270ms`) | **`65ms`** med (`git apply --check` `5/5` OK) | `163ms` med (`170ms` P95) | `139ms` med | `16.495s` med / `16.89s` P95 |

- **Notes on measurements and footprint variance:**
  - Single-run values (`n=1`, including all Variant B cells and Variant A's `Create + Start` / `ContainerRemove` from the separate inspection run) are observed single measurements, not medians.
  - Repo footprint differences reflect checkout/test mechanics: Variant B included `+1` marker file (`1,411` total) and default Windows CRLF line endings (`22.93 MB`), Variant B2 Optimized grew to `44.0 MB` inside the container because `git init && git commit` generated fresh `.git` objects instead of a thin shallow packfile (`37.0`–`38.2 MB`), and Variant A's earlier clone observed `1,392` files (`22.68 MB`).
  - On run 1 of each larger-repo suite (`measureDisk = true`), `du -sh /workspace` executed before `walkStart`, adding `~2s` on the Windows bind-mount to Variant B's `Total End-to-End` (`27.978s`) outside `TreeWalk I/O` (`22.44s`).
  - All `10/10` small-repo and `11/11` larger-repo `NetworkMode: "none"` runs (`1` Variant B + `5` B2 Baseline + `5` B2 Optimized; `36/36` total benchmark runs including Variant A's `15/15`) achieved `100%` HTTP/DNS isolation with `0` leftover containers or temp directories.
  - Excluding `.git/` + buffering `256 KiB` eliminated the `20.391s` `CopyToContainer` tail spike (`12.845s` max), sped up diff extraction by `6.2x` (`402ms` -> `65ms`), and kept `.git/` completely out of the sandbox. Because shallow `--depth 1` clones hold most files in the working tree (`1,410` files), Windows host `CopyToContainer` time remained dominated by per-file NTFS/Defender `open` + `stat` overhead (~8.8 ms/file).

### 3. Resource Caps, Timeout Termination & Zero-Orphan Verification

- **Cgroups v2 Enforcement:** `NanoCPUs: 1_000_000_000`, `Memory: 1 << 30`, and `PidsLimit: 256` produced exact container limits: `cpu.max=100000 100000`, `memory.max=1073741824`, and `pids.max=256`. Inspection runs (`includeDetails = true`) confirmed containers created with `NetworkMode: "none"` exposed only `lo` under `/sys/class/net` (no `eth0`).
- **Timeout Termination (`sh -c "sleep 60"` under `5s` `context.WithTimeout`):**
  - Context deadline expired at `5s` (`context deadline exceeded`), closing the hijacked attach connection.
  - Calling `ExecInspect` `+100ms` after cancellation (before container removal) returned **`Running=true PID=3961 ExitCode=0`** — confirming that Go context cancellation alone does **not** terminate processes inside the container.
  - Calling `ContainerRemove(..., Force: true, RemoveVolumes: true)` with a fresh `15s` cleanup context tore down the container in **`49ms`**.
  - Post-suite `docker ps -a --filter label=graft.spike=001` and `docker volume ls --filter dangling=true` confirmed `0` orphan containers and `0` leaked volumes.

### 4. Windows vs. Linux Docker Socket & Bind-Mount Observations

| Probe Target | Socket / Transport | Ping Latency / API | Observed Permissions & Behavior |
| :--- | :--- | :--- | :--- |
| **Windows Host (`windows/amd64`)** | `npipe:////./pipe/docker_engine` | `3.25ms` (`API 1.56`) | Default non-admin user connects without privilege elevation; bind-mount files show `0:0 777`/`0:0 644` inside container and delete cleanly on host. |
| **Linux Container as `root` (`0:0`)** | `unix:///var/run/docker.sock` | `6.183ms` (`API 1.56`) | `stat`: `srw-rw---- (660) 0:0 /var/run/docker.sock`. Connects cleanly via Unix socket. |
| **Linux Container as `1000:1000`** | `unix:///var/run/docker.sock` | `Failed (exit 1)` | `permission denied while trying to connect to the docker API at unix:///var/run/docker.sock` without matching socket group (`docker` / `0`) or socket proxy. |
| **Linux Container `--user 1000:1000` Bind-Mount** | `alpine:3.21` host mount | `N/A` (`exit 0`) | Created/deleted files as `1000:1000 644`, confirming `Config.User` alignment prevents `root:root`-owned bind-mount artifacts on native Linux workers. |

## Key Learnings

1. **Docker Engine 29+ Go SDK Migration:** `github.com/docker/docker` is deprecated upstream; Engine 29+ targets `github.com/moby/moby/client` (`v0.6.3`) and `github.com/moby/moby/api` (`v1.57.0`). `client.NewClientWithOpts` is deprecated in favor of `client.New(client.FromEnv)`, which defaults to `npipe:////./pipe/docker_engine` (Windows) or `unix:///var/run/docker.sock` (Linux) and negotiates API versions (`1.40`–`1.57`) automatically.
2. **`NetworkMode: "none"` Wins on Security & Baseline Overhead:** Starting without a network interface (`lo` only) removes `bridge` setup/teardown cost (~`24%` lower median lifecycle overhead on small repos: `347ms` vs. `458ms`), eliminates dynamic `NetworkDisconnect`/`NetworkConnect` failure modes, and keeps GitHub credentials strictly on the worker host.
3. **Bind-Mounts Do Not Scale on Windows Local Dev (`87x` I/O Penalty):** While `os.RemoveAll` stayed fast on small repos (`9ms` med) and acceptable on ~1,410 files (`139ms`–`244ms`), container-side full-tree reads over a Windows↔WSL2 NTFS bind-mount dropped from `257ms` (native `ext4`) to **`22.44s`** (and exceeded `30s` initially). Abstracting workspace injection behind `SandboxManager` allows using native `ext4`/ephemeral volumes in Linux production while supporting `CopyToContainer` (excluding `.git/`) + unified `git diff` extraction (`65ms` med) on Windows (measured) and optionally macOS / non-Linux hosts without native `ext4`.
4. **Never Leave Host `.git` or Tokens Accessible to the Sandbox:** Host-side cloning must disable hooks/autocrlf (`-c core.autocrlf=false -c core.hooksPath=/dev/null -c core.fsmonitor=false -c submodule.recurse=false`) and pass tokens ephemerally (never embedded in `.git/config`). Even on native Linux mounts/volumes, the host's `.git/` directory must never be mounted writable inside the sandbox (e.g., clone with `--separate-git-dir` outside the mount point or mount a clean working tree and apply validated unified diffs back to the host clone), preventing `.git/config` or `.git/hooks/` tampering from achieving host worker RCE. When using `CopyToContainer`, omitting `.git/` prevents exposing repository history or git configs altogether.
5. **Context Cancellation Alone Does Not Kill Container Execs:** Canceling `ExecAttach`'s context only closes the client HTTP stream (`Running=true` persisted inside the container). Every unrecoverable exec/job timeout must immediately invoke `ContainerRemove(..., Force: true, RemoveVolumes: true)` with a fresh bounded cleanup context (backed by the 10-minute crash-recovery reaper).
6. **Linux Socket & Non-Root File Ownership:** Native Linux deployments expose `/var/run/docker.sock` as `0660`; non-root worker containers require socket GID alignment (e.g., `GroupAdd`) or a socket proxy, and any Linux bind-mount should set `Config.User` to the worker's `uid:gid`.

## Impact on Requirements & ADRs

- **ADRs:** Finalizes [ADR-002](../adr/002-docker-sdk-sandbox.md) (`Accepted`) using `github.com/moby/moby/client`, `NetworkMode: "none"`, abstracted workspace transport (`ext4` native mount/volume without host `.git/` on Linux; `CopyToContainer` w/o `.git/` on Windows and optionally macOS/non-Linux hosts), and mandatory forced teardown on timeout.
- **Requirements ([requirements.md](../requirements.md)):** Provides measured baselines for container provisioning (`127ms`–`147ms` med `Create + Start` on cached image across `NetworkMode: "none"` suites), teardown (`68ms`–`163ms` med / `170ms` P95 across `NetworkMode: "none"` suites, `58ms` med in Variant A, `49ms` `n=1` in forced timeout termination), resource limits (`1 vCPU`, `1 GiB RAM`, `256` PIDs), and host-transfer overhead budgets to be consolidated under Phase 0 Gate (Issue `#4`).
- **Threat Model ([threat-model.md](../threat-model.md)):** Validates `NetworkMode: "none"` (`100%` HTTP/DNS block, `lo` only in inspection probes), host-only Git credential boundary, `.git/` isolation on both Linux mounts and `CopyToContainer`, and Docker socket `0660` privilege hardening (and notes that any pre-hydrated read-only `:ro` dependency caches for offline tests should be populated inside a separate credential-less preparer container rather than running `npm`/`pip` install scripts on the worker host).

## Recommendation & Next Steps

1. **Adopt Variant B (`NetworkMode: "none"` + Host-Side Git) in [ADR-002](../adr/002-docker-sdk-sandbox.md)** and reject dynamic `NetworkDisconnect`/`NetworkConnect`.
2. **Design `SandboxManager` with pluggable workspace provisioning** (`ext4` bind-mount/ephemeral volume with host `.git/` kept outside the mount on Linux production; `CopyToContainer` excluding `.git/` + diff extraction on Windows and optionally macOS/non-Linux hosts, plus optional `:ro` dependency-cache mounts for offline test execution).
3. **Enforce a strict cleanup contract:** wrap `ExecAttach` stream reads with `context.AfterFunc` to unblock on cancellation and always tear down timed-out sandboxes via `ContainerRemove(Force: true, RemoveVolumes: true)` on a fresh context.
4. **Keep Spike-001 PoC code (`tmp/spike-001/`) out of `main`**, proceed to [Spike-002](./002-treesitter-go-bindings.md) and [Spike-003](./003-ollama-tool-calling.md), and freeze [requirements.md](../requirements.md) and [threat-model.md](../threat-model.md) once all three spikes complete.
