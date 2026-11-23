# Consolidated Spike Learnings

| Field | Value |
| :--- | :--- |
| **Status** | `In Progress (1/3 Spikes Completed)` |
| **Date** | `2026-11-22` |
| **Related Docs** | [Spikes Index](./README.md), [Requirements](../requirements.md), [Threat Model](../threat-model.md), [ADRs](../adr/README.md) |

> [!NOTE]
> Use this file to synthesize empirical findings across [Spikes 001–003](./README.md) into concrete engineering rules before freezing [requirements.md](../requirements.md) and populating [threat-model.md](../threat-model.md). Record only verified measurements and observed behavior.

---

## 1. Consolidated Spike Summary

| Spike | Result | Key Measured Numbers | Chosen Approach | Docs Updated |
| :--- | :--- | :--- | :--- | :--- |
| [Spike-001: Docker Network Isolation](./001-docker-network-isolation.md) | `Completed` | Variant B `347ms` med / `379ms` P95 overhead vs. Variant A `458ms` / `476ms`; `10/10` HTTP+DNS blocked; `49ms` forced timeout teardown; `22.44s` Windows bind-mount vs. `260ms` native `TreeWalk I/O` (`1,410` files) | `NetworkMode: "none"` + host-side Git; workspace transport abstracted (`ext4` native mount/volume on Linux, buffered `CopyToContainer` w/o `.git/` on Windows/macOS) | [ADR-002](../adr/002-docker-sdk-sandbox.md) |
| [Spike-002: Tree-sitter Go Bindings](./002-treesitter-go-bindings.md) | `Pending` | `TBD` | `TBD` | `TBD` |
| [Spike-003: Ollama Tool Calling](./003-ollama-tool-calling.md) | `Pending` | `TBD` | `TBD` | `TBD` |

---

## 2. Platform & Environment Notes (Windows vs. Linux / Docker)

- **Spike-001 (Docker SDK & Host OS):**
  - **Engine 29+ Go SDK:** `github.com/docker/docker` is deprecated; use `github.com/moby/moby/client` (`v0.6.3`) and `github.com/moby/moby/api` (`v1.57.0`) with `client.New(client.FromEnv)` (automatic API version negotiation `1.40`–`1.57`).
  - **Docker Socket Permissions:** On Windows Docker Desktop, non-admin processes connect directly via `npipe:////./pipe/docker_engine` (`3.25ms` Ping). Inside Linux containers (`alpine:3.21`), `/var/run/docker.sock` has permissions `srw-rw---- (660) 0:0` (`6.183ms` Ping as `root`), rejecting non-root UID `1000:1000` with `permission denied` unless granted socket group access or fronted by a socket proxy.
  - **Cross-VM Bind-Mount I/O vs. Native `ext4`:** On a `~1,410`-file TypeScript repo (`22.9 MB` working tree), reading all files inside the container took `257ms`–`260ms` on native `overlayfs`/`ext4` vs. `22.44s` (and `>30s` on first attempt) across a Windows↔WSL2 NTFS bind-mount (~`87x` slower), while host `os.RemoveAll` scaled from `9ms` (`Graft`) to `139ms`–`244ms`.
  - **Linux Bind-Mount Ownership:** Running a Linux bind-mount test with `--user 1000:1000` created `1000:1000 644` files cleanly; native Linux workers mounting host dirs should set `Config.User` to match the worker `uid:gid`.
- *Spike-002 & Spike-003 platform notes:* `TBD`

---

## 3. Evidence for Requirements

- **Spike-001 (Sandbox Provisioning, Limits & Cleanup Baselines):**
  - **Container startup (`Create + Start`):** `127ms`–`139ms` med (`147ms`–`154ms` P95) with `NetworkMode: "none"` on pre-cached `alpine:3.21` fixture; `ContainerRemove(Force: true, RemoveVolumes: true)` completed in `49ms`–`163ms` med with `0` leaked containers/volumes across all runs.
  - **Resource enforcement:** `NanoCPUs: 1e9`, `Memory: 1 GiB`, and `PidsLimit: 256` mapped accurately to Linux cgroups v2 (`cpu.max=100000 100000`, `memory.max=1073741824`, `pids.max=256`).
  - **Host-to-container transfer budget:** On Windows hosts, buffered `CopyToContainer` (`256 KiB`, excluding `.git/`) stabilized at `12.48s` med / `12.79s` P95 for `1,410` files (`~22.9 MB`), followed by `737ms` in-container `git init` and `65ms` diff extract + `git apply --check`.
- *Spike-002 & Spike-003 requirement baselines:* `TBD` (to be consolidated into [requirements.md](../requirements.md) under Issue `#4`).

---

## 4. Evidence for Threat Model

- **Spike-001 (Network & Credential Isolation Verified):**
  - **Zero-network guarantee:** `NetworkMode: "none"` containers exposed only `lo` under `/sys/class/net` (no `eth0`) and blocked outbound HTTP (`curl` exit `6`) and DNS (`nslookup` exit `1`) across `100%` of tested runs (`10/10` small repo, `16/16` larger repo runs across B and B2).
  - **Host-only Git credentials:** Performing `git clone`/`git push` on the worker host (passing tokens ephemerally without writing credentials to `.git/config`, with hooks disabled) and excluding `.git/` when using `CopyToContainer` prevents any GitHub credential or git hook from entering the sandbox.
  - **Dependency-cache hydration boundary (design input):** Running arbitrary repo install scripts (`npm`/`pip`) on the worker host would break isolation; any read-only (`:ro`) dependency caches mounted into `NetworkMode: "none"` sandboxes must be hydrated in a separate credential-less preparer container.
  - **Runaway process termination:** Canceling `context.WithTimeout` on `ExecAttach` left `sleep 60` `Running=true` (`PID=3961`); explicit `ContainerRemove(Force: true, RemoveVolumes: true)` on a fresh cleanup context is mandatory to prevent zombie/forked processes inside containers.
- *Spike-002 & Spike-003 security findings:* `TBD` (to be consolidated into [threat-model.md](../threat-model.md) under Issue `#4`).

---

## 5. Dead Ends, Surprises & Rejected Approaches

- **Deprecated `github.com/docker/docker` SDK:** Deprecated for Docker Engine 29+ (`client.NewClientWithOpts` deprecated in `moby/client v0.6.3`); replaced with `github.com/moby/moby/client` (`client.New(client.FromEnv)`).
- **Dynamic `NetworkDisconnect` / `NetworkConnect` (Variant A):** Rejected — higher baseline lifecycle overhead (`458ms` vs. `347ms` med), starts containers with outbound access prior to disconnection, and risks exposing Git credentials inside the container for private repositories.
- **Relying on Go `context.WithTimeout` Alone for Exec Termination:** Rejected — closing the hijacked HTTP attach stream does not signal or kill the in-container process (`ExecInspect` confirmed `Running=true` `+100ms` post-cancel).
- **Cross-VM NTFS Bind-Mounts for Multi-File Container I/O:** Rejected as default on Windows/macOS local dev due to `~87x` read degradation (`22.44s` vs. `257ms` on `1,410` files). Also refuted the assumption that `bufio` + skipping `.git/` would drop Windows `CopyToContainer` to `<2s` on shallow clones: per-file NTFS `open`/`stat` dominates (`12.48s` med), though excluding `.git/` eliminated `20.4s` tail spikes (`P95` `12.79s`) and cut diff extraction `6.2x` (`402ms` -> `65ms`).
