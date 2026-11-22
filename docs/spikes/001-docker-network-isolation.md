# Spike-001: Container Isolation and Network Control with Docker SDK (Go)

- **Status:** `Pending`
- **Date:** `2026-11-21`
- **Timebox:** `0.5–1 day`
- **Informs / Blocks:** [ADR-002](../adr/002-docker-sdk-sandbox.md), [Threat Model](../threat-model.md), [Requirements](../requirements.md)

## Objective & Question

**Objective:** Using the Docker Engine SDK in Go, create an ephemeral container, clone a repository, isolate its network, and execute a command under a strict timeout (`context.WithTimeout`).

**Core Question:** What is the latency and operational impact of dynamically disconnecting a container's network interface via the Docker Engine API after cloning, compared to creating the container with `NetworkMode: "none"` from the start (and running `git clone`/`git push` on the worker host via a bind-mounted ephemeral workspace)?

## Context

[RFC-001](../architecture.md) specifies isolating the sandbox during analysis and test execution (`1 vCPU`, `1 GB RAM`, `PidsLimit`, no network), and we need to evaluate whether running `git clone` or `git push` inside the sandbox risks exposing GitHub credentials (feeding into [Threat Model](../threat-model.md)). Additionally, we need hands-on clarity on Docker socket permissions across Windows (Docker Desktop named pipe) and Linux (`/var/run/docker.sock`) before finalizing `SandboxManager`.

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

`Pending`

## Key Learnings

- `TBD` — Expected observations to capture:
  - Docker socket permissions and host OS quirks on Windows vs. Linux.
  - Exact `HostConfig` configuration for `NetworkMode: "none"`, CPU/RAM cgroups, and `PidsLimit`.
  - Latency delta between dynamic `NetworkDisconnect` and `NetworkMode: "none"`.
  - Whether host-side `git clone`/`git push` cleanly eliminates credentials inside the container without bind-mount permission friction.

## Impact on Requirements & ADRs

- **Requirements ([requirements.md](../requirements.md)):** Will establish concrete container provisioning timeouts, test execution timeouts, and baseline resource limits.
- **ADRs & Threat Model:** Will finalize the network isolation strategy in [ADR-002](../adr/002-docker-sdk-sandbox.md) and inform the git-push network boundary in [Threat Model](../threat-model.md).

## Recommendation & Next Steps

- `TBD`
