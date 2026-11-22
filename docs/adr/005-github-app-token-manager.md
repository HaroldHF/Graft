# ADR-005: GitHub App Installation Token Manager (In-Memory Cache & Singleflight)

- **Status:** `Accepted`
- **Date:** `2026-11-21`
- **Related Spikes:** None (resolved via proven Go patterns; validated directly via Phase 1 unit tests instead of a disposable spike)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Requirements](../requirements.md), [Threat Model](../threat-model.md)

## Context

Each Graft worker interacts with GitHub APIs to clone private repositories, push fix branches, open Pull Requests, and post failure diagnostics on issues. GitHub Apps authenticate in two steps: signing a short-lived `RS256` JWT with the App's private key, then exchanging it per `installation_id` for an **Installation Access Token** valid for **1 hour**.

Under concurrent worker execution:

- Minting a fresh Installation Access Token on every GitHub request wastes API rate limits and adds avoidable network round-trips.
- Caching tokens naively without synchronization triggers thundering-herd token exchanges near expiration or fails persistently on early token revocation (`401 Unauthorized`).
- Persisting short-lived tokens in shared external storage (such as Redis) enlarges the credential blast radius for modest worker counts.

Because this is a well-understood Go concurrency problem with established libraries and patterns, we can accept the architecture directly and validate it through deterministic production unit tests in **Phase 1** rather than running a throwaway spike.

## Options Considered

1. **Option A (`Per-Worker In-Memory TokenManager with singleflight`):**
   - Maintain a per-worker in-memory cache mapping `installation_id -> {token, expiresAt}` behind a small `TokenManager` Go interface, using `golang.org/x/sync/singleflight` (or wrapping `bradleyfalzon/ghinstallation/v2`) to coalesce concurrent exchanges.
   - Proactively refresh when remaining TTL drops below `< 5 minutes`, and explicitly invalidate + retry once on `401 Unauthorized`.
   - *Strengths:* Zero external secret persistence, minimal complexity, easily unit-tested with injectable clock and HTTP transport.
   - *Weaknesses:* Each worker process maintains its own cache (at most 1 token exchange per active `installation_id` per hour per worker).
2. **Option B (`Shared Redis Token Cache + Distributed Lock`):**
   - Store Installation Access Tokens centrally in Redis with TTL margins and distributed locking.
   - *Strengths:* Cluster-wide deduplication across large worker pools.
   - *Weaknesses:* Requires encrypting tokens at rest in Redis, adds Redis failure modes to auth, and increases secret exposure if Redis is compromised.
3. **Option C (`Per-Request Token Minting Without Caching`):**
   - Sign a JWT and exchange for an Installation Access Token on every GitHub call.
   - *Strengths:* Stateless.
   - *Weaknesses:* Unnecessary GitHub API rate-limit consumption and higher latency per operation.

## Decision

We accept **Option A (`Per-Worker In-Memory TokenManager with singleflight`)** for **Phase 1 (Implementation)** without running a pre-implementation spike:

- Expose a focused consumer interface (e.g., `Token(ctx context.Context, installationID int64) (string, error)` and `Invalidate(installationID int64)`) backed by an injectable clock and GitHub token-exchange client so behavior is deterministic in unit tests.
- Keep Installation Access Tokens strictly in worker process memory—never persisted to Redis or disk, and never injected into sandbox container environment variables.
- Backdate JWT `iat` claims (~60s) for clock-skew tolerance and refresh proactively when remaining TTL is `< 5 minutes`.

## Phase 1 Implementation & Verification Strategy

Instead of a disposable spike branch, concurrency and revocation guarantees must be proven by the production unit test suite for `TokenManager` running continuously under `go test -race ./...`:

1. **Concurrency & Expiry Boundary (`go test -race`):**
   - Spawn **20 concurrent goroutines** requesting tokens across **3 distinct `installation_id`s** right at the expiration / proactive-refresh boundary (`< 5 minutes` remaining TTL), using an injectable mock clock and mock token endpoint.
   - Assert zero data races (`-race`), valid tokens returned to all callers, and exact `singleflight` deduplication (one exchange per expiring `installation_id`).
2. **`401 Unauthorized` Revocation & Recovery:**
   - Simulate GitHub returning `401 Unauthorized` for an otherwise unexpired cached token (early revocation).
   - Assert cache invalidation via `Invalidate(installationID)`, a single fresh token exchange, and bounded transparent retry (maximum 1 retry to prevent auth loops).
3. **Clock Skew & Context Cancellation:**
   - Verify `iat` clock-skew handling (~60s) on generated JWT assertions and prompt cancellation when callers pass an expiring or canceled `context.Context`.

## Consequences

- **Positive:** Eliminates a redundant spike, keeps credentials out of Redis and sandboxes, provides clean testability with `go test -race` in both `pre-push` hooks and CI, and relies on proven Go concurrency primitives.
- **Negative / Trade-offs:** Multiple worker replicas independently mint tokens per active `installation_id` (~1 exchange/hour per worker). If Graft later scales to hundreds of worker replicas on the same installation, a subsequent ADR can supersede this decision with encrypted shared caching.

## References

- [RFC-001: Graft Architecture — Section 6](../architecture.md#6-open-questions-and-uncertainties-spike-inputs)
- [Threat Model](../threat-model.md)
