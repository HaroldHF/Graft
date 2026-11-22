# ADR-004: OpenAI-Compatible Interface for LLM Provider Abstraction

- **Status:** `Proposed`
- **Date:** `2026-11-21`
- **Related Spikes:** [Spike 003: Ollama Tool Calling](../spikes/003-ollama-tool-calling.md)
- **Related Docs:** [RFC-001: Graft Architecture](../architecture.md), [Benchmarks](../benchmarks.md)

## Context

Graft must run completely locally during development and privacy-sensitive workflows (via Ollama) while also supporting frontier cloud LLMs for comparison in the mini SWE-bench suite and higher-complexity production issues. Coupling the Agent Runtime to vendor-specific SDKs would duplicate tool-calling, retry, and telemetry logic across providers.

## Options Considered

1. **Option A (`Standard OpenAI-Compatible v1/chat/completions Client`):** Implement one configurable Go client speaking the OpenAI `v1/chat/completions` HTTP schema (base URL, API key, model name, tool definitions), compatible with both Ollama's OpenAI endpoint and cloud providers.
2. **Option B (`Per-Provider Native SDKs`):** Maintain separate adapter implementations for Ollama's native REST API, Anthropic SDK, OpenAI SDK, and Gemini SDK.

## Decision

We propose **Option A (`Standard OpenAI-Compatible v1/chat/completions Client`)** because:

- **Zero-Friction Provider Switching:** Switching between local Qwen 2.5 Coder models on Ollama and cloud APIs requires only configuration changes (endpoint URL, credentials, model tag), enabling apples-to-apples comparisons in [benchmarks.md](../benchmarks.md).
- **Minimal Adapter Surface:** Keeping one structured request/response pipeline simplifies token accounting, JSON tool-call parsing, and SSE event emission.

*Pending [Spike 003](../spikes/003-ollama-tool-calling.md):* We will validate whether local Ollama models (`Qwen 2.5 Coder 7B` vs. `14B`) emit reliable `tool_calls` payloads over `v1/chat/completions` or require fallback JSON prompt-parsing guardrails.

## Consequences

- **Positive:** Fast switching between local and cloud models, unified telemetry/token tracking, and minimal external SDK dependencies.
- **Negative / Trade-offs:** Provider-specific advanced features (e.g., vendor prompt-caching headers or native reasoning fields) are not automatically exposed without optional extensions.

## References

- [Spike 003: Ollama Tool Calling](../spikes/003-ollama-tool-calling.md)
- [Benchmarks: Local vs. Cloud](../benchmarks.md)
