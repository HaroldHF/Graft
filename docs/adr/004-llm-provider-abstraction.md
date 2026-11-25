# ADR-004: LLM Provider Abstraction (`LLMClient` over OpenAI-Compatible & Ollama Native Transports)

- **Status:** `Accepted`
- **Date:** `2026-11-25`
- **Related Spikes:** [Spike 003: Ollama Tool Calling](../spikes/003-ollama-tool-calling.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [ADR-003: Tree-sitter & MCP](./003-treesitter-and-mcp.md), [Benchmarks](../benchmarks.md), [Requirements](../requirements.md)

## Context

Graft must run completely locally during development and privacy-sensitive workflows (via Ollama) while also supporting frontier cloud LLMs for comparison in the mini SWE-bench suite and higher-complexity production issues. Coupling the Agent Runtime to vendor-specific SDKs would duplicate tool-calling, retry, and telemetry logic across providers.

However, [Spike 003](../spikes/003-ollama-tool-calling.md) demonstrated that relying solely on Ollama's OpenAI-compatible `v1/chat/completions` endpoint without provider-aware handling introduces silent failures on consumer hardware (`8 GiB` VRAM): Ollama `v0.40.18` ignored per-request `num_ctx` options on `/v1/chat/completions` (locking `default_num_ctx=4096` and truncating `7,312` tokens down to `limit=2050` without an HTTP error), and `qwen2.5-coder:7b/14b` emitted `0/64` native `tool_calls`, returning JSON inside `message.content` instead.

## Options Considered

1. **Option A (`Naive OpenAI v1/chat/completions Client Only`):** Send identical `v1/chat/completions` payloads to all providers and parse only `choices[0].message.tool_calls`.
2. **Option B (`Unified Go LLMClient Interface with OpenAI Cloud + Ollama-Aware Adapter`):** Define one minimal `LLMClient` interface in Go (`ChatTurn`) backed by a standard OpenAI `v1/chat/completions` transport for cloud/OpenAI-compatible APIs and an Ollama-aware adapter path (`/api/chat` with explicit `options.num_ctx`, or explicit server/Modelfile `num_ctx`), paired with multi-stage tool-call extraction, pre-request vs. `prompt_eval_count` truncation auditing, and 1-turn non-oracle prompt-repair retry.
3. **Option C (`Per-Provider Heavy Native SDKs`):** Pull vendor-specific Go SDKs for Anthropic, OpenAI, Gemini, and Ollama with separate runtime loops.

## Decision

We adopt **Option B (`Unified Go LLMClient Interface with OpenAI Cloud + Ollama-Aware Adapter`)** — explicitly distinguishing between **empirically validated local Ollama behavior** from [Spike 003](../spikes/003-ollama-tool-calling.md) and **design-adopted cloud `v1/chat/completions` transport** (to be benchmarked against cloud providers in Phase 4):

- **Single `LLMClient` Contract, Two Lightweight HTTP Transports:** Keep one Agent Runtime loop (`MCP` tools, token telemetry, SSE events, `context.WithTimeout` cancellation). Use standard `POST /v1/chat/completions` for cloud/OpenAI-compatible providers (design-adopted), and use Ollama's `POST /api/chat` (`options.num_ctx`) for local Ollama instances (empirically validated in Spike-003 so `8k`/`16k` context windows are honored per job and native `prompt_eval_count`, `prompt_eval_duration`, and `eval_duration` metrics are returned directly).
- **Silent-Truncation Auditing:** Compare returned `prompt_eval_count` / `prompt_tokens` against the **pre-request estimated input token count** (failing or re-scoping if `prompt_eval_count` drops significantly below the estimated input tokens, since a truncated `2,050`-token count from a `7,312`-token prompt looks valid if checked only against a `4,096` or `8,192` window), and verify `context_length` via `/api/ps` (or pin `OLLAMA_NUM_PARALLEL=1` on `8 GiB` VRAM hosts).
- **Multi-Stage Tool-Call Extraction & Non-Oracle Prompt-Repair:** Instruct single-tool-per-turn output in the system prompt (or parse sequential discovery calls, preventing `7B`'s `5` multi-JSON `{` parse failures). Extract calls from native `message.tool_calls` first, then fall back deterministically to `message.content` (`<tool_call>...</tool_call>`, markdown-fenced JSON, raw JSON) with string-literal control-character sanitization (`\t`, `\n`, `\r` — which lifted `qwen2.5-coder:14b` parse accuracy from `91.7%` to `100.0%` across `36` turns), followed by at most `1`–`N` structured prompt-repair turns returning exact parse/test errors without injecting oracle target files (requiring `read_file(path)` first if the target file has not been read).
- **Scoped Local vs. Cloud Operational Boundary:** Based on Spike-003 measurements on an `8 GiB` `RTX 4060 Laptop GPU` (`qwen2.5-coder:7b` at `50.0 tok/s` full VRAM; `14b` at `7.8 tok/s` with `20/49` layers spilled to CPU at `num_ctx=8192`), local `14B` models using `search_replace` (`5/5` clean `git apply --check`, `3/5` passing `go test` vs. `0/5` for raw `apply_patch`) are scoped strictly to localized single-file fixes within `~2k–6k` prompt tokens (`8k`–`16k` `num_ctx`), whereas multi-file or broad-context issues require cloud models via the same `LLMClient` interface.
- **Rejected Alternatives:**
  - *Option A (Naive OpenAI Client Only):* Fails `0/64` turns when `tool_calls` is empty on local `qwen2.5-coder` and silently truncates prompts (`7,312 -> 2,050` tokens) on `8 GiB` GPUs via `/v1/chat/completions`.
  - *Option C (Heavy Per-Vendor SDKs):* Duplicates retry, telemetry, and timeout handling across external vendor dependencies.

## Consequences

- **Positive:** Seamless switching between local Ollama and cloud APIs via `LLMClient` configuration, deterministic protection against Ollama silent truncation, `100.0%` tool-call recovery on `qwen2.5-coder:14b`, and clean `250ms`-verified HTTP cancellation (`context.WithTimeout`).
- **Negative / Trade-offs:**
  - Requires maintaining two small JSON payload mappers (`v1/chat/completions` and Ollama `/api/chat`) inside the `llm` package rather than one raw HTTP struct.
  - On `8 GiB` consumer GPUs, `14B` (`Q4_K_M`) spills `40%–60%` of weights/KV cache to host RAM (`7.8–9.0 tok/s` decode, `4.98s–5.24s` cold prefill, `32.4s` P95 turn latency at `8k` ctx), making local runs substantially slower and narrower in scope than cloud endpoints.
  - Cloud provider `v1/chat/completions` tool-calling accuracy and `Pass@1` remain design-adopted until Phase 4 mini SWE-bench evaluation ([benchmarks.md](../benchmarks.md)).

## References

- [Spike 003: Ollama Tool Calling](../spikes/003-ollama-tool-calling.md)
- [ADR-003: Tree-sitter AST Mapping & MCP Tools](./003-treesitter-and-mcp.md)
- [Benchmarks: Local vs. Cloud](../benchmarks.md)
