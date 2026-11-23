# ADR-002: Direct Docker Engine SDK for Ephemeral Sandboxes

- **Status:** `Accepted`
- **Date:** `2026-11-22`
- **Related Spikes:** [Spike 001: Docker Network Isolation](../spikes/001-docker-network-isolation.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Threat Model](../threat-model.md)

## Context

Graft runs untrusted repository code and LLM-generated patches. We must strictly isolate network access, CPU, memory (`1 vCPU`, `1 GB RAM` baseline), and process counts (`PidsLimit`), enforce command timeouts, stream container logs live, and guarantee clean container/volume teardown even after worker crashes.

## Options Considered

1. **Option A (`Go Docker Engine SDK + Ephemeral Containers`):** Drive the Docker Engine API programmatically through a Go `SandboxManager` interface, provisioning a fresh ephemeral container per job with cgroup limits, ephemeral volumes, and programmatic network isolation.
2. **Option B (`Shelling Out to docker CLI`):** Spawn `os/exec` subprocesses running `docker run`, `docker exec`, `docker network disconnect`, and `docker rm`.
3. **Option C (`Persistent / Pooled Long-Lived Containers`):** Keep warm containers running across multiple jobs and reset the working tree between issues.

## Decision

We adopt **Option A (`Go Docker Engine SDK + Ephemeral Containers`)** with the following finalized contracts validated in [Spike 001](../spikes/001-docker-network-isolation.md):

- **Typed Lifecycle & Error Handling:** Direct API calls provide structured container IDs (UUID tracking for the 10-minute crash-recovery reaper), reliable stream multiplexing (`stdcopy.StdCopy`) for SSE logs, and programmatic enforcement of cgroup/PID limits without fragile CLI output parsing.
- **Clean Isolation Per Job:** Ephemeral containers eliminate state bleed, leftover background processes, or cross-repository contamination that pooled persistent containers risk.
- **Official Moby Go SDK (`v29+` Compatible):** Use `github.com/moby/moby/client` (`client.New(client.FromEnv)`) and `github.com/moby/moby/api/types` rather than deprecated `github.com/docker/docker` or `NewClientWithOpts`.
- **Network & Git Credential Boundary (Variant B):** Create every sandbox container with `HostConfig.NetworkMode = "none"` from creation (`lo` interface only; `100%` HTTP/DNS blocked in benchmarks and `~24%` lower baseline lifecycle overhead than `NetworkDisconnect`: `347ms` vs. `458ms` median). Keep all authenticated `git clone` and `git push` operations on the worker host (with git hooks/autocrlf disabled and tokens passed ephemerally without persisting them into `.git/config`). Dynamic `NetworkDisconnect`/`NetworkConnect` is rejected.
- **Abstracted Workspace Provisioning & Optional `:ro` Caches:** Keep workspace transport behind the `SandboxManager` interface:
  - On native Linux workers/containers, use `ext4`/`tmpfs` ephemeral mounts or Docker volumes with `Config.User` aligned to the worker `uid:gid` (avoiding `root:root` artifacts and achieving ~`257ms` container read I/O on `~1,400` files).
  - On Windows/macOS hosts (where cross-VM NTFS bind-mounts degraded full-tree container reads to `22.44s`), inject the working tree via buffered `CopyToContainer` excluding `.git/` (restoring `260ms` native container I/O and keeping `.git/` outside the sandbox) and extract unified diffs back to the host (`65ms` median diff + `git apply --check`).
  - Any pre-hydrated dependency caches mounted read-only (`:ro`) for offline test execution must be populated in a separate credential-less preparer container rather than running repository install scripts (`npm`/`pip`) on the worker host.
- **Mandatory Forced Teardown on Timeout:** Because canceling `ExecAttach`'s `context.WithTimeout` only closes the client stream and leaves in-container processes running (`Running=true` verified empirically), `SandboxManager` must follow timeout expiry with `ContainerRemove(..., Force: true, RemoveVolumes: true)` on a fresh bounded cleanup context (`49ms` measured teardown), backed by the 10-minute crash-recovery reaper.
- **Verified Resource Limits:** Enforce `NanoCPUs: 1_000_000_000` (`1 vCPU`), `Memory: 1 << 30` (`1 GiB`), and `PidsLimit: 256` (verified via Linux cgroups v2).

## Consequences

- **Positive:** Zero network stack (`lo` only) and zero GitHub credentials inside sandboxes, deterministic cgroup v2 enforcement, clean diff-based host push flow, and `~49ms`–`170ms` teardown with zero orphan containers/volumes.
- **Negative / Trade-offs:** Higher per-job startup latency compared to warm containers; workers require Docker socket access (`npipe:////./pipe/docker_engine` on Windows; `0660` `/var/run/docker.sock` on Linux requiring socket GID alignment or a socket proxy, plus matching `Config.User` on Linux bind-mounts); Windows/macOS local `CopyToContainer` incurs host NTFS file-open overhead (`~12.5s` for `1,410` files) unless running workers inside Linux containers.

## References

- [Spike 001: Docker Network Isolation](../spikes/001-docker-network-isolation.md)
- [Threat Model and Security Guardrails](../threat-model.md)
