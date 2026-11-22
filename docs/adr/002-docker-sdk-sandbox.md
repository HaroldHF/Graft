# ADR-002: Direct Docker Engine SDK for Ephemeral Sandboxes

- **Status:** `Proposed`
- **Date:** `2026-11-21`
- **Related Spikes:** [Spike 001: Docker Network Isolation](../spikes/001-docker-network-isolation.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Threat Model](../threat-model.md)

## Context

Graft runs untrusted repository code and LLM-generated patches. We must strictly isolate network access, CPU, memory (`1 vCPU`, `1 GB RAM` baseline), and process counts (`PidsLimit`), enforce command timeouts, stream container logs live, and guarantee clean container/volume teardown even after worker crashes.

## Options Considered

1. **Option A (`Go Docker Engine SDK + Ephemeral Containers`):** Drive the Docker Engine API programmatically through a Go `SandboxManager` interface, provisioning a fresh ephemeral container per job with cgroup limits, ephemeral volumes, and programmatic network isolation.
2. **Option B (`Shelling Out to docker CLI`):** Spawn `os/exec` subprocesses running `docker run`, `docker exec`, `docker network disconnect`, and `docker rm`.
3. **Option C (`Persistent / Pooled Long-Lived Containers`):** Keep warm containers running across multiple jobs and reset the working tree between issues.

## Decision

We propose **Option A (`Go Docker Engine SDK + Ephemeral Containers`)** because:

- **Typed Lifecycle & Error Handling:** Direct API calls provide structured container IDs (UUID tracking for the 10-minute crash-recovery reaper), reliable stream multiplexing for SSE logs, and programmatic enforcement of cgroup/PID limits without fragile CLI output parsing.
- **Clean Isolation Per Job:** Ephemeral containers eliminate state bleed, leftover background processes, or cross-repository contamination that pooled persistent containers risk.

*Pending [Spike 001](../spikes/001-docker-network-isolation.md):* We still need to finalize whether the container starts with `NetworkMode: "none"` (performing `git clone`/`git push` from the worker host over a shared ephemeral volume) or disconnects the network dynamically via the Docker Engine API after cloning.

## Consequences

- **Positive:** Deterministic resource enforcement, straightforward cleanup by container UUID, and stronger security boundaries aligned with the [Threat Model](../threat-model.md).
- **Negative / Trade-offs:** Higher per-job startup latency compared to warm containers; workers require access to the Docker socket, which demands careful socket/privilege hardening.

## References

- [Spike 001: Docker Network Isolation](../spikes/001-docker-network-isolation.md)
- [Threat Model and Security Guardrails](../threat-model.md)
